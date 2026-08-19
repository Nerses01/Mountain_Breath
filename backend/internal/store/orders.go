package store

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

// CreateOrder turns the user's cart into an order — atomically. Everything
// happens in ONE transaction: either the whole checkout succeeds, or the
// database is left exactly as it was.
//
// `view` carries the two edge-negotiated facts the order is stamped with
// (F2 widened this from a bare Currency — the Parameter Object paying out
// exactly as its comment promised). The CURRENCY is what the customer is
// CHARGED in: a cart is a live thing that can be read in either market; an
// order is a fact about a transaction that happened in exactly one of them,
// which is why nothing below is dual and every number that follows is
// denominated in that one currency. The LOCALE is the language the
// checkout happened in, snapshotted for the same reason as the prices: a
// status email sent weeks later is triggered by the ADMIN's request, and
// the customer's language is a fact about the order, not about whoever
// pressed the button.
//
// E6 added `in` — the customer's CHOICES (address, method, note), already
// validated by the API layer. Note what the function still computes for
// itself: the subtotal from locked shelf prices, the shipping from the
// rates table, the totals from domain arithmetic. Nothing in `in` is money,
// so nothing a client sends can change what is charged.
func (s *Store) CreateOrder(ctx context.Context, userID int64, view domain.View, in domain.CheckoutInput) (domain.Order, error) {
	currency := view.EffectiveCurrency()
	locale := view.EffectiveLocale()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.Order{}, fmt.Errorf("beginning checkout tx: %w", err)
	}
	// Rollback after a successful Commit is a harmless no-op; this line
	// guarantees cleanup on every early return / panic path.
	defer func() { _ = tx.Rollback(ctx) }()

	// E7: lock the customer's own user row FIRST. This serializes two
	// parallel checkouts by the SAME user, which closes two races in one
	// statement: both counting zero prior orders (two free first
	// deliveries), and both redeeming a once-per-customer code (the unique
	// index would refuse the second anyway, but as a constraint explosion
	// mid-transaction rather than a clean wait-then-recount). Different
	// users' checkouts don't touch each other's row, so the parallelism
	// that matters — strangers buying the same jar — is untouched.
	//
	// It also EXTENDS the deadlock-avoidance ordering, which every new lock
	// must re-prove: user row → cart variants (ascending id) → promo row →
	// products (ascending id). Every checkout acquires in that sequence, so
	// no two can hold what the other wants.
	if _, err := tx.Exec(ctx,
		`SELECT FROM users WHERE id = $1 FOR UPDATE`, userID); err != nil {
		return domain.Order{}, fmt.Errorf("locking user row: %w", err)
	}

	// Lock the variant rows for this cart (FOR UPDATE OF v): concurrent
	// checkouts of the same variant must wait for us to finish, so stock
	// cannot be oversold. ORDER BY gives all transactions the same locking
	// order — the classic deadlock avoidance rule.
	//
	// The price arrives as a scalar SUBQUERY rather than a join, deliberately:
	// FOR UPDATE names `v`, and keeping the priced view out of the join tree
	// keeps the locking clause reading on exactly one table. It is also
	// LEFT-join-shaped by nature — it may legitimately return nothing.
	rows, err := tx.Query(ctx, `
		SELECT ci.variant_id, ci.qty, v.stock_qty, v.label, p.name, p.id,
		       p.is_cold_chain,
		       (SELECT ep.price_minor
		          FROM variant_effective_prices ep
		         WHERE ep.variant_id = v.id AND ep.currency = $2) AS price_minor
		FROM cart_items ci
		JOIN product_variants v ON v.id = ci.variant_id
		JOIN products p ON p.id = v.product_id
		WHERE ci.user_id = $1
		ORDER BY ci.variant_id
		FOR UPDATE OF v`,
		userID, currency)
	if err != nil {
		return domain.Order{}, fmt.Errorf("locking cart variants: %w", err)
	}

	type line struct {
		variantID   int64
		qty         int
		stockQty    int
		isColdChain bool
		// A POINTER, because the answer to "what does this cost in drams?"
		// can genuinely be "nothing on file". Browsing degrades over that;
		// checkout must not, so it becomes an error below rather than a zero.
		priceMinor *int64
		label      string
		name       string
		productID  int64
	}
	var lines []line
	for rows.Next() {
		var l line
		if err := rows.Scan(&l.variantID, &l.qty, &l.stockQty, &l.label, &l.name, &l.productID,
			&l.isColdChain, &l.priceMinor); err != nil {
			rows.Close()
			return domain.Order{}, fmt.Errorf("scanning cart line: %w", err)
		}
		lines = append(lines, l)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return domain.Order{}, fmt.Errorf("iterating cart lines: %w", err)
	}

	if len(lines) == 0 {
		return domain.Order{}, domain.ErrEmptyCart
	}

	// Validate stock under the lock — nobody can change it while we hold it.
	var subtotal int64
	var hasColdChain bool
	for _, l := range lines {
		if l.priceMinor == nil {
			return domain.Order{}, fmt.Errorf("%w: %s (%s) in %s",
				domain.ErrPriceUnavailable, l.name, l.label, currency)
		}
		if l.stockQty < l.qty {
			return domain.Order{}, fmt.Errorf("%w: %s (%s)", domain.ErrInsufficientStock, l.name, l.label)
		}
		subtotal += *l.priceMinor * int64(l.qty)
		hasColdChain = hasColdChain || l.isColdChain
	}

	// Shipping, read INSIDE the transaction like everything else the order
	// depends on. A market with no rate row cannot be charged shipping,
	// which means it cannot be charged at all — same refusal, same sentinel,
	// as a variant with no price.
	var rate domain.ShippingRate
	err = tx.QueryRow(ctx, `
		SELECT base_minor, cold_chain_surcharge_minor, free_over_minor
		FROM shipping_rates
		WHERE method = 'standard' AND currency = $1`, currency,
	).Scan(&rate.BaseMinor, &rate.ColdChainSurchargeMinor, &rate.FreeOverMinor)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Order{}, fmt.Errorf("%w: no shipping rate in %s",
				domain.ErrPriceUnavailable, currency)
		}
		return domain.Order{}, fmt.Errorf("reading shipping rate: %w", err)
	}

	// E7: the promo the cart is carrying, locked FOR UPDATE. The lock is
	// what turns max_redemptions from a hope into a rule — the count below
	// happens while no other checkout can insert a redemption for this code,
	// exactly the stock pattern one table over. FOR UPDATE OF p keeps the
	// lock off cart_promos, which nobody else is waiting on.
	var promo *domain.Promo
	{
		p, err := scanPromo(tx.QueryRow(ctx, `
			SELECT `+promoColumns+`
			FROM cart_promos cp
			JOIN promo_codes p ON p.id = cp.code_id
			WHERE cp.user_id = $1
			FOR UPDATE OF p`,
			userID))
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			// No code applied — the normal cart.
		case err != nil:
			return domain.Order{}, fmt.Errorf("locking cart promo: %w", err)
		default:
			if err := attachPromoValues(ctx, tx, &p); err != nil {
				return domain.Order{}, err
			}
			promo = &p
		}
	}

	// The hive-club fact, read inside the transaction (and safe from the
	// same-user race by the user-row lock above): how many non-cancelled
	// orders came before this one.
	var priorOrders int
	if err := tx.QueryRow(ctx, `
		SELECT count(*)::int FROM orders
		WHERE user_id = $1 AND status <> 'cancelled'`,
		userID).Scan(&priorOrders); err != nil {
		return domain.Order{}, fmt.Errorf("counting prior orders: %w", err)
	}

	// One calculator for every screen: the same domain.Price the cart page
	// and the checkout preview rendered from now decides what is charged.
	// If the code stopped being valid between apply and this moment
	// (expired, sold out, basket shrank below its floor), the checkout
	// REFUSES rather than silently charging a total the customer never saw.
	breakdown := domain.Price(domain.PriceInput{
		Currency:      currency,
		SubtotalMinor: subtotal,
		HasColdChain:  hasColdChain,
		Rate:          rate,
		PriorOrders:   priorOrders,
		Promo:         promo,
		Now:           time.Now(),
	})
	if promo != nil && breakdown.PromoIssue != "" {
		return domain.Order{}, fmt.Errorf("%w: %s (%s)",
			domain.ErrPromoInvalid, promo.Code, breakdown.PromoIssue)
	}
	totals := breakdown.OrderTotals

	// The rate on file at this instant, purely so the order stays reportable
	// later. Nothing above depends on it: the total is a sum of shelf prices.
	// No rows when the order is in the base currency — fx_rates forbids
	// base = quote — so absence needs no separate branch.
	var fxRate *string
	err = tx.QueryRow(ctx, `
		SELECT f.rate::text
		FROM fx_rates f
		JOIN currencies b ON b.code = f.base AND b.is_base
		WHERE f.quote = $1
		ORDER BY f.as_of DESC
		LIMIT 1`, currency).Scan(&fxRate)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return domain.Order{}, fmt.Errorf("reading fx rate: %w", err)
	}

	order := domain.Order{
		UserID: userID, Status: domain.OrderPending, TotalMinor: totals.TotalMinor,
		Currency: currency, Locale: locale, FxRateUsed: fxRate,
		ShipTo: &in.Address, DeliveryNote: in.DeliveryNote,
		LeaveWithNeighbour: in.LeaveWithNeighbour,
		Totals:             totals,
		PaymentMethod:      in.PaymentMethod,
		// Every method lands unpaid — card is a stub until Phase 11, a bank
		// transfer has not cleared, and cash has not been handed over. The
		// admin flips it; the column exists so that flip is a recorded fact.
		PaymentStatus: domain.PaymentUnpaid,
	}
	// The code's TEXT is snapshotted onto the order (NULL when none), the
	// same rule as product names in order_items: the receipt keeps saying
	// "HONEY10" whatever later happens to the code row.
	var promoCode *string
	if promo != nil {
		promoCode = &promo.Code
		order.PromoCode = promo.Code
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO orders (user_id, status, total_minor, currency, fx_rate_used,
		                    subtotal_minor, shipping_minor, discount_minor, tax_minor,
		                    member_discount_minor, promo_discount_minor, promo_code,
		                    payment_method, payment_status,
		                    ship_first_name, ship_last_name, ship_phone, ship_street,
		                    ship_city, ship_postal_code, ship_country,
		                    delivery_note, leave_with_neighbour, locale)
		VALUES ($1, $2, $3, $4, $5::numeric,
		        $6, $7, $8, $9, $10, $11, $12, $13, $14,
		        $15, $16, $17, $18, $19, $20, $21, $22, $23, $24)
		RETURNING id, created_at`,
		userID, order.Status, totals.TotalMinor, currency, fxRate,
		totals.SubtotalMinor, totals.ShippingMinor, totals.DiscountMinor, totals.TaxMinor,
		totals.MemberDiscountMinor, totals.PromoDiscountMinor, promoCode,
		order.PaymentMethod, order.PaymentStatus,
		in.Address.FirstName, in.Address.LastName, in.Address.Phone, in.Address.Street,
		in.Address.City, in.Address.PostalCode, in.Address.Country,
		in.DeliveryNote, in.LeaveWithNeighbour, locale,
	).Scan(&order.ID, &order.CreatedAt)
	if err != nil {
		return domain.Order{}, fmt.Errorf("inserting order: %w", err)
	}

	// A2 (log #85): the first history row — same transaction, and the SAME
	// timestamp the order row got, so "placed" and "created" can never
	// disagree by the microseconds between two now() calls.
	if _, err := tx.Exec(ctx, `
		INSERT INTO order_status_events (order_id, status, created_at)
		VALUES ($1, $2, $3)`,
		order.ID, domain.OrderPending, order.CreatedAt); err != nil {
		return domain.Order{}, fmt.Errorf("recording status event: %w", err)
	}
	order.Events = []domain.OrderEvent{{Status: domain.OrderPending, CreatedAt: order.CreatedAt}}
	order.HasColdChain = hasColdChain

	// Record the redemption — the row the UNIQUE (code_id, user_id) index
	// guards. Under the user-row and promo-row locks this insert cannot
	// race, so a violation here means a bug in the locking, not a customer:
	// translated to the sentinel anyway, because a 500 would tell the
	// customer the shop broke when what happened is the code was spent.
	if promo != nil {
		if _, err := tx.Exec(ctx, `
			INSERT INTO promo_redemptions (code_id, user_id, order_id)
			VALUES ($1, $2, $3)`,
			promo.ID, userID, order.ID); err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
				return domain.Order{}, fmt.Errorf("%w: %s (%s)",
					domain.ErrPromoInvalid, promo.Code, domain.ValidationPromoUsed)
			}
			return domain.Order{}, fmt.Errorf("recording promo redemption: %w", err)
		}
	}

	// The address book, updated in the SAME transaction: if the order fails,
	// the book does not learn a half-checked-out address.
	if err := upsertDefaultAddress(ctx, tx, userID, in.Address, in.LeaveWithNeighbour); err != nil {
		return domain.Order{}, err
	}

	for _, l := range lines {
		var item domain.OrderItem
		err = tx.QueryRow(ctx, `
			INSERT INTO order_items (order_id, variant_id, name_snapshot, label_snapshot, price_minor_snapshot, qty)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id`,
			order.ID, l.variantID, l.name, l.label, l.priceMinor, l.qty,
		).Scan(&item.ID)
		if err != nil {
			return domain.Order{}, fmt.Errorf("inserting order item: %w", err)
		}
		item.VariantID = l.variantID
		item.Name = l.name
		item.Label = l.label
		item.PriceMinor = *l.priceMinor // nil was rejected above
		item.Qty = l.qty
		order.Items = append(order.Items, item)

		if _, err := tx.Exec(ctx, `
			UPDATE product_variants SET stock_qty = stock_qty - $1 WHERE id = $2`,
			l.qty, l.variantID); err != nil {
			return domain.Order{}, fmt.Errorf("decrementing stock: %w", err)
		}
	}

	// Maintain the denormalized popularity counter (migration 000010) — the
	// signal behind the Shop page's "Most loved" sort. Free to do here: the
	// transaction and the row locks are already open, so the counter can
	// never disagree with the order that moved it.
	//
	// Written as one UPDATE per PRODUCT, in ascending id order, for the same
	// reason the cart is locked ORDER BY variant_id. UPDATE takes a row lock,
	// so two checkouts touching the same two products in opposite orders
	// would deadlock; every transaction taking product locks in ascending id
	// order makes that impossible. Quantities are summed per product first,
	// because one cart can hold two variants of the same jar.
	sold := make(map[int64]int, len(lines))
	for _, l := range lines {
		sold[l.productID] += l.qty
	}
	productIDs := make([]int64, 0, len(sold))
	for id := range sold {
		productIDs = append(productIDs, id)
	}
	// Map iteration order in Go is deliberately RANDOMISED, so this sort is
	// not tidiness — without it the lock order would differ run to run.
	slices.Sort(productIDs)
	for _, id := range productIDs {
		if _, err := tx.Exec(ctx,
			`UPDATE products SET sales_count = sales_count + $1 WHERE id = $2`,
			sold[id], id); err != nil {
			return domain.Order{}, fmt.Errorf("incrementing sales count: %w", err)
		}
	}

	if _, err := tx.Exec(ctx,
		`DELETE FROM cart_items WHERE user_id = $1`, userID); err != nil {
		return domain.Order{}, fmt.Errorf("clearing cart: %w", err)
	}
	// The applied code goes with the cart it was applied to — a redeemed
	// code left attached would greet the NEXT basket with "already used".
	if _, err := tx.Exec(ctx,
		`DELETE FROM cart_promos WHERE user_id = $1`, userID); err != nil {
		return domain.Order{}, fmt.Errorf("clearing cart promo: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Order{}, fmt.Errorf("committing checkout: %w", err)
	}
	return order, nil
}

// orderColumns is the one list every order read shares — E4's "same
// predicate by construction" rule applied to a SELECT list, after E6 grew it
// to twenty-odd columns and three hand-copied variants stopped being
// reviewable.
//
// Every column is QUALIFIED with the table name, and that is a bug fix, not
// style: the admin listing joins users, whose own `id` made a bare `id`
// ambiguous — Postgres 42702 on every /admin/orders request. Aliasing the
// OTHER table does not help (the alias renames it, it does not remove its
// columns from the namespace); only qualifying these does. Qualified names
// cost nothing in the single-table queries, so the shared constant carries
// them everywhere. Found in the RUNNING shop, not by the suite — the fake
// store cannot mis-parse SQL, and no integration test called ListAllOrders;
// TestListAllOrders now does.
const orderColumns = `orders.id, orders.user_id, orders.status,
	orders.total_minor, orders.created_at,
	orders.currency, orders.fx_rate_used::text,
	orders.subtotal_minor, orders.shipping_minor, orders.discount_minor,
	orders.tax_minor, orders.payment_method, orders.payment_status,
	orders.ship_first_name, orders.ship_last_name, orders.ship_phone,
	orders.ship_street, orders.ship_city, orders.ship_postal_code,
	orders.ship_country, orders.delivery_note, orders.leave_with_neighbour,
	orders.member_discount_minor, orders.promo_discount_minor,
	COALESCE(orders.promo_code, ''), orders.locale`

// scanOrder reads one row of orderColumns (plus whatever the caller
// appended) into a domain.Order. The ship_* columns are nullable — orders
// made before E6 genuinely had no address — so they scan into pointers and
// only a complete row becomes a ShipTo.
func scanOrder(row pgx.Row, extra ...any) (domain.Order, error) {
	var o domain.Order
	var first, last, phone, street, city, postal, country *string
	dest := []any{&o.ID, &o.UserID, &o.Status, &o.TotalMinor, &o.CreatedAt,
		&o.Currency, &o.FxRateUsed,
		&o.Totals.SubtotalMinor, &o.Totals.ShippingMinor,
		&o.Totals.DiscountMinor, &o.Totals.TaxMinor,
		&o.PaymentMethod, &o.PaymentStatus,
		&first, &last, &phone, &street, &city, &postal, &country,
		&o.DeliveryNote, &o.LeaveWithNeighbour,
		&o.Totals.MemberDiscountMinor, &o.Totals.PromoDiscountMinor,
		&o.PromoCode, &o.Locale}
	if err := row.Scan(append(dest, extra...)...); err != nil {
		return domain.Order{}, err
	}
	o.Totals.TotalMinor = o.TotalMinor
	if street != nil {
		o.ShipTo = &domain.Address{
			FirstName: *first, LastName: *last, Phone: *phone,
			Street: *street, City: *city, PostalCode: *postal, Country: *country,
		}
	}
	return o, nil
}

func (s *Store) ListOrdersByUser(ctx context.Context, userID int64) ([]domain.Order, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT `+orderColumns+`
		FROM orders
		WHERE user_id = $1
		ORDER BY created_at DESC`,
		userID)
	if err != nil {
		return nil, fmt.Errorf("querying orders: %w", err)
	}
	defer rows.Close()

	orders, err := scanOrders(rows)
	if err != nil {
		return nil, err
	}
	if err := s.attachOrderItems(ctx, orders); err != nil {
		return nil, err
	}
	if err := s.attachOrderEvents(ctx, orders); err != nil {
		return nil, err
	}
	return orders, nil
}

// ListAllOrders is the admin view: every order, newest first, with the
// customer's email joined in.
func (s *Store) ListAllOrders(ctx context.Context) ([]domain.Order, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT `+orderColumns+`, u.email
		FROM orders
		JOIN users u ON u.id = orders.user_id
		ORDER BY orders.created_at DESC
		LIMIT 200`)
	if err != nil {
		return nil, fmt.Errorf("querying all orders: %w", err)
	}
	defer rows.Close()

	orders := make([]domain.Order, 0)
	for rows.Next() {
		var email string
		o, err := scanOrder(rows, &email)
		if err != nil {
			return nil, fmt.Errorf("scanning order row: %w", err)
		}
		o.UserEmail = email
		orders = append(orders, o)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating order rows: %w", err)
	}
	if err := s.attachOrderItems(ctx, orders); err != nil {
		return nil, err
	}
	if err := s.attachOrderEvents(ctx, orders); err != nil {
		return nil, err
	}
	return orders, nil
}

// UpdateOrderStatus applies a state-machine transition. Cancelling an order
// returns its items to stock — in the same transaction.
func (s *Store) UpdateOrderStatus(ctx context.Context, orderID int64, to string) (domain.Order, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.Order{}, fmt.Errorf("beginning status tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	o, err := scanOrder(tx.QueryRow(ctx, `
		SELECT `+orderColumns+`
		FROM orders WHERE id = $1
		FOR UPDATE`,
		orderID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Order{}, domain.ErrNotFound
		}
		return domain.Order{}, fmt.Errorf("locking order: %w", err)
	}

	if !domain.ValidOrderTransition(o.Status, to) {
		return domain.Order{}, fmt.Errorf("%w: %s → %s", domain.ErrInvalidTransition, o.Status, to)
	}

	if err := applyOrderStatusTx(ctx, tx, orderID, to); err != nil {
		return domain.Order{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Order{}, fmt.Errorf("committing status change: %w", err)
	}
	o.Status = to
	return o, nil
}

// applyOrderStatusTx is the WRITE half of a status change, shared by the
// admin's UpdateOrderStatus and the customer's CancelOrderByCustomer (F2)
// so the two doors can never drift: whoever opened the transaction has
// already locked the row and decided the transition is allowed; this
// applies it and its side effects, and the caller commits.
func applyOrderStatusTx(ctx context.Context, tx pgx.Tx, orderID int64, to string) error {
	if _, err := tx.Exec(ctx,
		`UPDATE orders SET status = $1 WHERE id = $2`, to, orderID); err != nil {
		return fmt.Errorf("updating status: %w", err)
	}

	// A2 (log #85): the transition becomes a recorded fact, in the same
	// transaction as the update it describes — the history cannot commit
	// without the state change, nor the state change without its history.
	if _, err := tx.Exec(ctx, `
		INSERT INTO order_status_events (order_id, status)
		VALUES ($1, $2)`, orderID, to); err != nil {
		return fmt.Errorf("recording status event: %w", err)
	}

	if to == domain.OrderCancelled {
		// Give the reserved stock back.
		if _, err := tx.Exec(ctx, `
			UPDATE product_variants v
			SET stock_qty = v.stock_qty + oi.qty
			FROM order_items oi
			WHERE oi.order_id = $1 AND v.id = oi.variant_id`,
			orderID); err != nil {
			return fmt.Errorf("restoring stock: %w", err)
		}
		// ...and the promo code, the same way: cancelling undoes the order's
		// side effects, and a redemption is one of them. The order keeps its
		// promo_code SNAPSHOT (what happened is still what happened); only
		// the redemption row — the thing that blocks reuse and fills the
		// global cap — is released. No-op for promo-less orders.
		if _, err := tx.Exec(ctx, `
			DELETE FROM promo_redemptions WHERE order_id = $1`,
			orderID); err != nil {
			return fmt.Errorf("releasing promo redemption: %w", err)
		}
	}
	return nil
}

// CancelOrderByCustomer is the customer's one self-service transition
// (F2). Ownership is checked HERE, like Reorder's, because the call
// WRITES — a stranger's order id answers ErrNotFound, the same
// existence-hiding as the order page. The window is the DOMAIN's rule
// (CustomerMayCancelOrder: pending only — narrower than the machine,
// whose confirmed → cancelled arrow belongs to the admin), and it is
// checked under the same FOR UPDATE lock that the flip uses: without it,
// an admin confirming at the same moment could slip a confirmation
// between the customer's check and their cancel.
func (s *Store) CancelOrderByCustomer(ctx context.Context, userID, orderID int64) (domain.Order, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.Order{}, fmt.Errorf("beginning cancel tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	o, err := scanOrder(tx.QueryRow(ctx, `
		SELECT `+orderColumns+`
		FROM orders WHERE id = $1
		FOR UPDATE`,
		orderID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Order{}, domain.ErrNotFound
		}
		return domain.Order{}, fmt.Errorf("locking order: %w", err)
	}
	if o.UserID != userID {
		return domain.Order{}, domain.ErrNotFound
	}

	if !domain.CustomerMayCancelOrder(o.Status) {
		return domain.Order{}, domain.ErrTooLateToCancel
	}

	if err := applyOrderStatusTx(ctx, tx, orderID, domain.OrderCancelled); err != nil {
		return domain.Order{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Order{}, fmt.Errorf("committing cancel: %w", err)
	}
	o.Status = domain.OrderCancelled
	return o, nil
}

// UpdateOrderPaymentStatus flips the OTHER state machine (F2): whether
// money has arrived, orthogonal to where the parcel is. Same skeleton as
// UpdateOrderStatus — lock, validate in the domain, write — but with no
// side effects: nothing to restock, and no event row, because the column
// itself is the recorded fact E6 modelled ("the admin flips it; the column
// exists so that flip is a recorded fact").
//
// The read-then-write NEEDS the transaction + FOR UPDATE even though it is
// two tiny statements: without the lock, two admins clicking "mark paid"
// and "mark refunded" at once could both read `unpaid`, both pass
// validation, and the losing write would land a refund on money the record
// says never arrived. Same reasoning as the checkout's stock decrement,
// at one row's scale.
func (s *Store) UpdateOrderPaymentStatus(ctx context.Context, orderID int64, to string) (domain.Order, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.Order{}, fmt.Errorf("beginning payment tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	o, err := scanOrder(tx.QueryRow(ctx, `
		SELECT `+orderColumns+`
		FROM orders WHERE id = $1
		FOR UPDATE`,
		orderID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Order{}, domain.ErrNotFound
		}
		return domain.Order{}, fmt.Errorf("locking order: %w", err)
	}

	if !domain.ValidPaymentTransition(o.PaymentStatus, to) {
		return domain.Order{}, fmt.Errorf("%w: payment %s → %s", domain.ErrInvalidTransition, o.PaymentStatus, to)
	}

	if _, err := tx.Exec(ctx,
		`UPDATE orders SET payment_status = $1 WHERE id = $2`, to, orderID); err != nil {
		return domain.Order{}, fmt.Errorf("updating payment status: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Order{}, fmt.Errorf("committing payment change: %w", err)
	}
	o.PaymentStatus = to
	return o, nil
}

// Reorder merges an order's lines back into the cart (A2, decision log
// #86): one transaction, and PARTIAL SUCCESS is the designed outcome, not
// an error — an order from last month meets today's stock, and the result
// says line by line what happened. The interesting choices:
//
//   - Ownership is checked HERE (unlike GetOrder, which leaves it to the
//     handler): reorder is a WRITE to the caller's cart, so "may you" and
//     "do it" cannot be separated without a gap between them. A wrong
//     owner gets ErrNotFound — same existence-hiding as the order page.
//   - Quantities MERGE (cart qty + order qty, capped): a customer with 1
//     jar already in the cart reordering 2 wants 3, not "2 replaces 1".
//     The cap is the variant's stock and the cart's own 99-per-line rule.
//   - No FOR UPDATE: the cart never guarantees availability — checkout
//     re-validates under locks. The stock read here only shapes an honest
//     REPORT, so locking would buy nothing but contention.
func (s *Store) Reorder(ctx context.Context, userID, orderID int64) (domain.ReorderResult, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.ReorderResult{}, fmt.Errorf("beginning reorder tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var owner int64
	err = tx.QueryRow(ctx, `SELECT user_id FROM orders WHERE id = $1`, orderID).Scan(&owner)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ReorderResult{}, domain.ErrNotFound
		}
		return domain.ReorderResult{}, fmt.Errorf("reading order owner: %w", err)
	}
	if owner != userID {
		return domain.ReorderResult{}, domain.ErrNotFound
	}

	rows, err := tx.Query(ctx, `
		SELECT oi.name_snapshot, oi.label_snapshot, oi.qty, oi.variant_id,
		       v.stock_qty, p.is_active, COALESCE(ci.qty, 0)
		FROM order_items oi
		JOIN product_variants v ON v.id = oi.variant_id
		JOIN products p ON p.id = v.product_id
		LEFT JOIN cart_items ci ON ci.user_id = $2 AND ci.variant_id = oi.variant_id
		WHERE oi.order_id = $1
		ORDER BY oi.id`,
		orderID, userID)
	if err != nil {
		return domain.ReorderResult{}, fmt.Errorf("reading order lines: %w", err)
	}

	type line struct {
		name, label string
		qty         int
		variantID   int64
		stockQty    int
		isActive    bool
		inCart      int
	}
	var lines []line
	for rows.Next() {
		var l line
		if err := rows.Scan(&l.name, &l.label, &l.qty, &l.variantID,
			&l.stockQty, &l.isActive, &l.inCart); err != nil {
			rows.Close()
			return domain.ReorderResult{}, fmt.Errorf("scanning order line: %w", err)
		}
		lines = append(lines, l)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return domain.ReorderResult{}, fmt.Errorf("iterating order lines: %w", err)
	}

	// maxCartLine mirrors the API's qty validation (1–99 on PUT /cart/items):
	// reorder must not build a cart the customer could not have built by hand.
	const maxCartLine = 99

	var result domain.ReorderResult
	for _, l := range lines {
		out := domain.ReorderLine{Name: l.name, Label: l.label}
		room := min(l.stockQty, maxCartLine) - l.inCart
		switch {
		case !l.isActive:
			out.Issue = domain.ReorderUnavailable
		case room <= 0:
			out.Issue = domain.ReorderOutOfStock
		default:
			add := min(l.qty, room)
			if _, err := tx.Exec(ctx, `
				INSERT INTO cart_items (user_id, variant_id, qty)
				VALUES ($1, $2, $3)
				ON CONFLICT (user_id, variant_id)
				DO UPDATE SET qty = cart_items.qty + EXCLUDED.qty`,
				userID, l.variantID, add); err != nil {
				return domain.ReorderResult{}, fmt.Errorf("merging cart line: %w", err)
			}
			out.Qty = add
			if add < l.qty {
				out.Issue = domain.ReorderReduced
			}
		}
		result.Lines = append(result.Lines, out)
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.ReorderResult{}, fmt.Errorf("committing reorder: %w", err)
	}
	return result, nil
}

func scanOrders(rows pgx.Rows) ([]domain.Order, error) {
	orders := make([]domain.Order, 0)
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning order row: %w", err)
		}
		orders = append(orders, o)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating order rows: %w", err)
	}
	return orders, nil
}

// GetOrder loads one order with its items — the confirmation page's read.
// It does NOT filter by user: ownership is the API layer's question (the
// admin may see any order), and answering it here would force two methods
// for one query.
func (s *Store) GetOrder(ctx context.Context, orderID int64) (domain.Order, error) {
	o, err := scanOrder(s.pool.QueryRow(ctx, `
		SELECT `+orderColumns+`
		FROM orders WHERE id = $1`, orderID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Order{}, domain.ErrNotFound
		}
		return domain.Order{}, fmt.Errorf("querying order %d: %w", orderID, err)
	}

	orders := []domain.Order{o}
	if err := s.attachOrderItems(ctx, orders); err != nil {
		return domain.Order{}, err
	}
	if err := s.attachOrderEvents(ctx, orders); err != nil {
		return domain.Order{}, err
	}
	return orders[0], nil
}

// attachOrderItems batch-loads items for all orders in one query (same
// N+1 avoidance as product variants).
func (s *Store) attachOrderItems(ctx context.Context, orders []domain.Order) error {
	if len(orders) == 0 {
		return nil
	}

	ids := make([]int64, len(orders))
	byID := make(map[int64]*domain.Order, len(orders))
	for i := range orders {
		ids[i] = orders[i].ID
		byID[orders[i].ID] = &orders[i]
		orders[i].Items = make([]domain.OrderItem, 0)
	}

	// A2: the joins bring the LIVE cold-chain flag along — has_cold_chain
	// labels the parcel's handling today, it is not part of the frozen
	// financial record, so it reads like the cart's flag does.
	rows, err := s.pool.Query(ctx, `
		SELECT oi.id, oi.order_id, oi.variant_id, oi.name_snapshot,
		       oi.label_snapshot, oi.price_minor_snapshot, oi.qty,
		       p.is_cold_chain
		FROM order_items oi
		JOIN product_variants v ON v.id = oi.variant_id
		JOIN products p ON p.id = v.product_id
		WHERE oi.order_id = ANY($1)
		ORDER BY oi.id`,
		ids)
	if err != nil {
		return fmt.Errorf("querying order items: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var orderID int64
		var it domain.OrderItem
		var coldChain bool
		if err := rows.Scan(&it.ID, &orderID, &it.VariantID, &it.Name, &it.Label, &it.PriceMinor, &it.Qty, &coldChain); err != nil {
			return fmt.Errorf("scanning order item: %w", err)
		}
		o := byID[orderID]
		o.Items = append(o.Items, it)
		o.HasColdChain = o.HasColdChain || coldChain
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterating order items: %w", err)
	}
	return nil
}

// attachOrderEvents batch-loads each order's recorded timeline (A2), the
// same one-query shape as attachOrderItems.
func (s *Store) attachOrderEvents(ctx context.Context, orders []domain.Order) error {
	if len(orders) == 0 {
		return nil
	}

	ids := make([]int64, len(orders))
	byID := make(map[int64]*domain.Order, len(orders))
	for i := range orders {
		ids[i] = orders[i].ID
		byID[orders[i].ID] = &orders[i]
		orders[i].Events = make([]domain.OrderEvent, 0)
	}

	rows, err := s.pool.Query(ctx, `
		SELECT order_id, status, created_at
		FROM order_status_events
		WHERE order_id = ANY($1)
		ORDER BY created_at, id`,
		ids)
	if err != nil {
		return fmt.Errorf("querying order events: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var orderID int64
		var ev domain.OrderEvent
		if err := rows.Scan(&orderID, &ev.Status, &ev.CreatedAt); err != nil {
			return fmt.Errorf("scanning order event: %w", err)
		}
		o := byID[orderID]
		o.Events = append(o.Events, ev)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterating order events: %w", err)
	}
	return nil
}

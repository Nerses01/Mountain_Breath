package store

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/jackc/pgx/v5"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

// CreateOrder turns the user's cart into an order — atomically. Everything
// happens in ONE transaction: either the whole checkout succeeds, or the
// database is left exactly as it was.
//
// `currency` is what the customer is CHARGED in, and the order is stamped
// with it. A cart is a live thing that can be read in either market; an
// order is a fact about a transaction that happened in exactly one of them,
// which is why nothing below is dual and every number that follows is
// denominated in this one currency.
func (s *Store) CreateOrder(ctx context.Context, userID int64, currency domain.Currency) (domain.Order, error) {
	if currency == "" {
		currency = domain.DefaultCurrency
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.Order{}, fmt.Errorf("beginning checkout tx: %w", err)
	}
	// Rollback after a successful Commit is a harmless no-op; this line
	// guarantees cleanup on every early return / panic path.
	defer func() { _ = tx.Rollback(ctx) }()

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
		variantID int64
		qty       int
		stockQty  int
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
		if err := rows.Scan(&l.variantID, &l.qty, &l.stockQty, &l.label, &l.name, &l.productID, &l.priceMinor); err != nil {
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
	var total int64
	for _, l := range lines {
		if l.priceMinor == nil {
			return domain.Order{}, fmt.Errorf("%w: %s (%s) in %s",
				domain.ErrPriceUnavailable, l.name, l.label, currency)
		}
		if l.stockQty < l.qty {
			return domain.Order{}, fmt.Errorf("%w: %s (%s)", domain.ErrInsufficientStock, l.name, l.label)
		}
		total += *l.priceMinor * int64(l.qty)
	}

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
		UserID: userID, Status: domain.OrderPending, TotalMinor: total,
		Currency: currency, FxRateUsed: fxRate,
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO orders (user_id, status, total_minor, currency, fx_rate_used)
		VALUES ($1, $2, $3, $4, $5::numeric)
		RETURNING id, created_at`,
		userID, order.Status, total, currency, fxRate,
	).Scan(&order.ID, &order.CreatedAt)
	if err != nil {
		return domain.Order{}, fmt.Errorf("inserting order: %w", err)
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

	if err := tx.Commit(ctx); err != nil {
		return domain.Order{}, fmt.Errorf("committing checkout: %w", err)
	}
	return order, nil
}

func (s *Store) ListOrdersByUser(ctx context.Context, userID int64) ([]domain.Order, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, user_id, status, total_minor, created_at, currency, fx_rate_used::text
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
	return orders, nil
}

// ListAllOrders is the admin view: every order, newest first, with the
// customer's email joined in.
func (s *Store) ListAllOrders(ctx context.Context) ([]domain.Order, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT o.id, o.user_id, o.status, o.total_minor, o.created_at,
		       o.currency, o.fx_rate_used::text, u.email
		FROM orders o
		JOIN users u ON u.id = o.user_id
		ORDER BY o.created_at DESC
		LIMIT 200`)
	if err != nil {
		return nil, fmt.Errorf("querying all orders: %w", err)
	}
	defer rows.Close()

	orders := make([]domain.Order, 0)
	for rows.Next() {
		var o domain.Order
		if err := rows.Scan(&o.ID, &o.UserID, &o.Status, &o.TotalMinor, &o.CreatedAt,
			&o.Currency, &o.FxRateUsed, &o.UserEmail); err != nil {
			return nil, fmt.Errorf("scanning order row: %w", err)
		}
		orders = append(orders, o)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating order rows: %w", err)
	}
	if err := s.attachOrderItems(ctx, orders); err != nil {
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

	var o domain.Order
	err = tx.QueryRow(ctx, `
		SELECT id, user_id, status, total_minor, created_at, currency, fx_rate_used::text
		FROM orders WHERE id = $1
		FOR UPDATE`,
		orderID,
	).Scan(&o.ID, &o.UserID, &o.Status, &o.TotalMinor, &o.CreatedAt, &o.Currency, &o.FxRateUsed)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Order{}, domain.ErrNotFound
		}
		return domain.Order{}, fmt.Errorf("locking order: %w", err)
	}

	if !domain.ValidOrderTransition(o.Status, to) {
		return domain.Order{}, fmt.Errorf("%w: %s → %s", domain.ErrInvalidTransition, o.Status, to)
	}

	if _, err := tx.Exec(ctx,
		`UPDATE orders SET status = $1 WHERE id = $2`, to, orderID); err != nil {
		return domain.Order{}, fmt.Errorf("updating status: %w", err)
	}

	if to == domain.OrderCancelled {
		// Give the reserved stock back.
		if _, err := tx.Exec(ctx, `
			UPDATE product_variants v
			SET stock_qty = v.stock_qty + oi.qty
			FROM order_items oi
			WHERE oi.order_id = $1 AND v.id = oi.variant_id`,
			orderID); err != nil {
			return domain.Order{}, fmt.Errorf("restoring stock: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Order{}, fmt.Errorf("committing status change: %w", err)
	}
	o.Status = to
	return o, nil
}

func scanOrders(rows pgx.Rows) ([]domain.Order, error) {
	orders := make([]domain.Order, 0)
	for rows.Next() {
		var o domain.Order
		if err := rows.Scan(&o.ID, &o.UserID, &o.Status, &o.TotalMinor, &o.CreatedAt,
			&o.Currency, &o.FxRateUsed); err != nil {
			return nil, fmt.Errorf("scanning order row: %w", err)
		}
		orders = append(orders, o)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating order rows: %w", err)
	}
	return orders, nil
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

	rows, err := s.pool.Query(ctx, `
		SELECT id, order_id, variant_id, name_snapshot, label_snapshot, price_minor_snapshot, qty
		FROM order_items
		WHERE order_id = ANY($1)
		ORDER BY id`,
		ids)
	if err != nil {
		return fmt.Errorf("querying order items: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var orderID int64
		var it domain.OrderItem
		if err := rows.Scan(&it.ID, &orderID, &it.VariantID, &it.Name, &it.Label, &it.PriceMinor, &it.Qty); err != nil {
			return fmt.Errorf("scanning order item: %w", err)
		}
		o := byID[orderID]
		o.Items = append(o.Items, it)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterating order items: %w", err)
	}
	return nil
}

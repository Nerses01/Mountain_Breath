package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

// CreateOrder turns the user's cart into an order — atomically. Everything
// happens in ONE transaction: either the whole checkout succeeds, or the
// database is left exactly as it was.
func (s *Store) CreateOrder(ctx context.Context, userID int64) (domain.Order, error) {
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
	rows, err := tx.Query(ctx, `
		SELECT ci.variant_id, ci.qty, v.stock_qty, v.price_minor, v.label, p.name
		FROM cart_items ci
		JOIN product_variants v ON v.id = ci.variant_id
		JOIN products p ON p.id = v.product_id
		WHERE ci.user_id = $1
		ORDER BY ci.variant_id
		FOR UPDATE OF v`,
		userID)
	if err != nil {
		return domain.Order{}, fmt.Errorf("locking cart variants: %w", err)
	}

	type line struct {
		variantID  int64
		qty        int
		stockQty   int
		priceMinor int64
		label      string
		name       string
	}
	var lines []line
	for rows.Next() {
		var l line
		if err := rows.Scan(&l.variantID, &l.qty, &l.stockQty, &l.priceMinor, &l.label, &l.name); err != nil {
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
		if l.stockQty < l.qty {
			return domain.Order{}, fmt.Errorf("%w: %s (%s)", domain.ErrInsufficientStock, l.name, l.label)
		}
		total += l.priceMinor * int64(l.qty)
	}

	order := domain.Order{UserID: userID, Status: domain.OrderPending, TotalMinor: total}
	err = tx.QueryRow(ctx, `
		INSERT INTO orders (user_id, status, total_minor)
		VALUES ($1, $2, $3)
		RETURNING id, created_at`,
		userID, order.Status, total,
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
		item.PriceMinor = l.priceMinor
		item.Qty = l.qty
		order.Items = append(order.Items, item)

		if _, err := tx.Exec(ctx, `
			UPDATE product_variants SET stock_qty = stock_qty - $1 WHERE id = $2`,
			l.qty, l.variantID); err != nil {
			return domain.Order{}, fmt.Errorf("decrementing stock: %w", err)
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
		SELECT id, user_id, status, total_minor, created_at
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
		SELECT o.id, o.user_id, o.status, o.total_minor, o.created_at, u.email
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
		if err := rows.Scan(&o.ID, &o.UserID, &o.Status, &o.TotalMinor, &o.CreatedAt, &o.UserEmail); err != nil {
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
		SELECT id, user_id, status, total_minor, created_at
		FROM orders WHERE id = $1
		FOR UPDATE`,
		orderID,
	).Scan(&o.ID, &o.UserID, &o.Status, &o.TotalMinor, &o.CreatedAt)
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
		if err := rows.Scan(&o.ID, &o.UserID, &o.Status, &o.TotalMinor, &o.CreatedAt); err != nil {
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

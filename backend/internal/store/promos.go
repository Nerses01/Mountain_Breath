package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

// querier is the one method attachPromoValues needs, declared at the
// consumer exactly like the API layer's store interfaces: *pgxpool.Pool and
// pgx.Tx both satisfy it implicitly, so the same helper serves a plain read
// (apply, preview) and the checkout transaction without knowing which it is
// on. The C++ equivalent would be a template parameter or an abstract base;
// Go's structural interfaces make it one line neither caller declares.
type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// The SELECT list every promo read shares (the orderColumns idiom). The two
// subqueries fill the LIVE counts domain.Promo carries so that Issue() can
// stay a pure function: how many times the code has been redeemed in total,
// and whether the asking shopper has redeemed it. $1 is always the user id
// — first, because this fragment needs it in every query it appears in,
// while whatever the outer WHERE matches on comes after.
//
// Reading counts this way — without a lock — is fine for apply and preview,
// where the answer is advisory (the worst case is a code that sells out a
// moment later). The one place the count DECIDES something irreversible is
// CreateOrder, which re-reads it under FOR UPDATE of the promo row.
const promoColumns = `p.id, p.code, p.kind, COALESCE(p.percent, 0),
	p.starts_at, p.ends_at, p.max_redemptions, p.active,
	(SELECT count(*) FROM promo_redemptions r WHERE r.code_id = p.id)::int,
	EXISTS (SELECT 1 FROM promo_redemptions r
	        WHERE r.code_id = p.id AND r.user_id = $1)`

func scanPromo(row pgx.Row) (domain.Promo, error) {
	var p domain.Promo
	err := row.Scan(&p.ID, &p.Code, &p.Kind, &p.Percent,
		&p.StartsAt, &p.EndsAt, &p.MaxRedemptions, &p.Active,
		&p.Redemptions, &p.UsedByShopper)
	return p, err
}

// attachPromoValues loads the per-market money rows (promo_code_values)
// into the Promo. Querier rather than *Store so CreateOrder can call it on
// its open transaction.
func attachPromoValues(ctx context.Context, q querier, p *domain.Promo) error {
	rows, err := q.Query(ctx, `
		SELECT currency, amount_minor, min_subtotal_minor
		FROM promo_code_values
		WHERE code_id = $1`, p.ID)
	if err != nil {
		return fmt.Errorf("querying promo values: %w", err)
	}
	defer rows.Close()

	p.Values = make(map[domain.Currency]domain.PromoValue)
	for rows.Next() {
		var currency string
		var v domain.PromoValue
		if err := rows.Scan(&currency, &v.AmountMinor, &v.MinSubtotalMinor); err != nil {
			return fmt.Errorf("scanning promo value: %w", err)
		}
		p.Values[domain.Currency(currency)] = v
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterating promo values: %w", err)
	}
	return nil
}

// PromoForUser resolves a typed code for a specific shopper — the apply
// endpoint's read. The code is normalized here AND matched against the
// upper() expression index, so "honey10" finds HONEY10 without a sequential
// scan. ErrNotFound covers a code that simply does not exist; every other
// reason a code cannot apply is domain.Promo.Issue's verdict, not an error.
func (s *Store) PromoForUser(ctx context.Context, code string, userID int64) (domain.Promo, error) {
	p, err := scanPromo(s.pool.QueryRow(ctx, `
		SELECT `+promoColumns+`
		FROM promo_codes p
		WHERE upper(p.code) = $2`,
		userID, domain.NormalizePromoCode(code)))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Promo{}, domain.ErrNotFound
		}
		return domain.Promo{}, fmt.Errorf("querying promo code: %w", err)
	}
	if err := attachPromoValues(ctx, s.pool, &p); err != nil {
		return domain.Promo{}, err
	}
	return p, nil
}

// CartPromoForUser is the code this user's cart is carrying, or nil — nil
// rather than ErrNotFound, because "no promo applied" is the normal state
// of most carts, not a failure to find something that should exist.
func (s *Store) CartPromoForUser(ctx context.Context, userID int64) (*domain.Promo, error) {
	p, err := scanPromo(s.pool.QueryRow(ctx, `
		SELECT `+promoColumns+`
		FROM cart_promos cp
		JOIN promo_codes p ON p.id = cp.code_id
		WHERE cp.user_id = $1`,
		userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("querying cart promo: %w", err)
	}
	if err := attachPromoValues(ctx, s.pool, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// SetCartPromo remembers the cart's code — one per user, so applying a
// second code replaces the first (the design draws one promo box, not a
// list). The same no-read-then-write upsert as the default address.
func (s *Store) SetCartPromo(ctx context.Context, userID, codeID int64) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO cart_promos (user_id, code_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id) DO UPDATE SET code_id = EXCLUDED.code_id, applied_at = now()`,
		userID, codeID)
	if err != nil {
		return fmt.Errorf("setting cart promo: %w", err)
	}
	return nil
}

func (s *Store) ClearCartPromo(ctx context.Context, userID int64) error {
	if _, err := s.pool.Exec(ctx,
		`DELETE FROM cart_promos WHERE user_id = $1`, userID); err != nil {
		return fmt.Errorf("clearing cart promo: %w", err)
	}
	return nil
}

// PriorOrders is the hive-club fact: how many non-cancelled orders this
// customer already has. 0 = first delivery free; ≥1 = member pricing.
// Cancelled orders do not count — an order that was undone must not burn
// the customer's free first delivery.
func (s *Store) PriorOrders(ctx context.Context, userID int64) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `
		SELECT count(*)::int FROM orders
		WHERE user_id = $1 AND status <> 'cancelled'`,
		userID).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("counting prior orders: %w", err)
	}
	return n, nil
}

// UpsellForGap answers the free-shipping banner's call to action: the
// cheapest product that would, on its own, close the remaining gap to the
// threshold — the design's "Add pollen · $16" on an "$8 away" cart. The
// cheapest QUALIFYING product, not the cheapest product: a $2 item on an
// $8 gap would leave the bar unfilled and the promise broken.
//
// nil when nothing qualifies (a gap larger than every price on the shelf) —
// the banner then shows the message without a button, which is still true.
func (s *Store) UpsellForGap(ctx context.Context, view domain.View, gapMinor int64) (*domain.Upsell, error) {
	var u domain.Upsell
	err := s.pool.QueryRow(ctx, `
		SELECT v.id, p.slug, COALESCE(tr.name, p.name), ep.price_minor
		FROM product_variants v
		JOIN products p ON p.id = v.product_id AND p.is_active
		JOIN variant_effective_prices ep ON ep.variant_id = v.id AND ep.currency = $1
		LEFT JOIN product_translations tr ON tr.product_id = p.id AND tr.locale = $2
		WHERE ep.price_minor >= $3 AND v.stock_qty > 0
		ORDER BY ep.price_minor, p.id
		LIMIT 1`,
		view.EffectiveCurrency(), view.Locale, gapMinor,
	).Scan(&u.VariantID, &u.Slug, &u.Name, &u.PriceMinor)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("querying upsell: %w", err)
	}
	return &u, nil
}

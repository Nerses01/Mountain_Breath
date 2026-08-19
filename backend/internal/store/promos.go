package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

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

// ── F2: the admin's CRUD (decision #94) ───────────────────────────────────
//
// No Delete: promo_redemptions hangs off the code (ON DELETE CASCADE), so
// a hard delete would erase the once-per-customer history and free the
// text for reuse against a wiped record. `active = false` is the off
// switch; the update endpoint flips it.

// ListPromos is the admin table's read: every code, newest first. It
// reuses promoColumns with user id 0 — no real user has id 0, so the
// used-by-shopper EXISTS is honestly false; the admin table reads the
// TOTAL count next to it, which is the one it shows.
func (s *Store) ListPromos(ctx context.Context) ([]domain.Promo, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT `+promoColumns+`
		FROM promo_codes p
		ORDER BY p.created_at DESC, p.id DESC`,
		int64(0))
	if err != nil {
		return nil, fmt.Errorf("querying promos: %w", err)
	}
	defer rows.Close()

	var promos []domain.Promo
	for rows.Next() {
		p, err := scanPromo(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning promo: %w", err)
		}
		promos = append(promos, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating promos: %w", err)
	}
	// One values query per code — an N+1 the admin table can afford: the
	// family runs a handful of codes, not a catalog. The tripwire is
	// written here: batch like attachOrderItems the day the list grows.
	for i := range promos {
		if err := attachPromoValues(ctx, s.pool, &promos[i]); err != nil {
			return nil, err
		}
	}
	return promos, nil
}

// insertPromoValues writes the per-market money rows — the whole-value
// half of both create and update.
func insertPromoValues(ctx context.Context, tx pgx.Tx, codeID int64, values map[domain.Currency]domain.PromoValue) error {
	for currency, v := range values {
		if _, err := tx.Exec(ctx, `
			INSERT INTO promo_code_values (code_id, currency, amount_minor, min_subtotal_minor)
			VALUES ($1, $2, $3, $4)`,
			codeID, currency, v.AmountMinor, v.MinSubtotalMinor); err != nil {
			return fmt.Errorf("inserting promo value: %w", err)
		}
	}
	return nil
}

// CreatePromo stores a new code, NORMALIZED — the seed and the apply
// endpoint already speak trimmed-uppercase, and storing anything else
// would make the admin table show a form of the code no lookup uses. The
// upper(code) unique index answers a duplicate in any case with
// ErrPromoCodeTaken.
func (s *Store) CreatePromo(ctx context.Context, in domain.PromoInput) (domain.Promo, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.Promo{}, fmt.Errorf("beginning promo tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var id int64
	err = tx.QueryRow(ctx, `
		INSERT INTO promo_codes (code, kind, percent, starts_at, ends_at, max_redemptions, active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`,
		domain.NormalizePromoCode(in.Code), in.Kind, in.Percent,
		in.StartsAt, in.EndsAt, in.MaxRedemptions, in.Active,
	).Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
			return domain.Promo{}, domain.ErrPromoCodeTaken
		}
		return domain.Promo{}, fmt.Errorf("inserting promo: %w", err)
	}
	if err := insertPromoValues(ctx, tx, id, in.Values); err != nil {
		return domain.Promo{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Promo{}, fmt.Errorf("committing promo: %w", err)
	}
	return s.promoByID(ctx, id)
}

// UpdatePromo is whole-value, like the variant editor's prices: the form
// shows every field so it sends every field, and a currency absent from
// Values is REMOVED (delete-then-insert inside the transaction). The
// redemption history is untouched — usage is a fact about the past, not a
// property being edited.
func (s *Store) UpdatePromo(ctx context.Context, id int64, in domain.PromoInput) (domain.Promo, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.Promo{}, fmt.Errorf("beginning promo tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `
		UPDATE promo_codes
		SET code = $2, kind = $3, percent = $4, starts_at = $5, ends_at = $6,
		    max_redemptions = $7, active = $8
		WHERE id = $1`,
		id, domain.NormalizePromoCode(in.Code), in.Kind, in.Percent,
		in.StartsAt, in.EndsAt, in.MaxRedemptions, in.Active)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
			return domain.Promo{}, domain.ErrPromoCodeTaken
		}
		return domain.Promo{}, fmt.Errorf("updating promo: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.Promo{}, domain.ErrNotFound
	}

	if _, err := tx.Exec(ctx,
		`DELETE FROM promo_code_values WHERE code_id = $1`, id); err != nil {
		return domain.Promo{}, fmt.Errorf("clearing promo values: %w", err)
	}
	if err := insertPromoValues(ctx, tx, id, in.Values); err != nil {
		return domain.Promo{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Promo{}, fmt.Errorf("committing promo update: %w", err)
	}
	return s.promoByID(ctx, id)
}

// promoByID is the fresh read both writes answer with, so the admin form
// re-renders what the DATABASE holds (normalized code included), not what
// the request hoped.
func (s *Store) promoByID(ctx context.Context, id int64) (domain.Promo, error) {
	p, err := scanPromo(s.pool.QueryRow(ctx, `
		SELECT `+promoColumns+`
		FROM promo_codes p
		WHERE p.id = $2`,
		int64(0), id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Promo{}, domain.ErrNotFound
		}
		return domain.Promo{}, fmt.Errorf("querying promo: %w", err)
	}
	if err := attachPromoValues(ctx, s.pool, &p); err != nil {
		return domain.Promo{}, err
	}
	return p, nil
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

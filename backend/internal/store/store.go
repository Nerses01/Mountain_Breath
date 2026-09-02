package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

// Postgres error code for a UNIQUE constraint violation.
// Full list: https://www.postgresql.org/docs/current/errcodes-appendix.html
const uniqueViolation = "23505"

// Store gives the rest of the app access to the database. All SQL lives here.
type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// ListCategories returns categories with their names in the requested
// language.
//
// The name resolves through three levels: the requested locale, then
// English, then the legacy categories.name column. That last step is not
// belt-and-braces paranoia — CreateCategory still writes only the parent
// column, so a category added since migration 000007 has no translation rows
// at all and would otherwise come back blank. It disappears when the admin
// write path learns to write translations, and the column is dropped with it.
//
// Ordering by the RESOLVED name means each language gets its own alphabetical
// order, which is the point.
func (s *Store) ListCategories(ctx context.Context, locale domain.Locale) ([]domain.Category, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT c.id, c.slug,
		       COALESCE(t.name, en.name, c.name) AS name,
		       c.sort_order, c.created_at
		FROM categories c
		LEFT JOIN category_translations t
		       ON t.category_id = c.id AND t.locale = $1
		LEFT JOIN category_translations en
		       ON en.category_id = c.id AND en.locale = 'en'
		ORDER BY c.sort_order, COALESCE(t.name, en.name, c.name)`,
		locale)
	if err != nil {
		return nil, fmt.Errorf("querying categories: %w", err)
	}
	defer rows.Close()

	// Start with an empty (non-nil) slice: a nil slice marshals to JSON
	// null, an empty one to [] — and the API must return [].
	cats := make([]domain.Category, 0)
	for rows.Next() {
		var c domain.Category
		if err := rows.Scan(&c.ID, &c.Slug, &c.Name, &c.SortOrder, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning category row: %w", err)
		}
		cats = append(cats, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating category rows: %w", err)
	}
	return cats, nil
}

// CreateCategory inserts c with its translations and fills in the
// DB-generated fields (ID, CreatedAt).
//
// Transactional because a category and its translations are one fact: a
// category that exists with no English translation row would read back
// through the legacy-column fallback and look fine, which is exactly the kind
// of half-written state that is worse than a clean failure.
func (s *Store) CreateCategory(ctx context.Context, c *domain.Category) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning create-category tx: %w", err)
	}
	// Rollback after a successful Commit is a no-op; this line guarantees no
	// path leaves the transaction open.
	defer func() { _ = tx.Rollback(ctx) }()

	err = tx.QueryRow(ctx, `
		INSERT INTO categories (slug, name, sort_order)
		VALUES ($1, $2, $3)
		RETURNING id, created_at`,
		c.Slug, c.Name, c.SortOrder,
	).Scan(&c.ID, &c.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
			return domain.ErrSlugTaken
		}
		return fmt.Errorf("inserting category: %w", err)
	}

	// English is written from Name, not from Translations — the API rejects
	// "en" as a translations key precisely so there is one source for it.
	names := map[domain.Locale]string{domain.DefaultLocale: c.Name}
	for locale, name := range c.Translations {
		names[locale] = name
	}
	for locale, name := range names {
		if _, err := tx.Exec(ctx, `
			INSERT INTO category_translations (category_id, locale, name)
			VALUES ($1, $2, $3)`,
			c.ID, locale, name,
		); err != nil {
			return fmt.Errorf("inserting %s category translation: %w", locale, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing create-category: %w", err)
	}
	return nil
}

// ── F2: category management (decision #95) ────────────────────────────────

// AdminCategories is the EDITOR's read, distinct from ListCategories the
// storefront uses: Name is the raw English (never locale-resolved — the
// form must show what it would be editing), and each row carries its
// non-default translations, because an editor that cannot see the Armenian
// name cannot fix it.
func (s *Store) AdminCategories(ctx context.Context) ([]domain.Category, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT c.id, c.slug, COALESCE(en.name, c.name), c.sort_order, c.created_at
		FROM categories c
		LEFT JOIN category_translations en
		       ON en.category_id = c.id AND en.locale = 'en'
		ORDER BY c.sort_order, c.id`)
	if err != nil {
		return nil, fmt.Errorf("querying admin categories: %w", err)
	}
	defer rows.Close()

	cats := make([]domain.Category, 0)
	byID := make(map[int64]*domain.Category)
	for rows.Next() {
		var c domain.Category
		if err := rows.Scan(&c.ID, &c.Slug, &c.Name, &c.SortOrder, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning admin category: %w", err)
		}
		c.Translations = make(map[domain.Locale]string)
		cats = append(cats, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating admin categories: %w", err)
	}
	for i := range cats {
		byID[cats[i].ID] = &cats[i]
	}

	trRows, err := s.pool.Query(ctx, `
		SELECT category_id, locale, name
		FROM category_translations
		WHERE locale <> 'en'`)
	if err != nil {
		return nil, fmt.Errorf("querying category translations: %w", err)
	}
	defer trRows.Close()
	for trRows.Next() {
		var id int64
		var locale, name string
		if err := trRows.Scan(&id, &locale, &name); err != nil {
			return nil, fmt.Errorf("scanning category translation: %w", err)
		}
		if c, ok := byID[id]; ok {
			c.Translations[domain.Locale(locale)] = name
		}
	}
	if err := trRows.Err(); err != nil {
		return nil, fmt.Errorf("iterating category translations: %w", err)
	}
	return cats, nil
}

// UpdateCategory is whole-value, the CreateCategory shape re-applied: the
// legacy name column and the 'en' translation row are both written from
// Name (one source), and the non-default translation rows are replaced
// with exactly what the input carries — a language left out falls back to
// English again, which is what leaving it out means everywhere else.
func (s *Store) UpdateCategory(ctx context.Context, c *domain.Category) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning update-category tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `
		UPDATE categories SET slug = $2, name = $3, sort_order = $4
		WHERE id = $1`,
		c.ID, c.Slug, c.Name, c.SortOrder)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
			return domain.ErrSlugTaken
		}
		return fmt.Errorf("updating category: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	if _, err := tx.Exec(ctx,
		`DELETE FROM category_translations WHERE category_id = $1`, c.ID); err != nil {
		return fmt.Errorf("clearing category translations: %w", err)
	}
	names := map[domain.Locale]string{domain.DefaultLocale: c.Name}
	for locale, name := range c.Translations {
		names[locale] = name
	}
	for locale, name := range names {
		if _, err := tx.Exec(ctx, `
			INSERT INTO category_translations (category_id, locale, name)
			VALUES ($1, $2, $3)`,
			c.ID, locale, name); err != nil {
			return fmt.Errorf("inserting %s category translation: %w", locale, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing update-category: %w", err)
	}
	return nil
}

// DeleteCategory removes an EMPTY category. The rule "not while it holds
// products" is not checked here with a SELECT — it is the schema's
// ON DELETE RESTRICT doing what E7's unique index did for promo reuse:
// the constraint IS the check, race-proof by construction, and this
// method only translates its refusal into the domain's word.
func (s *Store) DeleteCategory(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM categories WHERE id = $1`, id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == foreignKeyViolation {
			return domain.ErrCategoryInUse
		}
		return fmt.Errorf("deleting category: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// ReorderCategories rewrites sort_order from a position list: first id
// gets 10, the next 20, and so on — spaced so a later manual INSERT can
// still slot between without renumbering. Ids come as ONE ordered list
// (the UI sends its whole list after every move); an id that matches no
// row aborts the transaction with ErrNotFound rather than half-applying
// an order the admin never saw.
func (s *Store) ReorderCategories(ctx context.Context, ids []int64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning reorder tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for i, id := range ids {
		tag, err := tx.Exec(ctx,
			`UPDATE categories SET sort_order = $2 WHERE id = $1`, id, (i+1)*10)
		if err != nil {
			return fmt.Errorf("reordering category %d: %w", id, err)
		}
		if tag.RowsAffected() == 0 {
			return domain.ErrNotFound
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing reorder: %w", err)
	}
	return nil
}

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

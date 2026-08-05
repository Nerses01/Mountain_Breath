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

// CreateCategory inserts c and fills in its DB-generated fields (ID, CreatedAt).
func (s *Store) CreateCategory(ctx context.Context, c *domain.Category) error {
	err := s.pool.QueryRow(ctx, `
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
	return nil
}

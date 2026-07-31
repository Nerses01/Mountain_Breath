-- Full-text search over products.
--
-- search_tsv is a GENERATED column: Postgres recomputes it on every
-- INSERT/UPDATE of name/description — application code cannot forget to.
-- setweight marks name matches (A) as more important than description
-- matches (B); ts_rank uses the weights when ordering results.
ALTER TABLE products ADD COLUMN search_tsv tsvector
    GENERATED ALWAYS AS (
        setweight(to_tsvector('english', coalesce(name, '')), 'A') ||
        setweight(to_tsvector('english', coalesce(description, '')), 'B')
    ) STORED;

-- GIN = inverted index (word -> rows containing it), the index type built
-- for tsvector; without it every search would scan all products.
CREATE INDEX idx_products_search ON products USING GIN (search_tsv);

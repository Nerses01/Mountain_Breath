-- Search v2: pg_trgm breaks text into 3-letter fragments ("honey" ->
-- {" h"," ho",hon,one,ney,"ey "}) and measures overlap — which gives
-- substring matching and typo tolerance, independent of language.
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- A trigram GIN index accelerates both ILIKE '%...%' and similarity
-- operators on name — the two new fallback paths.
CREATE INDEX idx_products_name_trgm ON products USING GIN (name gin_trgm_ops);

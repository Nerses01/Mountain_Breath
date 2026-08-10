-- One badge per product card — "Best seller", "New", "Cold chain"
-- (docs/PLAN_ERA_2.md, phase E2). A column rather than a product_badges
-- table because the design draws exactly one per card; a table would model a
-- set that nothing in the design or the API ever asks for.
--
-- The column stores a KEY, not the English words. Badges are user-facing
-- text, so under decision #6 they would otherwise need a translation table
-- of their own — but unlike a product name they are a small CLOSED set that
-- the shop owner never invents at runtime, which makes them UI vocabulary
-- rather than content. The three message catalogues own the wording, the
-- same codes-not-prose contract E1.5 established for validation errors
-- (backend/internal/domain/validation.go).
--
-- The CHECK constraint is the enforcement. Postgres has a real ENUM type,
-- but altering one is awkward (ADD VALUE cannot run in a transaction before
-- PG12 and still cannot be removed), whereas a CHECK is dropped and
-- recreated by any migration that adds a badge. NULL means "no badge" —
-- CHECK passes on NULL, since SQL's three-valued logic treats an unknown as
-- neither pass nor fail, so no `OR badge IS NULL` is needed.
ALTER TABLE products
    ADD COLUMN badge TEXT
        CHECK (badge IN ('best_seller', 'new', 'cold_chain',
                         'for_makers', 'immunity', 'protein')),
    -- How it looks, not what it means: the frontend's BadgeTone union
    -- (frontend/src/components/ui/Badge.tsx) is exactly these three.
    ADD COLUMN badge_tone TEXT NOT NULL DEFAULT 'honey'
        CHECK (badge_tone IN ('honey', 'dark', 'outline'));

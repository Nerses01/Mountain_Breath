# Mountain Breath — Architecture

## System Overview

```
                        ┌──────────────────────────────────────────┐
                        │              Docker Compose              │
                        │                                          │
  Browser ──HTTPS──►  Nginx (reverse proxy)                        │
                        │   ├── /            → frontend (static React build)
                        │   └── /api/*       → backend  (Go API, port 8080)
                        │                          │
                        │                          ▼
                        │                     PostgreSQL 17
                        │                     (volume-backed)
                        └──────────────────────────────────────────┘
```

- **During development:** Vite dev server (port 5173) proxies `/api` to the Go API
  (port 8080); Postgres runs via `docker-compose.dev.yml`.
- **In production (Phase 8+):** everything is containerized behind Nginx.

## Backend Layout (Go)

Standard Go service layout, introduced in Phase 1:

```
backend/
├── cmd/api/main.go          # entry point: wiring, config, server start
├── internal/
│   ├── api/                 # HTTP layer: handlers, middleware, routing, DTOs
│   ├── domain/              # core types & business rules (no DB, no HTTP imports)
│   ├── store/               # repositories: SQL via pgx
│   └── config/              # env-based configuration
├── migrations/              # golang-migrate SQL files (NNNN_name.up/down.sql)
└── go.mod
```

Dependency direction: `api → domain ← store`. The domain package imports neither
HTTP nor SQL — this keeps business logic testable in isolation.

## Domain Model

Food & beverages catalog. Products have **variants** for weight/volume
(e.g., "Mountain Herbal Tea" → 100 g / 250 g / 500 g), each with its own price,
SKU, and stock.

```
Category 1───* Product 1───* ProductVariant
                                   │
User 1───* Order 1───* OrderItem ──┘   (references variant, snapshots price)
User 1───1 Cart  1───* CartItem ───────(references variant)
```

| Entity | Key fields | Notes |
|---|---|---|
| `Category` | id, slug, name, sort_order | flat list first; nesting only if needed |
| `Product` | id, category_id, slug, name, description, image_url, is_active | |
| `ProductVariant` | id, product_id, sku, label ("250 g" / "0.5 L"), price_minor, stock_qty | **money in integer minor units (kopecks/cents), never float** |
| `User` | id, email, password_hash, role (`customer`\|`admin`), created_at | |
| `Cart` / `CartItem` | cart: user_id or session_id; item: variant_id, qty | server-side cart |
| `Order` | id, user_id, status, total_minor, created_at | status state machine below |
| `OrderItem` | order_id, variant_id, name_snapshot, price_minor_snapshot, qty | snapshots survive later catalog edits |

**Order status state machine:**
`pending → confirmed → shipped → delivered`, with `cancelled` reachable from
`pending`/`confirmed` only. Transitions validated in the domain layer.

**Checkout invariant:** order creation runs in a single DB transaction — check stock,
decrement stock, insert order + items, or roll everything back. Guarded against
concurrent checkouts (row locking `SELECT ... FOR UPDATE`).

## API Conventions

- Base path `/api/v1`
- Plural resource nouns: `GET /api/v1/products`, `GET /api/v1/products/{slug}`,
  `POST /api/v1/cart/items`, `POST /api/v1/orders`
- JSON everywhere; errors use one envelope:
  ```json
  { "error": { "code": "validation_failed", "message": "...", "fields": { "name": "required" } } }
  ```
- List endpoints support `?page=` / `?per_page=` (offset pagination first; cursor later if needed);
  paginated responses use the envelope `{"items": [...], "page": N, "per_page": N, "total": N}`
- Auth: session cookie (`HttpOnly`, `Secure`, `SameSite=Lax`); admin endpoints under `/api/v1/admin/*`
- Versioning in the path so a future v2 can coexist

## Frontend Layout (React + TypeScript)

```
frontend/
├── src/
│   ├── api/          # typed API client + TanStack Query hooks
│   ├── components/   # reusable UI components
│   ├── pages/        # route-level components (catalog, product, cart, checkout, admin)
│   ├── auth/         # auth context, guards
│   └── main.tsx
├── index.html
└── vite.config.ts    # dev proxy: /api → localhost:8080
```

## Decisions Log

| # | Decision | Rationale | Date |
|---|---|---|---|
| 1 | Go + chi over a full framework | learn stdlib fundamentals | 2026-07-02 |
| 2 | Sessions before JWT | understand the classic model and its trade-offs first | 2026-07-02 |
| 3 | Money as integer minor units | float money is a classic bug class | 2026-07-02 |
| 4 | Server-side cart | teaches state management on the backend; works logged-out via session | 2026-07-02 |
| 5 | REST before GraphQL/gRPC | foundation first | 2026-07-02 |
| 6 | Store interfaces defined at the consumer (`api.CategoryStore`), not in `store` | Go idiom; lets tests swap in fakes without touching the store package | 2026-07-26 |
| 7 | DB errors translated to domain sentinel errors (`ErrSlugTaken`) at the store boundary | API layer maps domain errors to HTTP codes without knowing SQL details | 2026-07-26 |
| 8 | Sessions in Postgres, cookie carries raw token, DB stores its SHA-256 | leak-resistant; server-side revocation and instant role changes | 2026-07-28 |
| 9 | Cart requires login (amends #4); `PUT /cart/items` with set-semantics | avoids a parallel anonymous-token system for now; idempotent cart writes | 2026-07-29 |
| 10 | Checkout locks variant rows (`FOR UPDATE`, ordered) inside one transaction | overselling impossible under concurrency; proven with a live race test | 2026-07-29 |
| 11 | Repo stays private; no enforced branch protection | business code privacy over free protection; CI status + discipline instead | 2026-07-29 |
| 12 | Host on own hardware via classic port forwarding; Cloudflare Tunnel declined | keep TLS end-to-end (Caddy/Let's Encrypt), no third party in the traffic path; Phase 9 paused until ISP unlocks 80/443 | 2026-07-30 |
| 13 | Brand orange `#e4761f` split into a decorative fill and a darker `#b8541a` "brand ink" for anything carrying text | the design's orange measures 2.9:1 under cream text and 2.7:1 as text — both below WCAG AA's 4.5:1; fixed at token-definition time so it is never baked into a component | 2026-08-04 |
| 14 | Catalog becomes apiary-only; tea and coffee retired | the Era II design's nav, hero and benefits system assume bee products throughout — adapting the copy instead would have made half of it untrue | 2026-08-05 |
| 15 | Three languages (en default, hy, ru) via per-entity `*_translations` tables, not JSONB columns | constraints and FK safety on translated text; per-field English fallback is a `COALESCE`. Costs one migration per translatable entity and a `LEFT JOIN … ON locale = ?` per read | 2026-08-05 |
| 16 | Prices stored per market in `variant_prices` (USD + AMD), not converted from one base at a live FX rate | a shelf price is a business decision held steady, not a derived number that moves between page loads; FX kept only as fallback and for reporting | 2026-08-05 |
| 17 | URL is the single source of truth for locale; no browser language detector wired in | a detector reading `navigator.language` is a second opinion that can disagree with the address bar — the failure mode is a page whose URL says Armenian while its text says English. `useLocale` syncs i18next *from* the route, one direction | 2026-08-05 |
| 18 | English served unprefixed at `/`; only `/hy` and `/ru` carry a prefix, and the prefixes are **enumerated**, not matched with `/:locale` | matches the stated default exactly, so links written in later phases keep working. A `/:locale` param binds greedily and would treat `/cart` as a language, silently rendering the home page there | 2026-08-05 |
| 19 | Per-script font fallback by `unicode-range`, appended to the type stacks — not `:lang()` overrides | CSS resolves `font-family` per CHARACTER, so an Armenian name inside an English page renders correctly with no wrapper; and `unicode-range` gates the download, so English visitors fetch neither Noto file | 2026-08-05 |
| 20 | Per-locale `search_tsv` generated with a `CASE` over **literal** text-search configuration names | generated columns must be IMMUTABLE, which rules out `to_tsvector(locale::regconfig, …)` — the cast reads the catalog and is only STABLE. Verified against the live database before the migration was written | 2026-08-05 |
| 21 | API answers validation failures with **codes** (`slug_format`), never English prose | prose baked one language into the contract — a Russian page showed an English error under the input. The client renders the code; a fourth language needs no backend change. Codes are contract: renaming one is as breaking as renaming a JSON field | 2026-08-05 |
| 22 | Admin writes take `translations` **additively**: `name`/`description` stay required and mean English | mirrors storage exactly — those are still the parent columns the read fallback terminates at. `"en"` inside `translations` is rejected, so there is never a rule needed for which of two copies wins | 2026-08-05 |
| 23 | Variant labels are **pure measurements** ("500 g", "30 ml"); the container noun moves into the product's translatable copy | a measurement means the same thing in three languages, so `label` stays locale-invariant beside `sku` and `price_minor` and needs no fourth translation table, no extra join on every product read, and no extra input set in the admin form. Costs a little fidelity to the mock's "500 g jar" | 2026-08-10 |
| 24 | `products.badge` stores a **closed key** (`best_seller`) with a CHECK, not badge text | badges are user-facing text but a small fixed set the shop owner never invents at runtime — UI vocabulary, not content. The catalogues own the wording, the same codes-not-prose contract as #21. A CHECK rather than a Postgres ENUM because a CHECK is dropped and recreated by any migration that adds a badge | 2026-08-10 |
| 25 | Popularity is a **denormalized `products.sales_count`**, maintained inside the checkout transaction, not an aggregate over `order_items` | the aggregate reads every order line ever written to sort six products, grows without bound, and collides with pagination (GROUP BY before LIMIT). The counter costs one UPDATE inside a lock already held, and is atomic with the order it counts. Accepted cost: derived data that can drift, so the one write path is tested. Deliberately not decremented on cancellation — "most loved" measures interest over time | 2026-08-10 |
| 26 | The catalog's SQL is assembled from **Go constants** shared by the list, count and facet queries | three readers of one definition of "which products match" cannot be kept in step by a comment. Every fragment is a compile-time constant with no runtime value in it, and user input still arrives only as bound parameters — the rule against building SQL by concatenation is about user input, not about `+` | 2026-08-10 |
| 27 | Sort is a **domain-layer whitelist** (`ParseProductSort`), and the store selects a constant ORDER BY from it | `ORDER BY` cannot be a bound parameter — Postgres plans the sort at parse time, so `ORDER BY $1` sorts by the constant string. The whitelist is therefore the only thing between a query param and the planner, which makes it a security boundary and puts it in `domain` beside the other rules rather than next to the SQL | 2026-08-10 |
| 28 | A facet group's **own filter does not narrow its own counts**; every value stays listed even at 0, but a value with nothing behind it at all is dropped (`HAVING`) | otherwise clicking "Honey" leaves every other category reading 0 and the only way out of a filter is the back button. A zero caused by a filter is information; a permanently empty row is noise | 2026-08-10 |
| 29 | Shop filter state lives **entirely in the query string**, never in React state | `useState` would break the back button, shared links, reload and open-in-new-tab all at once, and would pass every click-based test while doing so. The search params ARE the state, parsed on each render, so there is no second copy to fall out of sync | 2026-08-10 |
| 30 | The API client attaches the active locale to **every translated read**, and the locale is part of every such **query key** | E1.5 shipped a trilingual API and a frontend that never asked for a language: `/hy/shop` rendered an Armenian shell around an English catalog, and nothing failed because the backend's fallback returns valid English. The query key half matters equally — a switch changes the URL but not the key, so the cached English response would be served for the Armenian page | 2026-08-10 |

Add a new row whenever we make a significant technical decision.

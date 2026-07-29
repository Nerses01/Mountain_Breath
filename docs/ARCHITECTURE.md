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

Add a new row whenever we make a significant technical decision.

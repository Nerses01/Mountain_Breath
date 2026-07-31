# Learning Log

A journal of what was learned, session by session. Newest entries on top.

Template for an entry:

```markdown
## YYYY-MM-DD — Phase N: <topic>

**Worked on:** what was built/changed
**Learned:**
- concept 1 — one-line takeaway
- concept 2 — one-line takeaway
**Questions / to revisit:**
- open question
```

---

## 2026-07-30 — Incident: web container crash-loop ("host not found in upstream")

**What happened:** nginx crash-looped on `host not found in upstream "api"` while api was healthy. Root cause: the web container was *created* during an earlier failed `up` (port-80 conflict interrupted its network attachment) → it existed with NO networks; restart policies rerun the broken container as-is, never re-create it. Fixed with `up --force-recreate web`.
**Hardening:** nginx.conf now uses `resolver 127.0.0.11` + a variable in `proxy_pass`, deferring DNS to request time — nginx boots even when api is absent (relevant after host restarts, where `depends_on` ordering does not apply). Verified: web restarts cleanly with api stopped; static pages 200, api routes 502, self-heals when api returns.
**Lessons:** `docker inspect <container>` → Networks when behavior "can't happen"; restart ≠ recreate; literal `proxy_pass` binds DNS at boot; graceful degradation beats crash-looping.

## 2026-07-30 — Phase 10 started: Prometheus metrics + Grafana dashboard

**Worked on:** metrics middleware (`mb_http_requests_total`, duration histogram — labeled by chi route PATTERN, never raw path), `/metrics` endpoint (unproxied by nginx → never public), custom `PoolCollector` for pgxpool stats, `mb_orders_created_total` business counter, metrics test; Prometheus v3.5 + Grafana 12 joined the compose stack with provisioned-from-git datasource and 6-panel RED dashboard. Verified: target up, queries answering, traffic visible.
**Learned:**
- Prometheus is PULL-based: services answer /metrics, the server visits. Counters only grow; `rate()` derives per-second speed; histograms bucket latencies so `histogram_quantile()` can compute p95 later.
- Label cardinality is the classic self-inflicted wound: label by route pattern, not URL; every unique label combo is a stored time series.
- The RED method (Rate, Errors, Duration) is the standard first dashboard for a request-driven service.
- Business metrics (orders placed) matter as much as infrastructure metrics — servers can be green while sales are zero.
- Grafana provisioning = dashboards as code in git, surviving container wipes.
**Questions / to revisit:**
- Alerting rules; OpenTelemetry tracing; k6 load test to make these graphs interesting.

## 2026-07-30 — Phase 9 prepared: deploy artifacts + runbook (server pending)

**Worked on:** `deploy/docker-compose.prod.yml` (pulls GHCR images, Caddy TLS edge, MB_ENV=prod); `deploy/Caddyfile` (auto-Let's Encrypt in one line); `deploy/backup.sh` (nightly pg_dump + rotation); CD `deploy` job in CI (SSH pull-and-restart, dormant behind `DEPLOY_ENABLED` variable); `docs/DEPLOYMENT.md` — the full runbook from empty Ubuntu VPS to live HTTPS shop with CD.
**Learned:**
- CI vs CD as separate concerns: certify artifacts vs move them; connected by the registry.
- Caddy's value proposition: TLS issuance/renewal/redirects as defaults, not configuration.
- The server pulls (git + registry) with read-only credentials: deploy key + read:packages PAT — never push credentials on the server.
- Gate not-yet-usable pipeline stages behind repository variables so CI stays green.
- A backup that was never restored is a hope, not a backup.
**Questions / to revisit:**
- Execute the runbook once VPS + domain exist; flip MB_ENV story ends (Secure cookies live).

## 2026-07-29 — Phase 8 complete: the stack containerized

**Worked on:** multi-stage Dockerfiles (Go: deps-layer caching, CGO_ENABLED=0 static build, distroless nonroot final 22.5MB; frontend: npm ci → vite build → nginx 93MB); `api healthcheck` self-probe subcommand (distroless has no curl); nginx.conf (SPA try_files fallback, immutable asset caching, /api proxy via Docker DNS); production compose with health-gated startup chain (postgres healthy → migrate completes → api healthy → web); `.dockerignore` keeping `.env` out of images; GHCR publish job in CI (green master only).
**Learned:**
- Multi-stage builds: the toolchain (~800MB) never ships; only the artifact does. Layer order matters — lockfiles first.
- Distroless: no shell = tiny attack surface; healthchecks must be built into your binary.
- `depends_on` conditions (`service_healthy`, `service_completed_successfully`) encode startup order declaratively; migrations as a one-shot service.
- Not publishing postgres's port = the DB is unreachable from outside the compose network. Security by topology.
- The nginx config is the production twin of Vite's dev proxy — same origin story, same SPA fallback.
- MB_ENV stays dev until HTTPS exists: Secure cookies would break plain-http localhost logins.
**Questions / to revisit:**
- Phase 9: VPS, domain, TLS, CD pulling the GHCR images; MB_ENV=prod then.

## 2026-07-29 — Phase 7: GitHub Actions CI

**Worked on:** `.github/workflows/ci.yml` with three parallel jobs — backend (vet, golangci-lint pinned to local version, build, `go test -race` incl. testcontainers on the runner's Docker), frontend (npm ci, lint, vitest, tsc+build), e2e (Postgres service container with healthcheck, migrations via `go run migrate@version`, seed via psql, Playwright with trace upload on failure). README badge. `reuseExistingServer: !process.env.CI` for Playwright.
**Learned:**
- CI = the local quality gate, executed by a robot on every push; jobs run in parallel on clean machines, so "works on my machine" can't hide.
- `npm ci` vs `npm install`: ci installs exactly the lockfile and never mutates it — the only correct choice for CI.
- Service containers are CI's docker-compose: Postgres per job, health-gated.
- Pin tool versions in CI to match local (golangci-lint) or chase phantom differences.
- `-race` runs on Linux runners for free (cgo present) — the reason it's delegated to CI from Windows.
- Branch protection is plan-gated for private repos: decided CI-status-without-enforcement (decision #11).
**Questions / to revisit:**
- Start the PR habit with the next feature; CI badge stays red/green truth.

## 2026-07-29 — Phase 6 complete: frontend tests (Vitest + Playwright)

**Worked on:** Vitest + Testing Library setup (jsdom, setup file with jest-dom matchers + cleanup); formatPrice unit tests (locale pinned to en-US for determinism); ProductCard component tests (render → assert via accessible queries, MemoryRouter for Link context); Playwright e2e purchase flow with dual webServer config (auto-starts Go + Vite, reuses running ones); trace-on-failure debugging.
**Learned:**
- Component tests query the DOM like a user would (`getByRole`, `getByText`) — testing behavior, not implementation details.
- The e2e failure snapshot exposed a REAL accessibility bug: a Link wrapping a card computed no accessible name; `aria-label` fixed both the screen-reader experience and the test.
- Vitest and Playwright must not see each other's test files (`test.exclude`).
- Locale-dependent formatting is a test flakiness source — pin it.
- The pyramid's economics end-to-end: 7 component tests in ~150ms; 1 browser e2e in ~1.3s covering the whole stack.
**Questions / to revisit:**
- More e2e scenarios later (admin flow, insufficient stock); coverage reporting in CI.

## 2026-07-29 — Phase 6 (backend): the test pyramid

**Worked on:** table-driven domain tests (state machine, cart math, validation); api handler tests with an in-memory `fakeStore` satisfying `api.Store` + `httptest` through the real router/middleware (auth matrix 401/403/201); testcontainers integration tests — one throwaway Postgres per package run (`TestMain`), real migrations via migrate-as-library (iofs source), `resetDB` truncation between tests; checkout tests incl. rollback-on-insufficient-stock and the formalized concurrency race (10 goroutines, stock 3).
**Learned:**
- The pyramid in practice: domain tests run in μs, fake-store handler tests in ms, container tests in seconds — write many of the cheap ones, few of the expensive ones.
- Consumer-side interfaces made the fake store trivial — the design decision from Phase 2 paid off exactly as promised.
- `TestMain` = per-package setup/teardown; `flag.Parse()` needed before `testing.Short()` there.
- Test against the real migrations — the test schema can't drift from prod.
- Verify effects in the DB, not just return values (stock decremented, cart cleared, rollback left everything untouched).
- `go test -short` as the no-Docker fast path; `-race` needs cgo (absent on Windows) → run it in Linux CI.
- errcheck caught an unchecked `Close()` in my own test code — linters lint tests too.
**Questions / to revisit:**
- Vitest component tests + Playwright e2e (rest of Phase 6).
- Auth handler tests (register/login) against the fake; coverage reporting.

## 2026-07-29 — Phase 5 complete: shopping UI

**Worked on:** cart/orders/admin-orders API client + hooks; CartPage (qty +/−, remove, checkout with error surface); OrdersPage; AdminOrdersPage (status transition buttons from a client-side mirror of the state machine); real AddToCartButton on ProductPage (out-of-stock / sign-in-to-buy / in-cart states); header cart badge with live count.
**Learned:**
- `enabled:` option gates queries on preconditions (no cart fetch while anonymous).
- Derived UI state: the add-to-cart button's four states all derive from existing queries — no new state needed.
- Mutation → invalidation fan-out: checkout touches cart+orders+products (stock!), and the UI updates everywhere without a reload.
- Client-side mirror of a server state machine is fine for UX as long as the server enforces (409 on race).
**Questions / to revisit:**
- Product/variant management UI for admin; anonymous carts; payments (Phase 10).

## 2026-07-29 — Phase 5 (backend): cart, transactional checkout, order state machine

**Worked on:** migration 000004 (cart_items with composite PK, orders, order_items with snapshots); cart store (upsert via `ON CONFLICT DO UPDATE`, FK violation → 404); `CreateOrder` transaction (`FOR UPDATE OF v` row locks, deterministic lock order, stock check → decrement → snapshot → clear cart); order state machine with cancel-restores-stock; `requireUser` middleware; endpoints `/cart`, `/cart/items`, `/orders`, admin `/admin/orders(+/{id}/status)`; Postman folders. Live concurrency test: stock=1, two parallel checkouts → exactly one 201.
**Learned:**
- A transaction = all-or-nothing; `defer tx.Rollback()` after `Begin` guarantees cleanup on every path (RAII feeling); rollback after commit is a no-op.
- `SELECT ... FOR UPDATE` locks rows so concurrent transactions queue; `ORDER BY` in the locking query = consistent lock order = no deadlocks.
- Snapshots (`price_minor_snapshot`, `name_snapshot`) make orders immune to later catalog edits.
- Composite primary keys model "one row per (user, variant)" naturally; upsert = `INSERT ... ON CONFLICT ... DO UPDATE`.
- State machines as data (`map[from][]to`) keep transition rules in one testable place.
- PUT with set-semantics is idempotent — retries are safe (matters on flaky networks).
**Questions / to revisit:**
- Frontend cart/checkout/orders UI — next session.
- Payments stub; expired pending-order cleanup; cart price-change warnings.

## 2026-07-28 — Phase 4 complete: frontend auth UI

**Worked on:** `useMe` (401→null mapping), login/register/logout/createCategory mutations; LoginPage (controlled inputs, mode toggle, per-field API errors); AuthStatus header widget (Sign in ↔ email + Sign out + Admin link); AdminPage with category form + list; routes /login and /admin; generic `request<T>` extended for POST/204.
**Learned:**
- `useMutation` for writes vs `useQuery` for reads; after a write either `setQueryData` (we hold the fresh value) or `invalidateQueries` (server knows best).
- Controlled inputs: React state owns the field value; `e.preventDefault()` stops the browser's own form submission.
- Anonymous is a state, not an error — mapping 401 to `null` keeps `useMe` clean.
- Client-side route guards are UX only; the backend middleware is the actual security boundary.
- The browser attaches HttpOnly cookies automatically — frontend code never sees or touches the session token.
**Questions / to revisit:**
- Product management UI for admin (with Phase 5+); logout everywhere; email verification (Phase 10 maybe).

## 2026-07-28 — Phase 4 (backend): sessions, bcrypt, admin gate

**Worked on:** migration 000003 (users, sessions with hashed tokens); bcrypt registration/login; session cookie (HttpOnly, SameSite=Lax, Secure=!dev, 7d TTL); `withUser` context middleware + `requireAdmin`; `/api/v1/auth/*` endpoints; category POST moved to `/api/v1/admin/categories`; Postman restructured (Auth + Admin folders, cookie-jar flows). Full lifecycle verified with curl cookie jar, including SQL role promotion taking effect on the live session.
**Learned:**
- bcrypt: deliberately slow + salted → rainbow tables and brute force become impractical; `CompareHashAndPassword` never exposes the hash comparison.
- Session tokens: crypto/rand (never math/rand); DB stores SHA-256 of the token so a DB leak can't be replayed; raw token exists only in the cookie.
- Cookie flags: HttpOnly (XSS can't read), Secure (HTTPS only), SameSite=Lax (CSRF baseline).
- Same 401 for wrong email and wrong password — user enumeration defense.
- 401 vs 403: unauthenticated vs authenticated-but-not-allowed.
- `context.WithValue` with an unexported key type: per-request data flows through the call chain without globals.
- Server-side sessions mean role changes apply instantly (the JOIN reads current role per request) — a real trade-off vs JWT, where stale claims live until expiry.
**Questions / to revisit:**
- Frontend auth UI (forms, auth context, admin area) — next session.
- Later hardening: login rate limiting, session renewal, expired-session cleanup job.

## 2026-07-28 — Phase 3 complete: react-router + product detail page

**Worked on:** react-router v7 (`BrowserRouter`, `Routes`, `Link`, `useParams`); `/products/:slug` detail page with variant picker (selected-variant state, out-of-stock disabling, 404 handling via `ApiError.status`); ProductCard wrapped in `Link`; `formatPrice` moved to `src/lib/format.ts`; disabled "Add to cart" placeholder for Phase 5.
**Learned:**
- Client-side routing: the URL changes but no server request happens — the router renders a different component; deep links work because the dev server (later Nginx) serves index.html for any non-file path (history fallback).
- `useParams` reads URL segments; the URL is state too — shareable/bookmarkable, unlike `useState`.
- Distinguishing 404 from other errors on the frontend via the typed `ApiError`.
- Vite HMR picked up new files into the running dev server without restart.
**Questions / to revisit:**
- (none)

## 2026-07-26 — Phase 3 started: React + TypeScript catalog page

**Worked on:** Vite react-ts scaffold; Tailwind v4 (`@tailwindcss/vite` plugin); typed API layer (`src/api/types.ts` mirrors Go DTOs, `client.ts` with `ApiError` + generic `request<T>`, `hooks.ts` with TanStack Query); CatalogPage with category filter chips, loading/error states, ProductCard with variant prices; Vite dev proxy `/api → :8080` (no CORS needed). Verified end-to-end: browser → proxy → Go → Postgres.
**Learned:**
- Frontend types don't travel over the wire — `types.ts` is a *promise* about JSON shape; the compiler enforces our own consistency, not the backend's (OpenAPI codegen later closes that gap).
- TanStack Query: `queryKey` identifies the cache entry; params in the key = per-filter caching; `isPending/isError/data` replaces hand-rolled fetch state.
- React mental model: UI = f(state); `useState` + re-render instead of imperative DOM updates.
- `erasableSyntaxOnly`: constructor parameter properties are TS syntax that *generates* JS — modern configs forbid non-erasable TS.
- Vite dev proxy = same-origin in dev, mirroring Nginx in prod — CORS becomes unnecessary in both.
**Questions / to revisit:**
- TypeScript handbook + React docs reading still pending (checkbox open).
- Product detail page + react-router next.

## 2026-07-26 — Phase 2: products + variants, pagination, seed data

**Worked on:** migration 000002 (products, product_variants: FKs with RESTRICT/CASCADE, CHECK constraints, composite UNIQUE, explicit FK indexes); idempotent seed script (VALUES-join inserts, ON CONFLICT DO NOTHING); `ListProducts` (filter + pagination + total) and `GetProductBySlug`; N+1 avoided via `WHERE product_id = ANY($1)`; generic `paginated[T]` envelope; chi URL params; interface embedding (`Store` = `CategoryStore` + `ProductStore`). All 6 endpoint paths verified.
**Learned:**
- FK delete policies say what deletion *means*: RESTRICT (category with products = error) vs CASCADE (variants die with product).
- Postgres does not auto-index FK columns — create those indexes yourself.
- The N+1 problem and the batch fix: load children for a whole page in one `= ANY(ids)` query.
- `($1 = '' OR col = $1)` — optional filters without string-building SQL.
- Go generics (`paginated[T any]`) — like C++ templates but constraint-based, no header bloat.
- Query params are untrusted input: parse with defaults, clamp ranges.
- Smart App Control blocks fresh unsigned `go build` output; `go run` (build cache) passes — needs a permanent decision (see below).
**Questions / to revisit:**
- Smart App Control vs local dev: decide whether to turn it off (Windows Security → App & browser control) — it will keep blocking `bin\api.exe` and possibly air's `tmp\api.exe`.

## 2026-07-26 — Phase 2: pgx store layer + first real endpoints

**Worked on:** `internal/domain` (Category + validation + `ErrSlugTaken` sentinel), `internal/store` (pgxpool, ListCategories, CreateCategory), consumer-side `CategoryStore` interface in `api`, DTOs, `GET/POST /api/v1/categories` with 201/400/409 handling, `main` refactored to `run() error` pattern, godotenv for dev, Postman Catalog folder with pre-request scripts and tests. All 6 request paths verified with curl.
**Learned:**
- Connection pool: handlers borrow/return connections concurrently; `pgxpool.New` + startup `Ping` = fail fast.
- Parameterized queries (`$1, $2`) — the only defense against SQL injection; never string-concatenate SQL.
- Sentinel error flow across layers: pg error 23505 → `errors.As` → `domain.ErrSlugTaken` → `errors.Is` in handler → HTTP 409. Layers stay decoupled.
- Interfaces belong at the consumer (`api.CategoryStore`), satisfied implicitly — enables fake stores in tests.
- `INSERT ... RETURNING` fetches DB-generated values in the same round-trip.
- nil slice marshals to JSON `null`, empty slice to `[]` — APIs must return `[]`.
- Request hygiene: `http.MaxBytesReader` (body cap) + `DisallowUnknownFields` (typo'd JSON keys fail loudly).
- `run() error` pattern: `os.Exit` skips `defer`s, so main delegates to a function that returns errors.
- PowerShell 5.1 mangles embedded quotes for native exes — pass JSON to curl via `-d "@file"`.
**Questions / to revisit:**
- Products + variants schema and endpoints; pagination; seed script.

## 2026-07-24 — Phase 2 started: Postgres in Docker, first migration

**Worked on:** `deploy/docker-compose.dev.yml` (postgres:17-alpine, named volume, healthcheck, env-based secrets with git-ignored `.env` + committed `.env.example`); installed `migrate` CLI; migration 000001 (categories table); tested up → down → up; fixed broken `.gitignore` inline comments and untracked air build logs.
**Learned:**
- `.gitignore` does not support inline `#` comments — the comment becomes part of the pattern.
- Compose: `${VAR:-default}` vs `${VAR:?error}`; named volumes survive `docker compose down`; healthcheck gates dependent services.
- Migrations are versioned, append-only schema changes; `schema_migrations` table tracks the current version + a dirty flag; every up needs a working down.
- Postgres: `GENERATED ALWAYS AS IDENTITY` (modern SERIAL), `TEXT` is idiomatic (no VARCHAR(n) needed), `TIMESTAMPTZ` for timestamps, UNIQUE constraint creates an index automatically.
- Windows: `migrate -path` needs forward-slash/relative paths (it builds a `file://` URL).
**Questions / to revisit:**
- (none yet)

## 2026-07-24 — Phase 1 complete: structured API, middleware, graceful shutdown

**Worked on:** split `main.go` into `internal/api` + `internal/config`; chi router; request-logging and panic-recovery middleware; JSON respond helpers with the standard error envelope; env-based config; `http.Server` timeouts; graceful shutdown with `signal.NotifyContext` + `srv.Shutdown`; dev-only `/debug/slow` endpoint; Delve + VS Code F5 debugging; `.air.toml` and `.golangci.yml`.
**Learned:**
- Middleware = decorator pattern over `http.Handler`; chain order matters; closures capture dependencies.
- `defer` + `recover` only work together during panic unwinding — same mechanism as destructors during C++ exception unwinding.
- Every request runs in its own goroutine → handlers must be concurrency-safe.
- `select` waits on multiple channels at once (like `WaitForMultipleObjects`); `r.Context()` is cancelled when the client disconnects.
- Graceful shutdown: `signal.NotifyContext` → `<-ctx.Done()` → `srv.Shutdown(ctxWithTimeout)`; `ListenAndServe` returns `http.ErrServerClosed` on purpose then.
- `errcheck` forces every error to be handled or explicitly discarded (`_ =`) — `[[nodiscard]]` everywhere.
**Questions / to revisit:**
- Test graceful shutdown manually (Ctrl+C during `/debug/slow`).

## 2026-07-02 — Phase 0: Environment Setup ✅

**Worked on:** full dev environment on Windows — installed Go 1.26.4, Node.js 24.18 LTS, golangci-lint, air, VS Code extensions (Go, ESLint, Prettier, Tailwind); verified Docker; created private GitHub repo and pushed (`gh repo create ... --push`).
**Learned:**
- `go install <module>@latest` compiles and installs Go CLI tools into `%USERPROFILE%\go\bin` — that dir must be on PATH.
- Git identity resolution: repo-local config overrides global (`git config user.email` vs `--global`).
- `gh` (GitHub CLI) can create a repo from an existing local one and set up the `origin` remote + tracking branch in one command.
**Questions / to revisit:**
- Is `ner.manukyan@gmail.com` added to the GitHub account (Nerses01)? If not, commits won't link to the profile — check GitHub → Settings → Emails.

## 2026-07-02 — Project kickoff

**Worked on:** project planning — goals, tech stack, architecture, phased roadmap; repository initialized.
**Learned:**
- How a full-stack project is structured: monorepo with backend / frontend / deploy / CI separation.
- Why Go fits a C++ developer (compiled, typed) and why sessions-before-JWT, money-as-integers, REST-before-GraphQL.
**Questions / to revisit:**
- (none yet)

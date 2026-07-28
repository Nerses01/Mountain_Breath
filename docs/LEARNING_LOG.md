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

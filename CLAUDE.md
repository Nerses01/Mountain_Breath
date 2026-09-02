# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Working agreement

This is an e-commerce store **and** a deliberate learning project (C++ developer
learning full-stack). [docs/RULES.md](docs/RULES.md) is binding — read it at the
start of a session. The parts that change how you work here:

- **Learning over speed** (#1). The explanation is the deliverable, not a
  courtesy attached to one.
- **Claude writes, developer studies** (#3). Write the code with detailed
  explanations; the developer reads and must understand every line.
- **Teach the concepts, not just the decisions** (#4). Explain a new concept
  *before* writing code with it, name the C++ analogue or deliberate
  difference (error values vs exceptions, GC vs RAII, structural interfaces vs
  virtual inheritance, SQL `IMMUTABLE` vs `constexpr`), and **close every
  response with a short "What you learn here" section**. One line if the
  response only rearranged existing patterns. This is explicit because the
  abstract version of the rule was not producing it in practice.
- **Never run Git commands that change the repository** (#9) — including
  `git add`. Finish the work, state the suggested commit message, stop.
  Reading `git status`/`log`/`diff` is fine.
- **The design canvas is the source of UI truth** (#16): two canvases,
  `docs/design/Mountain Breath Store.dc.html` (screens 01–06) and
  `docs/design/Mountain Breath Account.dc.html` (07–10, the logged-in
  account area — plan in docs/PLAN_ACCOUNT.md), greppable rather than
  re-fetched.
  Three standing exceptions — accessibility overrides it, states it never
  draws (focus/error/loading/empty/disabled/hover) are yours to design, and
  post-mock requirements like the three languages have no guidance in it.
  Departures from the canvas get a line saying why.
- **One phase at a time** — follow the roadmap in
  [docs/PROJECT_PLAN.md](docs/PROJECT_PLAN.md) (Era I, phases 0–11) and
  [docs/PLAN_ERA_2.md](docs/PLAN_ERA_2.md) (Era II, phases E1–E10: building the
  designed storefront). Finish a phase's definition of done before moving on.
- After a significant session, append an entry to
  [docs/LEARNING_LOG.md](docs/LEARNING_LOG.md) (newest first, dated).
- Significant technical choices get a row in the Decisions Log in
  [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).
- **Every commit that adds/changes/removes a route updates
  `docs/api/mountain-breath.postman_collection.json` in the same commit** — it
  is the living API contract. Collection and code disagreeing is a bug.
- Conventional Commits (`feat:`, `fix:`, `docs:`, `refactor:`, `test:`,
  `chore:`). Daily work goes to `dev`; batch PRs merge `dev` → `master` with a
  merge commit (never squash), then `dev` is synced back from `master`.

## Commands

Windows host, PowerShell. Postgres always runs in Docker.

```powershell
# Dev stack: DB + mail catcher in Docker, servers on the host
docker compose -f deploy/docker-compose.dev.yml up -d   # Postgres :5432, Mailpit SMTP :1025 / mailbox UI http://localhost:8025
cd backend;  air                                        # API :8080, hot reload (or: go run ./cmd/api)
cd frontend; npm run dev                                # Vite :5173, proxies /api + /uploads → :8080
# Outgoing mail (reset links, order confirmations) goes to Mailpit when
# MB_SMTP_ADDR=localhost:1025 is set in backend/.env; unset, it lands in the
# API log. Google sign-in needs MB_GOOGLE_CLIENT_ID/SECRET (.env.example).

# Full containerized stack (nginx on :80, + Prometheus :9090, Grafana :3000)
docker compose -f deploy/docker-compose.yml up --build
```

Both stacks read `deploy/.env` (copy from `deploy/.env.example`); the host-run
API reads `backend/.env` (copy from `backend/.env.example`). Only
`MB_DATABASE_URL` is mandatory — `MB_ADDR`, `MB_ENV`, `MB_UPLOADS_DIR` have
defaults ([config.go](backend/internal/config/config.go)).

### Tests

```powershell
cd backend
go test ./...                       # store tests start a throwaway Postgres via testcontainers (needs Docker)
go test -short ./...                # skip everything Docker-backed
go test ./internal/store -run TestCreateOrder_ConcurrentCheckoutsDoNotOversell -v
go test -count=1 ./internal/api     # -count=1 defeats the test result cache

cd frontend
npm test                            # Vitest (jsdom), one shot
npm run test:watch
npx vitest run src/lib/format.test.ts        # single file
npm run e2e                                  # Playwright; starts/reuses both dev servers, needs Postgres up
npx playwright test --debug
```

CI runs `go test -race`; on Windows `-race` needs a cgo C toolchain, so plain
`go test` locally is the norm.

### Lint / typecheck / build

```powershell
cd backend;  go vet ./...; golangci-lint run   # v2.12.2 — keep in sync with .github/workflows/ci.yml
cd frontend; npm run lint                      # oxlint
cd frontend; npm run build                     # tsc -b (typecheck) + vite build
```

### Migrations & seed

```powershell
cd backend
migrate create -ext sql -dir migrations -seq add_something          # writes NNNNNN_*.up.sql + .down.sql
migrate -path migrations -database "$env:MB_DATABASE_URL" up        # forward-slash/relative paths on Windows
migrate -path migrations -database "$env:MB_DATABASE_URL" down 1

# seed the dev database, then promote yourself to admin (register through the UI first)
docker compose -f deploy/docker-compose.dev.yml cp backend/seed/seed.sql postgres:/tmp/seed.sql
docker compose -f deploy/docker-compose.dev.yml exec -T postgres psql -U mb -d mountain_breath -v ON_ERROR_STOP=1 -f /tmp/seed.sql
docker compose -f deploy/docker-compose.dev.yml exec postgres psql -U mb -d mountain_breath -c "UPDATE users SET role='admin' WHERE email='you@example.com';"
```

**Copy the file in, don't pipe it.** `Get-Content seed.sql | … psql` re-encodes
the stream through the console code page, which silently double-encodes every
non-ASCII byte: the Armenian and Russian translations land as `Õ„Õ¡Ö„…`, and
`4 × 100 g` becomes `4 Ã— 100 g`. Nothing errors — the mojibake is valid UTF-8,
just wrong — so it is only ever caught by looking at a rendered page. `cp`
transfers raw bytes and has no encoding step to get wrong. (The `migrate` CLI
reads its files directly and was never affected.)

Every `up` migration needs a working `down` — the up→down→up cycle is expected
to be tested. The compose stacks apply migrations with a one-shot
`migrate/migrate` job that the API waits on.

Load test: `k6 run load/catalog-test.js` (SLO thresholds are in the script).

## Architecture

Full design in [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md). What matters when
editing:

**Layering: `api → domain ← store`.** `domain` imports neither HTTP nor SQL, so
business rules are testable in isolation. All SQL lives in `store`.

**Store interfaces are declared at the consumer**, in
[server.go](backend/internal/api/server.go) (`CategoryStore`, `ProductStore`,
… composed into `Store`), and satisfied implicitly by `*store.Store`. Adding a
store method means adding it to the right interface there **and** to the
in-memory `fakeStore` in [api_test.go](backend/internal/api/api_test.go), which
is what lets handler tests run with no database.

**Errors cross layers as domain sentinels.** The store translates driver errors
(e.g. Postgres `23505`) into `domain.ErrSlugTaken` / `ErrSKUTaken` /
`ErrNotFound`; the API maps those to status codes. The API layer never inspects
SQL errors. Responses use one envelope —
`{"error":{"code","message","fields"}}` via `respondError` /
`respondValidationError` in [respond.go](backend/internal/api/respond.go).
Validation field keys use the JSON path the frontend form uses
(`variants[0].sku`), so the React admin form can attach errors to inputs.

**Money is always integer minor units** (`price_minor`, `total_minor`) — never
float, in Go, SQL, or TypeScript.

**Auth.** `withUser` runs on all of `/api/v1` and only *attaches* a user;
`requireUser` (401) and `requireAdmin` (401 anon / 403 non-admin) gate route
groups. The cookie carries a raw token; the `sessions` table stores its SHA-256.
Login returns identical errors for unknown email and wrong password.

**Checkout is the critical section.** `store.CreateOrder` opens one
transaction, locks the cart's variant rows with `FOR UPDATE OF v` in a
deterministic order, validates and decrements stock, snapshots names and
prices into `order_items`, and clears the cart. Order status transitions are
validated in `domain` (`pending → confirmed → shipped → delivered`, `cancelled`
only from pending/confirmed, and cancelling restores stock). The invariant is
covered by a 10-goroutine race test.

**Metrics.** The registry is per-`Server` and owns the RED middleware.
`metricsMiddleware` labels by chi's **route pattern**, never the raw path —
labeling by path would create unbounded label cardinality. `/metrics` is
deliberately not proxied by nginx/Caddy.

**Search** ([products.go](backend/internal/store/products.go)) is a hybrid: a
generated weighted `search_tsv` + GIN for full-text (`websearch_to_tsquery`),
plus `pg_trgm` similarity for prefix/typo/substring matching, with rank =
FTS rank + name similarity. websearch operators are stripped from the fuzzy
branch, and LIKE wildcards escaped.

**Frontend.** [client.ts](frontend/src/api/client.ts) is the only place that
calls `fetch`; it unwraps the error envelope into `ApiError` (status, code,
fields). Components never call it directly — they go through the TanStack Query
hooks in [hooks.ts](frontend/src/api/hooks.ts), which own query keys and cache
invalidation. Types in `api/types.ts` are hand-maintained to mirror the backend
JSON. The browser always talks to one origin (Vite proxy in dev, nginx/Caddy in
prod), so there is no CORS handling anywhere by design.

## Testing layers

| Layer | Location | Dependencies |
|---|---|---|
| Domain unit tests, table-driven | `backend/internal/domain/*_test.go` | none |
| HTTP handler tests, real middleware chain + `fakeStore` | `backend/internal/api/*_test.go` | none |
| Repository integration tests | `backend/internal/store/*_test.go` | Docker (testcontainers); `TestMain` migrates one container, `resetDB(t)` truncates between tests |
| Component tests | `frontend/src/**/*.test.ts(x)` | jsdom |
| End-to-end purchase journey | `frontend/e2e/` | Playwright + running Postgres |

New code comes with tests (rule #11).

## CI/CD

[.github/workflows/ci.yml](.github/workflows/ci.yml): `backend` (vet, lint,
build, `-race` tests) and `frontend` (lint, tests, typecheck+build) run on every
push to `master`/`dev` and on PRs. The expensive `e2e` job runs only on PRs and
`master`. Green `master` publishes `ghcr.io/nerses01/mountain-breath-{api,web}`
images; the `deploy` job is dormant until the repo variable `DEPLOY_ENABLED` is
`true` (Phase 9 is blocked on hosting — see
[docs/DEPLOYMENT.md](docs/DEPLOYMENT.md)).

# Mountain Breath — Project Plan

> E-commerce website for the Mountain Breath food & beverages business, built as a
> long-term **full-stack learning project**: backend, frontend, testing, CI/CD, and
> deployment — the complete lifecycle of a modern web application.

---

## 1. Goals

Ranked by priority (agreed at project start):

1. **Learn backend & full-stack development** — from a C++ desktop background to a
   job-ready full-stack skill set.
2. **Learn the complete delivery pipeline** — architecture → code → tests → CI →
   containers → deployment → monitoring.
3. **Learn new languages and tools** — Go and TypeScript, plus the professional
   tooling around them.
4. **Build a portfolio project** that helps in job hunting.
5. **Serve the business** — a real, working online store for Mountain Breath.

Learning takes priority over shipping fast. Time budget: **10+ hours/week**, no deadline.

## 2. Tech Stack & Rationale

| Layer | Choice | Why this choice |
|---|---|---|
| Backend language | **Go 1.24+** | Compiled, statically typed, performant — a natural transition from C++. One of the most in-demand backend/cloud languages. Small language spec, fast to become productive. |
| HTTP layer | **`net/http` + [chi](https://github.com/go-chi/chi) router** | Learn the standard library fundamentals first instead of hiding them behind a big framework. `chi` is a thin, idiomatic router on top of `net/http`. |
| Database | **PostgreSQL 17** | The industry-standard open-source relational database. Runs in Docker locally. |
| DB access | **[pgx](https://github.com/jackc/pgx)** driver | The de-facto standard Postgres driver for Go. Write real SQL first; consider `sqlc` later once SQL is comfortable. |
| Migrations | **[golang-migrate](https://github.com/golang-migrate/migrate)** | Versioned schema migrations are a core backend skill. |
| Frontend | **React 19 + TypeScript + [Vite](https://vitejs.dev)** | The biggest frontend job market by far. TypeScript is the second new language to learn. Vite gives a fast, modern dev experience. |
| Styling | **Tailwind CSS** | Utility-first CSS, extremely common in modern React codebases. |
| Server state | **TanStack Query** | The standard way to fetch/cache API data in React. |
| API style | **REST + JSON** | The foundation; OpenAPI spec added later. GraphQL/gRPC can be explored in late phases. |
| Containers | **Docker + Docker Compose** | The whole stack (API + DB + frontend) runs locally with one command. |
| VCS / hosting | **Git + GitHub** | Industry standard; required for CI and portfolio visibility. |
| CI/CD | **GitHub Actions** | The most common CI system in job postings; free for public repos. |
| Backend testing | **`go test`** + [testcontainers-go](https://golang.testcontainers.org/) | Unit tests with the built-in toolchain; integration tests against a real Postgres in Docker. |
| Frontend testing | **Vitest** + **Playwright** | Component/unit tests + real end-to-end browser tests of the purchase flow. |
| Dev tooling | **golangci-lint**, **Air** (hot reload), **ESLint + Prettier** | Professional workflow from day one — linters run locally and in CI. |

## 3. Repository Layout (monorepo)

```
Mountain_Breath/
├── README.md
├── docs/
│   ├── PROJECT_PLAN.md      # this file — the master plan
│   ├── RULES.md             # working rules for the collaboration
│   ├── ARCHITECTURE.md      # system design, domain model, API conventions
│   └── LEARNING_LOG.md      # journal: what was learned in each phase
├── backend/                 # Go API service            (created in Phase 1)
├── frontend/                # React + TypeScript app    (created in Phase 3)
├── deploy/                  # Dockerfiles, docker-compose, nginx configs
└── .github/workflows/       # CI pipelines              (created in Phase 7)
```

See [ARCHITECTURE.md](ARCHITECTURE.md) for the system design and domain model.

## 4. Learning Roadmap

Each phase has **goals**, a **you-will-learn** list, **tasks**, and a **definition of
done**. Phases are sequential but revisiting earlier phases to refactor is expected —
that is part of learning.

---

### Phase 0 — Environment Setup

**Goal:** a working professional development environment on Windows.

**You will learn:** the toolchain of a full-stack developer; Git basics if any gaps.

**Tasks:**
- [x] Install Go (latest stable), verify with `go version` — Go 1.26.4 via winget
- [x] Install Node.js LTS, verify `node -v`, `npm -v` — Node 24.18.0 / npm 11.16 via winget
- [x] Install Docker Desktop, verify — Docker 29.6.1, engine running
- [x] Configure Git (`user.name`, `user.email`), create GitHub repo, push the initial commit — https://github.com/Nerses01/Mountain_Breath (private)
- [x] Install VS Code extensions: Go, ESLint, Prettier, Tailwind CSS IntelliSense
- [x] Install `golangci-lint` (2.12.2 via winget) and `air` (1.65.3 via `go install`)

**Done when:** `go version`, `node -v`, `docker ps` all work; the repo is on GitHub. ✅ **Completed 2026-07-02**

---

### Phase 1 — Go Fundamentals & First HTTP API

**Goal:** be comfortable with Go syntax and idioms; stand up a minimal API.

**You will learn:** Go modules, packages, structs, interfaces, error handling
(`error` values vs C++ exceptions), goroutines/channels basics, `net/http`,
JSON marshalling, project layout conventions.

**Tasks:**
- [x] Complete the [Tour of Go](https://go.dev/tour/) (1–2 evenings)
- [x] Read [Effective Go](https://go.dev/doc/effective_go) sections as needed
- [x] Create `backend/` with `go mod init`
- [x] Build a minimal API: `GET /health` returning JSON, structured logging with `log/slog`
- [x] Add `chi` router, middleware (request logging, panic recovery)
- [x] Set up `air` for hot reload and `golangci-lint` with a config file
- [x] Understand and document the chosen project layout (`cmd/`, `internal/`)
- [x] Graceful shutdown (signal → drain in-flight requests → exit) + Delve debugging set up in VS Code

**Done when:** `go run ./cmd/api` serves `/health`; linter passes; layout documented. ✅ **Completed 2026-07-24**

---

### Phase 2 — Database & Catalog API

**Goal:** the core backend skill set — relational modeling, SQL, migrations, CRUD.

**You will learn:** PostgreSQL basics, SQL (joins, indexes, constraints), schema
migrations, the repository pattern, `pgx`, connection pooling, environment-based
configuration, Docker Compose for dev dependencies.

**Tasks:**
- [x] Write `deploy/docker-compose.dev.yml` with PostgreSQL 17 (+ `.env`/`.env.example` secrets pattern)
- [x] Design the catalog schema (see ARCHITECTURE.md): categories, products, variants (FKs, CHECK constraints, FK indexes)
- [x] Create migrations with `golang-migrate`; learn up/down migrations (000001 categories, 000002 products+variants; full up→down→up cycle tested)
- [x] Implement repository layer with `pgx` (pgxpool, consumer-side `CategoryStore` interface in `api`)
- [x] REST endpoints: categories (list+create), products (paginated list with category filter + detail by slug); product writes deferred to Phase 4 admin
- [x] Input validation and a consistent JSON error format (domain validation, 400/404/409 mapping, sentinel errors)
- [x] Seed script with sample Mountain Breath products (`backend/seed/seed.sql`, idempotent)

**Done when:** full catalog CRUD works via `curl`/Postman against Postgres in Docker.

---

### Phase 3 — Frontend Foundations

**Goal:** learn TypeScript and React; display the live catalog.

**You will learn:** TypeScript type system, React components/hooks/state, Vite,
Tailwind, TanStack Query, CORS (and why it exists), talking to your own API.

**Tasks:**
- [ ] TypeScript fundamentals (official handbook) + React quick start
- [ ] Scaffold `frontend/` with Vite (react-ts template), ESLint + Prettier, Tailwind
- [ ] Pages: product list (grid with categories filter), product detail with variant picker (weight/volume)
- [ ] Fetch data with TanStack Query; typed API client sharing types with backend JSON
- [ ] Handle loading/error states properly; add CORS middleware to the Go API

**Done when:** the catalog from Phase 2 renders in the browser from the real API.

---

### Phase 4 — Authentication & Authorization

**Goal:** one of the most asked-about backend topics in interviews.

**You will learn:** password hashing (bcrypt/argon2), sessions vs JWT (implement
sessions with secure cookies first — understand the trade-offs), middleware-based
auth, role-based access (customer vs admin), CSRF basics.

**Tasks:**
- [ ] `users` table + migration (email, password hash, role)
- [ ] Register / login / logout endpoints; session store; secure cookie handling
- [ ] Auth middleware; protect admin-only catalog CRUD
- [ ] Frontend: login/register forms, auth context, admin area for managing products

**Done when:** only a logged-in admin can modify the catalog; sessions survive restart.

---

### Phase 5 — Cart & Orders

**Goal:** real business logic — the heart of e-commerce.

**You will learn:** DB transactions and isolation (compare with C++ concurrency
intuition), designing state machines (order status), handling money correctly
(integer minor units, never floats), concurrency around stock.

**Tasks:**
- [ ] Cart: server-side cart tied to session/user; add/update/remove items
- [ ] Checkout: create order in a transaction — validate stock, decrement, snapshot prices
- [ ] Order status flow: `pending → confirmed → shipped → delivered / cancelled`
- [ ] Frontend: cart page, checkout flow, "my orders" page; admin order list
- [ ] Payments: stub/manual confirmation now; real provider integration in Phase 10

**Done when:** a customer can buy a product end-to-end and stock is correctly reduced,
even with concurrent checkouts (write a test that proves it).

---

### Phase 6 — Testing

**Goal:** the full test pyramid; testing is applied *retroactively and continuously*
from here on — new code in later phases arrives with tests.

**You will learn:** table-driven tests in Go, mocking vs real dependencies,
integration tests with testcontainers, HTTP handler tests, Vitest component tests,
Playwright end-to-end tests, coverage reports.

**Tasks:**
- [ ] Unit tests for domain logic (cart math, order state transitions)
- [ ] Integration tests for repositories against real Postgres (testcontainers-go)
- [ ] HTTP-level tests for API endpoints (auth, error cases)
- [ ] Vitest tests for key React components
- [ ] Playwright e2e: browse → add to cart → checkout
- [ ] Concurrent-checkout stock test from Phase 5 formalized

**Done when:** one command runs each test suite; the purchase flow is covered e2e.

---

### Phase 7 — Continuous Integration

**Goal:** nothing reaches `master` unverified.

**You will learn:** GitHub Actions (workflows, jobs, matrices, caching), branch
protection, PR-based workflow, status checks, artifacts.

**Tasks:**
- [ ] Workflow: lint + build + unit/integration tests for backend on every push/PR
- [ ] Workflow: lint + typecheck + test + build for frontend
- [ ] Playwright e2e job (against docker-compose services)
- [ ] Branch protection on `master`: PRs only, checks must pass
- [ ] Adopt PR workflow even solo — small PRs with self-review

**Done when:** a broken test blocks a PR; green checks required to merge.

---

### Phase 8 — Full Containerization

**Goal:** the entire stack runs identically anywhere with one command.

**You will learn:** writing production-grade multi-stage Dockerfiles (tiny Go static
binaries!), image layers and caching, docker-compose networking, Nginx as reverse
proxy and static file server, environment/secret handling, health checks.

**Tasks:**
- [ ] Multi-stage `Dockerfile` for the Go API (scratch/distroless final image)
- [ ] `Dockerfile` for frontend (build → static files served by Nginx)
- [ ] `deploy/docker-compose.yml`: nginx + api + postgres, one network, volumes, healthchecks
- [ ] Nginx config: serve frontend, proxy `/api` to the Go service
- [ ] CI builds and pushes images (GitHub Container Registry)

**Done when:** `docker compose up` on a clean machine serves the whole store.

---

### Phase 9 — Deployment & Continuous Delivery

**Goal:** the store is live on the internet with automated deploys.

**You will learn:** Linux server administration basics, SSH, firewalls, DNS,
HTTPS/TLS with Let's Encrypt, zero-ish-downtime deploys, database backups,
deploy automation from CI.

**Tasks:**
- [ ] Choose hosting (recommendation: cheap VPS — Hetzner/DigitalOcean — for maximum learning)
- [ ] Harden the server: non-root user, SSH keys, firewall, fail2ban
- [ ] Domain + DNS + TLS certificates (Let's Encrypt via certbot or Caddy/Traefik)
- [ ] CD: on merge to `master`, CI builds images and deploys to the server
- [ ] Automated Postgres backups + tested restore procedure

**Done when:** merging a PR automatically updates the live site over HTTPS.

---

### Phase 10 — Production Skills & Extensions

**Goal:** the topics that distinguish senior backend engineers. Pick by interest.

- **Observability:** structured logs aggregation, metrics with Prometheus + Grafana, alerting, request tracing
- **Payments:** integrate a real payment provider (Stripe or a local one)
- **Performance:** load testing (k6), query optimization, caching (Redis), CDN
- **Email:** transactional emails (order confirmation)
- **Search:** product search (Postgres full-text first)
- **API evolution:** OpenAPI spec + generated clients; explore gRPC or GraphQL
- **Infrastructure as Code:** Terraform/Ansible for the server setup
- **Kubernetes:** optional; only after Docker Compose feels limiting

---

## 5. Milestone Summary

| Milestone | After phase | You can put on a CV |
|---|---|---|
| API + DB running locally | 2 | Go REST API with PostgreSQL |
| Visible product catalog | 3 | Full-stack feature with React + TypeScript |
| Working store (auth, cart, orders) | 5 | Complete e-commerce backend with transactions |
| Tested + CI | 7 | Test pyramid + GitHub Actions pipeline |
| Live on the internet | 9 | Dockerized deployment with CD, HTTPS, backups |

## 6. Working Documents

- [RULES.md](RULES.md) — how we work together (read this first each session)
- [ARCHITECTURE.md](ARCHITECTURE.md) — system design and domain model
- [LEARNING_LOG.md](LEARNING_LOG.md) — journal of learned topics per session/phase

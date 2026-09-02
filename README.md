# Mountain Breath

![CI](https://github.com/Nerses01/Mountain_Breath/actions/workflows/ci.yml/badge.svg)

E-commerce website for the Mountain Breath food & beverages business — and a
long-term **full-stack learning project** covering the complete lifecycle of a
modern web application: Go backend, React + TypeScript frontend, PostgreSQL,
Docker, GitHub Actions CI/CD, and deployment.

Built solo, from an empty directory to a containerized app with a tested
CI pipeline, one phase at a time. The roadmap, the architecture decisions, and
a journal of everything learned along the way are all in [docs/](docs/).

## Stack

Go 1.26 · chi · pgx · PostgreSQL 17 · React 19 · TypeScript · Vite ·
Tailwind CSS v4 · TanStack Query · Docker Compose · Caddy · GitHub Actions ·
Prometheus + Grafana · Playwright · k6

## What's built

**Catalog** — categories, products with variants, pagination and filtering,
full-text search with trigram fuzzy matching (PostgreSQL FTS), admin product
and image management.

**Auth** — registration and login with bcrypt, server-side sessions in
Postgres (the DB stores only a SHA-256 of the token), HttpOnly cookies,
role-based `requireAdmin` middleware, and identical errors for wrong email vs.
wrong password to prevent user enumeration.

**Cart & orders** — server-side cart, checkout in a single transaction with
`FOR UPDATE` row locks so concurrent buyers cannot oversell the last item,
price snapshotting, and an order state machine
(`pending → confirmed → shipped → delivered / cancelled`) where cancelling
restores stock.

**Operations** — multi-stage Docker builds (distroless Go image, 22.5 MB,
non-root, self-probing healthcheck), health-gated compose startup with a
one-shot migration job, Prometheus metrics with Grafana dashboards and
Alertmanager rules, nightly `pg_dump` backups, and Caddy for automatic
Let's Encrypt TLS.

**Testing** — table-driven domain unit tests, repository integration tests
against a real Postgres via testcontainers, HTTP-level API tests with a fake
store and the real middleware chain, Vitest component tests, a Playwright e2e
purchase journey, and a k6 load script. A concurrency test spawns 10
simultaneous checkouts against stock of 3 and asserts exactly 3 succeed.

**CI** — every push runs vet, lint, race-enabled backend tests, frontend
lint/typecheck/tests, and the e2e suite; green builds on `master` push images
to GHCR.

## Documentation

| Document | Purpose |
|---|---|
| [docs/PROJECT_PLAN.md](docs/PROJECT_PLAN.md) | Goals, tech stack rationale, and the phased learning roadmap |
| [docs/PLAN_ERA_2.md](docs/PLAN_ERA_2.md) | Era II roadmap: turning the design into the real storefront |
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | System design, domain model, API conventions, decisions log |
| [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md) | Server setup runbook: hardening, TLS, CD, backups |
| [docs/RULES.md](docs/RULES.md) | Working rules for the collaboration |
| [docs/LEARNING_LOG.md](docs/LEARNING_LOG.md) | Journal of learned topics |

## Running it locally

Requires Docker. The whole stack — database, migrations, API, and web — comes
up with one command:

```bash
cp deploy/.env.example deploy/.env    # then fill in a password
docker compose -f deploy/docker-compose.yml up --build
```

The app is served at `http://localhost` and the API at
`http://localhost/api/v1`.

To work on the code instead, run the database in Docker and the two dev
servers on the host:

```bash
docker compose -f deploy/docker-compose.dev.yml up -d   # Postgres only
cp backend/.env.example backend/.env                    # then fill in the DSN
cd backend && air                                       # API on :8080, hot reload
cd frontend && npm install && npm run dev               # Vite on :5173
```

Tests:

```bash
cd backend  && go test ./...        # add -short to skip the Docker-backed ones
cd frontend && npm test             # component tests
cd frontend && npm run e2e          # Playwright journey
```

## Repository layout

```
Mountain_Breath/
├── docs/          # planning, architecture, runbooks, learning journal
├── backend/       # Go API service — cmd/, internal/{api,domain,store,config}, migrations/, seed/
├── frontend/      # React + TypeScript app — src/{api,components,pages,lib}, e2e/
├── deploy/        # Docker Compose stacks, Caddy, observability, backup script
├── load/          # k6 load test
└── .github/       # CI/CD workflows
```

## Status

Phases 0–8 complete (environment, Go API, database & catalog, frontend, auth,
cart & orders, testing, CI, containerization). **Phase 9 — deployment and
continuous delivery** is in progress: the CD job and server runbook are
written and waiting on hosting. See the roadmap in
[docs/PROJECT_PLAN.md](docs/PROJECT_PLAN.md).

## License

This repository is public for **portfolio and evaluation purposes only**
(recruiting, code review, education). It is proprietary, closed-source
software — no permission is granted to use, copy, modify, or run this
code. See [LICENSE](LICENSE) for details.

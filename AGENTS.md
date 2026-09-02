# Repository Guidelines

## Project Structure & Architecture

Mountain Breath is a Go, React, and PostgreSQL e-commerce application. The Go API lives in `backend/`: `cmd/api` starts the service, `internal/api` contains HTTP handlers, `internal/domain` holds business rules, and `internal/store` owns SQL access. Database changes belong in paired `backend/migrations/*.{up,down}.sql`; development data is in `backend/seed/`.

The React/TypeScript application is in `frontend/`. Keep API requests in `src/api/`, reusable UI in `src/components/`, routes in `src/pages/`, and client utilities in `src/lib/`. Component tests sit beside source as `*.test.ts(x)`; Playwright tests are in `frontend/e2e/`. Deployment, observability, and Compose files are in `deploy/`; architecture and API contracts are in `docs/`.

## Build, Test & Development

Run Postgres and Mailpit locally with `docker compose -f deploy/docker-compose.dev.yml up -d`. Start the API with `cd backend; go run ./cmd/api` (or `air`) and the web app with `cd frontend; npm run dev`.

- `cd backend; go test ./...` runs all Go tests; add `-short` to skip Docker-backed store tests.
- `cd backend; go vet ./...; golangci-lint run` performs backend checks.
- `cd frontend; npm run lint && npm run build` lints, type-checks, and builds the client.
- `cd frontend; npm test` runs Vitest; `npm run e2e` runs Playwright and requires the dev database.

## Style, Tests & Contracts

Format Go with `gofmt`; keep domain code independent of HTTP and SQL. Use TypeScript with 2-space indentation, PascalCase components (for example `ProductCard.tsx`), camelCase functions, and `*.test.tsx` test names. `oxlint` and `golangci-lint` must pass. Add tests with every behavior change.

Keep money in integer minor units, never floats. User-facing text belongs in the locale catalogues, not inline code. For route changes, update `docs/api/mountain-breath.postman_collection.json`. UI should follow the design canvases in `docs/design/`; document intentional departures.

## Commits & Pull Requests

Use focused Conventional Commits such as `feat(cart): cap quantity by stock` or `fix(api): return validation fields`. Work lands on `dev`; batch PRs merge `dev` into `master` with a merge commit, not a squash. PRs should explain the user-visible change, link the relevant issue/decision, list verification commands, and include screenshots for UI changes. Never commit `.env` files; use the provided `.env.example` templates.

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

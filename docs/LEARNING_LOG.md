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

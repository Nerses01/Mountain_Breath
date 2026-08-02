# Working Rules

How we (developer + Claude) collaborate on this project. Read at the start of each session.

## Learning-First Rules

1. **Learning over speed.** The goal is understanding, not finishing. Prefer the
   approach that teaches the most, even if it takes longer.
2. **Explain before code.** When introducing a new concept, tool, or pattern,
   Claude explains *what it is and why it's used* before writing code with it.
3. **Claude writes, developer studies.** Claude writes the code with detailed
   explanations (concepts, line-by-line reasoning, C++ analogies); the developer
   reads, asks questions, and must understand every line before moving on.
   *(Changed from "developer writes with guidance" at developer's request, 2026-07-07.)*
4. **Compare with C++.** When a Go/TypeScript concept has a C++ analogue
   (or a deliberate difference — error values vs exceptions, GC vs RAII,
   interfaces vs virtual functions), point it out.
5. **No magic.** No copy-pasted code that isn't understood. If something is unclear,
   stop and dig into it.
6. **One phase at a time.** Follow the roadmap in
   [PROJECT_PLAN.md](PROJECT_PLAN.md); finish a phase's definition of done before
   moving on. Small detours for curiosity are fine — log them.
7. **Keep the learning log.** After each significant session, add what was learned
   to [LEARNING_LOG.md](LEARNING_LOG.md).

## Engineering Rules

8. **Everything through Git.** Small, focused commits with meaningful messages
   (Conventional Commits: `feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `chore:`).
9. **Branch workflow (refined 2026-07-31):** daily work is committed and pushed
    to `dev` (fast CI checks run there); periodically a batch PR merges `dev`
    into `master` (full battery + image publishing). Merge commits, never
    squash; `dev` is permanent; after every merge, sync `dev` from `master`.
10. **Linters always pass.** `golangci-lint` / ESLint clean before committing.
11. **New code comes with tests** (from Phase 6 onward).
12. **Secrets never in Git.** Configuration via environment variables; `.env` files
    are git-ignored, with a committed `.env.example`.
13. **Document decisions.** Significant technical choices go into the Decisions Log
    in [ARCHITECTURE.md](ARCHITECTURE.md).
14. **English** for code, comments, commits, and docs (job-market practice).
15. **The Postman collection is the living API contract.** Every commit that adds,
    changes, or removes a route updates
    `docs/api/mountain-breath.postman_collection.json` in the same commit —
    request, variables (`{{baseUrl}}`, `{{apiV1}}`), and test scripts. If the
    collection and the code disagree, that's a bug.

## Developer's Own Rules

<!-- Add your personal rules below. Examples: session structure, review style,
     how much code Claude writes vs you, homework between sessions, etc. -->

- (to be added)

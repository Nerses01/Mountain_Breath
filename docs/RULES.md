# Working Rules

How we (developer + Claude) collaborate on this project. Read at the start of each session.

## Learning-First Rules

1. **Learning over speed.** The goal is understanding, not finishing. Prefer the
   approach that teaches the most, even if it takes longer.
2. **Explain before code.** When introducing a new concept, tool, or pattern,
   Claude explains *what it is and why it's used* before writing code with it.
3. **The developer writes code too.** Where practical, Claude guides and reviews
   while the developer writes the code — especially for core learning topics
   (Go idioms, SQL, React hooks). Claude writes boilerplate/scaffolding.
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
9. **PR workflow from Phase 7 onward.** Feature branches + pull requests, even solo.
10. **Linters always pass.** `golangci-lint` / ESLint clean before committing.
11. **New code comes with tests** (from Phase 6 onward).
12. **Secrets never in Git.** Configuration via environment variables; `.env` files
    are git-ignored, with a committed `.env.example`.
13. **Document decisions.** Significant technical choices go into the Decisions Log
    in [ARCHITECTURE.md](ARCHITECTURE.md).
14. **English** for code, comments, commits, and docs (job-market practice).

## Developer's Own Rules

<!-- Add your personal rules below. Examples: session structure, review style,
     how much code Claude writes vs you, homework between sessions, etc. -->

- (to be added)

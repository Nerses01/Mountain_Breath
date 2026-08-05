# Working Rules

How we (developer + Claude) collaborate on this project. Read at the start of
each session.

*Consolidated 2026-08-05: four rules about explaining concepts had grown up
separately and said overlapping things, and one real rule (the design canvas)
was living only inside `PLAN_ERA_2.md`. Merged and renumbered — see the
mapping at the bottom if you meet an old number.*

## Learning first — why this project exists

1. **Learning over speed.** The goal is understanding, not finishing. Prefer
   the approach that teaches the most, even if it takes longer. Nothing below
   outranks this one.

2. **No magic.** No code that isn't understood — copy-pasted, generated, or
   otherwise. The developer must be able to explain every line before it is
   built on. If something is unclear, stop and dig into it; that detour *is*
   the work, not an interruption to it.

3. **Claude writes, developer studies.** Claude writes the code; the developer
   reads it, asks questions, and understands it rather than typing it from
   scratch.
   *(Changed from "developer writes with guidance" at the developer's request,
   2026-07-07 — he learns better from well-explained working code than from
   fighting syntax.)*

4. **Teach the concepts, not just the decisions.** Because of #3, the
   explanation *is* the deliverable, so it has a required shape:
   - **Explain before code.** A new concept, tool or pattern gets explained —
     what it is, why it is the right tool here — before code using it appears.
   - **Compare with C++.** Where a Go/TypeScript/SQL concept has a C++
     analogue, or a deliberate difference (error values vs exceptions, GC vs
     RAII, structural interfaces vs virtual inheritance, `IMMUTABLE` vs
     `constexpr`), say so.
   - **Close every response with "What you learn here"** — the concept named,
     why it was the right tool, and the C++ analogue where one exists. Not a
     summary of what changed; that is the rest of the response. One line when
     a response only rearranged things that already existed.

   The test for all three: could the developer explain it to someone else
   afterwards?
   *(Merged 2026-08-05 from three rules that kept restating each other. The
   response-closing format was added because the abstract versions were not
   producing it in practice.)*

5. **One phase at a time.** Follow the roadmaps — [PROJECT_PLAN.md](PROJECT_PLAN.md)
   for Era I (phases 0–11), [PLAN_ERA_2.md](PLAN_ERA_2.md) for Era II
   (E1–E10). Finish a phase's definition of done before moving on. Small
   detours for curiosity are fine — log them.

6. **Keep the learning log.** After each significant session, add what was
   learned to [LEARNING_LOG.md](LEARNING_LOG.md).

## Git

7. **Small, focused commits with meaningful messages.** Conventional Commits:
   `feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `chore:`.

8. **Branch workflow.** Daily work is committed and pushed to `dev` (fast CI
   checks run there); periodically a batch PR merges `dev` into `master` (full
   battery + image publishing). Merge commits, never squash; `dev` is
   permanent; after every merge, sync `dev` from `master`.
   *(Refined 2026-07-31.)*

9. **Claude never runs Git commands that change the repository.** No
   `git add`, `commit`, `push`, `branch`, `merge` or `checkout` — **staging
   included**. At each logical end of a piece of work, Claude states the
   suggested commit message (rule #7) and stops there; the developer stages
   and commits himself.
   *Reading* state — `git status`, `git log`, `git diff` — is fine, and is how
   Claude checks its own work.
   *(Added 2026-08-05. Git is part of what this project exists to teach, and
   every commit should be a deliberate act by the person learning it.)*

## Engineering

10. **Linters always pass.** `golangci-lint` and the frontend linter clean
    before committing.

11. **New code comes with tests** (from Phase 6 onward).

12. **Secrets never in Git.** Configuration via environment variables; `.env`
    files are git-ignored, with a committed `.env.example`.

13. **Document decisions.** Significant technical choices get a row in the
    Decisions Log in [ARCHITECTURE.md](ARCHITECTURE.md).

14. **English is the *source* language** — code, comments, commit messages and
    docs, as job-market practice.
    This is about the repository, not the product: since E1.5 the shop itself
    speaks English, Armenian and Russian. User-facing strings belong in the
    message catalogues, never inline in source.

## Contracts that must not drift

Two artefacts outside the code are authoritative. Both have the same failure
mode — they rot silently, and nothing fails until someone trusts them.

15. **The Postman collection is the living API contract.** Every commit that
    adds, changes, or removes a route updates
    `docs/api/mountain-breath.postman_collection.json` in the same commit —
    request, variables (`{{baseUrl}}`, `{{apiV1}}`), and test scripts. If the
    collection and the code disagree, that is a bug.

16. **The design canvas is the source of UI truth.** Colours, spacing, type,
    copy, layout and component structure come from
    `docs/design/Mountain Breath Store.dc.html` — read it rather than
    inventing a value. It is a working copy of the claude.ai/design project;
    refresh it with `DesignSync`, never hand-edit it.
    **Three standing exceptions**, because a static mock cannot express
    everything a running site needs:
    1. **Accessibility overrides the design.** Where a design value fails WCAG
       AA, the accessible value wins (E1 did this to the brand orange and to
       both muted inks).
    2. **States the mock never draws are ours to design** — focus, error,
       loading, empty, disabled, hover appear nowhere in it.
    3. **Requirements added after the mock** have no canvas guidance and are
       designed to fit it; the three languages were the first.

    Anything taken from the canvas needs no justification. Anything departing
    from it gets a line saying why, where it happened.
    *(Moved here from `PLAN_ERA_2.md` §6 on 2026-08-05 — it applies to all UI
    work, not just Era II, and a rule that lives inside one phase plan is a
    rule waiting to be missed.)*

## Renumbering map (2026-08-05)

| Old | New | Note |
|---|---|---|
| 1 | 1 | unchanged |
| 2, 4, 17 | **4** | merged — explain before code / C++ analogies / "What you learn here" |
| 3 | 3 | unchanged in substance |
| 5 | **2** | promoted next to #1; it is a learning rule, not an aside |
| 6 | **5** | now cites both roadmaps |
| 7 | **6** | |
| 8 | **7** | |
| 9 | **8** | |
| 10–15 | 10–15 | unchanged |
| 16 | **9** | moved next to the other Git rules |
| — | **16** | new here; moved in from `PLAN_ERA_2.md` §6 |

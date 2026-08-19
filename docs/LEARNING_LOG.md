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

## 2026-08-19 — Phase F2, part three: the customer's cancel button

**Worked on:** `POST /orders/{id}/cancel` — the customer's one
self-service transition, while the order is still `pending` (decision
#93). The admin's cancel and the customer's now share one transaction body
(`applyOrderStatusTx`: status write + history event + restock + promo
release), with different gates in front: the admin gets the machine's
rules, the customer gets ownership (existence-hiding 404) plus the
pending-only window (`ErrTooLateToCancel` → 409 `too_late_to_cancel`).
The cancelled letter goes out for self-cancels too, through the shared
`sendOrderStatusMail`. On the order page: the address book's armed-button
pattern (ask again, disarm after 3 s), pending orders only, with the 409
branch explaining "the hive already confirmed — get in touch" over the
freshly-refetched tracker. Canvas draws no cancel control, so the design
is ours (standing exception #2). ×3 locales.

**Learned:**
- *Permission is not transition* — the machine says confirmed → cancelled
  exists; policy says only the admin may drive it. Encoding the customer's
  window as its own domain function (rather than a second transition
  table) means the admin's arrows can grow without silently widening the
  customer's.
- *Check the policy under the same lock as the write* — the pending-only
  check runs under the cancel's own FOR UPDATE, so a concurrent admin
  confirmation cannot slip between "still pending?" and "cancel it".
  TOCTOU closed by construction, not by luck.
- *Extract the transaction body, keep the gates apart* — two doors, one
  room: `applyOrderStatusTx` holds the side effects both paths must never
  disagree on, while each caller keeps its own locking, validation, and
  error vocabulary.
- *Distinct sentinels earn distinct client copy* — `ErrTooLateToCancel`
  exists so the page can say "contact us" instead of a generic failure,
  while a stranger still sees only 404. Error design is UX design.
- *On a 409, refetch what refuted you* — the cancel hook invalidates the
  order caches on conflict: the truest thing to show under "too late" is
  the confirmed status that made it so.

**Questions / to revisit:**
- Should a cancelled-by-customer order be distinguishable from
  cancelled-by-admin in the history (an actor column on
  order_status_events)? Today both read "cancelled"; F4's webhook will add
  a third writer and may force the question.

## 2026-08-18 — Phase F2, part two: status-change emails (and the order learns its language)

**Worked on:** the second F2 item — confirmed/shipped/delivered/cancelled
now mail the customer through the existing Mailer. The prerequisite nothing
had recorded: WHOSE language? The trigger is the admin's request, so the
admin's negotiated locale is wrong by design. Migration 000024 snapshots
the checkout's locale onto the order (default `'en'` doubles as backfill);
`CreateOrder` widened from a bare `Currency` to `domain.View`; new
`GetUserByID` feeds the handler the email plus the `notify_order_updates`
toggle A5 shipped with exactly this sender promised (decision #87, honored;
decision #92 records the rest). Trilingual copy ×4 statuses, flagged for
F3's native review. No mail for `pending` (the confirmation letter is that
step) and none for payment flips.

**Learned:**
- *"In the order's locale" is a schema requirement, not a template
  argument* — a mail sent later than its trigger's request must read
  language from a recorded fact, and the only honest record is a snapshot
  at checkout, exactly like prices. Deferred sending changes where data
  must live.
- *The Parameter Object pays out on schedule* — View's comment promised
  "add a field without editing every signature again"; widening
  Currency→View cost one signature and a mechanical sed, not a redesign.
  The C++ analogue held: `const Options&` over growing positional args.
- *Preferences gate effects, not actions* — the toggle-off test pins that
  the status still changes and only the MAIL is suppressed; a preference
  that silently blocked the state change would be a bug wearing a
  setting's clothes.
- *Non-fatal side effects sit AFTER the commit and log instead of erroring*
  — same pattern as the confirmation mail, now stated twice, which makes
  it a rule: the transition happened; the letter about it is best-effort.
- *A fake graduates when there is something to observe* — UpdateOrderStatus
  was a bare ErrNotFound stub until a 200 had an observable consequence
  (the mail); then it started borrowing the domain's real transition table.

**Questions / to revisit:**
- The mail's link uses the legacy `/orders/{id}` path (A1's redirect covers
  it) — switch the mail builders to `/account/orders/{id}` someday, or keep
  legacy paths in emails that outlive URL schemes?

## 2026-08-18 — Phase F2 begins: the payment-status write path (Era III §1's top gap)

**Worked on:** the first Era III cut, exactly the one §4 nominated: E6
modelled `payment_status` as the orthogonal twin of order status and nobody
ever built the flip — a bank-transfer order could never become `paid`. Now:
a second tiny state machine in `domain` (`unpaid → paid → refunded`),
`store.UpdateOrderPaymentStatus` (same lock-validate-write skeleton as its
status sibling, minus every side effect), `PATCH /admin/orders/{id}/payment`,
mark-paid/mark-refunded buttons on the admin orders table, tests at all
three layers, decision #91.

**Learned:**
- *No backward arrows in money machines* — bookkeeping corrects a mistake
  with a compensating entry (the refund), never by erasing the fact, so
  `paid → unpaid` deliberately does not exist. Same reason the order
  machine has no undo; now stated as a principle rather than an accident.
- *Two orthogonal machines beat one merged enum* — a cancelled order can
  still owe a refund; combining parcel-state and money-state into one
  status would make that unrepresentable. C++ analogue: two small enums
  over one enum with every combination spelled out.
- *400 vs 409 is "not a word" vs "not from here"* — `ValidPaymentStatus`
  answers the request-shaped question (422-ish validation), the
  transition table answers the state-shaped one (409 conflict). Two
  functions because they back two different HTTP answers.
- *A behaving fake beats a stub when the brain lives in `domain`* — the
  fakeStore's payment method calls the same `ValidPaymentTransition` the
  real store calls, so handler tests walk 200/409/404 with zero duplicated
  logic. That's the payoff of the `api → domain ← store` layering, visible.
- *FOR UPDATE even for a two-statement flip* — two admins clicking
  paid/refunded at once could both read `unpaid` and both pass validation;
  the row lock serializes them. The checkout's reasoning at one row's scale.

**Questions / to revisit:**
- F4 will flip this same column from a provider webhook — does the webhook
  call the same store method, or does a provider event deserve its own
  recorded fact (a payments/events table) the way status transitions got one?

## 2026-08-18 — Phase A5: settings (canvas 10) — and the account plan closes

**Worked on:** migration 000023 (`full_name`, `phone`,
`notify_order_updates`), PATCH /account/profile (echoing the fresh
user), POST /account/password (current verified, other sessions
revoked), the notifications endpoints incl. the tokenless newsletter
unsubscribe, the canvas-10 screen (profile/password panels, language
links + currency display segments per #89, real+stub toggles via a new
`Switch`, the F2 delete stub), `lib/displayName` unifying identity in
the rail and header pill, strings ×3, tests at every layer. **PLAN_ACCOUNT
A1–A5 complete.**

**Learned:**

- **Two revocation policies, one sentence apart:** reset revokes ALL
  sessions (the password was suspect — the thief's cookie must die,
  even at the cost of the owner's); change-password revokes all EXCEPT
  the caller's (the owner is at the keyboard; logging them out punishes
  the right behaviour). The policy difference is the threat-model
  difference, recorded at the interface so neither gets "unified" later.
- **Wrong password ≠ unauthorized:** the failed bcrypt compare returns
  a FIELD error on current_password, not a 401 — the session
  authenticated fine; one input is wrong. HTTP status speaks about the
  request's standing, the body about the form.
- **Double opt-in forces a three-state toggle.** A boolean switch lies
  about the newsletter: between flip and inbox click the truthful state
  is "pending", so the API says none|pending|subscribed and the UI
  shows the middle state instead of pretending. Consent flows make
  binary UI dishonest.
- **A capability can have two keys.** Unsubscribe-by-token exists for
  the anonymous emailed link; unsubscribe-by-session exists because
  being signed in to the account IS proof of owning the email. Same
  row, two doors, each keyed to who is knocking.
- **`role="switch"` vs checkbox:** a switch takes effect immediately,
  a checkbox implies a form submitted later — the ARIA role encodes the
  timing contract, not the shape. Fourth popup/control pattern named in
  this project (combobox, disclosure, menu, switch), each chosen by JOB.
- **Echo the fresh resource from a PATCH** and the client can
  `setQueryData` instead of refetching — one round-trip saved, and the
  cache can never disagree with what the server just confirmed.

**Questions / to revisit:**
- F2: wire the delete button, and the status-change mailer must read
  BOTH `notify_order_updates` and the order's locale.
- The currency display preference is localStorage-only (#89) — promote
  to a users column only on a real two-devices complaint.

---

## 2026-08-18 — Phase A4: the addresses screen (canvas 09)

**Worked on:** migration 000022 (`addresses.leave_with_neighbour`,
decision #88), the flag through domain/store/API/book form, checkout
prefill widened to the full entry AND the checkout teaching the flag
back to the book, the canvas-09 card grid (note row, dashed add card,
two-step remove, pickup honest-stub), strings ×3, tests at every layer.

**Learned:**

- **Suggestion vs record, applied twice.** The address flag PREFILLS
  the checkout; the order still snapshots what was chosen. And the
  checkout writes the choice back to the book — prefill learns, the
  record freezes. Naming which columns are suggestions and which are
  records is what keeps the two from being conflated later.
- **A two-step arming button is a confirm without a dialog:** the same
  control re-labels to "Remove?" and disarms after three seconds — one
  tab stop, announced by its own text. Dialogs earn their cost when
  there is more to say than "are you sure?"; here there is not.
- **Testing time-and-state together needs BOTH act() and fake timers:**
  the timeout's setState is a React update, so advancing the clock
  outside act() leaves the re-render pending — and TanStack mutations
  run on microtasks, so asserting a DELETE right after click needs
  waitFor. Two different asynchronies, two different tools.
- **The mojibake trap is real and it is MINE too:** a PowerShell
  `Get-Content | Set-Content` one-liner to rename a translation key
  re-encoded the file through the console code page and turned every
  em-dash into `â€”` — exactly the CLAUDE.md seed-file warning. Caught
  by reading the diff; fixed by rewriting the file. Text transforms on
  UTF-8 files go through tools that never re-encode (Edit/Write), not
  through shell pipes.
- **Widening a return type beats a parallel endpoint:** DefaultAddress
  returning the entry (id/label riding along harmlessly) gave the
  checkout the flag with zero new routes — the consumer strips what it
  does not need, guarded by DisallowUnknownFields on the way back in.

**Questions / to revisit:**
- The checkout still shows its own neighbour checkbox; F5's address
  PICKER will need the per-address flag to swap in as the selection
  changes — wire that when the picker lands.

---

## 2026-08-18 — Phase A3: the wishlist screen (canvas 08)

**Worked on:** `saved_at` on the wishlist read (`domain.WishlistItem`
embedding Product; the DTO embedding productResponse for flat JSON),
`POST /wishlist/add-all` reusing A2's `ReorderResult` contract, the
canvas-08 frontend — `WishlistCard` (saved-ago via
`Intl.RelativeTimeFormat`), worth-total header, add-all with the shared
`ReorderReport` banner, dashed save-more slot, the price-drop-alerts
honest stub in the rail — strings ×3, tests at every layer.

**Learned:**

- **Embedding is the Go answer twice in one feature:** domain
  `WishlistItem{Product; SavedAt}` promotes every Product field (no
  is-a inheritance, just forwarding), and the DTO embeds
  `productResponse` so JSON marshals FLAT — a wishlist row is a product
  card plus one field, not a wrapper the client unwraps.
- **A shared result contract is worth more than a shared endpoint.**
  Add-all is not reorder, but returning the same `{lines: [...]}` shape
  meant zero new client vocabulary: same banner, same issue codes, same
  translations. Design the RESPONSE for reuse even when the requests
  differ.
- **Two queries can beat one parameter.** `saved_at` could have been
  threaded through the shared `productCards` helper — four callers,
  three of which would ignore the new column. A second tiny query in
  the one caller that needs it keeps the helper's signature honest.
- **Absent beats stubbed for per-item controls.** The screen-level
  alerts card can carry a "not wired yet" note; a per-card "Notify me"
  cannot — it would be twenty dead buttons. The deferral pattern
  scales by SURFACE: one honest card, zero fake buttons.
- **`Intl.RelativeTimeFormat` + fake timers:** the platform formats
  "2 weeks ago" in all three languages; the test freezes `Date` (and
  ONLY Date — faking all timers would stall the test's own promises)
  so "14 days before now" is a constant.

**Questions / to revisit:**
- Add-all picks a product's first variant by id, not the cheapest
  in-stock one — revisit if variant counts grow beyond royal jelly's
  three.

---

## 2026-08-18 — Phase A2: the orders screen (canvas 07)

**Worked on:** settled PLAN_ACCOUNT.md's remaining decisions (#2–#8, log
#85–#89) by explicit user confirmation; then A2 end to end — migration
000021 (`order_status_events` + backfill, cycle-tested), event inserts in
both order write paths, events + `has_cold_chain` on the order DTO,
`POST /orders/{id}/reorder` with per-line issue codes, Postman, and the
canvas-07 frontend: `OrderTracker`, filter pills, the featured-order
card, history rows with collapse, the reorder banner, the rail's reorder
card, `#MB-` display numbers, strings ×3. Tests at every layer.

**Learned:**

- **The cheapest event sourcing: keep the state column, add the log.**
  `orders.status` still answers "what is it now?" in one read; the
  append-only events table answers "how and when did it get here?". Full
  event sourcing would DERIVE state from the log; storing both costs one
  insert per transition and keeps every existing query untouched.
- **Record the transition in the transaction that causes it.** The event
  insert lives beside the UPDATE (and CreateOrder's event reuses the
  order row's own timestamp), so history and state cannot disagree —
  the same atomicity instinct as stock decrement inside checkout.
- **A backfill may only assert what it knows.** Old orders got exactly
  one synthetic `pending` event from `created_at`; their later steps
  happened but are unrecoverable, and the tracker shows a dash rather
  than fiction. Honesty is a schema property, not just a UI one.
- **Partial success as a RESULT, not an error.** Reorder returns 200
  with per-line fates ("reduced", "out_of_stock", "unavailable") — codes
  the client translates, the promo_issue contract again. HTTP status
  codes describe the request's fate; the body describes the domain's.
- **Ownership placement is per-method, not per-layer.** GetOrder leaves
  "may you look?" to the handler (admin may); Reorder answers it in the
  store because the check and the cart write must share a transaction.
  The interface comment records WHY the two differ.
- **`aria-pressed` vs tabs; `aria-current="step"`:** filter pills toggle
  a list in place (no panels → not tabs), and a tracker is an `<ol>`
  whose current step carries the spec's exact value for "you are here".

**Questions / to revisit:**
- Old orders' trackers show dashes for pre-021 steps — acceptable
  forever, or worth a one-off manual backfill by the family? (Data
  entry, not code.)
- The status chip's delivered green is the canvas's raw pair
  (#4C7A3D on #EAF2E3, AA-checked) — promote to tokens if a second
  green consumer appears.

---

## 2026-08-18 — Phase A1: the account shell (PLAN_ACCOUNT.md)

**Worked on:** the account canvas (07–10) arrived; wrote PLAN_ACCOUNT.md
(gap audit, 8 decisions, phases A1–A5); decided #1 (nest under
`/account/*`, decision log #84); built A1 — `AccountLayout` (rail +
guard + `<Outlet/>`), nested routes with `LegacyRedirect` for the old
paths, the header `AccountMenu` pill, pages stripped to panes, the old
AccountPage split into rail/AddressesPage/interim SettingsPage, strings
×3, component tests for guard/redirects/menu keyboard.

**Learned:**

- **A layout route is Template Method with the router doing the
  virtual dispatch.** The frame (rail, guard) is written once;
  `<Outlet/>` is the hole; children are chosen by URL, not by a call.
  The guard living in the layout deleted three per-page copies — the
  same de-duplication a base class buys, without inheritance.
- **NavLink over manual `pathname === href`:** active-ness is a PREFIX
  question (`/account/orders/42` should light "My orders"), and NavLink
  answers it and sets `aria-current` from one source. The header's
  manual comparison would have said "nowhere".
- **Three popups, three ARIA patterns, chosen by JOB:** PillSelect is a
  combobox (picks a value, points with `aria-activedescendant`), the
  mobile nav is a disclosure (shows a region, no focus trap), the new
  AccountMenu is a menu (list of commands, arrows move REAL focus —
  roving tabindex). The role dictates the keyboard contract; right role
  + wrong keys is worse than divs.
- **Refs are populated at commit, not at `setState`.** "Open on the
  last item" computed `items.length - 1` before the menu existed —
  length 0, focus lost, caught by the test. The index math had to move
  inside the `requestAnimationFrame` with the focus call: state now,
  DOM later.
- **A count is a thing that arrives.** The rail test asserted the badge
  the instant the pane rendered; the orders query was still in flight.
  `waitFor` is the test-side acknowledgment that queries resolve
  independently.

**Questions / to revisit:**
- Decisions #2–#8 in PLAN_ACCOUNT.md §2 are still open; #3 (status
  history) gates A2 and gets more expensive the longer it waits.
- The interim SettingsPage shows profile facts read-only until A5.

---

## 2026-08-15 — Phase E10: the audit phase, and the end of Era II

**Worked on:** the mobile chrome (nav sheet, filter drawer, sticky
add-to-cart, responsive type); seven CI-blocking axe scans and the fixes
they forced; the 375px and keyboard-only purchase journeys; `usePageMeta`,
JSON-LD, the backend sitemap, robots.txt; Lighthouse CI with budgets; the
k6 re-baseline and the JIT finding; cache headers. Era II's last phase.

**Learned:**

- **An audit you run once decays; a gate compounds.** The phase's whole
  design was turning "audited" into executable statements: axe scans that
  fail the build, a purchase journey that never calls `click()`, Lighthouse
  budgets in CI. The gates paid for themselves immediately — six real
  violations on day one, several of them MINE from earlier phases.
- **Measure contrast against the worst surface a token can sit on.** E1
  measured against `panel`; the hero sits on `page`, which is darker, and
  two tokens quietly failed there for eight phases. Same class of lesson as
  the WCAG large-text exemption: it starts at 18.66px BOLD, not "17px looks
  big enough" — read the spec's numbers, not its vibe.
- **The bottleneck was a compiler, not a query.** The plan predicted the
  facet counts; k6 measured them at 113ms p50 and pointed instead at every
  price read: Postgres was JIT-compiling 49 LLVM functions per query
  (~250ms, never cached) because the effective-prices view's NUMERIC math
  inflated the cost estimate. `jit = off` for the app's pool — one runtime
  parameter — took p95 from 3,090ms to 11.6ms. Predictions are for
  deciding WHERE to measure; only the measurement decides what is true.
- **Deciding what NOT to build is E10 work too.** No image pipeline (there
  are no images yet — tuning srcset against designed placeholders is
  optimizing blind), no catalog cache (k6 says 6ms average; a cache without
  a measured need is an invalidation bug on layaway), no bottom-sheet
  summary (the stacked layout already serves; a fourth disclosure pattern
  for one screen is complexity, not fidelity).
- **Overlay semantics follow coverage, not style**: in-flow disclosure
  (nav sheet — page stays interactive, no focus trap owed) versus dialog
  (filter drawer — covers the page, so Escape/focus-return are owed).
  Naming the rule once beats re-deriving it per overlay.
- **Frameworks hide invalid HTML until something else surfaces it.** The
  nested checkout forms (E7's PromoBox inside E6's form) rendered fine and
  passed every test — a React console warning in the e2e logs was the only
  witness. The `form` attribute exists precisely for "the submit button
  lives outside its form".

**Questions / to revisit (the Phase 11 handoff):**

- Prerendering/SSR for meta that reaches non-rendering crawlers — behind
  hosting, with the image pipeline behind photography.
- The `agentic-browsing` Lighthouse category (new, unasserted) — watch it.
- The k6 SLO could tighten (p95 11.6ms vs the 200ms threshold) once the
  stack runs on the real host rather than a dev machine.

**Era II is complete.** Ten phases (E1–E10 plus E1.5): the design became a
trilingual, dual-currency shop with a real checkout, promotions, accounts,
content, and now the hardening to stand behind it. What remains before
customers is Phase 9's blocked hosting — and photographs.

---

## 2026-08-15 — Phase E9: content without a CMS, consent despite the robots

**Worked on:** migration 000020 (`newsletter_subscribers`, double opt-in);
subscribe/confirm/unsubscribe endpoints with the trilingual confirmation
mail; the markdown pipeline (glob import, `marked`, hand-rolled frontmatter
and `.prose` styles); 27 content files — six pages and three journal posts
in three languages; the journal list and post pages; the emailed-link
landing pages; the footer form gone live; every waiting nav and footer link
wired; the link-resolution e2e journey.

**Learned:**

- **A bundler can be a content pipeline.** `import.meta.glob` with `eager`
  and `?raw` turns a directory of markdown into an object in the bundle —
  "compiled at build time" without any build step to maintain. A missing
  translation becomes a missing KEY, the same failure shape as the message
  catalogues, so the same per-file English fallback covers it. The whole
  CMS is forty lines and a naming convention.
- **Know when a format needs a parser and when it does not.** Markdown got
  a dependency (`marked`) because its grammar is genuinely recursive and a
  homemade parser fails on real documents. Frontmatter did NOT: three
  `key: value` lines between fences is a ten-line loop, and importing a
  YAML engine for it would be the same trap pointed the other way. The
  skill is telling the two cases apart, not a blanket rule.
- **Consent flows are adversarial against robots, not people.** Mail
  scanners prefetch GET links — so the confirm link cannot be a mutating
  GET (the plan's original shape, deliberately abandoned). Some scanners
  execute JavaScript — so the landing page cannot auto-POST on load either.
  The only click a robot never makes is a form submission, which is why the
  button IS the consent, and why a test asserts zero requests fire on load.
- **Not every token is a secret with a fuse.** After three uses of
  raw-token/SHA-256-with-expiry-and-single-use, the newsletter token broke
  the habit on purpose: it is a permanent capability, because the
  unsubscribe link in a year-old email must still work. Recognizing which
  properties of a pattern are essential (hashing at rest) and which were
  contextual (expiry, single use) is what stops a pattern from becoming a
  cargo cult.
- **Lifecycle as timestamps beats lifecycle as enum.** `confirmed_at` +
  `unsubscribed_at` made every rule a one-line SQL predicate — including
  the two subtle ones: a live re-subscriber must get no mail (and keep
  their token), and a returning unsubscriber is a new consent on a fresh
  token. An enum would have stored the conclusion and lost the evidence.
- **Unsanitized HTML can be a correct decision if you write down its
  boundary.** Repo-authored markdown needs no DOMPurify — its authors can
  already commit code. The comment in `Markdown.tsx` states the tripwire
  (any database- or form-sourced content) so the decision cannot silently
  outlive its premise.

**Questions / to revisit:**

- Actually SENDING a newsletter (composing an issue to all live
  subscribers) is deliberately unbuilt — the list is collected with
  consent, the sender is a Phase 11 conversation.
- Native review of ~4,500 words of new Armenian/Russian content copy — the
  legal pages especially.
- If the family ever needs to edit pages without a commit, decision #3's
  markdown migrates to DB-backed pages without URL changes; the journal
  would go first.

---

## 2026-08-15 — Phase E8: accounts, tokens, and the flows that leave the app

**Worked on:** migration 000019 (`wishlist_items`, `password_reset_tokens`,
`oauth_identities`, address labels); the `mail` package with Mailpit in the
dev stack; the full password-reset loop; keep-me-signed-in TTLs; login rate
limiting; the wishlist with save-for-later; the account page's address book;
Google OAuth against a fake and the Apple stub; the two-panel sign-in,
forgot/reset pages, hearts everywhere; the account e2e journey.

**Learned:**

- **One pattern, three tables.** Sessions, reset tokens and (in E9's plan)
  newsletter confirmations are the same design: the client holds a raw
  random value, the database holds its SHA-256. What varies is the policy
  bolted on — sessions expire late, reset tokens expire in an hour and die
  on first use, and consuming one revokes the sessions too. Recognizing a
  pattern means the third use costs a comment, not a design session.
- **Absence of information is information.** Three separate rules exist to
  say nothing: forgot-password answers 204 for strangers and members alike,
  a dead reset token never says WHICH kind of dead it is, and E7's
  "promo_unknown" covers disabled codes. All are the same idea — an error
  channel is an oracle unless you flatten it — and all trace back to
  login's identical-errors rule from Phase 4.
- **OAuth is three redirects and one back-channel call.** Naming the legs
  demystified it: state cookie out (CSRF proof), one-time code back, the
  SERVER swaps code for token (the browser never sees the secret), userinfo
  over our own TLS connection — which is exactly why no JWT signature
  verification is needed: the JWKS dance exists for tokens you did not
  fetch yourself. The subject is the identity; the email is a one-time,
  verified-only linking hint, because providers let people change email.
- **An empty bcrypt hash is a lock with no key.** OAuth-born accounts store
  `password_hash = ''` and bcrypt refuses to match anything against it —
  password login fails closed with zero special-case code, and
  forgot-password doubles as "add a password to my Google account". The
  cheapest possible design fell out of understanding the library instead of
  adding a nullable column.
- **The dev sink should be the production path.** Mailpit speaks real SMTP,
  and building against it (not a mock) is what surfaced RFC 2047: an
  unencoded Armenian subject line arrives as mojibake. A `t.Log`-style
  fake mailer would have shipped that bug to the first real relay.
- **Rate limiting is a keying problem more than an algorithm problem.**
  Fixed-window vs token bucket mattered far less than (IP, email) as the
  key — either half alone is a lockout attack or a botnet pass — and than
  running the check BEFORE bcrypt, so guessing cannot spend the shop's own
  hashing time.
- **A grain change can be a feature.** The cart line (variant × qty) becomes
  a wishlist row (product) in one `DELETE … RETURNING` transaction — the
  transfer drops precision on purpose, because later-you wants to remember
  the jar, not the Tuesday's quantity.

**Questions / to revisit:**

- Going live with Google needs the OAuth client created in the console and
  its redirect URI updated when Phase 9 lands a real domain.
- The limiter is per-process; if the compose stack ever runs two API
  replicas, it moves to storage (the caveat is written at the type).
- The account page could later let the checkout PICK among saved addresses
  (`address_id`); today it prefills the default only — backlog, noted in
  the plan.
- Native review of the new Armenian/Russian copy (auth screens + the two
  email templates) — the reset email especially, since it lands in
  strangers' inboxes.

---

## 2026-08-15 — Phase E7: one calculator, and locks that make promises

**Worked on:** migration 000018 (`promo_codes`, per-market
`promo_code_values`, `promo_redemptions` with its unique index,
`cart_promos`, the order's discount split); `domain.Price` as the one pure
calculator (deleting `ComputeTotals` and `QuoteCart`); the hive club derived
from order history instead of stored; `POST /checkout/preview`,
`POST|DELETE /cart/promo`, hive standing on `/auth/me`; the redemption race
test; the designed cart page with the honey banner, progress bar, upsell CTA
and promo box; discount lines on summary and receipt; the member badge.

**Learned:**

- **A pure function is the load-bearing wall of pricing.** `domain.Price`
  takes plain values and returns the whole breakdown — the C++ instinct is
  `constexpr`: no I/O, no hidden clock (`now` is a parameter precisely so a
  test can hold time still at an expiry boundary). Cart, preview and the
  checkout transaction call the same function with differently-sourced
  inputs, so "every screen agrees to the dram" is a property of the design,
  and the table test gets to pin nine interacting-rule cases without a
  database.
- **Two kinds of impossibility, and when each applies.** Once-per-customer
  is a UNIQUE index — a fact of the storage, like a `static_assert`, immune
  to check-then-insert races by construction. The global cap is a COUNT, and
  a count cannot be an index, so it is enforced the way stock is: a `FOR
  UPDATE` lock on the promo row and a count taken under it. Knowing which
  tool fits which invariant is the actual lesson; the race test (ten
  checkouts, one code, one winner, nine clean refusals) is its proof.
- **Every new lock re-opens the deadlock proof.** E2 taught it with
  `sales_count`; E7 recites it: user row → cart variants (ascending) →
  promo row → products (ascending), every transaction in the same order.
  The user-row lock was not in the plan — it exists because "how many orders
  does this customer have" became money (the first-delivery perk), and two
  parallel checkouts by one customer could both read zero.
- **Derive what is derivable.** The design's own copy defines the hive club
  as having an account, so membership is `count(non-cancelled orders)` — no
  tier column, no second copy to drift (E5's `price_minor` lesson, avoided
  in advance this time). Corollary: the perks needed no admin UI, no
  enrollment flow, and cancelling your first order honestly makes your next
  one "first" again.
- **A discount changed who may answer "what does delivery cost".** The cart
  response lost its shipping and total: they now depend on the CUSTOMER and
  the code, so a contents-only quote was E6's "Total" lie one phase later.
  Removing fields from a response is also a contract change — cart tests,
  Postman and the frontend moved in the same commit, which is what rule #15
  is for.
- **Validity is a moment, not a property.** An applied code is one row of
  cart state; whether it WORKS is re-judged on every read and once more
  under lock at charge time. The three verdicts have three audiences: apply
  answers the form field, preview names the issue beside the code, checkout
  refuses with 409 — because silently repricing an approved total is worse
  than asking the customer to look again.
- **Error text can be an oracle.** "promo_unknown" deliberately covers
  disabled and not-yet-started codes: distinguishing them would let anyone
  enumerate which codes exist before their campaign starts — the 404-not-403
  reasoning, wearing a promo box.
- **VAT follows the money that moves.** With discounts real, the contained
  tax is carved from `subtotal − discount`: the discounted price IS the
  price, and its receipt line must contain the tax of what was actually
  paid.

**Questions / to revisit:**

- Promo codes have no admin CRUD — the family edits SQL (or the seed) until
  a phase needs the form. Fine for three codes; revisit when it is not.
- The upsell suggests the cheapest gap-closing product; a smarter pick
  (benefit overlap with the basket, like E3's related fallback) is a
  one-query change if the banner earns it.
- Percent promos and the member 8% stack additively on the shelf subtotal.
  If the family ever wants exclusive codes ("not combinable with member
  pricing"), that is a `stackable` boolean on `promo_codes` and one branch
  in `Price` — noted so the conversation starts from the right place.

---

## 2026-08-14 — Phase E6: the server owns every number

**Worked on:** migration 000017 (`addresses`, order snapshot columns, the
five-figure money breakdown with its balance CHECK, `payment_method`/
`payment_status`, per-currency `shipping_rates`); `POST /orders` with a real
body; `GET /orders/{id}`; the cart's shipping quote; the designed checkout
screen with its own chrome, hand-rolled validation and the shared
`OrderSummary`; the confirmation page; the Playwright journey extended
through an address.

**Learned:**

- **A snapshot is a design pattern, and this was its third application.**
  Prices froze in Phase 5, the currency in E5, the address now — same
  sentence every time: an order is a closed record, and nothing the customer
  can later edit may reach into it. The tell that it is a pattern and not a
  trick: the address table and the snapshot columns hold the same seven
  fields and there is deliberately NO foreign key between them. The test
  edits the address book after checkout and asserts the order did not move.
- **A CHECK constraint is the database's assert().** `subtotal + shipping −
  discount = total` is enforced on every row, forever, including rows written
  by a future bug or by someone in psql at midnight. It caught its first
  victim the day it landed — a hand-written test fixture inserting
  `total_minor = 1000` over defaulted zeros. That is the constraint working,
  not the constraint being annoying.
- **Contained VAT is division by 120, not multiplication by 20%.** "Prices
  include VAT" means the shelf price already holds the tax, so the invoice
  figure is `subtotal × rate / (100 + rate)` — a 120-dram subtotal contains
  20 of tax, where the naive `× 20%` says 24 and overstates by a sixth.
  Integer arithmetic, round-half-up, one function owning the rounding so
  every invoice line agrees.
- **"The client never sends money" has a stronger form than ignoring.** The
  request struct simply has no money field, and `DisallowUnknownFields`
  turns a body that smuggles `total_minor` into a 400 before any handler
  code runs. Refused beats ignored: an ignored field lets the client believe
  it worked. The same reasoning keeps card numbers out of the API entirely —
  a stub that stores card data buys PCI scope for nothing.
- **Method and status are different facts.** A bank transfer is `confirmed`
  before it is `paid`; a cash order is `delivered` at the moment it stops
  being `unpaid`. Folding payment into the order state machine would have
  encoded a false dependency; two columns encode the truth.
- **Fees are shelf prices too.** `shipping_rates` is keyed (method,
  currency) with no conversion fallback — E5's argument applied to delivery:
  the family sets 1,900 ֏, not $4-times-a-rate. And the free-shipping
  threshold waives only the BASE; the mock's own cart charges chilled
  shipping past $70, because the cold box costs real money either way. Read
  §1.4 closely enough and the design is a requirements document.
- **The quote and the charge must be one arithmetic.** `QuoteCart` (cart
  page, checkout sidebar) and `ComputeTotals` (the transaction) share
  `ShippingFor`; an integration test places an order and fails if the number
  the customer read differs from the number charged. Two implementations of
  money arithmetic is how the "but the cart said…" ticket is born.
- **404 and 403 trade places when the resource is private.** E4 gave a
  non-purchaser 403 — the product is public, only the action was denied.
  Someone else's ORDER is 404: a 403 would confirm to an id-enumerator that
  the order exists and is somebody's, which is the very fact being fished
  for. The pair is now the project's rule of thumb.
- **Validation can mirror across the wire.** The client checks presence with
  the server's own field keys (`address.postal_code`), so local errors and
  server errors land on the same inputs through one rendering path; the
  richer rules (cash-is-AMD-only) come only from the server, which is the
  authority anyway. No form library needed yet — the plan asked for the
  decision to be noted, and the note lives on the CheckoutPage doc comment.
- **A shared SQL constant can hide a query that only breaks in its variant
  shape.** `orderColumns` served four single-table reads perfectly and broke
  the fifth — the admin listing JOINs `users`, whose own `id` made every
  bare column ambiguous (42702), and /admin/orders 500ed in the running
  shop. Aliasing the *other* table does not fix ambiguity (an alias renames
  it; its columns stay in the namespace) — qualifying the shared columns
  does, and costs nothing in the single-table queries. The suite missed it
  for a nameable reason: handler tests run on the fake store, which cannot
  mis-parse SQL, and no integration test called `ListAllOrders`. Fourth
  phase running where the defect that mattered surfaced only in the running
  app — but this one's lesson is sharper: every store method with its OWN
  SQL shape needs its own integration test, however thin.
- **Testing Library normalizes the DOM's text but not your matcher.** The
  price's non-breaking space collapses to a plain space in the element's
  normalized text, so `getByText('15,300 ֏')` finds nothing while
  `getByText('15,300 ֏')` (plain space) matches. An hour of "but the string
  IS in the innerHTML" went into that sentence.

**Questions / to revisit:**

- The E7 discount slots are already plumbed (`discount_minor` in the CHECK,
  the parameter in `ComputeTotals`, the row in the receipt) but always zero.
  E7's pure `domain.Price` calculator should REPLACE the inline arithmetic in
  `CreateOrder`, not sit beside it — one calculator for every screen is that
  phase's own headline.
- `GET /account/address` returns one default. When E8 builds the account
  page, the book becomes a list (`is_default` is already modelled); the
  checkout then grows an address picker over `address_id`.
- The step indicator marks steps 1–2 together on the single checkout page.
  If a future phase splits the page into real steps, the indicator's state
  model is already there (`step: 1 | 3`).
- Order emails ("your order is in") are conspicuously absent — E9's
  newsletter infrastructure is probably where mail enters the project.

---

## 2026-08-11 — Phase E5: money is harder than a multiplication

**Worked on:** migration 000016 (`currencies`, `variant_prices`, `fx_rates`,
the `variant_effective_prices` view, `orders.currency`/`fx_rate_used`, and the
removal of `product_variants.price_minor`); currency negotiation at the edge;
per-market catalog filtering, sorting and facet bounds; dual prices on the
card, the buy box and the cart; the footer's "USD / AMD" switcher; the admin
editor's price box per market.

**Learned:**

- **A price is not a property of a product.** It is a property of a
  *(product, market)* pair. Every awkward thing about the old schema
  dissolved once that was taken literally, and everything the phase built —
  the join table, the view, the map in the JSON — is one sentence of
  consequence.
- **Per-market prices, not live conversion.** A shelf price is a business
  decision: a shop picks a round figure and holds it. Conversion would move
  the price tag between two page loads and print 6,743 ֏ where a human would
  write 6,700. The rate still earns its table — as the *fallback* for a
  market nobody has priced yet, and as the record that keeps an old order
  reportable next year at the rate that was true then.
- **`price_minor / 100` is a bug, not a convention.** The scale of a minor
  unit belongs to the currency. USD has two decimals; a dram has none in
  circulation, so its minor unit IS the dram. Every `/100` in the codebase
  was a hidden assumption that there was only ever one currency, and the
  compiler could not see a single one of them.
- **Summing and converting do not commute.** Convert a total and you round
  once; total the conversions and you round per line — and three lines that
  each round up leave a total that disagrees with the numbers the customer
  just added up on screen. Per-market integer prices make the disagreement
  impossible rather than small. The domain test picks fixture prices that are
  deliberately *not* a fixed multiple, and fails if they ever become one,
  because a test where both answers agree proves nothing.
- **Currency is not a display concern.** The price slider's ends, the
  `price_asc` ordering and `min_price`/`max_price` are all denominated in it,
  so the *cheapest product can genuinely differ between markets*. Treating it
  as formatting would have produced a correctly-shaped answer to the wrong
  question, silently — which is the same failure mode as E1.5's missing
  `?lang=`.
- **Degrade on reads, refuse on charges.** A card with no dram price shows
  one line instead of two; `CreateOrder` returns `ErrPriceUnavailable` in the
  same situation. The asymmetry is the rule: the alternative to failing at
  checkout is billing someone zero.
- **Define it once, in SQL.** Seven callers ask "what does this variant cost
  in that currency?". `variant_effective_prices` is a plain view, so Postgres
  inlines it and pushes `WHERE currency = $1` down into it — the same instinct
  as E2's shared Go constants, one layer lower, where the database can hold
  the definition.
- **`ON currencies ((TRUE)) WHERE is_base`** — a unique index on a *constant*
  over a filtered subset means "at most one row can have this flag". The
  singleton form of E3's one-primary-image index.
- **A decimal in JSON should be a string.** `fx_rate_used` is
  `NUMERIC(18,8)`; `JSON.parse` turns every JSON number into a double. Sending
  the digits as text is the only way the exact value survives the trip.
- **When you must duplicate a set across layers, make the duplication
  testable.** Go needs the currency codes (to reject `?currency=ZZZ` without a
  query); SQL needs the properties (that is where rounding happens). E1.5 met
  the same problem for locales and wrote a comment asking future readers to
  keep three places in sync. A comment is a wish. `TestCurrenciesMatchTheDatabase`
  is the version that fails a build.
- **Language and currency are not the same shape of problem.** `/hy/shop` is
  a different *document* — different `<html lang>`, separately shareable,
  separately indexable — so the locale belongs in the path. A currency is a
  lens on the same document, so it belongs in storage. It also needs a cookie,
  because the server is the one that decides what a checkout charges.
- **`Intl.NumberFormat` knows too much.** With `style: 'currency'` it takes
  symbol placement from the *display locale*, so 6,700 drams renders "֏6,700"
  for an English reader and "6 700 ֏" for an Armenian one — a price tag that
  changes shape with the site language. Intl still does the hard part
  (grouping, decimal places) against a pinned locale; the symbol is placed
  from the currency's own row.
- **"from {{price}}" is a suffix in Armenian** (`{{price}}-ից`). The `Price`
  component takes a *callback* rather than a prefix string for exactly that
  reason — the message decides where the word goes, not the component.
- **Piping a file through PowerShell re-encodes it.** `Get-Content seed.sql |
  … psql` runs the stream through the console code page, and non-ASCII either
  double-encodes (`4 Ã— 100 g`) or is destroyed outright (46 Armenian
  characters became 46 `?`). Nothing errors — the result is valid UTF-8, just
  wrong. `docker compose cp` moves raw bytes and has no encoding step to get
  wrong. Found by *looking at a rendered page*, not by a test, which is now
  three phases running.

**Questions / to revisit:**

- The FX rate is a bootstrap row in the migration. A real shop wants a daily
  feed writing a new `as_of` — trivial to add, and worth doing before the
  fallback path prices anything a customer actually buys.
- `rounding_step` only applies to *converted* prices right now. If E7's promo
  discounts land as percentages, the same rounding question reappears for
  computed amounts and the answer should come from the same column.
- Nothing yet reads `variant_effective_prices.is_converted`. The admin screen
  is the natural consumer: an editor ought to see which figures the shop chose
  and which the exchange rate chose.
- `sales_count` is currency-blind — one counter across both markets. Fine for
  ranking; wrong the moment anyone wants revenue per market, which needs the
  order snapshots and `fx_rate_used` rather than a counter.

---

## 2026-08-11 — Phase E4: keeping a denormalized aggregate honest

**Worked on:** migration 000015 (`reviews` with a status workflow and a
UNIQUE per person per product; `products.rating_avg`/`rating_count`);
verified-purchase enforcement; the moderation queue; `sort=rating`; an
accessible `Stars` used by the card, the detail and every review row; the
Reviews tab the design draws and E3 deliberately left empty.

**Learned:**

- **Recompute, don't nudge.** The tempting version of a stored average keeps
  a running total and adjusts it by each review's delta. It is cheaper and it
  drifts — and worse, it is *order-dependent*: publish, reject, re-publish
  has to land back on the same number, which incremental arithmetic does not
  guarantee. Reading one product's reviews and recomputing is bounded, exact,
  and idempotent. The test walks that whole lifecycle for exactly this reason.
- **Why a stored average and not `sum` + `count`.** The pair is exact by
  construction and updates incrementally — and cannot be indexed for
  `ORDER BY rating_avg`, which is the only reason to denormalize at all.
  Knowing *why the obvious better idea is wrong here* is the actual lesson.
- **`avg()` over zero rows is NULL, not 0.** Reject a product's last review
  and the `NOT NULL` column rejects the write unless you `coalesce`. SQL's
  "no rows" and "zero" are different answers to different questions.
- **A default value can be a security control.** `status` defaults to
  `pending`, so forgetting to moderate fails CLOSED: the worst outcome of a
  bug is an unpublished review, not an unmoderated one on the storefront.
  The public endpoint pins the status server-side for the same reason —
  `?status=pending` must not be a thing a URL can ask for.
- **A constraint beats a check under concurrency.** "Have you already
  reviewed this?" asked in application code passes for *both* of two
  simultaneous submissions. `UNIQUE (product_id, user_id)` cannot.
- **403 and 404 answer different questions.** A non-purchaser gets 403: the
  product exists and we know who is asking; what is missing is standing. A
  404 would tell the client the product had vanished.
- **`can_review` is a hint, not a permission.** It exists so the UI does not
  reimplement the rule and guess — and the store checks the rule again,
  because anyone can POST.
- **Fractions and floating point, again.** `(4.67 / 5) * 100` is
  `93.39999999999999`, and that whole string was going into a style
  attribute on every card. Same family as the money rule; caught by a test
  that asserted the exact width.
- **A partial fill beats half-star rounding.** Drawing five outlines and
  clipping a filled layer to the exact percentage means 4.67 renders as 4.67
  — no rounding, no per-star branching, and any precision the backend picks
  renders correctly.

**The decision the plan left to me, and why it went the boring way.** The
plan said "Most loved" could now mean sales or rating — pick one. Rating is
the more literal reading and the wrong answer: an average over few reviews is
violently unstable, so as the *default* sort one five-star review would
outrank a jar that has sold 148 times, and the home page would reshuffle
every time anyone submitted anything. Ranking honestly by stars needs a
Bayesian prior — weight each average toward the catalog mean by how few
ratings it has — which is real work a six-product shop cannot justify. So
there are two sorts, each meaning exactly what it says, and the cheap half of
the prior (tie-break by `rating_count`) is in the ORDER BY. The choice is
pinned by a test, because it lives in a constant and would otherwise look
like an accident.

**A smaller thing worth keeping.** An orphaned `api.exe` from earlier in the
session was still holding :8080, so the browser verification would have
tested pre-E4 code against post-E4 expectations. The harness reported its
task as dead; the port disagreed. Checking the port rather than the task
status is what caught it — the same lesson as before, arriving unprompted.

**Questions / to revisit:**

- Reviews are moderated by hand. At what volume does that stop scaling, and
  is the first automation a spam heuristic or just "auto-publish 4★+ from
  verified purchasers"?
- `rating_avg` is recomputed synchronously inside the moderation transaction.
  Fine at this size; at a million reviews the recompute is the slow part of
  publishing. Where would that move — a trigger, a queue, a materialised view?
- The review form is shown only when `can_review`. Someone who has bought the
  product but is signed out sees nothing explaining why. Worth a "sign in to
  review" prompt in E8, when the account area is built?

---

## 2026-08-10 — Phase E3: modelling editorial content, and two ARIA patterns

**Worked on:** migrations 000011–000014 (`product_images` + alt translations
with a partial unique index; `product_highlights` / `product_usage_cards`
keyed by locale; product metadata split invariant/translatable;
`product_related`); detail-only store reads; `GET /products/{slug}/related`
with a computed fallback; admin endpoints for gallery, copy and curation; the
seeded editorial content; the rebuilt product page with a keyboard-navigable
gallery and hash-linkable tabs; the admin's content editor.

**Learned:**

- **A "translation table" is not one shape.** Decision #6 splits
  locale-invariant fields from prose. For a highlight bullet there IS no
  invariant field, so the split degenerates into a parent row holding only a
  `sort_order` — the same principle, applied honestly, produces a different
  table. Images went the other way for the same reason.
- **A PARTIAL unique index is "unique among SOME rows".** `UNIQUE (product_id,
  is_primary)` would forbid two non-primary images; `UNIQUE (product_id) WHERE
  is_primary` forbids two heroes and nothing else. Postgres has no
  constraint syntax for it — the index *is* the constraint.
- **A constraint changes the code that writes to it.** Because the index
  rejects the intermediate state, "set a new hero" has to clear the old flag
  first, in the same transaction. Verified by writing directly to the table
  in a test and asserting the database says no.
- **Fallback has a UNIT, and picking the wrong one interleaves languages.**
  Names fall back per field; bullet lists fall back per LIST, because a
  per-row fallback would put an English bullet in the middle of an Armenian
  panel. That choice follows directly from the rows being keyed by locale.
- **The ARIA tabs pattern is one tab stop, not N.** Roving `tabIndex`
  (`0` on the selected, `-1` on the rest) is what stops a five-image gallery
  costing a keyboard user five presses. Selection follows focus, and focus
  has to be moved imperatively — React state repaints the highlight and
  leaves the browser focused on the old element.
- **A fragment identifier means "a position within this document"**, which is
  exactly what a tab is — so the tab belongs in the hash, not a query param,
  and `replace: true` keeps three tab clicks from becoming three history
  entries.
- **A plan's rule can be dead on arrival against real data.** "Related =
  same category by popularity" is the standard rule and returns nothing in a
  catalog with one product per category. Shared benefits is both what works
  and what the panel actually claims.

**The recurring lesson, in a new costume.** E2's was "a green suite is not a
working app". E3's is narrower and sharper: **an admin tool has to know the
difference between what it stores and what it computes.** The related-products
picker read the storefront's endpoint, which answers "what should this panel
show" — curated list *or* computed fallback. Pre-filled from that, saving
would silently freeze a dynamic panel into a static one; left empty (as it
first was), one click of Save would wipe an existing curation. Neither
failure produces an error. The fix was a second, narrower question the API
can be asked — `?curated=true` — and it only surfaced because the editor was
opened in a browser against a product that was actually curated.

**Questions / to revisit:**

- The editor reads through the PUBLIC product endpoint, one locale at a time.
  That keeps the admin seeing what a shopper sees, fallbacks included, at the
  cost of a request per language tab. If the catalog grows, is an
  admin-shaped "every language at once" read worth the second resolution path
  it would create?
- `products.image_url` still exists and is still what the shop grid reads,
  even though the gallery now owns the images. Migration 000015 drops it —
  which read paths have to move first?
- The design's ★★★★★ and "Reviews (64)" tab are deliberately absent until E4.
  Does the meta row look unfinished without them, or better?

---

## 2026-08-10 — Phase E2: faceted shop, home page, and a bug only running found

**Worked on:** migrations 000008–000010 (benefit taxonomy + join table +
translations; `products.badge`/`badge_tone`; denormalized `sales_count` with a
backfill); `ProductFilter` extended with benefits, a price band and a sort
whitelist; `GET /catalog/facets` answering the whole sidebar in one round
trip; `sales_count` maintained inside the checkout transaction; seed rewritten
to the design's six hive products in three languages; `ShopPage` with
URL-driven filters, `HomePage`, redesigned `ProductCard`, `PriceRange` dual
slider, header search overlay; Postman gains a "Faceted catalog" folder.

**Learned:**

- **Many-to-many needs a table; one-to-many needs a column.** A product has
  one category (`category_id`, an FK) and any number of benefits
  (`product_benefits`, PK on the pair). The composite PK is what makes the
  relationship a *set* — no amount of double-clicking can make one product
  count twice in a facet total.
- **A composite index only helps queries that constrain its LEADING column.**
  The PK `(product_id, benefit_id)` answers "which benefits does this product
  have" but not "which products have this benefit", so the facet query needs
  the mirror index. Same rule as ordering fields in a C++ struct for lookup,
  except the database will silently do a sequential scan instead of telling
  you.
- **`count(*) FILTER (WHERE …)` is a per-aggregate WHERE.** Several
  differently-filtered aggregates share one scan, which is what lets one
  LATERAL subquery return a product's min price, max price and
  "how many variants are inside the requested band" together.
- **`LEFT JOIN` + `count(column)` ≠ `count(*)`.** On an unmatched row the
  joined column is NULL and `count(col)` ignores it, while `count(*)` counts
  the row itself and would report 1 for every empty category. The difference
  is the whole reason zero-count facets can be rendered at all.
- **`ORDER BY` cannot be a bound parameter.** Postgres plans the sort at parse
  time, so `ORDER BY $1` sorts by the constant string `$1`. That single fact
  is why the sort whitelist is a security boundary and lives in `domain`.
- **The rule against concatenating SQL is about user input, not about `+`.**
  Sharing the WHERE clause between the list, count and facet queries as Go
  constants is safe — the string is fixed before `main` runs, like a
  `constexpr` — and it removes the "these two must stay identical" comment
  that was the actual hazard.
- **Denormalizing is a trade with a named cost.** `sales_count` buys a cheap
  sort and pays with derived data that can drift; it is worth it only because
  exactly one write path touches it and that path already holds the locks.
- **Deadlocks come from lock ORDER, not lock count.** Adding a second
  `UPDATE` to checkout meant products could be locked in cart order; two
  carts holding the same two products in opposite orders would deadlock. Fix
  is the same rule the variant lock already used — sort the ids ascending.
  Go randomises map iteration order deliberately, so `slices.Sort` is not
  tidiness here, it is the fix.
- **Faceted search has one governing rule**: a facet group's own filter must
  not narrow its own counts. Everything else about the CTE follows from it.
- **The URL is a place to put state.** Filters in `useState` would have
  broken the back button, shared links, reload and open-in-new-tab — and
  would have passed every "does clicking work" test while doing so.
- **A percentage height resolves against the parent's HEIGHT.** `h-full`
  inside a container that only has `min-height` collapses; an aspect ratio
  derives the height from the width instead. Found by looking at a
  screenshot, not by typechecking.

**The one that matters most: a passing test suite is not a working app.**
E1.5 finished with 43 green tests, a Postman collection proving the API
answers in three languages, and a `Done when` that said the shell "reads
correctly in all three languages". Opening `/hy/shop` in a browser showed an
Armenian header wrapped around an entirely English catalog — the frontend had
never sent `?lang=`, a cookie, or `Accept-Language` to anything. Nothing
failed, because the backend's fallback chain returns perfectly valid English.
Two smaller versions of the same shape turned up beside it: `GetCart` had no
locale at all, and the footer's link columns were hardcoded English literals.
The lesson is not "write more tests" — it is that tests assert what you
thought to assert, and *running the thing* is a different question. The fix
now has a test that asserts on the request URL, which is the thing that was
actually wrong.

**Questions / to revisit:**

- Facet counts are recomputed on every filter change. Six products makes that
  free; at a few thousand it is the page's cost centre. Where does caching
  belong — HTTP headers, a materialised view, or the client's query cache?
- `sales_count` is not decremented on cancellation, on purpose. If the shop
  ever wants "most loved" to mean *kept*, that becomes a second write path
  and the counter needs the same care twice.
- The card shows a benefit from the taxonomy where the mock wrote a
  per-product phrase. If E3 adds editorial fields anyway, is a `tagline`
  worth it, or is one vocabulary better than two?

---

## 2026-08-05 — Phase E1.5: three languages, front to back

**Worked on:** i18next + react-i18next with the URL as the only source of
locale truth; `/hy` and `/ru` route prefixes (English unprefixed); Noto Sans
Armenian/Cyrillic fallback; migration 000007 (`product_translations`,
`category_translations`) with a per-locale generated tsvector; locale
negotiation middleware; translated reads with a three-level fallback;
validation codes replacing English prose; admin write path for translations;
Postman Localization folder.

**Learned:**
- **Postgres volatility classes.** `IMMUTABLE` / `STABLE` / `VOLATILE` are a
  promise about determinism. A `GENERATED … STORED` column *requires*
  immutable, because the value is computed once and written to disk — if the
  function could later answer differently, the stored value silently becomes
  a lie. `to_tsvector('english', x)` qualifies; `to_tsvector(locale::regconfig, x)`
  does not, because the cast reads the system catalog. A `CASE` over literal
  config names does. **C++:** `constexpr` vs `const` vs an ordinary function
  reading a global — a generated column is a `constexpr` context.
- **All three languages get real stemming.** Postgres ships 29 text search
  configurations including `armenian`; `SELECT cfgname FROM pg_ts_config`
  proved it before any code was written. Russian normalises ё→е, so "мед"
  finds "мёд".
- **`LEFT JOIN` + `COALESCE` is a declarative fallback chain.** Inner join
  drops unmatched rows; left join keeps them and fills NULL — join twice at
  different locales and `COALESCE` picks the first that exists. **C++:**
  chained `optional::value_or`, evaluated set-at-a-time.
- **A fallback chain must be complete.** Coalescing text three levels but the
  tsvector only two silently removed every untranslated product from
  full-text search. An existing test caught it — 2 results became 1.
- **CSS resolves `font-family` per character, not per element.** So appending
  Noto to the stack beats `:lang()` overrides: an Armenian name inside an
  English page just works. `unicode-range` also gates the download — English
  visitors fetch neither font. **C++:** overload resolution picking per
  argument, not per call site.
- **`context.WithValue` needs an unexported key type.** `type ctxKey struct{}`
  cannot be named by another package, so collision is impossible by
  construction rather than convention; zero-size, so free. **C++:** a private
  tag type, or a type in an anonymous namespace.
- **Accept-Language is q-values, not order.** `en;q=0.3,hy;q=0.9` means
  Armenian. Hand-parsed in ~30 lines rather than taking
  `golang.org/x/text/language`.
- **Negotiation must never fail.** An unknown tag or malformed `q` falls
  through to the next source. A shop that refuses to render because a header
  was odd is worse than one that renders in English.
- **CLDR plural categories.** Russian picks one/few/many from the last digit
  *and* the tens: 21 товар, 22 товара, 25 товаров. No `count === 1 ? a : b`
  expresses that — the concrete reason this project accepted an i18n
  dependency.
- **Error prose is part of the API contract.** Returning English sentences
  hardcoded one language into every client. Codes fixed it; the frontend had
  to change in the same commit, because all three forms printed
  `fields[x]` raw.
- **`ON CONFLICT … DO UPDATE`** is atomic write-whatever-the-current-state,
  with no window where the row is absent, and lets create and update share one
  path. `EXCLUDED` is the would-be-inserted row. **C++:**
  `map::insert_or_assign` vs erase-then-insert.
- **Transactions are about invariants, not statement counts.** `CreateCategory`
  needed one because "a category has text in at least English" is an
  invariant — and a half-written state would have *read back as fine* through
  the fallback.
- **Go generics that never inspect `T`.** `parseLocaleMap[T any]` re-keys a
  map for both `string` and a struct; no constraint needed. **C++:** a
  pass-through template, except Go compiles one shared implementation rather
  than one instantiation per type.

**Questions / to revisit:**
- Armenian and Russian copy is machine-assisted and wants a native speaker's
  eye — the apiary vocabulary especially (propolis, royal jelly, bee venom).
- Untranslated products match against an English-stemmed tsvector, so
  non-English full-text is weak for them; trigram still finds them. Revisit if
  the catalogue grows.
- `COALESCE(t.name, en.name, p.name)` cannot use the trigram index. Fine at
  six products; E10's k6 re-baseline should confirm.
- Variant labels ("500 g jar") are translatable text and are *not* covered —
  decide in E3 whether labels become pure measurements.
- `i18next-browser-languagedetector` is installed but unused; drop it if
  nothing needs it.

## 2026-08-04 — Phase E1: design system foundations

**Worked on:** Tailwind v4 `@theme` token layer, WCAG contrast corrections,
self-hosted Poppins/Karla, eleven UI primitives + `Field` + `cx`, a global
`:focus-visible` ring, and eight inline icon components.

**Learned:**
- **Tailwind v4 is CSS-first.** `@theme` declares tokens that become both
  custom properties and utilities: `--color-panel` generates `bg-panel`,
  `text-panel`, `border-panel`. **C++:** a single `constants.h` replacing
  magic numbers — except CSS custom properties resolve at *runtime* in the
  browser, closer to a global read through a pointer than to a compile-time
  substitution.
- **WCAG contrast is arithmetic, and the design failed it.** Relative
  luminance → ratio. The mock's orange measures 2.9:1 as a button fill and
  2.7:1 as text, both under 4.5:1; its muted browns fail at body sizes on
  *both* the light and dark surfaces. Fixing at token-definition time is much
  cheaper than after components consume them.
- **Fix contrast where the value is defined, not where it is used.** Baking
  `#e4761f` into `Button` would have meant repainting every screen in E10.
- **Subset your fonts.** `@fontsource/poppins/400.css` pulls every subset
  Google publishes — ~450 kB of unused Devanagari. Import
  `latin-400.css` instead.
- **Tailwind variants compile to real selectors.** `peer-checked:x` becomes
  `.peer:checked ~ .x` — a *sibling* combinator, so a nested element can never
  match. My first `Checkbox` nested the tick and it would have been invisible
  forever. When an abstraction misbehaves, expand it to the CSS it generates.
- **Types can enforce accessibility.** `IconButton`'s `label` prop is
  required, so `tsc` refuses an unnamed icon button — the compiler doing what
  a code review would otherwise have to catch.
- **`currentColor` makes icons free.** Inline SVG icons inherit whatever text
  colour the token system already applied, so no per-icon wiring — and they
  tree-shake, unlike a sprite.
- **Assertions must be able to fail.** The Postman validation test asserted
  the `fields` keys merely *existed*, which passed both before and after the
  contract changed. A test that cannot fail is not protecting anything.

**Questions / to revisit:**
- Of the eleven primitives, `Select`, `Checkbox` and `Breadcrumbs` have no
  consumer until E3–E6 — speculative work; demand-driven would have been
  defensible.
- The whole palette needs an axe re-verification in E10; the ratios here were
  computed by hand.

## 2026-07-31 — Search v2: trigram prefix + typo tolerance

**Worked on:** migration 000006 (`pg_trgm` extension + trigram GIN on name); three-door search predicate — FTS (raw query) OR name-substring (ILIKE) OR fuzzy (`word_similarity > 0.35`), hybrid ranking (ts_rank + similarity); `escapeLike` (user's `%`/`_` are literal); `fuzzyQuery` stripping websearch operators from the trigram doors; 4 new integration subtests (prefix, typo, mid-word, wildcard-literal).
**Learned:**
- Trigrams: text → 3-char fragments; overlap = similarity. Gives substring + typo matching and is language-blind (works for Armenian names where stemming can't).
- A trigram GIN index accelerates ILIKE '%…%' — the classic "can't index a leading-wildcard LIKE" rule has this exception.
- **Query-language leakage bug (caught by tests):** the FTS operator `-tea` reached the trigram doors as literal text and matched "Herbal Tea" — when one input feeds engines with different syntaxes, sanitize per engine.
- Guard empty fallback inputs: ILIKE '%%' matches everything.
- Layered search = layered ranking: sum the signals so exact beats fuzzy.
**Questions / to revisit:**
- Multilingual FTS configs if descriptions go Armenian/Russian; similarity threshold may need tuning with a real catalog.

## 2026-07-31 — Product search: Postgres full-text + debounced UI

**Worked on:** migration 000005 — GENERATED tsvector column (weighted: name=A, description=B) + GIN index; `websearch_to_tsquery` in ListProducts with rank ordering when searching (CASE in ORDER BY, one query for both modes); `q` param through public+admin endpoints; `useDebouncedValue` hook (300ms) with fake-timer Vitest test; search box with live result count on the catalog.
**Learned:**
- FTS pipeline: text → to_tsvector (lexemes, stemmed by language config) → @@ match against tsquery → ts_rank for ordering; GENERATED column means the DB keeps the index source in sync — application can't forget.
- GIN = inverted index (word → rows), the tsvector workhorse.
- websearch_to_tsquery accepts raw human input safely — quotes, OR, -exclusion — no parsing on our side.
- setweight lets structure (name vs description) influence ranking.
- Debouncing: input state updates instantly, the derived query value trails typing by 300ms — cancel-previous-timer in useEffect cleanup IS the mechanism; tested with vi.useFakeTimers.
**Questions / to revisit:**
- Multilingual search (Armenian/Russian product names need different ts configs); typo tolerance (pg_trgm) if ever needed.

## 2026-07-31 — Product images: first file upload

**Worked on:** `POST /admin/products/{id}/image` — multipart parsing, content type sniffed from magic bytes (`http.DetectContentType`; filename and client headers ignored as attacker-controlled), 5MB `MaxBytesReader` cap, server-generated filenames, orphan-file cleanup on failed DB update; Go `http.FileServer` at `/uploads/*`; nginx proxy location with cache headers; `uploads_data` volume in both compose stacks; `FormData` path in the frontend client (browser sets multipart boundary itself); ImageSlot thumbnail-as-button UI (hidden file input); images render in catalog/product pages.
**Learned:**
- Never trust uploads: sniff real content type, cap size, generate your own filenames — the three rules of accepting user files.
- The test caught a REAL platform bug: `os.Remove` on a still-open file fails on Windows (fine on Linux) — close before delete; deferred-close was too late. Tests of cleanup paths earn their keep.
- Files don't belong in the DB; the DB stores the URL, the filesystem (a volume, later S3) stores bytes.
- `FormData` uploads must NOT set Content-Type manually — the browser's boundary parameter is required.
- Hidden `<input type="file">` triggered by a styled button is the standard upload UX.
**Incident (same day):** in containers, upload failed with `permission denied` — the api runs as distroless **nonroot** (uid 65532) but Docker created the named volume owned by **root**. Fix: the Dockerfile now ships `/uploads` with `COPY --chown=65532:65532`; Docker copies that ownership into a named volume **on its first initialization** (had to `docker volume rm` the root-owned one). Lesson: non-root containers + named volumes = ownership must come from the image; "works with go run on the host" proves nothing about the container's permission reality.
**Questions / to revisit:**
- Image resizing/thumbnails; S3-compatible storage when hosting matures; nginx serving uploads from a shared volume directly (skip the proxy hop).

## 2026-07-31 — Admin products UI

**Worked on:** AdminProductsPage — create form with **array state** (dynamic variant rows added/removed immutably via map/filter), backend's `variants[i].field` errors mapped onto the exact form row, category `<select>`, major↔minor price conversion isolated at the UI edge (`toMinor`), per-variant inline editors with dirty-tracking save buttons, active/inactive toggle (full-fields PUT), shared `AdminNav` with `NavLink` active styling; product-write hooks invalidate admin + public caches together.
**Learned:**
- Array state in forms: rows live in one array; edits replace the array immutably — `map` for update, `filter` for remove, spread for append.
- Dirty tracking (compare draft vs source, show save only when changed) is derived state — no extra flags needed.
- Money conversion belongs at ONE edge: humans type 1500.00, everything else speaks minor units.
- `NavLink` gives active-tab styling from the router for free.
- Field-error contracts (JSON paths) designed on the backend pay off exactly here.
**Questions / to revisit:**
- Add-variant-to-existing-product; image upload; category management could join AdminNav pattern.

## 2026-07-31 — Admin product management (backend)

**Worked on:** `POST/PUT /admin/products`, `PATCH /admin/variants/{id}`, admin list with `IncludeInactive`; transactional CreateProduct (product+variants all-or-nothing); error mapping by pg **constraint name** (one SQLSTATE 23505 → ErrSlugTaken / ErrSKUTaken / ErrVariantLabelTaken); validation with `variants[i].field` JSON-path keys; slug immutability (public URLs must not break); tests at all three levels incl. rollback-no-orphan integration test; Postman admin requests with chained variables.
**Learned:**
- `pgErr.ConstraintName` distinguishes business meanings behind identical error codes — name constraints deliberately.
- Nested validation errors need addressable keys (`variants[0].sku`) so forms can target inputs.
- Immutable identifiers: anything that is a public URL (slug) or an external reference (SKU on invoices) should not be editable casually.
- Soft delete via `is_active` + a filter flag beats DELETE: orders reference products forever.
- Postman chaining: a test script saves a response id into a variable the next request uses.
**Questions / to revisit:**
- Admin products UI (next session); adding variants to an existing product; image upload.

## 2026-07-31 — Alerting: rules + Alertmanager, fire drill included

**Worked on:** `alerts.yml` with 4 rules (APIDown via the free `up` metric; HighErrorRate; SlowRequests p95; DBPoolSaturated via wait-rate); `evaluation_interval`, `rule_files` and `alerting:` blocks in prometheus.yml; Alertmanager v0.28 service (UI :9093, receiver empty until real hosting — Telegram config documented in place). Fire drill: stopped api → watched pending (45s) → firing (105s) → active in Alertmanager → restarted api → auto-resolved.
**Learned:**
- Alert anatomy: `expr` (a PromQL query) + `for` (flap protection — brief blips never page) + labels/annotations (severity, human-readable context).
- The `up` metric is monitoring-for-free: every scrape target gets liveness alerting without any instrumentation.
- pending → firing → resolved is the full lifecycle; Prometheus evaluates, Alertmanager routes/groups/throttles (group_wait, repeat_interval).
- Alert thresholds encode the SLOs we load-tested: the k6 baseline (p95 ~20ms) justifies alerting at 500ms — alerts and load tests reference the same numbers.
- Test the fire alarm by lighting a real (controlled) fire — an unverified alert rule is a hope, like an unrestored backup.

## 2026-07-31 — k6 load testing: baseline + a real bug found

**Worked on:** `load/catalog-test.js` — two k6 scenarios (ramping browsers 0→20 VUs; 5 constant buyers doing register→cart→checkout with per-VU sessions), SLO thresholds in code (p95<200ms, errors<1%). First run FAILED usefully: k6 v2 resets its cookie jar every iteration → sessions died after iteration 0 (255 × 401s). Fixed by capturing the session cookie into per-VU module state and sending it manually. Final: 3,986 reqs @ 38/s, p95 19ms client / 8.5ms server, 0 errors, ~250 orders; pool wait 0.17s cumulative on 12 conns → no contention.
**Learned:**
- Thresholds = SLOs as code; k6's exit code makes load tests CI-able.
- Client p95 vs server p95 differ by the proxy/network layer — measuring both localizes overhead.
- `mb_db_pool_acquire_wait_seconds_total` is the saturation signal: flat = pool comfortable, climbing = raise pool size or fix slow queries.
- Load tests find *behavioral* bugs too (session handling), not just slowness.
- "No bottleneck at this load" is a legitimate, valuable finding — measured, not assumed; the breaking-point hunt is a separate future experiment.
- Tooling: PowerShell `Select-Object -First` closes the pipe and kills the upstream process mid-run — capture to file instead.

## 2026-07-30 — Incident: web container crash-loop ("host not found in upstream")

**What happened:** nginx crash-looped on `host not found in upstream "api"` while api was healthy. Root cause: the web container was *created* during an earlier failed `up` (port-80 conflict interrupted its network attachment) → it existed with NO networks; restart policies rerun the broken container as-is, never re-create it. Fixed with `up --force-recreate web`.
**Hardening:** nginx.conf now uses `resolver 127.0.0.11` + a variable in `proxy_pass`, deferring DNS to request time — nginx boots even when api is absent (relevant after host restarts, where `depends_on` ordering does not apply). Verified: web restarts cleanly with api stopped; static pages 200, api routes 502, self-heals when api returns.
**Lessons:** `docker inspect <container>` → Networks when behavior "can't happen"; restart ≠ recreate; literal `proxy_pass` binds DNS at boot; graceful degradation beats crash-looping.

## 2026-07-30 — Phase 10 started: Prometheus metrics + Grafana dashboard

**Worked on:** metrics middleware (`mb_http_requests_total`, duration histogram — labeled by chi route PATTERN, never raw path), `/metrics` endpoint (unproxied by nginx → never public), custom `PoolCollector` for pgxpool stats, `mb_orders_created_total` business counter, metrics test; Prometheus v3.5 + Grafana 12 joined the compose stack with provisioned-from-git datasource and 6-panel RED dashboard. Verified: target up, queries answering, traffic visible.
**Learned:**
- Prometheus is PULL-based: services answer /metrics, the server visits. Counters only grow; `rate()` derives per-second speed; histograms bucket latencies so `histogram_quantile()` can compute p95 later.
- Label cardinality is the classic self-inflicted wound: label by route pattern, not URL; every unique label combo is a stored time series.
- The RED method (Rate, Errors, Duration) is the standard first dashboard for a request-driven service.
- Business metrics (orders placed) matter as much as infrastructure metrics — servers can be green while sales are zero.
- Grafana provisioning = dashboards as code in git, surviving container wipes.
**Questions / to revisit:**
- Alerting rules; OpenTelemetry tracing; k6 load test to make these graphs interesting.

## 2026-07-30 — Phase 9 prepared: deploy artifacts + runbook (server pending)

**Worked on:** `deploy/docker-compose.prod.yml` (pulls GHCR images, Caddy TLS edge, MB_ENV=prod); `deploy/Caddyfile` (auto-Let's Encrypt in one line); `deploy/backup.sh` (nightly pg_dump + rotation); CD `deploy` job in CI (SSH pull-and-restart, dormant behind `DEPLOY_ENABLED` variable); `docs/DEPLOYMENT.md` — the full runbook from empty Ubuntu VPS to live HTTPS shop with CD.
**Learned:**
- CI vs CD as separate concerns: certify artifacts vs move them; connected by the registry.
- Caddy's value proposition: TLS issuance/renewal/redirects as defaults, not configuration.
- The server pulls (git + registry) with read-only credentials: deploy key + read:packages PAT — never push credentials on the server.
- Gate not-yet-usable pipeline stages behind repository variables so CI stays green.
- A backup that was never restored is a hope, not a backup.
**Questions / to revisit:**
- Execute the runbook once VPS + domain exist; flip MB_ENV story ends (Secure cookies live).

## 2026-07-29 — Phase 8 complete: the stack containerized

**Worked on:** multi-stage Dockerfiles (Go: deps-layer caching, CGO_ENABLED=0 static build, distroless nonroot final 22.5MB; frontend: npm ci → vite build → nginx 93MB); `api healthcheck` self-probe subcommand (distroless has no curl); nginx.conf (SPA try_files fallback, immutable asset caching, /api proxy via Docker DNS); production compose with health-gated startup chain (postgres healthy → migrate completes → api healthy → web); `.dockerignore` keeping `.env` out of images; GHCR publish job in CI (green master only).
**Learned:**
- Multi-stage builds: the toolchain (~800MB) never ships; only the artifact does. Layer order matters — lockfiles first.
- Distroless: no shell = tiny attack surface; healthchecks must be built into your binary.
- `depends_on` conditions (`service_healthy`, `service_completed_successfully`) encode startup order declaratively; migrations as a one-shot service.
- Not publishing postgres's port = the DB is unreachable from outside the compose network. Security by topology.
- The nginx config is the production twin of Vite's dev proxy — same origin story, same SPA fallback.
- MB_ENV stays dev until HTTPS exists: Secure cookies would break plain-http localhost logins.
**Questions / to revisit:**
- Phase 9: VPS, domain, TLS, CD pulling the GHCR images; MB_ENV=prod then.

## 2026-07-29 — Phase 7: GitHub Actions CI

**Worked on:** `.github/workflows/ci.yml` with three parallel jobs — backend (vet, golangci-lint pinned to local version, build, `go test -race` incl. testcontainers on the runner's Docker), frontend (npm ci, lint, vitest, tsc+build), e2e (Postgres service container with healthcheck, migrations via `go run migrate@version`, seed via psql, Playwright with trace upload on failure). README badge. `reuseExistingServer: !process.env.CI` for Playwright.
**Learned:**
- CI = the local quality gate, executed by a robot on every push; jobs run in parallel on clean machines, so "works on my machine" can't hide.
- `npm ci` vs `npm install`: ci installs exactly the lockfile and never mutates it — the only correct choice for CI.
- Service containers are CI's docker-compose: Postgres per job, health-gated.
- Pin tool versions in CI to match local (golangci-lint) or chase phantom differences.
- `-race` runs on Linux runners for free (cgo present) — the reason it's delegated to CI from Windows.
- Branch protection is plan-gated for private repos: decided CI-status-without-enforcement (decision #11).
**Questions / to revisit:**
- Start the PR habit with the next feature; CI badge stays red/green truth.

## 2026-07-29 — Phase 6 complete: frontend tests (Vitest + Playwright)

**Worked on:** Vitest + Testing Library setup (jsdom, setup file with jest-dom matchers + cleanup); formatPrice unit tests (locale pinned to en-US for determinism); ProductCard component tests (render → assert via accessible queries, MemoryRouter for Link context); Playwright e2e purchase flow with dual webServer config (auto-starts Go + Vite, reuses running ones); trace-on-failure debugging.
**Learned:**
- Component tests query the DOM like a user would (`getByRole`, `getByText`) — testing behavior, not implementation details.
- The e2e failure snapshot exposed a REAL accessibility bug: a Link wrapping a card computed no accessible name; `aria-label` fixed both the screen-reader experience and the test.
- Vitest and Playwright must not see each other's test files (`test.exclude`).
- Locale-dependent formatting is a test flakiness source — pin it.
- The pyramid's economics end-to-end: 7 component tests in ~150ms; 1 browser e2e in ~1.3s covering the whole stack.
**Questions / to revisit:**
- More e2e scenarios later (admin flow, insufficient stock); coverage reporting in CI.

## 2026-07-29 — Phase 6 (backend): the test pyramid

**Worked on:** table-driven domain tests (state machine, cart math, validation); api handler tests with an in-memory `fakeStore` satisfying `api.Store` + `httptest` through the real router/middleware (auth matrix 401/403/201); testcontainers integration tests — one throwaway Postgres per package run (`TestMain`), real migrations via migrate-as-library (iofs source), `resetDB` truncation between tests; checkout tests incl. rollback-on-insufficient-stock and the formalized concurrency race (10 goroutines, stock 3).
**Learned:**
- The pyramid in practice: domain tests run in μs, fake-store handler tests in ms, container tests in seconds — write many of the cheap ones, few of the expensive ones.
- Consumer-side interfaces made the fake store trivial — the design decision from Phase 2 paid off exactly as promised.
- `TestMain` = per-package setup/teardown; `flag.Parse()` needed before `testing.Short()` there.
- Test against the real migrations — the test schema can't drift from prod.
- Verify effects in the DB, not just return values (stock decremented, cart cleared, rollback left everything untouched).
- `go test -short` as the no-Docker fast path; `-race` needs cgo (absent on Windows) → run it in Linux CI.
- errcheck caught an unchecked `Close()` in my own test code — linters lint tests too.
**Questions / to revisit:**
- Vitest component tests + Playwright e2e (rest of Phase 6).
- Auth handler tests (register/login) against the fake; coverage reporting.

## 2026-07-29 — Phase 5 complete: shopping UI

**Worked on:** cart/orders/admin-orders API client + hooks; CartPage (qty +/−, remove, checkout with error surface); OrdersPage; AdminOrdersPage (status transition buttons from a client-side mirror of the state machine); real AddToCartButton on ProductPage (out-of-stock / sign-in-to-buy / in-cart states); header cart badge with live count.
**Learned:**
- `enabled:` option gates queries on preconditions (no cart fetch while anonymous).
- Derived UI state: the add-to-cart button's four states all derive from existing queries — no new state needed.
- Mutation → invalidation fan-out: checkout touches cart+orders+products (stock!), and the UI updates everywhere without a reload.
- Client-side mirror of a server state machine is fine for UX as long as the server enforces (409 on race).
**Questions / to revisit:**
- Product/variant management UI for admin; anonymous carts; payments (Phase 10).

## 2026-07-29 — Phase 5 (backend): cart, transactional checkout, order state machine

**Worked on:** migration 000004 (cart_items with composite PK, orders, order_items with snapshots); cart store (upsert via `ON CONFLICT DO UPDATE`, FK violation → 404); `CreateOrder` transaction (`FOR UPDATE OF v` row locks, deterministic lock order, stock check → decrement → snapshot → clear cart); order state machine with cancel-restores-stock; `requireUser` middleware; endpoints `/cart`, `/cart/items`, `/orders`, admin `/admin/orders(+/{id}/status)`; Postman folders. Live concurrency test: stock=1, two parallel checkouts → exactly one 201.
**Learned:**
- A transaction = all-or-nothing; `defer tx.Rollback()` after `Begin` guarantees cleanup on every path (RAII feeling); rollback after commit is a no-op.
- `SELECT ... FOR UPDATE` locks rows so concurrent transactions queue; `ORDER BY` in the locking query = consistent lock order = no deadlocks.
- Snapshots (`price_minor_snapshot`, `name_snapshot`) make orders immune to later catalog edits.
- Composite primary keys model "one row per (user, variant)" naturally; upsert = `INSERT ... ON CONFLICT ... DO UPDATE`.
- State machines as data (`map[from][]to`) keep transition rules in one testable place.
- PUT with set-semantics is idempotent — retries are safe (matters on flaky networks).
**Questions / to revisit:**
- Frontend cart/checkout/orders UI — next session.
- Payments stub; expired pending-order cleanup; cart price-change warnings.

## 2026-07-28 — Phase 4 complete: frontend auth UI

**Worked on:** `useMe` (401→null mapping), login/register/logout/createCategory mutations; LoginPage (controlled inputs, mode toggle, per-field API errors); AuthStatus header widget (Sign in ↔ email + Sign out + Admin link); AdminPage with category form + list; routes /login and /admin; generic `request<T>` extended for POST/204.
**Learned:**
- `useMutation` for writes vs `useQuery` for reads; after a write either `setQueryData` (we hold the fresh value) or `invalidateQueries` (server knows best).
- Controlled inputs: React state owns the field value; `e.preventDefault()` stops the browser's own form submission.
- Anonymous is a state, not an error — mapping 401 to `null` keeps `useMe` clean.
- Client-side route guards are UX only; the backend middleware is the actual security boundary.
- The browser attaches HttpOnly cookies automatically — frontend code never sees or touches the session token.
**Questions / to revisit:**
- Product management UI for admin (with Phase 5+); logout everywhere; email verification (Phase 10 maybe).

## 2026-07-28 — Phase 4 (backend): sessions, bcrypt, admin gate

**Worked on:** migration 000003 (users, sessions with hashed tokens); bcrypt registration/login; session cookie (HttpOnly, SameSite=Lax, Secure=!dev, 7d TTL); `withUser` context middleware + `requireAdmin`; `/api/v1/auth/*` endpoints; category POST moved to `/api/v1/admin/categories`; Postman restructured (Auth + Admin folders, cookie-jar flows). Full lifecycle verified with curl cookie jar, including SQL role promotion taking effect on the live session.
**Learned:**
- bcrypt: deliberately slow + salted → rainbow tables and brute force become impractical; `CompareHashAndPassword` never exposes the hash comparison.
- Session tokens: crypto/rand (never math/rand); DB stores SHA-256 of the token so a DB leak can't be replayed; raw token exists only in the cookie.
- Cookie flags: HttpOnly (XSS can't read), Secure (HTTPS only), SameSite=Lax (CSRF baseline).
- Same 401 for wrong email and wrong password — user enumeration defense.
- 401 vs 403: unauthenticated vs authenticated-but-not-allowed.
- `context.WithValue` with an unexported key type: per-request data flows through the call chain without globals.
- Server-side sessions mean role changes apply instantly (the JOIN reads current role per request) — a real trade-off vs JWT, where stale claims live until expiry.
**Questions / to revisit:**
- Frontend auth UI (forms, auth context, admin area) — next session.
- Later hardening: login rate limiting, session renewal, expired-session cleanup job.

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
- Is the commit e-mail added to the GitHub account (Nerses01)? If not, commits won't link to the profile — check GitHub → Settings → Emails.

## 2026-07-02 — Project kickoff

**Worked on:** project planning — goals, tech stack, architecture, phased roadmap; repository initialized.
**Learned:**
- How a full-stack project is structured: monorepo with backend / frontend / deploy / CI separation.
- Why Go fits a C++ developer (compiled, typed) and why sessions-before-JWT, money-as-integers, REST-before-GraphQL.
**Questions / to revisit:**
- (none yet)

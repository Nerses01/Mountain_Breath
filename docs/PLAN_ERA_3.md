# Mountain Breath — Plan, Era III: from built to open for business

> Era I (Phases 0–11 in [PROJECT_PLAN.md](PROJECT_PLAN.md)) built the
> machine. Era II ([PLAN_ERA_2.md](PLAN_ERA_2.md), E1–E10) built the store
> the design drew. **Era III is everything between "the code is done" and
> "a customer in Yerevan receives a parcel"** — plus every debt the first
> two eras consciously took and wrote down.
>
> **How this document was made (2026-08-15):** a full re-read of both
> plans, every "Questions / to revisit" list in the learning log, the
> Phase 11 backlog, the deferral notes scattered through Era II's
> checkboxes — and two live checks against the running system that found
> gaps NO document had recorded (§1). Everything is here; nothing is
> committed-to until it moves into an active phase, exactly like Phase 11.
>
> Phases are numbered **F1–F6** (E was Era II's letter). Unlike Era II,
> these are not strictly sequential: F1 gates everything, but F2 and F3
> can interleave with it, and F5/F6 items are picked by interest and
> evidence. Phase 11 stays the shared parking lot for raw ideas.

---

## 1. Found by THIS audit — gaps no document recorded

The re-read's most valuable products, because nothing else would ever have
surfaced them:

- [x] ~~**`payment_status` has no write path.**~~ E6 modelled it ("the admin
      flips it; the column exists so that flip is a recorded fact") and
      nobody ever built the flip — no endpoint, no button. Consequence: a
      bank-transfer order could NEVER become `paid`. **Closed 2026-08-18
      (decision #91):** `PATCH /admin/orders/{id}/payment` drives a
      payment state machine in the domain (`unpaid → paid → refunded`, no
      backward arrows), with mark-paid/mark-refunded buttons on the admin
      orders table. F2's first item, done first as §4 suggested.
- [ ] **The privacy page promises what the API cannot do.** E9's copy says
      "delete your account entirely" and "we will show you what we store" —
      there is no deletion endpoint and no export. A published promise is a
      requirement; either build it (F2) or soften the page before launch.
      Deletion is also where `users` FKs get audited: orders must survive
      (bookkeeping, the page says so), sessions/hearts/addresses must go.
- [ ] **Placeholder facts inside real pages.** The E9 content ships
      `hive@mountain-breath.example` and `+374 91 00 00 00` on the Contact
      page, and §1.1's rule that *every price is placeholder* still stands.
      Real contact details and the family's real per-market prices are
      launch gates (F3) — the shop currently promises to answer an email
      address that does not exist.
- [x] ~~**Two small promised-and-forgotten reviews:**~~ E1.5's "revisit the
      footer placement of the LanguageSwitcher" — **closed 2026-08-18
      (decision log #90):** the footer switchers were REMOVED; the account
      settings screen (PLAN_ACCOUNT A5) is the one home for both
      language and currency, with the anonymous-visitor cost recorded.
      Era I's admin promotion via SQL remains open (folded into F2's
      admin batch).

---

## 2. Ledger corrections — Phase 11 lines Era II already closed

Recorded so the backlog stops advertising finished work (ticked in
PROJECT_PLAN.md with the same cross-references):

- ~~Login rate limiting~~ — E8 (fixed-window per IP+email, also guards
  forgot-password and newsletter subscribe).
- ~~Order e-mail confirmation~~ — E8 (trilingual, via the Mailer);
  *status-change* emails remain open → F2.
- ~~Product image galleries~~ — E3; *server-side thumbnails/resizing*
  remain open → F3, behind photography.
- ~~Multilingual content in FTS~~ — E1.5 (per-locale generated tsvector
  with the built-in `armenian`/`russian` configs).
- ~~k6 breaking-point (first pass)~~ — E10 found and fixed the real
  bottleneck (Postgres JIT, p95 3,090ms → 11.6ms); what remains is the
  re-run on real hardware → F6.

---

## 3. The phases

### Phase F1 — Go live (Phase 9, unfrozen)

**Goal:** the store is on the internet, over HTTPS, deploying itself from
green master. Everything else in this era presumes it.

**The blocker, restated:** hosting on own hardware is frozen on the ISP's
locked FiberHome terminal (no port forwarding; call pending since
2026-07-30). Every artifact is ready (`deploy/Caddyfile`, the dormant
`deploy` CI job, `backup.sh`, docs/DEPLOYMENT.md).

- [ ] **Decide the unblock**: either the ISP opens 80/443, or fall back to
      the original recommendation (cheap VPS — Hetzner/DO) rather than
      waiting indefinitely. *(User decision; the artifacts serve both.)*
- [ ] Harden the server (runbook §2), domain + DNS + Caddy TLS, flip
      `DEPLOY_ENABLED=true`, first CD deploy.
- [ ] Backups on cron **and a tested restore** — a backup that has never
      been restored is a hope, not a backup.
- [ ] The go-live config sweep the code already anticipates:
      `MB_PUBLIC_URL` to the real origin; the Google OAuth client gains the
      production redirect URI and the consent screen leaves Testing mode
      (E8's checklist); a real SMTP relay replaces Mailpit for production
      (`MB_SMTP_*` — the Mailer interface means zero code); `MB_ENV=prod`
      for Secure cookies.
- [ ] Prometheus alerting gains its notification channel (Telegram —
      parked in Phase 10 explicitly "for when it runs on the real server").

**Done when:** merging a PR updates the live site over HTTPS, a restore
drill has actually run, and an alert reaches a phone.

---

### Phase F2 — An operable shop (the admin's missing hands)

**Goal:** the family can run the store without psql. Era II built the
customer's every screen; the admin got only what each phase needed.

- [x] **`PATCH /admin/orders/{id}/payment` + buttons** — mark
      paid/refunded, the §1 gap. The state pair stays orthogonal to order
      status, as E6 modelled it. *Done 2026-08-18 (decision #91): domain
      machine + store write (FOR UPDATE, no side effects), the endpoint,
      the two buttons, tests at all three layers.*
- [x] **Order status-change emails** (the other half of Phase 11's
      "order e-mail notifications"): confirmed/shipped/delivered/cancelled
      notify the customer, through the existing Mailer + template pattern,
      in the order's locale. *Done 2026-08-18 (decision #92): migration
      000024 snapshots the checkout's locale onto the order,
      `CreateOrder` widened to take `domain.View`, the status handler
      sends non-fatally honoring `notify_order_updates` (#87's promise),
      trilingual copy flagged for the F3 native review.*
- [ ] **Customer-facing cancellation** while `pending` (Phase 11 line):
      one endpoint (the domain state machine already allows it and
      restores stock + promo redemptions), one button on the order page.
- [ ] **Promo code CRUD** in the admin (E7 shipped seed-only codes;
      "revisit when three codes stop being enough").
- [ ] **Category management**: edit / delete-or-deactivate / reorder
      (Phase 11 line; the footer and facets already read sort_order).
- [ ] **Admin user management**: promote/demote via UI instead of SQL
      (Era I leftover), guarded so the last admin cannot demote themself.
- [ ] **Account deletion + data view** (the privacy page's promises, §1):
      delete cascades the personal graph, orders are retained and
      detached per the stated bookkeeping rule.

**Done when:** a full order lifecycle — including getting paid, a status
email, and a customer cancellation — happens without anyone opening psql,
and the privacy page tells no lies.

---

### Phase F3 — Launch content (the business half, then its code)

**Goal:** the shop stops being a beautifully-typeset placeholder.

- [ ] **Photography.** The single biggest visible gap: every image slot on
      every screen is a designed placeholder. A jar, a comb, the meadow,
      the family — the mock's own shot list. *(Business task; everything
      below waits on it.)*
- [ ] **The E10-deferred image pipeline**, built against real files:
      server-side thumbnails/resizing on upload (Phase 11 line), `srcset`
      + AVIF/WebP, explicit dimensions everywhere an `<img>` renders.
      S3-compatible storage (Phase 11) joins here if the volume-on-VPS
      model chafes.
- [ ] **Real prices** in both markets (replacing §1.1's placeholders) and
      real shipping rates/threshold — the tables exist precisely so this
      is data entry, not a deploy.
- [ ] **Real contact details** on the Contact page + `MB_MAIL_FROM` on a
      real domain (§1).
- [ ] **Native review of the Armenian and Russian copy** — accumulated
      across E1.5→E9 and flagged every time: UI catalogues, auth screens,
      the three email templates, ~4,500 words of content pages. The legal
      pages and the reset email (it lands in strangers' inboxes) first.
- [ ] **The newsletter SENDER** (E9's deliberate other half): compose an
      issue, send to `confirmed AND NOT unsubscribed`, unsubscribe links
      carrying the permanent tokens the schema already promised. Start as
      an admin-triggered send; scheduling can wait.

**Done when:** a stranger could not tell which parts of the shop began as
placeholders.

---

### Phase F4 — Real payments

**Goal:** the card stops being a stub. The research was parked in Phase 11
from the start: **Idram / Ameriabank vPOS** (the mock's own ArCa hint), or
Stripe if the local rails disappoint.

- [ ] Research + pick the provider; sandbox account.
- [ ] Integration the way E6's stub anticipated: the provider hosts the
      card entry (the API never touches PANs — the PCI line E6 drew),
      webhook/callback flips `payment_status` through F2's new write path.
- [ ] Refunds wired to the same path; the `refunded` status finally
      reachable by a real flow.
- [ ] The checkout's decorative card fields become the provider redirect.

**Done when:** a real dram moves in a sandbox end to end, and the order
record shows paid-by-card without a human flipping anything.

---

### Phase F5 — Conversion & convenience (deferred UX, by evidence)

Deliberately after launch: each of these was cut with a reason, and real
traffic should confirm the reason wrong before it is built.

- [ ] **Anonymous carts** (+ merge into the account cart on login) and
      anonymous wishlists — Phase 11 + E8's backlog notes; the largest
      funnel-friction item on the list.
- [ ] **Checkout address picker** (`address_id`) over the E8 address book —
      today the default prefills; picking among several is the noted
      leftover.
- [ ] **Apple sign-in** — if the family approves $99/yr once a domain
      exists; the adapter work is scoped in E8's notes (form_post
      callback, JWT client secret, email-absent re-auth).
- [ ] **Smarter upsell** in the free-shipping banner (benefit-overlap
      ranking, E7's one-query note) and the `stackable` promo flag — each
      only when the current behaviour demonstrably misses.
- [x] ~~The LanguageSwitcher placement review~~ — settled 2026-08-18 the
      other way (decision #90): removed from the footer entirely, the
      account settings screen is the one home. Re-open only if anonymous
      traffic demonstrably suffers.

---

### Phase F6 — Engineering debts & evolution

The consciously-carried debts, each with its written tripwire:

- [ ] **Drop `products.image_url`** — E3's follow-up that lost its
      migration number to reviews: move the list read to `product_images`,
      stop double-writing on upload, then the drop (the add-backfill-drop
      pattern's last step, still owed).
- [ ] **Auth hardening batch** (Phase 11): security headers (CSP et al. at
      Caddy/nginx), an expired-`sessions` cleanup job (they accumulate —
      verified live), same sweep for spent reset tokens.
- [ ] **Rate limiter to shared storage** — tripwire written at the type:
      the day the API runs more than one replica.
- [ ] **OpenAPI spec + generated TS client** (Phase 11) — retiring
      hand-maintained `types.ts`; the Postman collection is the seed of
      truth to formalize.
- [ ] **Prerender/SSR for meta** — E10's honesty note: JS-managed tags
      reach only crawlers that render; behind hosting.
- [ ] **DB-backed pages/journal** — decision #77's tripwire: the day the
      family needs to edit copy without a commit. Also repairs the known
      hole of journal posts missing from the backend sitemap.
- [ ] **Observability round two** (Phase 10): log aggregation, OpenTelemetry
      tracing; k6 re-run on the real host — tighten the 200ms SLO the
      11.6ms p95 now mocks, then find the true breaking point there.
- [ ] **Only-if-measured shelf**: Redis caching, CDN, catalog cache
      headers (E10: "an invalidation bug on layaway"), Terraform/Ansible,
      Kubernetes ("only after Compose feels limiting") — each stays here
      until a measurement or a real pain names it.
- [ ] Watch the unasserted `agentic-browsing` Lighthouse category (E10).

---

## 4. Suggested order

F1 unblocks everything and needs mostly non-coding work (ISP or VPS call,
DNS, drills) — start it immediately and let it run in the background.
F2 is pure coding and small — it can be finished before hosting lands, so
launch day is not spent building the "mark paid" button. F3 waits on
photography but its code half (pipeline) is scopeable now. F4 needs F1's
domain for provider callbacks. F5 and F6 are evidence-driven and permanent.

The first cut worth a session: **F2's payment-status write path** — the
one item that is a bug-shaped hole in an already-shipped promise rather
than a feature.

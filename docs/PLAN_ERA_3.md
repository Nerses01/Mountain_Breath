# Mountain Breath — Plan, Era III: from built to open for business

> **📌 Historical since 2026-09-03.** F1 shipped 2026-09-02 (decisions
> #100/#101 — Cloudflare tunnel from a home laptop, *not* this file's
> VPS/port-forward plan; runbook: [DEPLOYMENT_HOME.md](DEPLOYMENT_HOME.md));
> F2 completed 2026-08-19 (decisions #91–#97). **Every still-open item
> from F1's residue and F3–F6 now lives in [BACKLOG.md](BACKLOG.md)** —
> the one list — and is maintained there, not here. This file stays as
> the era's record of what was planned and why.

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
- [x] ~~**The privacy page promises what the API cannot do.**~~ E9's copy says
      "delete your account entirely" and "we will show you what we store" —
      there was no deletion endpoint and no export. **Closed 2026-08-19
      (decision #97):** both promises built rather than softened —
      GET `/account/data` and DELETE `/account`; the FK audit ran as
      predicted (orders detached via migration 000025, reviews deleted,
      everything else cascades).
- [ ] **Placeholder facts inside real pages.** The E9 content ships
      `hive@mountain-breath.example` and `+374 91 00 00 00` on the Contact
      page, and §1.1's rule that *every price is placeholder* still stands.
      Real contact details and the family's real per-market prices are
      launch gates (F3) — the shop currently promises to answer an email
      address that does not exist. *→ open in [BACKLOG.md](BACKLOG.md) §2.*
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

✅ **SHIPPED 2026-09-02 — by a different road than planned** (decisions
#100/#101, runbook [DEPLOYMENT_HOME.md](DEPLOYMENT_HOME.md)): the ISP
blocker turned out to be CGNAT (unfixable by any router setting), so the
"ISP opens 80/443 vs VPS" decision resolved as *neither* — a home laptop
+ owned domain (`mountainbreath.net`) + **Cloudflare named tunnel**,
HTTPS at the edge, zero inbound ports, CD push over Tailscale (ephemeral
`tag:ci` runner nodes). Hardening done; `DEPLOY_ENABLED=true`; first
automated deploy green after a five-failure debugging arc (log
2026-09-02). The observability trio moved into prod, loopback-bound
(#101).

- [x] ~~Decide the unblock~~ — decided: tunnel (reverses #12).
- [x] ~~Harden, domain + DNS + TLS, flip `DEPLOY_ENABLED`, first CD
      deploy~~ — all live (TLS at Cloudflare's edge, not Caddy).
- [x] `MB_PUBLIC_URL`/`MB_ENV=prod` — derived from `DOMAIN` in the prod
      compose; Secure cookies on.

**Residue — open in [BACKLOG.md](BACKLOG.md) §1:** backups cron + tested
restore drill, real SMTP relay, Google OAuth prod redirect + consent
screen, the Telegram alert channel.

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
- [x] **Customer-facing cancellation** while `pending` (Phase 11 line):
      one endpoint (the domain state machine already allows it and
      restores stock + promo redemptions), one button on the order page.
      *Done 2026-08-19 (decision #93): `POST /orders/{id}/cancel`,
      pending-only window as a domain rule distinct from the machine
      (`ErrTooLateToCancel` → 409), shared cancel transaction with the
      admin path, cancelled letter sent, armed button on the order page
      ×3 locales.*
- [x] **Promo code CRUD** in the admin (E7 shipped seed-only codes;
      "revisit when three codes stop being enough"). *Done 2026-08-19
      (decision #94): GET/POST/PUT `/admin/promos`, domain validation
      mirroring migration 000018's CHECKs, whole-value updates, NO delete
      (redemption history hangs off the code — `active` is the off
      switch), admin Promos tab with list + form.*
- [x] **Category management**: edit / delete-or-deactivate / reorder
      (Phase 11 line; the footer and facets already read sort_order).
      *Done 2026-08-19 (decision #95): settled as delete-when-empty (the
      schema's RESTRICT, no is_active column), whole-value edit with the
      editor's own raw-English read, positional reorder endpoint, admin
      list with move/edit/delete. Also FOUND AND FIXED: the category
      form's and collection's nested translations shape never matched the
      backend's flat map — creating a category with a translation 400'd.*
- [x] **Admin user management**: promote/demote via UI instead of SQL
      (Era I leftover), guarded so the last admin cannot demote themself.
      *Done 2026-08-19 (decision #96): GET `/admin/users` +
      PATCH `/admin/users/{id}/role`; "at least one admin" enforced as a
      count-under-locks invariant (race-tested like the oversell), 409
      `last_admin`; admin Users tab with role badges and (you) marker.*
- [x] **Account deletion + data view** (the privacy page's promises, §1):
      delete cascades the personal graph, orders are retained and
      detached per the stated bookkeeping rule. *Done 2026-08-19
      (decision #97): migration 000025 (orders.user_id nullable, FK stays
      RESTRICT), DeleteAccount as one transaction honoring the page's
      sentence, GET /account/data composing the screens' own reads, the
      settings stub replaced by the real password-confirmed armed flow +
      data download ×3 locales.*

**Done when:** a full order lifecycle — including getting paid, a status
email, and a customer cancellation — happens without anyone opening psql,
and the privacy page tells no lies. ✅ **PHASE F2 COMPLETE 2026-08-19**
(decisions #91–#97, learning log entries of 2026-08-18/19): every item
shipped; the mark-paid button, status mails, customer cancel, promo CRUD,
category management, role management, and the privacy promises all exist.

---

### Phase F3 — Launch content (the business half, then its code)

**Goal:** the shop stops being a beautifully-typeset placeholder.

*All six items (photography, image pipeline, real prices, real contacts,
native hy/ru review, newsletter sender) → moved to
[BACKLOG.md](BACKLOG.md) §2 on 2026-09-03. Progress note kept: the home
hero carries its first real image since 2026-08-19 (an honestly-labelled
AI stand-in; the family's shot list still gates the rest).*

**Done when:** a stranger could not tell which parts of the shop began as
placeholders.

---

### Phase F4 — Real payments

**Goal:** the card stops being a stub. The research was parked in Phase 11
from the start: **Idram / Ameriabank vPOS** (the mock's own ArCa hint), or
Stripe if the local rails disappoint.

*All four items (provider research, hosted-entry integration via F2's
write path, refunds, checkout swap) → moved to [BACKLOG.md](BACKLOG.md)
§3 on 2026-09-03, plus the webhook-recording design question the F2 work
raised.*

**Done when:** a real dram moves in a sandbox end to end, and the order
record shows paid-by-card without a human flipping anything.

---

### Phase F5 — Conversion & convenience (deferred UX, by evidence)

Deliberately after launch: each of these was cut with a reason, and real
traffic should confirm the reason wrong before it is built.

*All open items (anonymous carts/wishlists, address picker, Apple
sign-in, smarter upsell + stackable) → moved to
[BACKLOG.md](BACKLOG.md) §4 on 2026-09-03, joined there by
PLAN_ACCOUNT's four deferrals (back-in-stock, apiary pickup, SMS
notices, wishlist price-drop sender).*

- [x] ~~The LanguageSwitcher placement review~~ — settled 2026-08-18 the
      other way (decision #90): removed from the footer entirely, the
      account settings screen is the one home. *(And re-reversed by #98
      on 2026-08-19: the footer switchers returned — the settings screen
      is sign-in-only, anonymous visitors had no control. Closed both
      ways; the log tells the story.)*

---

### Phase F6 — Engineering debts & evolution

The consciously-carried debts, each with its written tripwire:

- [x] ~~**Drop `products.image_url`**~~ — closed 2026-08-20 by the
      product-media work (decision #99, migration 000027) — the
      add-backfill-drop pattern's last step finally paid.
- *Everything else (auth hardening, rate-limiter tripwire, OpenAPI,
  prerender/SSR — now unblocked by hosting, DB-backed pages tripwire,
  observability round two + the k6 re-run on the real laptop, the
  only-if-measured shelf, the agentic-browsing watch) → moved to
  [BACKLOG.md](BACKLOG.md) §5 on 2026-09-03, joined there by the
  post-audit additions: dead i18n keys, admin code-split, CI hygiene,
  and the tripwire table (§6).*

---

## 4. Suggested order

*(Historical — both nominated phases are done: F2 closed 2026-08-19, F1
shipped 2026-09-02. Sequencing of what remains is
[BACKLOG.md](BACKLOG.md)'s job now; its §1 "go-live residue" — backups
drill first — is the successor of this section's advice.)*

F1 unblocks everything and needs mostly non-coding work (ISP or VPS call,
DNS, drills) — start it immediately and let it run in the background.
F2 is pure coding and small — it can be finished before hosting lands, so
launch day is not spent building the "mark paid" button. F3 waits on
photography but its code half (pipeline) is scopeable now. F4 needs F1's
domain for provider callbacks. F5 and F6 are evidence-driven and permanent.

The first cut worth a session: **F2's payment-status write path** — the
one item that is a bug-shaped hole in an already-shipped promise rather
than a feature.

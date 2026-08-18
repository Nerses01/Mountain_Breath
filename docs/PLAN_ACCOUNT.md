# Mountain Breath — Plan: the account section redesign (canvas 07–10)

> A second design canvas arrived on 2026-08-18:
> [docs/design/Mountain Breath Account.dc.html](design/Mountain%20Breath%20Account.dc.html)
> — four logged-in screens (**07 Orders, 08 Wishlist, 09 Addresses,
> 10 Settings**) whose numbering continues the store canvas's screens.
> They share one shell: the store header with the **account menu open
> state**, and a **left rail** that swaps content on the right.
>
> Era II's E8 built the *capabilities* (auth, wishlist, address book,
> order history); this canvas now draws the *rooms* they live in. So this
> plan is mostly frontend re-architecture plus a handful of genuinely new
> backend surface (profile fields, change-password, reorder). It is a
> separate file because it is neither Era II (closed) nor Era III's
> go-live track — phases here are lettered **A1–A5** (A for account) and
> can interleave with F-work. Where a checkbox overlaps an Era III item
> (account deletion, checkout address picker), the F-phase keeps
> ownership and this plan only draws its doorway.
>
> Rule #16 applies to this canvas exactly as to the first: it is the
> source of UI truth, greppable rather than re-fetched; accessibility
> overrides it; states it never draws are ours to design; post-mock
> requirements (the **three** languages — the canvas shows only two) have
> no guidance in it. **A1 adds this file to CLAUDE.md's rule-#16 line so
> the next session knows there are two canvases.**

---

## 1. What the canvas changes

### 1.1 Gap table: canvas → backend

Everything the four screens show that the current API cannot serve.

| Canvas element | Today | Gap | Phase |
|---|---|---|---|
| Profile card: "Anahit Sargsyan", initials avatar | `User` has only email/role/hive | `users.full_name` (+ phone), profile PATCH | A5 |
| Settings profile: Phone field | orders snapshot one, user stores none | same migration as above | A5 |
| Password "Change" (while signed in) | only the emailed reset flow (E8) | `POST /account/password` (current + new) | A5 |
| Order tracker with per-step dates (14 Aug / 15 Aug / Today) | only `orders.created_at` | status-change timestamps (decision #3) | A2 |
| "Reorder" on every past order | — | reorder path (decision #4) | A2 |
| "chilled parcel" tag on an order | cart has `has_cold_chain`, order doesn't expose it | expose on the order DTO | A2 |
| Wishlist "saved 2 weeks ago" | `GET /wishlist` returns bare products | `saved_at` in the response | A3 |
| Wishlist "Notify me" on out-of-stock | — | back-in-stock subscriptions — **deferred** (§2 #6) | — |
| "Price-drop alerts" toggle | — | notification preferences (decision #5) | A5 |
| Notifications panel: 4 toggles | newsletter double opt-in only | same decision #5 | A5 |
| Per-address "Leave with the neighbour" | `leave_with_neighbour` is per-ORDER (E6) | per-address flag prefill (decision #7) | A4 |
| "Pickup from the apiary" card | — | pickup as a delivery method — **deferred** (§2 #6) | — |
| "Delete account" row | — | **owned by Era III F2** (the privacy-page promise); A5 draws the UI when F2's endpoint exists | A5/F2 |
| "Show prices in USD+AMD / USD / AMD" | dual display always on (E5) | display preference (decision #8) | A5 |

### 1.2 Gap table: canvas → frontend

| Canvas element | Today | Phase |
|---|---|---|
| Shared account shell: left rail + swapped right pane | three unrelated pages, three `Shell`s | A1 |
| Rail: dark profile card (name, avatar, orders/member stats) | plain profile section on `/account` | A1 |
| Rail: nav with counts (orders 7, wishlist 5, addresses 2) | two bare links | A1 |
| Rail: contextual promo card (reorder / alerts / pickup) | — | per-screen, A2–A4 |
| Header: account pill "Anahit ▾" with open menu | icon link to `/account` | A1 |
| Orders: filter pills (All / On the way / Delivered) | flat list | A2 |
| Orders: highlighted active order with 4-step tracker | `OrderCard` list | A2 |
| Orders: "#MB-1047" order-number style | raw numeric id | A2 (display only) |
| Orders: "Show 3 older orders ▾" collapse | full list always | A2 |
| Wishlist: canvas card (saved date, Add / Notify me, badge) | reuses shop `ProductCard` | A3 |
| Wishlist: header worth-total + "Add all to cart" | — | A3 |
| Wishlist + addresses: dashed "add more" slot | — | A3/A4 |
| Addresses: card grid, Default badge, per-card note row | single-column list | A4 |
| Settings screen (profile, language & currency, notifications, delete) | does not exist | A5 |

### 1.3 Canvas departures already decided by standing rules

Recorded here so nobody "fixes" them back toward the mock:

- **Orders show ONE currency.** The canvas prints $64.00 over 30,600 ֏ on
  an order; E5/E6 decided an order carries the single currency the
  customer was charged in ("showing a converted alternative beside a
  charge invites *but you billed me the other number*"). The muted second
  line stays on carts and catalogs, **not** on orders. Departure noted per
  rule #16.
- **"Track parcel" has nothing to track.** No carrier integration exists
  or is planned before F-era; the button renders as the door to the order
  detail page (or not at all — see decision #2's tracker mapping).
- **Three languages, not two.** The settings language picker gets en/hy/ru
  segments; the canvas predates Russian (standing exception).
- **The canvas's figures are placeholders** (Era II §1.1 rule): "8%
  member price" happily matches the real `hive.member_discount_percent`,
  but 7 orders / $101 wishlist worth / Saturday pickup hours are
  illustrative.

---

## 2. Decisions — all settled 2026-08-18

The user's calls, each confirmed explicitly (log rows #84–#89 in
ARCHITECTURE.md). Numbers stay stable once referenced.

- [x] **1. Routes: nest under `/account/*`.** The canvas's shell
      implies `/account/orders`, `/account/wishlist`,
      `/account/addresses`, `/account/settings` as nested routes under
      one layout. This **reverses E8's recorded choice** ("order history
      deliberately STAYS at `/orders`" — AccountPage.tsx's own comment):
      that choice predated any design for the account area, and the
      canvas is the newer authority. Old paths (`/orders`,
      `/orders/:id`, `/wishlist`) become client redirects in every
      locale prefix — emailed links and bookmarks must keep working.
      *(Decided 2026-08-18, decision log #84.)*
- [x] **2. Tracker maps to the machine we have** — Placed (pending) →
      Confirmed → Shipped → Delivered; steps renamed in copy only, no
      invented courier states ("out for delivery" exists nowhere in the
      system). Cancelled orders show a flat cancelled band, a state the
      canvas never draws (ours to design).
      *(Decided by user 2026-08-18, log #85.)*
- [x] **3. Status history: the `order_status_events` table** — one row
      per transition, inserted inside the existing domain-validated
      transition path; existing orders backfill one synthetic `pending`
      event from `created_at`. Chosen over "dates only where known"
      because history not recorded now is unrecoverable later — the one
      item in this plan that gets more expensive by the day.
      *(Decided by user 2026-08-18, log #85.)*
- [x] **4. Reorder is a server endpoint** — `POST /orders/{id}/reorder`:
      one transaction, the server decides what is still purchasable,
      returns the refreshed cart plus what was skipped ("2 of 3 added,
      royal jelly sold out"). A3's "Add all to cart" reuses the same
      partial-success contract.
      *(Decided by user 2026-08-18, log #86.)*
- [x] **5. Notification preferences: real toggles + honest stubs.**
      Persist only channels with a sender: order-update emails (columns
      on `users`, read by F2's status-change mailer when it lands) and
      harvest notes wired to the existing newsletter subscription.
      Wishlist alerts and SMS render as E6/E8-style stubs — disabled,
      truth stated — until their machinery exists; their columns arrive
      with their senders. *(Decided by user 2026-08-18, log #87.)*
- [x] **6. Deferred to Phase 11's parking lot** (lines added there, this
      canvas as design reference): back-in-stock "Notify me", apiary
      pickup as a delivery method, SMS delivery-day notices, the
      wishlist price-drop sender. Whether their cards render in the E6
      stub style or not at all is settled per screen at A3/A4 time.
      *(Decided by user 2026-08-18.)*
- [x] **7. `leave_with_neighbour` becomes a per-address flag** (A4
      migration) that PREFILLS checkout; the order keeps its own
      snapshot — prefill is a suggestion, the order is a record, exactly
      the E6 address-snapshot pattern.
      *(Decided by user 2026-08-18, log #88.)*
- [x] **8. Currency display preference is client-side** — localStorage +
      context like the locale mechanism, zero backend; promoted to a
      `users` column only if a signed-in-on-two-devices complaint ever
      actually arrives. Orders are unaffected either way (§1.3).
      *(Decided by user 2026-08-18, log #89.)*

---

## 3. The phases

Sequential: A1 is the shell everything else mounts into; A2–A4 restyle
one screen each and can land in any order; A5 carries the only sizeable
migration. Every phase: strings ×3 locales, tests at the layers it
touches (rule #11), Postman collection in the same commit as any route
change, LEARNING_LOG entry after.

### Phase A1 — The shell: routes, rail, header menu

**Goal:** one `AccountLayout` owns the two-column frame; the four
screens become its children; the header grows the canvas's account menu.

- [x] Decisions #1 (routes) recorded (log #84); CLAUDE.md rule-#16 line
      lists the second canvas.
- [x] `AccountLayout`: the `300px + 1fr` grid (canvas), collapsing to a
      single column below `lg` — the canvas is a 1440px drawing; the
      small-screen rail (stacked above the pane) is ours to design.
      Signed-out guard lives here once, replacing the three per-page
      `signInRequired` blocks.
- [x] Rail part 1 — the dark profile card: initials avatar, name (email
      until A5 ships `full_name`), hive standing, the two stat tiles
      (orders count, member % — a first-delivery-free tile for
      non-members, a state the canvas never draws).
- [x] Rail part 2 — the nav list: four items with active state (the
      canvas's orange fill rendered as `--color-brand-ink`: white on
      #E4761F fails AA, the same substitution every button makes),
      counts on orders/wishlist/addresses, divider, **Log out**.
- [x] Nested routes per decision #1 + `LegacyRedirect` from the old
      paths, in all three locale prefixes (param-preserving:
      `/orders/42 → /account/orders/42`). The old AccountPage split
      apart: profile → rail, address book → AddressesPage, an interim
      read-only SettingsPage holds the pane until A5.
- [x] Header: the signed-in icon becomes the canvas's dark pill (avatar
      initial + email local part + ▾) opening an owned menu — the
      **menu button** ARIA pattern (`aria-haspopup="menu"`, roving
      tabindex, Escape returns focus), a sibling of PillSelect's
      combobox but a different pattern: menus fire actions, comboboxes
      pick values. Signed-out keeps the icon link to `/login`.
- [x] Component tests: layout guard, rail counts + aria-current, menu
      keyboard contract, param- and locale-preserving redirects.

**Done when:** all four paths render inside one shell with correct
active states, the old URLs redirect, and the header menu passes a
keyboard-only walk.

### Phase A2 — Orders (canvas 07)

**Goal:** the flat list becomes the canvas's page: filters, the
highlighted active order with its tracker, compact history rows.

- [x] Decisions #2, #3, #4 recorded first (log #85, #86).
- [x] `order_status_events` migration (000021, up→down→up tested): the
      insert inside both write paths (CreateOrder shares the order's own
      timestamp; UpdateOrderStatus in the same tx as its UPDATE), events
      in the order DTO, backfill = one synthetic `pending` event per
      existing order. Postman updated (order shape + new request). Bonus
      from the gap table: `has_cold_chain` exposed on the order DTO,
      derived live through the items' product joins.
- [x] Filter pills (All / On the way / Delivered) — client-side over the
      already-fetched list; a pill whose count is zero does not render.
      `aria-pressed` toggles, not tabs — there is no tabpanel here.
- [x] `OrderTracker`: the four mapped steps, done/current/future states,
      per-step dates only from RECORDED events (unrecorded steps show a
      dash); cancelled band. An `<ol>` with `aria-current="step"`.
- [x] The active-order card (the orange border keeps `--color-brand` —
      a border carries no text) for the newest still-moving order:
      tracker + items band + Details link standing in for "Track parcel"
      (§1.3). No active order → the history list leads. Item thumbnails
      dropped: order items snapshot no image, and inventing one from the
      live product would mislabel renamed products.
- [x] History rows (id, date · items, summary, total in the ORDER's one
      currency, status chip, Reorder on delivered/cancelled rows);
      "#MB-{{id}}" display format in all three catalogues (e2e heading
      regex updated with it).
- [x] Reorder per decision #4: the endpoint merges with caps (stock and
      the cart's 99-per-line rule) and reports per-line issue codes the
      client translates (the promo_issue contract); the page banner
      names the skips, and the rail's "Reorder in one tap" card (latest
      delivered order) calls the same path with its own confirmation.
- [x] "Show N older orders ▾" collapse past the first three history rows.
- [x] `OrderDetailPage` renders the same tracker (it is the "Details"
      destination and must not look like a different site).
- [x] Tests: tracker table-driven per status incl. cancelled and the
      dates-only-from-events rule; store suite for events (create,
      transition, attach on list/get) and reorder (merge, cap, out-of-
      stock, retired product, stranger → ErrNotFound); handler tests
      (200 report / 404 stranger / 401 anonymous); fakeStore grew
      Reorder with the real ownership contract.

**Done when:** a pending→delivered order walks the tracker with real
dates, reorder refills the cart and says what it skipped, and the page
matches canvas 07 at 1440px.

### Phase A3 — Wishlist (canvas 08)

**Goal:** the saved shelf gets its own card (not the shop's), a header
that totals it, and one-tap add-all.

- [x] `GET /wishlist` gains `saved_at` per item — `domain.WishlistItem`
      embeds Product, the DTO embeds productResponse (flat JSON, one new
      field). Postman updated.
- [x] `WishlistCard`: canvas layout — image, shared heart (un-heart IS
      removal), badge, name, size · saved-ago line
      (`Intl.RelativeTimeFormat` speaks all three locales), "from" price
      with muted second line, Add / out-of-stock state. **"Notify me" is
      ABSENT, not stubbed** (decision #6): a per-card dead button is the
      decorative control the project refuses — the sold-out label plus a
      disabled Add tells the truth without promising an alert. Departure
      recorded here.
- [x] Header row: count + worth-total (sum of the same card prices —
      display math, not money math) and **Add all to cart** =
      `POST /wishlist/add-all`, the reorder endpoint's sibling: same
      transaction shape, same per-line report (shared `ReorderReport`
      banner, extracted from OrdersPage). Per product the first variant
      with room is chosen, qty 1 merges.
- [x] The dashed "Save more from the shop" slot appended to the grid;
      the all-empty state keeps E8's existing copy inside the shell.
- [x] Rail promo card: "Price-drop alerts" — the E6/E8 honest stub per
      decision #87 (no live toggle; the card states it is not wired and
      what it will do when the mailer exists).
- [x] Tests: card component (relative dates under fake timers, in/out of
      stock, no-Notify-me, "from" price); store suite for saved_at and
      add-all (skip + merge-on-repeat); existing handler tests carry the
      shape change through the fakeStore.

**Done when:** the wishlist reads as canvas 08, remove/add still work,
and the worth-total agrees with the cards to the minor unit.

### Phase A4 — Addresses (canvas 09)

**Goal:** the E8 book, re-hung in the canvas's card grid. The CRUD,
validation-error plumbing and default-flag logic all survive untouched.

- [ ] Card grid (2-up ≥ lg): label + Default badge, address block, the
      note row, Edit / Make default / Remove as the canvas's text
      actions; the dashed "Add a new address" card replaces the header
      link as the primary add affordance.
- [ ] Decision #7: `addresses.leave_with_neighbour` migration + the flag
      in `AddressInput`/`AddressEntry`; checkout prefills it from the
      chosen address. Postman updated. (If declined: the note row renders
      only on the canvas's checked style when the DEFAULT address's last
      order said so — decide, don't fudge.)
- [ ] The add/edit form keeps its field keys (server errors must keep
      landing on inputs) but moves into the pane per the canvas —
      inline card-turned-form or a panel below the grid, ours to design;
      the canvas only draws the resting state.
- [ ] Delete confirms before firing (destructive, and the canvas never
      draws the confirm — ours).
- [ ] Pickup card per decision #6 (stub style or absent).
- [ ] Tests: existing AddressBook tests migrate with the markup; new
      flag round-trips through create/update.

**Done when:** the book matches canvas 09, a full
add→edit→default→remove cycle works with server-side validation errors
landing on the right inputs, and checkout's prefill honours the flag.

### Phase A5 — Settings (canvas 10) — the backend phase

**Goal:** the one screen with no predecessor, and the only real schema
work: who you are, how the site speaks to you, and the exit door.

- [ ] Migration: `users.full_name`, `users.phone` (nullable — every
      existing user lacks them; the profile card and checkout prefill
      cope with absence). Down migration drops them.
- [ ] `PATCH /account/profile` (name, phone) + `useUpdateProfile`;
      `/auth/me` and the `User` type carry the new fields; the rail's
      profile card and header pill prefer the name over the email.
- [ ] `POST /account/password` — current password verified, then the
      same hash-and-rotate the reset flow uses, **all other sessions
      revoked** (the reset flow's rule; changing a password while a
      stolen session stays alive would be theatre).
- [ ] Profile panel per canvas: read-only fields + Edit mode, password
      row opening the change form (three fields; the canvas draws none
      of it — ours, reusing the reset page's strength rules).
- [ ] Language & currency panel: language segments (en/hy/ru, §1.3)
      driving the existing locale mechanism; currency display segments
      per decision #8.
- [ ] Notifications panel per decision #5: real toggles persisted, dead
      channels in the stub style with the truth stated.
- [ ] Delete-account row: renders against **F2's** endpoint when it
      exists; until then the row is the honest stub linking the privacy
      page. The confirm flow (type-to-confirm or password re-entry) is
      designed here since the canvas only draws the button.
- [ ] Tests: password-change handler (wrong current, session
      revocation), profile validation, store round-trips; fakeStore
      grows the methods; Postman gains all new routes.

**Done when:** name/phone/password change end-to-end with sessions
revoked on password change, preferences persist, and no toggle on the
screen lies about what it does.

---

## 4. Order of work & cross-references

A1 first (everything mounts in it). A2 is the richest screen and the
best second. A5 last on purpose: its migration is this plan's only
schema risk, and by then the shell is proven. Interleaving with Era III:
**F2's account-deletion endpoint** is A5's missing half — if F2 lands
first, A5 just wires the button; **F5's checkout address picker** gets
easier after A4's card grid exists (same component, selectable). Neither
plan blocks the other.

Deferred to Phase 11's parking lot with this canvas as design
reference: back-in-stock notifications, apiary pickup as a delivery
method, SMS delivery-day notices, wishlist price-drop emails (the
sender half), "member price" strikethrough pricing on wishlist cards.

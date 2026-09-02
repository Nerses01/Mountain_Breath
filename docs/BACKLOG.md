# Mountain Breath — Backlog (the one list)

> **Made 2026-09-03, the day after launch** (mountainbreath.net live,
> CD green — decisions #100/#101), by a full re-read of every plan:
> PROJECT_PLAN Phases 9–11, PLAN_ERA_2's open decision, PLAN_ERA_3
> F1–F6, PLAN_ACCOUNT's deferrals, and every "Questions / to revisit"
> in the learning log since the 2026-08-15 audit.
>
> **This file replaces Phase 11 and the Era III open lists as the one
> home of open work.** The plan documents keep their history and point
> here; an item leaving this file means it shipped (tick it with the
> decision number) or was rejected (strike it with the reason). Anyone
> adds lines anytime — a line is enough; picking an item for work means
> scoping it into a session/phase with a definition of done, as always.
> Sources in parentheses.

---

## 0. Reconciliation — closed since the 2026-08-15 audit, differently or at all

- ~~F1 "Go live"~~ — **shipped 2026-09-02, but not as written.** The
  plan said VPS-or-ISP-unlock, Caddy TLS, port forwarding; reality
  (LIKENET runs CGNAT — no inbound at any setting) forced decision
  #100: home laptop + owned domain + **Cloudflare named tunnel**,
  HTTPS at the edge, zero open ports, CD push over Tailscale.
  Hardening, DNS, `DEPLOY_ENABLED=true`, first automated deploy: all
  done (runbook: DEPLOYMENT_HOME.md). Residue → §1.
- ~~F6 "Drop `products.image_url`"~~ — closed 2026-08-20 by decision
  #99 (migration 000027), before this consolidation.
- ~~"Notification channel when it runs on the real server"~~ — no
  longer parked: the observability trio runs **in prod** since decision
  #101; the channel itself is §1's alerts item.
- **Unblocked by hosting existing**: prerender/SSR for meta and the k6
  re-run were both "behind hosting" — they now have a machine to run
  against (§5).

---

## 1. Go-live residue — the do-next shelf

The gap between "deploys itself" and "operated like you mean it":

- [ ] **Backups: cron + a tested restore drill** (runbook §Backups;
      F1's line) — *the* open gate. An untested backup is a hope; do
      the drill before real customers exist.
- [ ] Off-machine backup copies — automate the `scp` over tailnet to
      the working PC (or rclone to any free object storage); the laptop
      is a single point of failure *in the house* (log 09-02).
- [ ] Verify prod is **seeded and an admin exists** (runbook first-boot
      tail) — confirm the live DB isn't running on manual fumes.
- [ ] **Real SMTP relay** for production mail (`MB_SMTP_*`,
      `MB_MAIL_FROM` on `mountainbreath.net`) — today reset links and
      order mails land in the api log (F1 sweep line).
- [ ] **Google sign-in on prod**: OAuth client gains the
      `https://mountainbreath.net` redirect URI; consent screen leaves
      Testing mode (E8's checklist; F1 sweep line).
- [ ] `www.mountainbreath.net` as a second tunnel public hostname
      (30 seconds in the Zero Trust dashboard; session 09-02).
- [ ] **Alerts reach a phone**: Telegram receiver — the recipe is
      commented in `deploy/observability/alertmanager.yml`; the token
      mounts as a file, chat id is not a secret (Phase 10's parked
      line; F1's last line).
- [ ] Confirm the laptop's tailnet **key expiry is disabled** (console
      → Machines → homeserver badge) — silent CD death in ~180 days
      otherwise (session 09-02).
- [ ] CI hygiene, when convenient: bump the deploy job's
      `actions/checkout@v4` → v5 (node20 deprecation warnings); cache
      the Playwright browser (`~/.cache/ms-playwright` keyed by
      Playwright version, ~1 min/run) (log 09-02).

## 2. Launch content (was Era III F3)

**Goal unchanged:** the shop stops being a beautifully-typeset
placeholder. Photography gates most of it.

- [ ] **Photography** — the family's real shot list: jars, combs, the
      meadow, the family. The home hero carries an honest AI stand-in
      since 08-19; every other slot is a designed placeholder.
- [ ] **Image pipeline**, built against the real files: server-side
      thumbnails/resizing on upload, `srcset` + AVIF/WebP, explicit
      dimensions at every `<img>`; a video **poster frame** extracted
      at upload joins here (log 08-20); S3-compatible storage if the
      volume model chafes (P11).
- [ ] **Real prices** in both markets + real shipping rates/threshold —
      data entry by design, not a deploy.
- [ ] **Real contact details** on the Contact page (it still promises
      `hive@mountain-breath.example` and `+374 91 00 00 00`).
- [ ] **Native review of Armenian and Russian copy** — accumulated
      E1.5→E9 + the three mail templates; legal pages and the reset
      email (it lands in strangers' inboxes) first.
- [ ] **Newsletter sender** (E9's other half): compose an issue, send
      to `confirmed AND NOT unsubscribed`, unsubscribe links on the
      schema's permanent tokens; admin-triggered first.
- [ ] Account **data export, human-readable** — the JSON answer keeps
      the promise; decide whether F3's content pass wants an HTML
      rendering (log 08-19).

## 3. Real payments (was Era III F4)

- [ ] Research + pick: **Idram / Ameriabank vPOS** (the mock's ArCa
      hint) or Stripe if local rails disappoint; sandbox account.
- [ ] Integration as E6's stub anticipated: provider hosts card entry
      (no PANs touch the API), webhook flips `payment_status` through
      F2's write path (#91).
- [ ] Design question to settle at the webhook: same store method, or
      a `payments/events` table so a provider event is its own
      recorded fact, as status transitions got (log 08-18)?
- [ ] Refunds on the same path; `refunded` finally reachable by flow.
- [ ] Checkout's decorative card fields become the provider redirect.

**Done when:** a sandbox dram moves end to end and the order shows
paid-by-card with no human flipping anything.

## 4. Conversion & convenience — by evidence (was F5 + PLAN_ACCOUNT deferrals)

Each was cut with a written reason; real traffic should prove the
reason wrong before it gets built.

- [ ] **Anonymous carts** (+ merge on login) and anonymous wishlists —
      the biggest funnel-friction item (P11/E8).
- [ ] **Checkout address picker** over the address book (`address_id`);
      wire the per-address `leave_with_neighbour` flag to swap with the
      selection (A4's note, log 08-18).
- [ ] **Apple sign-in** — if the family approves $99/yr; adapter scoped
      in E8's notes (form_post callback, JWT client secret).
- [ ] **Smarter upsell** in the free-shipping banner + the `stackable`
      promo flag — when current behaviour demonstrably misses (E7).
- [ ] **Back-in-stock "Notify me"** (wishlist canvas 08) — deferred
      2026-08-18 (PLAN_ACCOUNT §2 #6).
- [ ] **Apiary pickup** as a delivery method (canvas 09; touches
      checkout pricing) — same deferral.
- [ ] **SMS delivery-day notices** (+374; canvas 10's fourth toggle) —
      same deferral.
- [ ] **Wishlist price-drop / member-offer sender** (canvas 08's rail
      card promises it; the page says "not wired up yet") — same
      deferral.

## 5. Engineering debts & evolution (was F6, updated)

- [ ] **Auth hardening batch**: security headers — decide the home now
      that the edge is Cloudflare (edge rules vs nginx) — plus cleanup
      jobs for expired `sessions` rows (verified to accumulate) and
      spent reset tokens (P11).
- [ ] **Rate limiter to shared storage** — tripwire unchanged: the day
      the API runs a second replica.
- [ ] **OpenAPI spec + generated TS client** — retire hand-maintained
      `types.ts`; the Postman collection is the seed of truth (P11).
- [ ] **Prerender/SSR for meta** — now unblocked; JS-managed tags reach
      only rendering crawlers (E10).
- [ ] **DB-backed pages/journal** — tripwire: the family needs to edit
      copy without a commit (#77); also repairs journal posts missing
      from the backend sitemap.
- [ ] **Observability round two**: log aggregation, OpenTelemetry
      tracing; **k6 re-run against the laptop** — tighten the 200ms SLO
      the 11.6ms p95 mocks, then find the real breaking point (P10).
- [ ] **Dead i18n keys sweep** — `ordersTitle`, account `title` ("Your
      account"), `addresses.title` ("Address book") and whatever else
      the A-phases orphaned (log 09-02).
- [ ] **Code-split the admin area** (~97 KiB unused JS on customer
      pages per Lighthouse; `import()` so shoppers never download it)
      (session 09-02).
- [ ] **Only-if-measured shelf**, unchanged: Redis, CDN, catalog cache
      headers ("an invalidation bug on layaway"), Terraform/Ansible,
      Kubernetes ("only after Compose feels limiting").
- [ ] Watch the unasserted `agentic-browsing` Lighthouse category (E10).
- [ ] **Product editorial fields** (E3): explicit columns/child tables
      vs one JSONB `content` column — the one PLAN_ERA_2 decision still
      open (its §decisions #4).
- [ ] Watch the e2e purchase spec's cart-empty step — flaked once on a
      congested runner (09-02); two more and it earns investigation.
- [ ] **VPS someday**: keep the tunnel (works anywhere) or reinstate
      Caddy from the kept `deploy/Caddyfile` and reclaim #12's
      end-to-end TLS (log 09-02).

## 6. Tripwires & open design questions

Not tasks — questions with a named firing event. When the event
happens, the question gets a session:

| Question | Fires when | Source |
|---|---|---|
| Promo quick-flip: whole-value PUT vs a PATCH write path | a second quick-flip use case appears | log 08-19 |
| `order_status_events` actor column (customer vs admin vs webhook) | F4's webhook becomes the third writer | log 08-19 |
| Mail links use legacy `/orders/{id}` | decide: switch builders to `/account/orders/{id}` or keep legacy paths in emails that outlive URL schemes | log 08-18 |
| Currency display pref is localStorage-only | a real two-devices complaint | log 08-18 (#89) |
| Deleted customer's order lookup (anonymized token on the goodbye screen?) | first support case about an old order from a deleted account | log 08-19 |
| Revoke sessions on admin demotion (UI lags with stale role; API is safe) | first real confusion caused by the lag | log 08-19 |
| Grafana access via `tailscale serve` instead of `ssh -L` | the ssh tunnel dance gets annoying | session 09-03 |

## 7. Learning shelf

- [ ] TypeScript fundamentals (official handbook) + React quick start —
      Phase 3's line, long since absorbed in practice; kept as a
      deliberate study pass through the handbook (P3).

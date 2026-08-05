# Mountain Breath — Plan, Era II

> The first era (Phases 0–11 in [PROJECT_PLAN.md](PROJECT_PLAN.md)) built a
> working store: catalog, auth, cart, transactional checkout, tests, CI,
> containers, metrics. It looks like what it is — a learning app with default
> Tailwind styling and a one-click checkout.
>
> Era II turns it into the store in the design: `Mountain Breath Store.dc.html`,
> six desktop screens (Home, Shop, Product, Cart, Checkout, Account).
>
> **Design source.** claude.ai/design project `Mountain Breath E-commerce Store`,
> id `70fac810-0193-46d2-979e-d1c281beeae2`
> ([open](https://claude.ai/design/p/70fac810-0193-46d2-979e-d1c281beeae2?file=Mountain+Breath+Store.dc.html)).
> The design is **not** copied into this repo on purpose — it is read live with
> the `DesignSync` tool (`get_project` / `list_files` / `get_file` by that id),
> so a stale duplicate can never disagree with the canvas. When the design
> changes, re-read it; when this plan and the canvas disagree, the canvas wins.
>
> Phases are numbered **E1–E10** so they never collide with Era I's 0–11.
> Phase 11 (Idea Backlog) stays permanently open and feeds both eras.

---

## 1. What the design changes

### 1.1 The product domain moves to the hive

The design is not a food & beverages shop. It is a **single-family apiary**
selling six bee products, and the whole information architecture follows from
that: the nav says "Our hive", the benefits panel is about what bee products do
in the body, and every product page carries a harvest hive number.

The design ships its own seed content (in its `renderVals()` block), which is a
gift — the copy is written:

| # | Product | Category | Size | Badge | "Good for" | USD | AMD |
|---|---|---|---|---|---|---|---|
| 1 | Mountain Wildflower Honey | Honey | 500 g jar | Best seller | Natural energy | $14.00 | 6,700 ֏ |
| 2 | Pure Beeswax Blocks | Beeswax | 4 × 100 g | For makers | Balms & candles | $9.00 | 4,300 ֏ |
| 3 | Raw Propolis Tincture | Propolis | 30 ml dropper | Immunity | Antimicrobial | $19.00 | 9,100 ֏ |
| 4 | Fresh Royal Jelly | Royal jelly | 25 g jar | Cold chain | Vitality & skin | $32.00 | 15,300 ֏ |
| 5 | Bee Pollen Granules | Bee pollen | 250 g pouch | Protein | Protein & minerals | $16.00 | 7,600 ֏ |
| 6 | Bee Venom Serum | Bee venom | 15 ml bottle | New | Apitherapy | $28.00 | 13,400 ֏ |

Two things fall out of this table:

- **The current seed is wrong for the design.** `backend/seed/seed.sql` has
  herbal tea, coffee and wildflower honey. Six new categories replace three.
- **AMD prices are not computed.** 14.00 → 6,700 ֏ implies ≈478 ֏/$, but so do
  9 → 4,300 (478) and 16 → 7,600 (475). They are *rounded per market*, not
  converted at a live rate. That is evidence for how to model currency (E5).

The royal jelly page also shows **non-linear variant pricing** — 25 g $32,
50 g $58, 100 g $105 — which the existing per-variant `price_minor` already
supports. No change needed there.

### 1.2 Gap table: design → backend

Everything the six screens show that the current schema and API cannot serve.

| Design element | Today | Gap | Phase |
|---|---|---|---|
| Product gallery (hero + 4 thumbs) | one `products.image_url` | `product_images` table | E3 |
| Card badges (Best seller, New, Cold chain…) | — | badge field or table | E2 |
| "Good for" sidebar facet + counts | — | benefit taxonomy + faceted counts | E2 |
| Price range slider ($9–$32) | — | price filter + facet bounds | E2 |
| Sort: "Most loved" | fixed order | sort param + popularity signal | E2 |
| Category counts in sidebar | — | grouped counts endpoint | E2 |
| "What it does" bullets + disclaimer | `description` only | highlights + disclaimer | E3 |
| Harvest / Shipping / Lab report cards | — | product metadata fields | E3 |
| How to take it / Storage tabs | — | usage content | E3 |
| "Often taken together" | — | related products | E3 |
| ★★★★★ (64 reviews) | — | reviews + rating aggregate | E4 |
| Every price shown in USD **and** AMD | single currency, `/100` | currency model + FX | E5 |
| Delivery address, phone, name | orders have none | address snapshot on order | E6 |
| Card / bank transfer / cash on delivery | admin confirms manually | payment method + status | E6 |
| Subtotal / shipping / discount / total | one `total_minor` | totals breakdown | E6 |
| "Prices include VAT" | — | contained-VAT field | E6 |
| "$8 away from free shipping" | — | shipping rules + threshold | E7 |
| Promo code box | — | promo codes + redemptions | E7 |
| "Hive club discount − $4.00", 8% member price | — | membership tier | E7 |
| Wishlist hearts, "Save for later" | — | wishlist | E8 |
| "Forgot password?" | — | reset tokens + email | E8 |
| "Keep me signed in" | fixed session TTL | persistent sessions | E8 |
| Continue with Google / Apple | — | OAuth identities | E8 |
| Our hive / Benefits / Journal | — | content pages | E9 |
| Newsletter "Join" | — | subscribers + double opt-in | E9 |

### 1.3 Gap table: design → frontend

| Design element | Today | Phase |
|---|---|---|
| Whole visual language (palette, Poppins/Karla, radii, shadows) | `@import 'tailwindcss'` and nothing else | E1 |
| Site header with nav, search, wishlist, cart pill | one flat bar, emoji logo | E1 |
| Site footer (4 columns + newsletter + currency) | none | E1 |
| **Home page** | does not exist — `/` is the catalog | E2 |
| Shop page with filter sidebar, sort, pagination | search box + chips + grid | E2 |
| Product page: gallery, tabs, meta, related | name, variants, add-to-cart | E3 |
| Cart: free-shipping progress, promo, summary card | plain list | E6/E7 |
| **Checkout page** | does not exist — `POST /orders` is one click | E6 |
| Account page, two-panel sign-in | plain login form | E8 |
| Responsive below 1440 px | untested; design is desktop-only | E10 |

### 1.4 Inconsistencies in the mock (content questions, not bugs)

Worth resolving before the copy is treated as canonical:

- The cart line for **Raw Propolis Tincture reads $16.00 / 7,600 ֏**, which is
  the Bee Pollen price; the shop card says $19.00 / 9,100 ֏.
- The cart totals are internally consistent ($62 + $6 − $4 = $64), and
  "$8 away from free shipping" on a $62 subtotal implies a **$70 threshold** —
  but chilled shipping is still charged, so the threshold presumably applies to
  standard shipping only. Confirm the rule.
- "Prices include VAT" sits next to a separate discount line. Decide whether
  VAT is contained in the displayed price (Armenian retail convention) and only
  broken out on the invoice — E6 assumes yes.
- The design shows `+374` phone, Yerevan address and **ArCa** card network:
  Armenia is the primary market, which supports the Idram / Ameriabank vPOS
  research already parked in Phase 11.

---

## 2. Decisions to make before E1

These are the user's calls, not technical defaults. Each gets a row in the
[ARCHITECTURE.md](ARCHITECTURE.md) decisions log once made (RULES.md #13).

1. **Catalog scope.** Does Mountain Breath become an apiary-only store (drop
   tea and coffee), or does the hive line sit next to them? The design's nav,
   hero and benefits copy assume apiary-only. Everything in E2 depends on this.
2. **Currency.** Ship both USD and AMD (E5), or AMD only for an Armenian
   launch and treat USD as later work? The design shows both on every price.
3. **Content storage.** "Our hive"/"Benefits"/"Journal" as markdown in the repo
   (versioned with the code, zero backend) or DB-backed so the family edits
   without a deploy? E9 recommends markdown for v1.
4. **Product editorial fields** (E3): explicit columns and child tables, or one
   JSONB `content` column? Columns give constraints and clean admin forms;
   JSONB avoids a migration per field but loses FK safety.
5. **Social sign-in.** Google/Apple are two buttons in the mock and the most
   third-party-dependent item in the plan. Confirm they are in scope, or drop
   the buttons from the design.

---

## 3. The phases

Same shape as Era I: **Goal**, **You will learn**, **Backend**, **Frontend**,
**Done when**. One phase at a time (RULES.md #6).

---

### Phase E1 — Design system & app shell

**Goal:** the visual language exists as tokens and primitives, and every
existing page already renders inside it.

**You will learn:** Tailwind v4's CSS-first `@theme` configuration, design
tokens vs. ad-hoc utility classes, building a small component library,
WCAG contrast maths and why it belongs at token-definition time.

**Backend:** none.

**Frontend:**
- [ ] Tokens in `src/index.css` via `@theme`: surfaces (`#F3E2D0` page,
      `#FDEFE0` panel, `#FEF4E8` header, `#FFF8EE` card), ink (`#46281C` bark,
      `#5C3B2A`, `#6E4B36`, `#7C5A45`, `#A9714B` muted), brand (`#E4761F`
      orange, `#F6C244` honey), border `#EED9C0`.
- [ ] **Fix the contrast failures while defining the tokens, not in E10.**
      Measured against the `#FDEFE0` panel:

      | Pair | Ratio | Verdict |
      |---|---|---|
      | `#6E4B36` body text | 6.8:1 | passes AA |
      | `#7C5A45` secondary | 5.5:1 | passes AA |
      | `#A9714B` muted (13 px) | 3.6:1 | **fails** AA 4.5:1 |
      | `#E4761F` orange text/price | 2.7:1 | **fails** even AA-large 3:1 |
      | `#FFF8EE` on `#E4761F` (primary CTA) | 2.9:1 | **fails** AA 4.5:1 |
      | `#FFF8EE` on `#B8541A` | 4.6:1 | passes AA |

      Recommended fix: keep `#E4761F` as a decorative/large-display accent, and
      add `--color-brand-ink: #B8541A` (the design's own link-hover colour) for
      orange **text** and for the primary button background. Restrict
      `#A9714B` to ≥18.66 px or bump it darker. Re-verify with axe in E10.
- [ ] Fonts: Poppins (400–800, display) + Karla (400–700, body), self-hosted
      with `font-display: swap` rather than the Google CDN link — the CSP-free
      route and one less third party. Type scale from the mock: display
      68/46/44/42/38/34/32/26, body 18/17/16/15/14/13, eyebrow 11–13 uppercase
      at 0.18–0.24em tracking.
- [ ] Primitives in `src/components/ui/`: `Button` (primary pill with the
      orange glow shadow, dark pill, outline, ghost-underline), `Badge`,
      `Card`, `Input`, `Select`, `Checkbox`, `QtyStepper`, `IconButton`
      (38 px circle, 1.5 px border), `Breadcrumbs`, `SectionHeading`
      (eyebrow + title + trailing link), `Stat`.
- [ ] **Focus states**: the mock has none. Every interactive primitive gets a
      visible `:focus-visible` ring in the honey token — decided here, once.
- [ ] `SiteHeader` (logo mark, wordmark + tagline, 5 nav links, search and
      wishlist icon buttons, cart pill with count) and `SiteFooter`
      (4 columns, newsletter form, bottom bar with the currency switcher slot).
- [ ] `Layout` route wrapper in `App.tsx`; `/admin/*` keeps its own chrome.
- [ ] Extend `public/icons.svg` with the sprite the design needs: search,
      heart, arrow-right, chevron-down, minus, plus, check, star.
- [ ] Vitest: `Button` variants render, `QtyStepper` clamps at 1 and at stock.

**Done when:** all eight existing routes render inside the new shell with the
new palette, no raw hex outside the token block, and every control is reachable
and visibly focused by keyboard.

---

### Phase E2 — Catalog model, faceted shop, home page

**Goal:** the Shop screen's sidebar works for real, and the Home screen exists.

**You will learn:** faceted search and why counts are expensive, aggregate
queries with `FILTER`, many-to-many taxonomies, URL as state (deep-linkable
filters), keyset vs. offset pagination.

**Backend:**
- [ ] Migration: `benefits` (id, slug, name, sort_order) and `product_benefits`
      (product_id, benefit_id, PK on both) — Energy, Immunity, Skin, Recovery,
      Sweetening.
- [ ] Migration: `products.badge` (nullable TEXT) + `badge_tone` — one badge
      per card in the design; a `product_badges` table only if that changes.
- [ ] Extend `domain.ProductFilter`: `Benefits []string`, `PriceMinMinor`,
      `PriceMaxMinor`, `Sort` (`popular|price_asc|price_desc|newest`).
      Validate `Sort` in the domain layer against a whitelist — never
      interpolate it into SQL.
- [ ] Popularity signal for "Most loved": denormalized `products.sales_count`,
      incremented in the checkout transaction (it is already open), with a
      backfill query in the migration. Compare in the log with the alternative
      (aggregate `order_items` on every list query) and why denormalizing wins
      here.
- [ ] `GET /api/v1/catalog/facets` → category counts, benefit counts, price
      bounds, respecting the *other* active filters. One round trip, CTEs +
      `count(*) FILTER (WHERE …)`.
- [ ] Rewrite `seed/seed.sql` for the six hive products with the design's copy,
      badges, benefits and both currencies' prices (or USD only until E5).
- [ ] Tests: store tests for each filter and sort, a facet-count test that
      proves counts change with the active filter, domain test for sort
      whitelisting.
- [ ] Update the Postman collection (RULES.md #15).

**Frontend:**
- [ ] `HomePage` at `/`: hero (headline, subcopy, two CTAs, 3-stat strip),
      "How we harvest" dark card + "What the hive does for you" panel, six
      product cards, story band, all from the API — no hardcoded product copy.
- [ ] `ShopPage` at `/shop`: breadcrumbs, result count, sort select, sidebar
      (`CategoryFilter` with counts, `BenefitChips`, `PriceRange` dual slider,
      "Ask a beekeeper" card), grid, pagination.
- [ ] **All filter state lives in the query string** via `useSearchParams`, so
      back/forward work and a shared link reproduces the exact view.
- [ ] `ProductCard` redesigned to the mock: image, badge, category eyebrow,
      name, "size · benefit", dual price, Add button, wishlist heart (inert
      until E8).
- [ ] Search moves from the catalog body into a header overlay, keeping the
      existing 300 ms debounce and the trigram behaviour.
- [ ] Vitest: `PriceRange` emits clamped values; `ProductCard` renders badge
      and out-of-stock states.

**Done when:** every filter, the sort and the page number survive a reload and
a copy-pasted URL; sidebar counts match the grid; `/` is the designed home page.

---

### Phase E3 — Product detail

**Goal:** the third screen, rendered entirely from API data.

**You will learn:** modelling editorial content in a relational schema,
ordered child collections, the ARIA tabs pattern, image galleries without a
library.

**Backend:**
- [ ] Decide decision #4 (columns vs JSONB) and log it.
- [ ] Migration: `product_images` (product_id, url, alt, sort_order,
      is_primary) with a partial unique index enforcing one primary per
      product. Backfill from `products.image_url`, then drop the column in a
      follow-up migration once the admin UI writes the new table.
- [ ] Migration: `product_highlights` (product_id, sort_order, text) for the
      "What it does" bullets; `product_usage_cards` (kicker, title, body,
      sort_order) for Morning / Course / Pairs with.
- [ ] Migration: `products.disclaimer`, `storage_note`, `harvest_note`
      ("June 2026, Hive 41"), `shipping_note` ("Chilled, 2–4 days"),
      `lab_batch` ("RJ-0626"), `is_cold_chain`.
- [ ] Related products: `product_related` (product_id, related_id, sort_order)
      curated by the admin, falling back to same-category-by-popularity when
      empty. `GET /products/{slug}/related`.
- [ ] Extend `GET /products/{slug}`; keep the list payload lean — the card does
      not need highlights or usage cards.
- [ ] Admin: extend the product form for images (multi-upload, reorder,
      set primary), highlights, usage cards and metadata.
- [ ] Tests: store test for ordering and the one-primary-image constraint; API
      test for the fallback path of `/related`.

**Frontend:**
- [ ] `Gallery`: hero + 4 thumbnails, arrow-key navigable, `alt` from the API.
- [ ] `VariantPicker` as labelled price pills ("25 g · $32"), disabled and
      marked when out of stock.
- [ ] `QtyStepper` + `AddToCart` with the price in the button label.
- [ ] "What it does" panel with the disclaimer in muted small print.
- [ ] Meta row: Harvest / Shipping / Lab report cards.
- [ ] `Tabs` (How to take it · Storage · Reviews) using the ARIA tabs pattern,
      with the active tab in the URL hash so a tab is linkable.
- [ ] `RelatedProducts` grid.

**Done when:** no string on the product page is hardcoded, the gallery and tabs
are operable by keyboard alone, and the admin can produce a complete product
page without SQL.

---

### Phase E4 — Reviews & ratings

**Goal:** the ★★★★★ (64 reviews) on the card and the Reviews tab are real.

**You will learn:** denormalized aggregates and how to keep them honest
(trigger vs. application-level), moderation workflows, "verified purchase"
as a join, preventing review spam.

**Backend:**
- [ ] Migration: `reviews` (product_id, user_id, rating 1–5 CHECK, title, body,
      status `pending|published|rejected`, created_at, UNIQUE(product_id,
      user_id)).
- [ ] `products.rating_avg` + `rating_count`, recomputed when a review's status
      or rating changes. Implement application-side first (inside the same
      transaction), then write up in the learning log why a trigger is the
      other option and what each costs. The list query needs the aggregate, so
      denormalizing is not optional.
- [ ] Verified purchase: a user may review a product only if they have a
      `delivered` order containing one of its variants — one EXISTS query,
      enforced in the store, surfaced to the API as a domain error.
- [ ] `GET /products/{slug}/reviews?page=`, `POST /products/{slug}/reviews`
      (login + purchase required), `GET /admin/reviews?status=`,
      `PATCH /admin/reviews/{id}` (publish/reject).
- [ ] `GET /products/{slug}` gains `can_review` so the UI need not guess.
- [ ] `sort=rating` joins the sort whitelist; "Most loved" can now be defined
      as sales or rating — pick one and say which in the log.
- [ ] Tests: aggregate stays correct after publish → edit → reject; a
      non-purchaser gets 403; the unique constraint blocks a second review.

**Frontend:**
- [ ] `Stars` component: accessible (`role="img"` + `aria-label="4.6 out of
      5"`), half-star rendering, one implementation used by card, detail and
      the review list.
- [ ] Review list with pagination inside the tab; `ReviewForm` shown only when
      `can_review`; admin moderation table.

**Done when:** ratings everywhere come from real rows, a stranger cannot
review, and moderation changes the public average immediately.

---

### Phase E5 — Dual currency (USD + AMD)

**Goal:** every price in the design shows two currencies, and an order is
unambiguously charged in one of them.

**You will learn:** why money is harder than a multiplication, per-market
pricing vs. FX conversion, currencies with different minor units, snapshotting
rates for auditability.

**Backend:**
- [ ] Migration: `currencies` (code, symbol, minor_exponent, rounding_step) —
      USD has 2 decimals, AMD is priced in whole drams. The existing
      `formatPrice` assumption that everything is `/100` breaks here.
- [ ] Migration: `variant_prices` (variant_id, currency, price_minor,
      PK(variant_id, currency)). **Recommended over live conversion**, and the
      design proves why: 14.00 → 6,700 ֏ and 9.00 → 4,300 ֏ are rounded
      shelf prices at ≈475–479 ֏/$, not one rate applied twice.
- [ ] Migration: `fx_rates` (base, quote, rate, as_of) as the *fallback* for a
      currency with no explicit price, and for reporting.
- [ ] Currency resolution per request: `?currency=` → cookie → `Accept-Language`
      → default; validated against the allowed set, never trusted raw.
- [ ] Orders snapshot `currency` and `fx_rate_used` alongside the existing
      price snapshots — decision #3's reasoning extended one step.
- [ ] Migrate `product_variants.price_minor` into `variant_prices` and keep the
      column until the admin UI is converted, then drop it.
- [ ] Tests: totals reconcile in each currency; AMD rounds to whole drams; an
      unknown currency code is rejected, not silently defaulted.

**Frontend:**
- [ ] `CurrencyProvider` + switcher in the footer bar, persisted in
      `localStorage` *and* a cookie so the server sees the same choice.
- [ ] Replace `formatPrice` with `formatMoney(minor, currency)` built on
      `Intl.NumberFormat`, driven by `minor_exponent` — no `/100` anywhere.
- [ ] `Price` component rendering the primary amount plus the muted secondary,
      used by card, product, cart, checkout and order history.
- [ ] Update the existing `format.test.ts` for the new signature.

**Done when:** switching currency changes every price on the site, an order
records what it was charged in, and no total is ever off by a rounding step.

---

### Phase E6 — Real checkout

**Goal:** screen 05 — a checkout that collects an address and produces an
itemised order.

**You will learn:** multi-section form design and validation that mirrors
server rules, why the server must own every number, snapshotting addresses,
modelling tax that is contained in the price.

**Backend:**
- [ ] Migration: `addresses` (user_id, first_name, last_name, phone, street,
      city, postal_code, country, is_default) for the address book, **plus
      snapshot columns on `orders`** — an order must not change when the
      customer later edits their address, exactly as prices are snapshotted.
- [ ] Migration: `orders.subtotal_minor`, `shipping_minor`, `discount_minor`,
      `tax_minor`, `total_minor` with a CHECK that
      `subtotal + shipping − discount = total` (tax is *contained* in
      subtotal per "Prices include VAT" — confirm in §1.4 first).
- [ ] Migration: `orders.payment_method` (`card|bank_transfer|cash_on_delivery`)
      and `payment_status` (`unpaid|paid|refunded`). Card stays a stub; the
      real provider remains Phase 11 work.
- [ ] Shipping: `shipping_rates` (method, base_minor, cold_chain_surcharge,
      free_over_minor) rather than constants in code — the family will change
      these without a deploy.
- [ ] `POST /orders` grows a request body (address or address_id, payment
      method, delivery note, "leave with the neighbour"). It keeps the single
      transaction and the ordered `FOR UPDATE` locks — the oversell test must
      still pass unchanged.
- [ ] **The client never sends money.** It sends items, address and method; the
      server computes and returns every figure. Add an API test that proves a
      client-supplied total is ignored.
- [ ] Field-level validation reusing the existing `fields` envelope with JSON
      paths the form can attach to (`address.postal_code`).
- [ ] Tests: totals arithmetic table-driven in the domain; a cash-on-delivery
      order lands `unpaid`; address snapshot survives an address edit.

**Frontend:**
- [ ] `CheckoutPage` at `/checkout`: step indicator (Details → Payment → Done),
      Contact section, Delivery address section, Payment method cards, card
      fields (stub), summary sidebar with line items.
- [ ] Keep validation hand-rolled to mirror the backend's field keys — the
      project has deliberately avoided form libraries; revisit only if this
      hurts. Note the decision either way.
- [ ] `OrderSummary` component shared by cart and checkout.
- [ ] `/orders/:id` confirmation and detail view with the full breakdown.
- [ ] Extend the Playwright journey: browse → add → checkout **with an
      address** → confirmation → order visible.

**Done when:** a real order carries an address, a method and five money fields
that reconcile, and the checkout screen's numbers come from the server.

---

### Phase E7 — Promotions, shipping progress, hive club

**Goal:** the promo box, the "$8 away from free shipping" bar and the member
discount.

**You will learn:** keeping pricing rules as a pure function, enforcing
redemption limits under concurrency, why one calculator must serve every
screen.

**Backend:**
- [ ] `domain.Price(cart, user, promo, shippingRate) → Breakdown` as a **pure
      function** — no DB, no HTTP, fully table-testable. This is the Era I
      layering rule paying off; it is also the single source of truth for cart,
      checkout preview and order creation.
- [ ] Migration: `promo_codes` (code, kind `percent|fixed|free_shipping`,
      value, starts_at, ends_at, max_redemptions, per_user_limit,
      min_subtotal_minor, active) and `promo_redemptions` (code_id, user_id,
      order_id) with a unique index that makes over-redemption impossible
      rather than unlikely.
- [ ] Migration: membership — `users.tier` (`guest|hive_club`) or a
      `memberships` table if it needs dates. Rules from the design: 8% off
      every order after the first, first delivery free.
- [ ] `POST /api/v1/checkout/preview` → the breakdown for the current cart,
      promo and user, without creating anything. Cart, checkout and the
      progress bar all call this.
- [ ] Concurrency test in the spirit of the oversell test: N goroutines
      redeeming a code with `max_redemptions = 1` → exactly one succeeds.

**Frontend:**
- [ ] Promo input with inline success/error from the envelope's `code`.
- [ ] Free-shipping progress bar + the upsell CTA ("Add pollen · $16").
- [ ] Discount and member lines in `OrderSummary`; member badge in the header.
- [ ] Every money figure on cart and checkout comes from `/checkout/preview` —
      no client-side arithmetic beyond formatting.

**Done when:** cart, checkout and the created order agree to the dram, and a
one-use code cannot be used twice under parallel checkouts.

---

### Phase E8 — Accounts: wishlist, password reset, sign-in

**Goal:** screen 06, plus the hearts scattered across screens 01–04.

**You will learn:** single-use hashed tokens (the session pattern reused),
transactional email, session lifetime policy, OAuth's authorization-code flow.

**Backend:**
- [ ] Migration: `wishlist_items` (user_id, product_id, added_at, PK on both).
      Login required — consistent with decision #9 on carts; anonymous
      wishlists stay in the backlog.
- [ ] "Save for later" = move a line from `cart_items` to `wishlist_items` in
      one transaction.
- [ ] Migration: `password_reset_tokens` (user_id, token_sha256, expires_at,
      used_at) — the same leak-resistant design as sessions (decision #8),
      single use, short TTL, and the request endpoint must answer identically
      whether or not the email exists (no user enumeration, as login already
      does).
- [ ] Transactional email (pulled forward from Phase 11): provider or SMTP,
      templates for reset and order confirmation, and a dev sink so tests never
      send mail.
- [ ] "Keep me signed in": short session vs. 30-day persistent cookie, chosen
      at login; rotate the token on login either way.
- [ ] Login rate limiting (also from Phase 11) — this is the phase where auth
      is already open.
- [ ] OAuth (decision #5, optional): `oauth_identities` (provider, subject,
      user_id, UNIQUE(provider, subject)); account linking by verified email.
- [ ] Tests: a reset token works once and not after expiry; rate limiting
      trips and recovers; save-for-later moves exactly one row each way.

**Frontend:**
- [ ] `LoginPage` rebuilt as the two-panel design with the Hive club panel,
      show/hide password, keep-me-signed-in, forgot-password link.
- [ ] `/forgot-password` and `/reset-password/:token` pages.
- [ ] `WishlistPage`; heart toggles wired on card, product and cart.
- [ ] Account area: profile, address book, order history.

**Done when:** a forgotten password is recoverable end to end and hearts
survive a logout on another device.

---

### Phase E9 — Content pages, journal, newsletter

**Goal:** no navigation link 404s.

**You will learn:** content pipelines without a CMS, double opt-in and why it
is the legal default, build-time content indexing.

**Backend:**
- [ ] Migration: `newsletter_subscribers` (email, token_sha256, confirmed_at,
      unsubscribed_at) with **double opt-in** — reusing the token pattern from
      E8 a third time.
- [ ] `POST /newsletter/subscribe`, `GET /newsletter/confirm?token=`,
      unsubscribe link.
- [ ] Only if decision #3 chose DB-backed content: `pages` and `posts` tables
      plus admin CRUD. Otherwise no backend work.

**Frontend:**
- [ ] Content pages: Our hive, Benefits, Harvest log, Shipping, Contact, Terms,
      Privacy. Recommended: markdown in `src/content/` compiled at build time —
      versioned with the code, no runtime cost, no CMS to operate.
- [ ] Journal: post list + detail, shared card component with the product grid.
- [ ] Footer newsletter form wired with inline confirmation.

**Done when:** all five header links and every footer link resolve, and a
subscription requires confirming the email.

---

### Phase E10 — Responsive, accessibility, performance, SEO

**Goal:** the design is a 1440 px desktop mock; this is where it becomes a
site people can actually use.

**You will learn:** responsive strategy from a fixed-width design, the WCAG
criteria that fixed-width mocks always miss, image delivery, Core Web Vitals,
structured data.

**Frontend:**
- [ ] Breakpoint plan: **1440** as designed → **1024** (sidebar becomes a
      drawer, 3-col grids become 2) → **768** (nav collapses to a sheet, hero
      and product page stack, checkout becomes one column) → **375** (single
      column, sticky add-to-cart bar, summary as a bottom sheet).
- [ ] Accessibility audit with axe + manual keyboard pass. Known issues already
      visible in the mock: icon-only buttons need labels; the qty stepper's
      −/+ are plain text; tabs need the ARIA pattern (done in E3); form inputs
      need real `<label>` association and `aria-describedby` for errors; the
      contrast pairs fixed in E1 must be re-verified with a tool.
- [ ] Images: `srcset` + AVIF/WebP, explicit width/height to stop layout shift,
      lazy-loading below the fold, and server-side thumbnails (Phase 11 item)
      — the design has a hero, six cards and a five-image gallery per product.
- [ ] SEO: per-product title/meta/OG, JSON-LD `Product` + `Offer` +
      `AggregateRating` (E4 and E5 make these truthful), `sitemap.xml`,
      canonical URLs on filtered shop pages.

**Backend / CI:**
- [ ] Lighthouse CI with budgets as a job; fail on regressions.
- [ ] Re-baseline the k6 script against the new queries — the facet counts of
      E2 are the likely first bottleneck, which finally answers Phase 11's
      "find the breaking point" item with a query worth optimising.
- [ ] Cache headers for images and the catalog; revisit only if k6 says so.

**Done when:** the purchase journey works from 375 px to 1440 px, is completable
with a keyboard only, and CI blocks a Lighthouse or axe regression.

---

## 4. Backend track at a glance

| Phase | Migrations | New endpoints |
|---|---|---|
| E2 | `benefits`, `product_benefits`, `products.badge`, `sales_count` | `GET /catalog/facets` |
| E3 | `product_images`, `product_highlights`, `product_usage_cards`, product metadata, `product_related` | `GET /products/{slug}/related` |
| E4 | `reviews`, `products.rating_avg/count` | `GET|POST /products/{slug}/reviews`, `GET|PATCH /admin/reviews` |
| E5 | `currencies`, `variant_prices`, `fx_rates`, order currency snapshot | currency negotiation on every read |
| E6 | `addresses`, order totals + address snapshot, payment fields, `shipping_rates` | `POST /orders` (body), `GET /orders/{id}` |
| E7 | `promo_codes`, `promo_redemptions`, membership tier | `POST /checkout/preview`, promo apply/remove |
| E8 | `wishlist_items`, `password_reset_tokens`, `oauth_identities` | wishlist CRUD, reset request/confirm, OAuth callback |
| E9 | `newsletter_subscribers` (+ optional `pages`/`posts`) | subscribe/confirm/unsubscribe |

Invariants that must survive all of it: the checkout stays one transaction with
ordered `FOR UPDATE` locks; money stays integer minor units; the domain package
imports neither SQL nor HTTP; the server computes every total.

## 5. Frontend track at a glance

| Phase | Routes | Components |
|---|---|---|
| E1 | layout wrapper | tokens, 11 primitives, `SiteHeader`, `SiteFooter` |
| E2 | `/` home, `/shop` | `ProductCard` v2, filters, sort, pagination, search overlay |
| E3 | `/products/:slug` v2 | `Gallery`, `VariantPicker`, `Tabs`, meta cards, related |
| E4 | — | `Stars`, review list, review form, moderation table |
| E5 | — | `CurrencyProvider`, switcher, `Price`, `formatMoney` |
| E6 | `/checkout`, `/orders/:id` | step indicator, address form, payment picker, `OrderSummary` |
| E7 | — | promo input, free-shipping progress, discount lines |
| E8 | `/wishlist`, `/account/*`, reset flow | two-panel sign-in, heart toggles, address book |
| E9 | `/our-hive`, `/benefits`, `/journal/*`, legal | markdown pages, post cards, newsletter form |
| E10 | — | responsive passes, a11y fixes, image pipeline |

## 6. Rules that apply to every phase

Unchanged from [RULES.md](RULES.md), repeated because Era II is where they get
tested:

- Every route change updates `docs/api/mountain-breath.postman_collection.json`
  in the same commit (#15).
- New code comes with tests (#11) — and each phase above names which.
- Significant choices get a Decisions Log row in
  [ARCHITECTURE.md](ARCHITECTURE.md) (#13); §2 of this document is the queue.
- A learning-log entry per session (#7).
- Work lands on `dev`; batch PRs merge to `master` (#9).

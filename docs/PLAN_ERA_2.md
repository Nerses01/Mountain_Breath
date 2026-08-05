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
> ([open](https://claude.ai/design/p/70fac810-0193-46d2-979e-d1c281beeae2?file=Mountain+Breath+Store.dc.html)),
> with a working copy at `docs/design/mountain-breath-store.dc.html`.
>
> **Revised 2026-08-05.** This originally said the design was deliberately
> *not* copied into the repo, and was read live via `DesignSync` so no stale
> duplicate could contradict it. That traded the wrong way: `get_file` pulls
> the whole ~50 kB file into context on every look, while a local copy can be
> grepped for one section or one value at near-zero cost — and the mock is a
> delivered artifact, not a document changing daily. The canvas stays
> authoritative and the local file is a cache: refresh it from `DesignSync`
> when the design moves, never hand-edit it.
>
> "The canvas wins" was also too strong, and E1 disproved it three times over
> — see §6 for the rule that replaced it: the canvas is the default source
> for every UI decision, overridden only by accessibility, by states it never
> draws, and by requirements added after it was made.
>
> Phases are numbered **E1–E10** so they never collide with Era I's 0–11.
> **E1.5** is inserted between E1 and E2 (not a renumber — every "E2 depends
> on this"-style cross-reference in this document stays correct) once
> language support was added to scope: it has to land before E2 writes its
> first translatable-content migration, not after.
> Phase 11 (Idea Backlog) stays permanently open and feeds both eras.

---

## 1. What the design changes

### 1.1 The product domain moves to the hive

The design is not a food & beverages shop. It is a **single-family apiary**
selling six bee products, and the whole information architecture follows from
that: the nav says "Our hive", the benefits panel is about what bee products do
in the body, and every product page carries a harvest hive number.

The design ships its own seed content (in its `renderVals()` block), which is a
gift — the copy is written. **Prices excepted:** decision §1.4 confirmed every
figure below is placeholder, so the names, categories, sizes, badges and
benefits are usable as-is, and the two price columns are illustrative only.

| # | Product | Category | Size | Badge | "Good for" | USD | AMD |
|---|---|---|---|---|---|---|---|
| 1 | Mountain Wildflower Honey | Honey | 500 g jar | Best seller | Natural energy | $14.00 | 6,700 ֏ |
| 2 | Pure Beeswax Blocks | Beeswax | 4 × 100 g | For makers | Balms & candles | $9.00 | 4,300 ֏ |
| 3 | Raw Propolis Tincture | Propolis | 30 ml dropper | Immunity | Antimicrobial | $19.00 | 9,100 ֏ |
| 4 | Fresh Royal Jelly | Royal jelly | 25 g jar | Cold chain | Vitality & skin | $32.00 | 15,300 ֏ |
| 5 | Bee Pollen Granules | Bee pollen | 250 g pouch | Protein | Protein & minerals | $16.00 | 7,600 ֏ |
| 6 | Bee Venom Serum | Bee venom | 15 ml bottle | New | Apitherapy | $28.00 | 13,400 ֏ |

Two things fall out of this table:

- [x] **The current seed is wrong for the design.** `backend/seed/seed.sql`
      has herbal tea, coffee and wildflower honey — six new categories replace
      three. Finding recorded here; the fix itself is E2's own seed-rewrite
      checkbox, not duplicated as a second item.
- [x] **AMD prices are not computed.** 14.00 → 6,700 ֏ implies ≈478 ֏/$, but
      so do 9 → 4,300 (478) and 16 → 7,600 (475) — rounded per market, not
      converted at one live rate.
      **Weakened 2026-08-05:** with the prices confirmed placeholder, this
      shows only how the *mock's author* generated numbers, not how the
      business prices. E5's per-market `variant_prices` model still stands,
      but on the general argument — real shops set round shelf prices per
      market and do not let a fluctuating rate move a price tag daily — not
      on this table as evidence.

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

- [x] ~~The cart line for **Raw Propolis Tincture reads $16.00 / 7,600 ֏**,
      which is the Bee Pollen price; the shop card says $19.00 / 9,100 ֏.~~
      **Resolved 2026-08-05: moot.** Every figure in the mock is placeholder;
      real prices come from the business. E2 seeds whatever the family
      supplies, so the contradiction never reaches the database — but it does
      mean **no price in §1.1's table is authoritative**, only the product
      names, categories, sizes, badges and benefits are.
- [ ] The cart totals are internally consistent ($62 + $6 − $4 = $64), and
      "$8 away from free shipping" on a $62 subtotal implies a **$70
      threshold** — but chilled shipping is still charged, so the threshold
      presumably applies to standard shipping only. Confirm the rule (E7).
- [ ] "Prices include VAT" sits next to a separate discount line. Decide
      whether VAT is contained in the displayed price (Armenian retail
      convention) and only broken out on the invoice — E6 assumes yes.
- The design shows `+374` phone, Yerevan address and **ArCa** card network:
  Armenia is the primary market, which supports the Idram / Ameriabank vPOS
  research already parked in Phase 11. *(observation, not a decision to make)*

---

## 2. Decisions to make before E1.5

These are the user's calls, not technical defaults. Each gets a row in the
[ARCHITECTURE.md](ARCHITECTURE.md) decisions log once made (RULES.md #13).

**Originally headed "before E1", corrected once E1 was underway:** none of
these actually gate E1. Tokens, primitives and the focus ring are identical
whether the shop sells honey or tea, in one currency or three — so E1 was
started with all of them still open, deliberately. They bite from **E1.5**
onward, where the first migration has to commit to a storage shape. The
column below records which phase each one genuinely blocks, so nothing is
answered earlier than it needs to be.

| # | Blocks | Cost of deciding late |
|---|---|---|
| 1 Catalog scope | E2 | every category, seed row and page string |
| 6 Translation storage | E1.5 | the first translation migration |
| 2 Currency | E5, and E2's seed prices | re-seeding, `formatMoney` signature |
| 4 Editorial fields | E3 | product schema shape |
| 3 Content storage | E9 | page pipeline only |
| 5 Social sign-in | E8 | two buttons, one table |

- [x] **1. Catalog scope — apiary-only.** Six bee products; tea and coffee are
      dropped. The design's nav, hero and benefits copy are taken as literal,
      not adapted. E2 replaces all three existing categories rather than
      adding to them, and the tea/coffee seed rows go with them.
      *(Decided 2026-08-05.)*
- [x] **2. Currency — both USD and AMD.** The dual price in every mock screen
      is real, so E5 ships in full: `currencies`, per-market `variant_prices`,
      order currency snapshot. Note this lands *after* E2 seeds prices, so
      E2 seeds USD only and E5 backfills AMD — or E2 seeds both against a
      `variant_prices` table brought forward. Decide which when E2 starts;
      seeding twice is the cheaper mistake than migrating a live price column.
      *(Decided 2026-08-05.)*
- [ ] **3. Content storage.** "Our hive"/"Benefits"/"Journal" as markdown in
      the repo (versioned with the code, zero backend) or DB-backed so the
      family edits without a deploy? E9 recommends markdown for v1. Now a
      three-language question too: markdown means one file per locale
      (`our-hive.en.md` / `.hy.md` / `.ru.md`) per page; DB-backed means a
      `page_translations` table mirroring decision #6.
- [ ] **4. Product editorial fields** (E3): explicit columns and child
      tables, or one JSONB `content` column? Columns give constraints and
      clean admin forms; JSONB avoids a migration per field but loses FK
      safety.
- [ ] **5. Social sign-in.** Google/Apple are two buttons in the mock and the
      most third-party-dependent item in the plan. Confirm they are in scope,
      or drop the buttons from the design.
- [x] **6. Translatable content strategy — translation tables.**
      `product_translations` / `category_translations` in E1.5, and
      `benefit_translations` later in E2 alongside the `benefits` table it
      references (parent_id, locale, name, description, …), locale-invariant fields
      (slug, sku, price, stock) staying on the parent row. Chosen over JSONB
      for the constraints and FK safety, at the cost of one migration per
      translatable entity and a `LEFT JOIN … ON locale = ?` on every read.
      Per-field English fallback is a `COALESCE`, so a missing Armenian
      description degrades to English rather than blanking.
      **This also pre-commits decision #4** toward explicit columns — mixing
      relational translations with a JSONB editorial blob would mean two
      incompatible answers to "where does product text live".
      *(Decided 2026-08-05.)*

  *(Numbers are referenced by "decision #N" elsewhere in this document — keep
  them stable; if a decision list item is ever removed, leave its number
  retired rather than renumbering the rest.)*

---

## 3. The phases

Same shape as Era I: **Goal**, **You will learn**, **Backend**, **Frontend**,
**Done when**. One phase at a time (RULES.md #5).

---

### Phase E1 — Design system foundations

**Goal:** the visual language exists as tokens, primitives and icons — the
whole string-free layer of the design system, ready for the shell in E1.5 to
assemble into pages.

*(Renamed from "Design system & app shell" on 2026-08-05, when the shell
bullets moved to E1.5 — see the moved bullet below. Keeping "app shell" in
the title while the shell lives in another phase would have been the same
inconsistency in a different place.)*

**You will learn:** Tailwind v4's CSS-first `@theme` configuration, design
tokens vs. ad-hoc utility classes, building a small component library,
WCAG contrast maths and why it belongs at token-definition time.

**Backend:** none.

**Frontend:**
- [x] Tokens in `src/index.css` via `@theme`: surfaces (`#F3E2D0` page,
      `#FDEFE0` panel, `#FEF4E8` header, `#FFF8EE` card), ink (`#46281C` bark,
      `#5C3B2A`, `#6E4B36`, `#7C5A45`, `#A9714B` muted), brand (`#E4761F`
      orange, `#F6C244` honey), border `#EED9C0`.
- [x] **Fix the contrast failures while defining the tokens, not in E10.**
      Measured against the `#FDEFE0` panel:

      | Pair | Ratio | Verdict |
      |---|---|---|
      | `#6E4B36` body text | 6.8:1 | passes AA |
      | `#7C5A45` secondary | 5.5:1 | passes AA |
      | `#A9714B` muted (13 px) | 3.6:1 | **fails** AA 4.5:1 |
      | `#E4761F` orange text/price | 2.7:1 | **fails** even AA-large 3:1 |
      | `#FFF8EE` on `#E4761F` (primary CTA) | 2.9:1 | **fails** AA 4.5:1 |
      | `#FFF8EE` on `#B8541A` | 4.6:1 | passes AA |

      Implemented as recommended: `#E4761F` stays a decorative/large-display
      accent (`--color-brand`); `--color-brand-ink: #B8541A` (the design's own
      link-hover colour) carries orange **text** and the primary button
      background. `#A9714B` survives only as `--color-ink-faint` for
      large/bold use; body-size "muted" copy uses a darkened `--color-ink-muted:
      #93603c` (4.7:1) instead. Re-verify with axe in E10.
- [x] Fonts: Poppins (400–800, display) + Karla (400–700, body), self-hosted
      via `@fontsource` **latin subset only** (the unscoped import pulls in
      ~450 kB of unused Devanagari glyphs) — `font-display: swap` is the
      package default, confirmed rather than assumed. Display scale collapses
      the mock's nine headings into five steps (26/32/38/46/68, each with its
      own line-height and tracking); body text uses Tailwind's own xs–lg scale
      plus one added `text-2xs` (11px) for eyebrows, rather than replicating
      the mock's six-step body scale literally — fewer near-duplicate sizes to
      choose between later.
- [x] Primitives in `src/components/ui/`: `Button` (primary pill with the
      orange glow shadow, dark pill, outline, ghost-underline), `Badge`,
      `Card`, `Input`, `Select`, `Checkbox`, `QtyStepper`, `IconButton`
      (38 px circle, 1.5 px border), `Breadcrumbs`, `SectionHeading`
      (eyebrow + title + trailing link), `Stat`. Plus `Field` + a `cx`
      helper (nine lines, rather than depending on `clsx`), and one token
      the mock never provided: `--color-danger`, since the design draws no
      error state anywhere and form validation needs one. Accessibility
      folded in at build time rather than deferred to E10 — `IconButton`
      makes its `label` prop **required** so the type checker refuses an
      unnamed icon button, `QtyStepper`'s −/+ carry real labels, and every
      field wires `htmlFor` / `aria-describedby` / `aria-invalid`.
- [x] **Focus states**: the mock has none. A global `:focus-visible` rule in
      `@layer base` now rings every interactive element in `--color-brand-ink`
      (not honey — `#F6C244` reads too pale as a 2px outline on the `#FFF8EE`
      card surface). Decided once, here; primitives in the next bullet inherit
      it for free and only need an override if a specific control asks for one.
- [→] **Moved to E1.5, 2026-08-05:** `SiteHeader` / `SiteFooter` and the
      `Layout` route wrapper. They were listed here, but they are the first
      components in the rebuild that contain user-facing strings — building
      them in this phase means writing English into JSX and reopening both
      files a day later to extract every string into translation keys.
      Interleaving E1 and E1.5 to avoid that would have broken RULES.md #5
      ("finish a phase's definition of done before moving on"), so the
      bullets move to the phase that owns their dependency instead. E1 is now
      exactly the layer that has no strings in it.
- [x] ~~Extend `public/icons.svg` with the sprite~~ → **inline React icon
      components** in `src/components/ui/icons.tsx`: search, heart,
      arrow-right, chevron-down, minus, plus, check, star.
      **Deviation, 2026-08-05.** This bullet assumed `public/icons.svg` was
      the project's icon system. It was not — it is untouched Vite scaffold
      (bluesky/discord/github/X logos, hardcoded `#aa3bff` strokes) and
      nothing in the codebase referenced it, so it has been deleted rather
      than extended. Inline components win here on three counts: every icon
      draws with `currentColor`, so the E1 colour tokens already on a parent
      colour the glyph with no extra wiring; unused icons leave the bundle;
      and there is no second request nor the cross-document styling quirks
      external `<use>` sprites still carry. `QtyStepper` and `Checkbox` now
      render real icons instead of the `−` / `+` / `✓` text characters they
      were standing in with.
- [x] Vitest: `Button` variants render, `QtyStepper` clamps at 1 and at stock.
      13 new tests (21 total). The Button suite pins the contrast decision in
      place — it asserts `primary` resolves to `bg-brand-ink`, so restoring
      the mock's failing orange breaks a test rather than shipping quietly.

**Done when:** ~~all eight existing routes render inside the new shell~~ →
every token, primitive and icon the design needs exists and is tested, no raw
hex lives outside the token block, and every interactive primitive is
reachable and visibly focused by keyboard. **✅ Complete 2026-08-05.**

*(The original wording required the routes to render inside the shell, which
is no longer this phase's job — that clause moved to E1.5's definition of
done along with the components that satisfy it.)*

---

### Phase E1.5 — Internationalization: Armenian, Russian, English (default)

**Goal:** Armenian, Russian and English are real, switchable languages across
the app chrome and every page built from here on — not a retrofit bolted on
near launch. Inserted immediately after E1's inert token layer and before E2
writes its first migration, because two upcoming schema decisions (§2 #4 and
the new #6) need a translation strategy before they're written, and because
`SiteHeader`/`SiteFooter` (E1's still-open primitives bullet) are the very
first user-facing strings in the whole rebuild — the cheapest possible moment
to make them translatable instead of hardcoding English into JSX that then
has to be reopened.

*(Not a gap from the design — the six-screen mock is English-only and shows
no language switcher. This is a requirement added independently of the Claude
Design canvas, and the mock gives no layout guidance for it; the switcher
needs a slot of its own, most naturally beside E5's currency switcher in the
footer.)*

**You will learn:** `Accept-Language` negotiation, ICU message formatting
(pluralization, interpolation) vs. hand-rolled string concatenation, why
translated *content* and translated UI *chrome* are different problems with
different storage answers, Postgres's per-language text-search
configurations, font coverage per writing system.

**Backend:**
- [x] Migration `000007_translations`: `product_translations` /
      `category_translations` (parent_id, locale, name, description) — the
      relational half of decision #6. Locale-invariant fields (slug, sku,
      price, stock) stay on the parent row; only human-language text moves
      out. Verified up → down → up against the dev database.
      **Corrected 2026-08-05:** this bullet originally also listed
      `benefit_translations`, which cannot exist yet — the `benefits` table
      it would reference is not created until E2, so the FK has no parent.
      `benefit_translations` moves to E2, created in the same migration as
      `benefits` itself.
      **Search had to come along, and it constrained the design.**
      `products.search_tsv` (000005) is a GENERATED column reading
      `name`/`description`, and `idx_products_name_trgm` (000006) indexes
      `name` — so moving product text drags Era I's whole search
      implementation with it. The per-locale tsvector cannot use the obvious
      `to_tsvector(locale::regconfig, name)`, because a generated column must
      be IMMUTABLE and casting text to `regconfig` reads the catalog (only
      STABLE). A **CASE over literal config names** is immutable and does
      work — confirmed against the real database before the migration was
      written, with genuine per-language stemming:
      `Wildflower→wildflow`, `цветочный→цветочн` (and `мёд→мед`),
      `Լեռնային→լեռնայ`.
      The old columns are deliberately **not dropped yet**: the store still
      reads them, so they go in a follow-up once it reads translations —
      the same add-backfill-then-drop sequence this plan uses for
      `products.image_url`.
- [ ] **Variant labels are translatable text too, and are not covered here.**
      The design's sizes read "500 g jar", "30 ml dropper", "250 g pouch" —
      a measurement plus an English noun. Either `product_variants.label`
      gains a translation table, or labels become pure measurements ("500 g")
      with the container moved into translatable product copy. The second is
      cleaner and probably right; decide it in E3 when variants are reworked,
      not silently here.
- [x] Locale resolution: `?lang=` → cookie → `Accept-Language` → default
      `en`, validated against `{en, hy, ru}` — the same shape as E5's planned
      currency resolution. Worth merging into one "preferences" middleware
      that resolves both instead of writing two near-identical ones.
      Accept-Language is parsed by hand (q-values, highest first, region
      subtags cut) rather than pulling in `golang.org/x/text/language`: the
      rule is ~30 lines and the spec's script/wildcard matching has no use
      here. **Nothing in the chain can fail** — a malformed q or unknown tag
      falls through to the next source instead of 400ing, because a shop that
      refuses to render over an odd header is worse than one that renders in
      English.
- [x] `GET /categories`, `GET /products`, `GET /products/{slug}` resolve the
      requested locale server-side and fall back to `en` per-field when a
      translation row is missing, rather than 404ing or returning blank text.
      The fallback is three levels — requested locale → English translation →
      the legacy parent column — because `CreateCategory`/`CreateProduct`
      still write only the parent row, so anything added through the admin
      has no translation rows at all yet.
      **Slugs are deliberately not translated:** the slug is the product's
      identity, so `/products/wildflower-honey` resolves in every language
      and a link shared between speakers still works.
- [x] Validation messages in `domain.ValidateProduct` (and the rest of the
      `fields` envelope) become short keys (`"required"`, `"positive"`,
      `"slug_format"`) instead of English prose — the frontend's i18n layer
      renders them, and the backend stops hardcoding a language into its API
      contract. Codes live as constants in `domain/validation.go`, so a typo
      is a compile error and the vocabulary is greppable from one place;
      they are part of the public contract, and renaming one is as breaking
      as renaming a JSON field.
      No existing test asserted on the prose, so none needed updating — but
      the **frontend did**, and it had to change in the same commit: all
      three forms rendered `err.fields[x]` straight to the screen, so keys
      alone would have shown readers "slug_format". `useFieldErrors` is now
      the single place a code becomes a sentence, with an `unknown` fallback
      so a code the catalogue has not learned yet looks imprecise rather
      than leaking a raw identifier.
      The envelope's top-level `message` stays English on purpose: `code`
      exists precisely so a client can render its own text, leaving `message`
      as a developer-facing fallback.
- [x] Search: **verified against the running dev database — Postgres ships a
      built-in `armenian` text search configuration**, alongside `english`
      and `russian` ([source](https://www.postgresql.org/docs/current/textsearch-configuration.html)):

      ```
      SELECT cfgname FROM pg_ts_config ORDER BY cfgname;
      -- arabic, armenian, basque, … english, … russian, … (29 total)
      ```

      So all three locales get a real `websearch_to_tsquery('<locale>', …)`
      branch, not just `en`/`ru` with `hy` falling back to trigram-only as
      first assumed — spot-check Armenian stemming/stopword quality when this
      is implemented, since "the config exists" isn't the same as "it's
      well-tuned for this catalog's vocabulary." The trigram layer Era I
      already built stays as the fuzzy/typo-tolerant layer under all three,
      exactly as it is for English today.
- [x] Admin per-locale write path and forms. `POST /admin/categories`,
      `POST /admin/products` and `PUT /admin/products/{id}` now accept an
      optional `translations` map alongside the existing fields, so
      translations can be written through the API instead of raw SQL.
      **Additive, not breaking:** `name`/`description` stay required and mean
      *English*, which maps exactly onto storage — they are still the parent
      columns the read fallback ends at. An `"en"` key inside `translations`
      is rejected (`locale_is_default`) rather than accepted as a second place
      to write one value, and an unknown language is caught here as a field
      error rather than at the database's CHECK constraint, which would
      surface as a 500-shaped driver error.
      `CreateCategory` and `UpdateProduct` became transactional; `UpdateProduct`
      upserts with `ON CONFLICT (product_id, locale) DO UPDATE`, so create and
      update share one write helper and re-editing never has a window where a
      product has no text. Rollback is covered by a test that duplicates a slug
      and asserts no orphan translation rows survive.
      Also fixed here: two literal `"required"` strings in the update handler
      that the earlier codes change had missed.
      **Forms:** both admin forms grew a `Translations` fieldset, one input
      set per non-default language, with the English fields relabelled so it
      is obvious which language the required copy is. The rule the form has to
      honour is that **blank means omit** — `translationPayload` drops any
      language whose name is empty, because an ABSENT language falls back to
      English while a PRESENT-but-empty one is a validation error that would
      block the submit. Six tests pin that distinction.
      **Found while updating Postman:** `PUT /admin/products/{id}` has existed
      in the router since Era I Phase 10 and was **never in the collection** —
      a standing rule #15 violation, now documented along with a negative case
      asserting an `"en"` key returns `locale_is_default`.
      (tabs or stacked fields) so the family can leave a translation blank
      and it falls back to English rather than the form blocking submission.
- [x] Tests: a request with no `Accept-Language` gets English; an
      unsupported locale code falls back rather than 400s; a product missing
      its Armenian translation still returns, with English fallback fields.
      All three live in `internal/api/locale_test.go` (11 negotiation cases,
      table-driven) and `internal/store/translations_test.go`, which exercises
      the fallback against real Postgres rather than a fake.
- [x] Update the Postman collection (RULES.md #15) with a `lang` variable.
      New **Localization** folder (4 requests): Armenian via `?lang=`, Russian
      via `Accept-Language` q-values, an unsupported tag proving it returns
      **200 and not 400**, and per-locale search stemming. The three existing
      catalog requests gained a disabled `?lang={{lang}}` param and a note on
      the resolution order.
      The admin validation request's tests were **wrong after E1.5** and are
      fixed here: they now assert `fields.slug === "slug_format"` rather than
      merely that the key exists, which is what pins the codes-not-prose
      contract. 26 → 30 requests, no request lost.

**Frontend:**
- [x] i18n library: `i18next` + `react-i18next`. The one deliberate
      exception to the project's "no dependency without a reason" default —
      Russian selects between three plural forms by rules that depend on the
      last digit *and* the tens (21 товар, 22 товара, 25 товаров), which no
      `count === 1 ? a : b` can express. Tests pin exactly those cases.
      **`i18next-browser-languagedetector` installed but NOT wired in:** the
      URL is the single source of truth for locale, and a detector reading
      `navigator.language` or a cookie would be a second, competing opinion —
      the failure mode being a page whose URL says Armenian while its text
      says English. `useLocale` syncs i18next *from* the route, one direction
      only. Remove the package if E1.5 ends without a use for it.
- [x] Message catalogues (`common`, `footer`) in all three languages, with
      English as the reference and per-key fallback to it, so a partially
      translated page still reads. **Armenian and Russian copy is
      machine-assisted and flagged for native review** — the apiary
      vocabulary in particular (propolis, royal jelly, beeswax) is specialist
      and easy to get subtly wrong.
- [x] Route structure: bare `/…` serves **English, the stated default** — no
      prefix, no redirect, every link written in E2 onward keeps working
      unchanged; `/hy/…` and `/ru/…` prefix the other two explicitly.
      The prefixes are **enumerated** (`PREFIXED_LOCALES.map`), not matched
      with a `/:locale` param: a param binds greedily, so `/cart` would set
      locale="cart" and silently render the home page there. A regression
      test pins exactly that case. `useLocale` therefore parses
      `useLocation().pathname` rather than reading a route param, which also
      makes it independent of how the route tree is shaped.
- [x] **Font coverage gap found while building this — verified against the
      installed packages' own `unicode-range`s, not assumed:** neither
      Poppins nor Karla (chosen in E1) ships Cyrillic or Armenian glyphs.

      ```
      @fontsource/poppins subsets available: latin, latin-ext, devanagari
      @fontsource-variable/karla unicode-range: U+0000-00FF, U+0100-02BA, … (Latin only)
      ```

      Fixed with two hand-written `@font-face` rules (Noto Sans Cyrillic
      20 kB, Noto Sans Armenian 27 kB) appended to the existing
      `--font-display` / `--font-body` stacks.

      **Better than the `:lang()` approach this bullet originally proposed.**
      CSS resolves font-family *per character*, not per element, so an
      Armenian product name inside an English page picks up the Armenian face
      with no wrapper or language attribute — which `:lang(hy)` could not do.
      And because `unicode-range` gates the download, an English visitor
      fetches neither file: the cost of supporting two more scripts is zero
      bytes for anyone not reading them.
      Declared by hand rather than imported because
      `@fontsource-variable/noto-sans` ships eight subsets in one stylesheet
      (greek, devanagari, vietnamese…) and Vite would bundle all of them —
      the same trap as the Poppins Devanagari import in E1.
- [x] Translation namespaces per feature area (`common`, `catalog`, `cart`,
      `checkout`, `account`) rather than one growing file — mirrors this
      plan's page-per-phase structure, so each future phase adds one
      namespace instead of editing a shared file. `common` and `footer` exist
      now; the rest arrive with the phases that need them.
- [x] `LanguageSwitcher` primitive, built into `SiteFooter` from the start
      rather than retrofitted. Rendered as **links, not buttons** — each
      language is a real URL, so middle-click and "open in new tab" work and
      the choice survives a reload without any persistence code. Placed in
      the footer bottom bar beside the slot E5's currency switcher will take,
      following the design's own habit of putting locale-shaped controls
      there ("USD / AMD") rather than inventing a header position the mock
      gives no guidance for. Revisit in E10 if it proves too buried.
- [x] **`SiteHeader` / `SiteFooter` (moved here from E1, 2026-08-05.)**
      Header: logo mark, wordmark + tagline, 5 nav links, search and wishlist
      icon buttons, cart pill with count. Footer: 4 columns, newsletter form
      (inert until E9), bottom bar. Every string comes from the `common` /
      `footer` namespaces.
      Two departures from the mock, both §6 exception 2 (states it never
      draws): an **account control** — the design's header has none, because
      it never shows a signed-in state, but the app has auth and users need
      to reach it — and body-size footer text using `ink-on-dark-soft`
      instead of the design's `#a98a74`, which measures **4.2:1 on bark at
      the 13px the mock uses it at** and fails AA.
      Three nav destinations (Our hive, Benefits, Journal) render as plain
      text, not links, until E9 builds them — the header keeps the design's
      shape without shipping links to a blank page.
- [x] **`Layout` route wrapper (moved here from E1.)** Wraps the routes in
      header + footer; `/admin/*` stays outside with its own chrome. `App.tsx`
      grew the locale prefixes in the same edit, so the file was touched once
      rather than twice.
- [x] `useLocale()` hook wrapping `useParams()` + i18next, so every future
      page reads the active locale one way instead of each poking
      `i18next.language` directly. Also owns the `<html lang>` attribute
      (screen-reader pronunciation, browser translation offers) and exposes
      `hrefFor(locale)` so switching language keeps you on the same page
      rather than bouncing to the home page.
- [x] Vitest: `LanguageSwitcher` changes the route prefix and persists the
      choice; a missing translation key falls back to English rather than
      rendering blank. 22 new tests (43 total), including a routing smoke
      suite over `App` — the locale prefixes mount the same route list three
      times from an array of `<Route>` elements, and a passing typecheck says
      nothing about whether the router actually walks that shape.

**Done when:** the whole shell (header, footer, nav, auth forms, error toasts)
reads correctly in all three languages with no hardcoded English string left
in JSX, switching language never loses the current page, Armenian and Russian
render in a font that actually has their glyphs, and — inherited from E1 when
the shell bullets moved here — **all eight existing routes render inside that
shell with the new palette.**

---

### Phase E2 — Catalog model, faceted shop, home page

**Goal:** the Shop screen's sidebar works for real, and the Home screen exists.

**You will learn:** faceted search and why counts are expensive, aggregate
queries with `FILTER`, many-to-many taxonomies, URL as state (deep-linkable
filters), keyset vs. offset pagination.

**Backend:**
- [ ] Migration: `benefits` (id, slug, sort_order), `product_benefits`
      (product_id, benefit_id, PK on both) and `benefit_translations`
      (benefit_id, locale, name) in one migration — Energy, Immunity, Skin,
      Recovery, Sweetening. The translations table lands here, not in E1.5,
      because its FK parent is created here.
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
      badges, benefits and both currencies' prices (or USD only until E5) —
      seeded in all three languages from the start, not English-only with
      translations bolted on later.
      **Also update the Postman collection in the same commit:** three
      requests hardcode seed slugs that decision #14 deletes
      (`/products/armenian-coffee`, and the category/product create bodies
      still say herbal-tea). They will 404 the moment the seed changes, which
      is the collection-and-code disagreement rule #15 exists to prevent.
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
- [ ] Decide decision #4 (columns vs JSONB) and log it — together with
      decision #6 (E1.5's translation storage), since the two overlap directly.
- [ ] Migration: `product_images` (product_id, url, alt, sort_order,
      is_primary) with a partial unique index enforcing one primary per
      product. `alt` is translatable text (screen readers read it aloud in
      the visitor's language) and follows decision #6 like every other field.
      Backfill from `products.image_url`, then drop the column in a
      follow-up migration once the admin UI writes the new table.
- [ ] Migration: `product_highlights` (product_id, sort_order, text) for the
      "What it does" bullets; `product_usage_cards` (kicker, title, body,
      sort_order) for Morning / Course / Pairs with. Both carry translatable
      text per decision #6.
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
      PK(variant_id, currency)). **Chosen over live FX conversion** because a
      shelf price is a business decision, not a derived number: a shop sets a
      round price per market and holds it, rather than letting a fluctuating
      rate move the price tag between page loads. (The mock's own figures hint
      at the same habit, but they are placeholder — see §1.1 — so they are an
      illustration, not the argument.)
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
      versioned with the code, no runtime cost, no CMS to operate. One file
      per locale (`our-hive.en.md`, `.hy.md`, `.ru.md`) if decision #3 keeps
      markdown, per the note added to that decision.
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
      canonical URLs on filtered shop pages, `hreflang` alternates across the
      three locales from E1.5.

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
| E1.5 | `product_translations`, `category_translations`, `benefit_translations` | locale (`lang`) negotiation added to every existing read endpoint |
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
| E1.5 | `/:locale/*` (bare `/` = English) | i18n provider, `LanguageSwitcher`, `useLocale` |
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

**[RULES.md](RULES.md) is the single copy.** This section used to restate five
of its rules and additionally *held* one — the design canvas as source of UI
truth — that existed nowhere else, which is how a rule gets missed by anyone
not reading this particular plan. That rule now lives in RULES.md as **#16**,
and the restated ones are gone rather than kept in two places to drift apart.

The ones Era II leans on hardest, by number:

| Rule | Why it bites here |
|---|---|
| **#16** design canvas is UI truth | every phase from E1 on renders something the mock drew |
| **#15** Postman is the API contract | E1.5, E2, E3, E4, E6, E7 and E9 all add or change routes |
| **#11** new code comes with tests | each phase above names which tests it owes |
| **#13** decisions get logged | §2 of this document is the queue feeding that log |
| **#5** one phase at a time | E1.5 exists *because* interleaving E1 with it would have broken this |

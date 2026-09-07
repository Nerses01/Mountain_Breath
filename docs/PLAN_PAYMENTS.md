# Mountain Breath — Plan: real payments (BACKLOG §3, was Era III F4)

> **❄ Status 2026-09-07: frozen before P0** (decision #107). Every
> provider below requires an individual entrepreneur or a company; the
> family has not decided whether to register one
> ([PAPERWORK_ARMENIA.md](PAPERWORK_ARMENIA.md) has the steps, costs and
> the physical-person alternative). Until then the shop sells on bank
> transfer and cash only. Nothing in this plan changes on thaw — P0 is
> simply the first step again.

> **Written 2026-09-04** as the research step §3 asks for first. Everything
> below rests on the provider documentation and reference integrations
> read that day (sources at the end), on E6's checkout, and on decision
> #91's write path. The **provider pick is the developer's** — it is a
> merchant contract with a bank in the family's name, not a `go get`.
> This document gives the comparison, a recommendation, and the phases
> that follow whichever way the pick goes.
>
> **Done when** (unchanged from the backlog): a sandbox dram moves end to
> end and the order shows paid-by-card with no human flipping anything.

---

## 1. What already exists — the shape the provider must fit

E6 and F2 built the model deliberately half-finished, and the half that
exists constrains the half that does not:

| Already there | Where | What it fixes for us |
|---|---|---|
| `payment_method ∈ {card, bank_transfer, cash_on_delivery}` and `payment_status ∈ {unpaid, paid, refunded}` as **two separate facts** | migration 000017, [checkout.go](../backend/internal/domain/checkout.go) | The provider only ever touches `payment_status`; the method is the customer's choice at checkout and never changes |
| `paymentTransitions`: `unpaid → paid → refunded`, **no backward arrow**, `refunded` terminal | domain | A failed or abandoned bank session is *not* a transition — the order simply stays `unpaid`. Failure is a fact about the **attempt**, which is why attempts need their own table (§2.2) |
| `UpdateOrderPaymentStatus`: lock the row `FOR UPDATE`, validate the transition in the domain, write | [orders.go](../backend/internal/store/orders.go) | Decision #91 promised "F4's provider webhook will flip it through this same path". It will — the provider verification calls this, never `UPDATE orders` directly |
| `PATCH /admin/orders/{id}/payment` | api/orders.go | Stays as the human door (bank transfers, cash). Gains a provider call in front of it for `paid → refunded` on card orders (§4, P3) |
| `CreateOrder` decrements stock and clears the cart in one transaction | store | A card order holds stock from the moment it is placed — **before** the money moves. An abandoned bank page therefore leaks stock until something releases it (§2.4) |
| Checkout's card fields are `DisabledStub`s, `aria-hidden`, with a note underneath | [CheckoutPage.tsx](../frontend/src/pages/CheckoutPage.tsx) | E6's comment says it outright: "card numbers belong on the provider's servers, not this one". The stub is replaced by a redirect, not by real inputs |
| Session cookie is `SameSite=Lax` | api/auth.go | The customer **comes back from the bank with their session** — Lax sends the cookie on a top-level GET navigation, which is exactly what a provider's return redirect is. A provider that instead notifies by server-to-server POST gets no cookie and needs a signature |
| `MB_PUBLIC_URL` is the browser's origin | config | The return URL is built from it, like Google's callback |
| No background job mechanism exists (no ticker anywhere in the API) | — | The abandoned-attempt expiry (§2.4) needs one; so do §5's session/reset-token cleanups. Build it **once** |

Stripe-shaped assumptions the codebase does **not** make, fortunately:
no `payment_intent` column, no client-side SDK, no PAN anywhere. Every
option below is a *hosted page* — the customer leaves the site, pays on
the bank's page, and returns. That is the only shape a home-hosted shop
should ever accept: PCI scope stays with the bank.

## 2. The comparison

### 2.1 Candidates

| | **Ameriabank vPOS** | **ArCa IPay (via a partner bank)** | **Idram** | **Stripe** |
|---|---|---|---|---|
| What it is | Ameriabank's own gateway on the ArCa rails | The national scheme's gateway; credentials issued by whichever partner bank the merchant contracts with | E-wallet with a merchant interface; wallet-first, cards attached | — |
| Cards accepted | Visa, Mastercard, ArCa — the checkout's own `cardBlurb` | Visa, Mastercard, ArCa | Idram wallet; cards attached to a wallet | — |
| Who may sign | Legal entity **or individual entrepreneur** (stated) | Legal entity / IE through the bank | Business contract with Idram | **Armenia is not a supported country** — would need a foreign legal entity |
| Integration shape | **Redirect + pull**: `InitPayment` → send customer to `Payments/Pay?id=…` → customer returns to `BackURL` → we call `GetPaymentDetails` and trust only that | Same: `register.do` → `formUrl` → return → `getOrderStatusExtended.do` | **Push**: the customer pays on Idram's page; Idram's servers POST to our RESULT_URL twice (precheck, then confirmation with an MD5 checksum); we answer `OK` | — |
| Money on the wire | Decimal major units, ISO 4217 **numeric** currency (`051` AMD, `840` USD) | **Integer minor units**, numeric currency | Decimal `0.00` | — |
| Refund / reversal | `RefundPayment`, `CancelPayment` — API calls | `refund.do`, `reverse.do` | Not in the merchant interface (manual) | — |
| Sandbox | **Public test host** `servicestest.ameriabank.am/VPOS` with a public help page; test credentials emailed after creating an Ameria Business account; test amount 10 AMD; test order-id range assigned per client | `ipaytest.arca.am:8445` — credentials from the bank | None documented | — |
| Doc quality | Help page lists every operation with field names; no enum values published for `PaymentState`/`OrderStatus` (learned empirically in the sandbox); success is `ResponseCode == "00"` on details, `1` on init | A PDF manual (behind a 403 for us today); `orderStatus 2` = deposited, `3` reversed, `6` declined | A Russian-language spec deck; URLs and secret configured **by Idram staff at contract time** | — |
| Runs behind our tunnel? | Yes — outbound calls only; the "callback" is the customer's browser | Yes — same | Yes — inbound POST arrives through Cloudflare like any request | — |
| Standing conditions | Bank may terminate below **AMD 1,000,000 turnover per 180 days** | Per bank | Per contract | — |
| Fees | Not published on the pages we could read; the corporate tariff PDF covers accounts/transfers, not acquiring — **ask the bank** (regional benchmark ≈ 2.5%) | Per bank | Per contract | — |

### 2.2 Recommendation: Ameriabank vPOS first, Idram as a later second method

**Ameriabank vPOS.** It is the only option where a sandbox can be obtained
*before* any contract is signed, the API is four calls we need
(`InitPayment`, `GetPaymentDetails`, `RefundPayment`, `CancelPayment`),
it accepts exactly the three networks the checkout already promises, it
is redirect-and-pull (no inbound webhook to authenticate, and the
customer returns with their session), and refunds are a first-class
call — which is what makes `refunded` reachable *by flow*, the backlog's
own wording. The individual-entrepreneur eligibility matters: the
family does not need an LLC to start.

**Why not ArCa direct:** it is the same rails one layer down. It only
wins if the family's bank is not Ameria — then it is the same plan with
minor-unit amounts (which we would actually prefer) and a different
adapter. The `PaymentProvider` interface in §3 is written so that swap
costs one package.

**Why not Idram first:** a wallet, not a card acquirer; no sandbox; the
merchant URLs and secret are set by Idram's staff, so nothing can be
tried before the contract; and its push model means a public,
unauthenticated POST endpoint authenticated by an MD5 checksum over a
shared secret. All workable — a good **second** method for wallet users
once card payments exist — but the wrong first integration for a
learning project that wants to iterate in a sandbox.

**Stripe: rejected**, with the reason on record so it is not re-asked:
stripe.com/global lists no Caucasus or Central Asian country. A US or
EU entity would be a bigger undertaking than the shop.

### 2.3 The backlog's design question: same store method, or a table?

> "same store method, or a `payments/events` table so a provider event is
> its own recorded fact, as status transitions got (log 08-18)?"

**Both — they answer different questions.** The order's `payment_status`
column stays the single source of truth for *is this order paid*, and
it is flipped only through `UpdateOrderPaymentStatus` (decision #91).
But a **`payments` table gets one row per attempt**, because an attempt
is a fact the column cannot hold:

- A customer abandons the bank page and tries again → two attempts, one
  order. The bank needs a **unique `OrderID` per attempt**, so the
  attempt's own id is what we send, not `orders.id`.
- The return URL is hit twice (refresh, back button) → the second
  verification must find the attempt already `paid` and do nothing.
  **Idempotency needs a place to remember.**
- A refund needs the bank's `PaymentID` months later.
- "Why does the system think this order is paid?" needs the bank's
  `ResponseCode`, `rrn`, approved amount, and the timestamp — audit
  data that belongs next to the fact, not in a log line.

This is the same reasoning that made `order_status_events` its own table
in 000021 rather than a column: the column is the *current* state, the
table is the *history of facts* that produced it.

```
payments
  id                  BIGSERIAL PK          -- what the bank sees as OrderID
  order_id            BIGINT FK orders      -- ON DELETE CASCADE, like order_status_events
  provider            TEXT                  -- 'ameria' (later 'idram')
  provider_payment_id TEXT NULL UNIQUE      -- the bank's PaymentID once known
  amount_minor        BIGINT                -- what we asked for, in luma
  currency            TEXT                  -- 'AMD' | 'USD' (our words; the adapter maps to 051/840)
  state               TEXT CHECK IN ('initiated','paid','failed','abandoned','refunded')
  provider_response   JSONB NULL            -- GetPaymentDetails snapshot at verification
  created_at, updated_at
  INDEX (order_id, created_at)
```

Attempt states are **the attempt's** machine, distinct from the order's
payment machine: `initiated → paid | failed | abandoned`, `paid →
refunded`. Only `initiated → paid` is allowed to call
`UpdateOrderPaymentStatus(unpaid → paid)`, inside the same transaction.

The §6 tripwire "`order_status_events` actor column fires when the
webhook becomes the third writer" fires **here**: the verification
handler is neither the customer nor the admin. Settle it in P2 — an
`actor` column on `payments` at least (`customer_return` vs
`admin_refund`), and decide whether `order_status_events` gets one too.

### 2.4 Stock held by an unpaid card order

`CreateOrder` takes the stock. If the customer never pays, that stock is
invisible to other customers until someone cancels the order. Two
options, and the second is recommended:

1. Move the stock decrement to "after payment" for card orders — breaks
   E6's single critical section and lets two customers race for the last
   jar on the bank's page. No.
2. **Expire abandoned attempts.** A background job cancels `pending`
   card orders whose latest attempt is `initiated` and older than N
   minutes (the bank's own session `Timeout` bounds N) through the
   existing cancel path, which restocks and releases the promo. The
   customer sees "payment window expired — reorder" and the Reorder
   button (A2) already exists.

That job is the API's first background goroutine — a ticker under the
server's context, cancelled on shutdown. §5's expired-session and
reset-token cleanups are the same shape; **P3 builds the mechanism once
and hangs all three on it.**

## 3. Architecture of the integration

```
frontend                          api                           bank
────────                          ───                           ────
checkout, method=card
POST /orders            ───────▶  CreateOrder (unchanged; order unpaid)
POST /orders/{id}/payments ─────▶  ownership check
                                  INSERT payments (initiated)
                                  provider.Initiate(attempt) ──────▶ InitPayment
                                                          ◀────────  PaymentID
                                  UPDATE payments.provider_payment_id
                        ◀───────  {redirect_url}
window.location = redirect_url ──────────────────────────────────▶ hosted card page
                                                                    customer pays
        ◀───────────── 302 BackURL?orderID=&paymentID=&… ◀──────────
GET /api/v1/payments/return ────▶  (query params are UNTRUSTED — used only to find the attempt)
                                  provider.Verify(paymentID) ─────▶ GetPaymentDetails
                                                          ◀────────  ResponseCode, OrderID, Amount, Currency…
                                  tx: lock attempt; if already paid → no-op
                                      check OrderID == attempt.id, amount, currency
                                      payments.state = paid, snapshot response
                                      UpdateOrderPaymentStatus(unpaid → paid)   ← decision #91's path
                                  302 → /account/orders/{id}?payment=paid|failed
```

- **`PaymentProvider` is an interface declared at the consumer** in
  `api`, exactly like the store interfaces: `Initiate`, `Verify`,
  `Refund`. A `fakeProvider` in `api_test.go` drives the handler tests;
  the real adapter lives in `internal/payment/ameria` and is pinned by
  tests against a **scripted bank** (`httptest.Server` answering the
  four endpoints) — the same technique as #104's scripted SMTP relay.
- **The only float in the money path is the adapter's wire format.**
  `amount_minor` (int64) becomes Ameria's decimal at the boundary and is
  compared back as an integer after verification. The conversion is one
  function with table tests; nowhere else may `float64` touch a price.
- **Config** follows Google/SMTP: `MB_PAY_PROVIDER=none|ameria`,
  `MB_AMERIA_CLIENT_ID`, `MB_AMERIA_USERNAME`, `MB_AMERIA_PASSWORD`,
  `MB_AMERIA_SANDBOX=true`. With `none`, the checkout keeps today's stub
  — so **the code merges and deploys before any contract exists**, and
  prod changes behaviour only when the keys land in `deploy/.env`.
- **Return-URL states are ours to design** (the canvas never drew them):
  "taking you to the bank" (button state after `POST …/payments`),
  "paid" (the order page with a confirmation ribbon), "declined — try
  again" (a second attempt from the order page), "expired".
- **Bank transfer and cash are untouched.** The admin's PATCH remains
  their write path.

## 4. The phases — one per session

### P0 — The pick, and the accounts (developer)

- Decide the provider (§2.2). If Ameriabank: create the **Ameria
  Business account**, ask the vPOS team for **test credentials** and the
  test order-id range; ask for the acquiring **tariff** in writing; ask
  whether **USD** acquiring is on the contract (the checkout has a USD
  market — if not, card is AMD-only, a rule the domain states the way it
  states cash-is-AMD-only).
- The legal question for the family: **individual entrepreneur or
  LLC** — Ameria accepts both; the tax treatment differs and is not
  this project's to decide.
- Keys go to `deploy/.env` on the laptop; the runbook gets a step.

**Done when:** `curl` against `servicestest.ameriabank.am` with the
emailed credentials returns a `PaymentID`.

### P1 — Attempts and the adapter (backend, no routes yet)

- Migration: the `payments` table of §2.3 (+ down).
- Domain: `PaymentAttempt`, its state machine as data (the
  `paymentTransitions` pattern), `ValidateAttemptMatches(order)`.
- `PaymentProvider` interface in `api/server.go`; `fakeProvider`.
- `internal/payment/ameria`: `Initiate`, `Verify`, `Refund`; the
  minor-units ↔ decimal function; numeric currency map; scripted-bank
  tests pinning request and response JSON, the `"00"` / `1` success
  codes, timeouts, and a non-JSON reply.
- Store: `CreatePaymentAttempt`, `MarkAttemptPaid` (the transaction of
  §3), `LatestAttempt(orderID)`; integration tests including
  double-verification being a no-op.

**Done when:** `go test ./...` proves an attempt can be created, verified
once, and verified again harmlessly, with a bank that only exists in
`httptest`.

### P2 — The customer flow (routes + frontend)

- `POST /orders/{id}/payments` (requireUser, owner only, `card` only,
  `unpaid` only) → `{redirect_url}`.
- `GET /payments/return` (public; identifies the attempt by
  `paymentID`, trusts nothing else) → verify → 302.
- Checkout: `card` submits the order, then requests the attempt and
  navigates to the bank; the stub fields and their i18n keys go.
- Order page: the four return states, in three languages.
- The §6 actor tripwire is settled here (§2.3).
- Postman collection updated in the same commit (the living contract).
- Playwright: the journey with a **stub provider** the dev API exposes
  when `MB_PAY_PROVIDER=stub` — a page that has "Pay" and "Decline"
  buttons and redirects like a bank would. Real banks stay out of CI.

**Done when:** in dev, a card checkout leaves the site, returns, and the
order page shows *paid* with no admin involved; a declined payment
leaves the order unpaid with a retry.

### P3 — Refunds, expiry, and the first background job

- `paid → refunded` on a card order calls `provider.Refund` **before**
  `UpdateOrderPaymentStatus`; a bank failure leaves the order `paid` and
  reports why. Admin cancel of a paid card order refunds through the
  same call.
- The ticker: `internal/jobs` (or in `api`), one `time.Ticker` per job
  under the server context; **expire abandoned attempts** first, then
  §5's expired sessions and spent reset tokens hang on the same loop.
  Metrics: a counter per job run and per outcome, on the existing
  registry.
- Alert rule: a verification that fails with a *bank* error (not a
  decline) pages, via #105's Telegram road.

**Done when:** an admin refund of a sandbox payment shows `RefundedAmount`
in `GetPaymentDetails`, and an abandoned attempt restocks the jar
within N minutes with a log line and a metric.

### P4 — Sandbox on prod, then the switch

- Deploy with `MB_AMERIA_SANDBOX=true` and the test credentials; buy a
  10-AMD test jar on mountainbreath.net end to end — **the backlog's
  definition of done**.
- Contract signed → live credentials, `MB_AMERIA_SANDBOX=false`, one
  real dram, refunded the same day.
- Runbook step; decision-log row; learning-log entry.

## 5. Open questions carried into P0

- USD acquiring on the contract, or card = AMD only?
- Does the bank's `Timeout` on `InitPayment` bound the abandoned-attempt
  window, or do we set our own (shorter) N?
- Is the test order-id range something we must honour by **offsetting**
  `payments.id`, or does the sandbox accept any unique integer? (Two
  reference integrations show two different ranges — they look assigned
  per client.)
- Idram as a second method: after P4, as its own backlog line.

---

### Sources read on 2026-09-04

- Ameriabank vPOS test API help: <https://servicestest.ameriabank.am/VPOS/help> — operations `InitPayment`, `GetPaymentDetails`, `RefundPayment`, `CancelPayment`, `ConfirmPayment`, bindings; request/response field names.
- Ameriabank eligibility, test-credentials process, turnover clause: search results incl. <https://ameriabank.am/en/business/micro/more/other-services/ecommerce>, <https://support.ucraft.com/hc/ucraft-knowledge-base/articles/how-to-setup-ameriabank-vpos>, <https://arka.am/en/news/business/ameriabank_offers_innovative_solution_for_opening_e_stores/>.
- Reference integrations (field names, success codes, payment-page URL, test amounts/ranges): <https://github.com/ayvazyan10/AmeriaBankvpos>, <https://dev.to/boolfalse/ameriabank-idram-v-pos-integration-to-your-laravel-php-app-e55>, <https://github.com/hos/ameria-sdk-js>.
- ArCa IPay: <https://github.com/DeReNiKGab/wc-arca-gateway>, <https://github.com/hexdivision/woocommerce-arca> (test/live hosts, `register.do`, `getOrderStatusExtended.do`, `orderStatus` meanings, minor units, `051`); manual at <https://www.arca.am/file_manager/Merchant%20Manual_1.55.1.0.pdf> (403 today).
- Idram merchant interface: <https://www.slideshare.net/slideshow/idram-merchant-interface/228444357> (fields, precheck/confirm, checksum formula, `OK`), <https://www.zegashop.com/web/tutorial/idram-vpos-payment-method/>.
- Stripe country list: <https://stripe.com/global>.
- Armenian PSP fee benchmark: <https://payatlas.com/countries/armenia-am>.

# Mountain Breath — Paperwork in Armenia before real payments

> **❄ Status 2026-09-07: on hold.** The family will decide between
> registering an individual entrepreneur (§1–§4) and staying a natural
> person selling own honey (§7). Until then the shop takes bank transfer
> and cash only, which needs none of §2's steps 1–4, 10 and 11. Steps
> 5–9 (food-chain registration, sanitary booklet, honey examination,
> labels, legal pages) apply to the product and the site regardless of
> the seller's status and can proceed at any time.

> **Written 2026-09-04** as the legal half of [PLAN_PAYMENTS.md](PLAN_PAYMENTS.md)
> P0. Everything below comes from the sources listed at the end, read that
> day. Rates and deadlines in Armenia change every January; **items marked
> ⚠ verify** are the ones where sources disagreed or were silent, and are
> the questions to put to an accountant, the bank, or the Food Safety
> Inspection Body (ՍԱՏՄ) before relying on them. This is orientation, not
> legal advice.
>
> Context that shapes it: the family keeps its own apiary and sells its
> own honey and bee products through the site. That makes the business
> **agricultural production sold online**, not resale trade — and the tax
> treatment of the two differs (§3).

---

## 1. The one decision first: individual entrepreneur or LLC

| | **Individual entrepreneur — ԱՁ (անհատ ձեռնարկատեր)** | **LLC — ՍՊԸ** |
|---|---|---|
| Registration | State duty **3,000 AMD**, same day | **Free**, by law within 2 working days |
| Who owns it | One person, personally liable | Legal entity, several owners possible |
| Profit tax | none — turnover tax or micro regime replaces it | 18% under the general regime, or the same turnover tax; dividends taxed 5% |
| Fixed monthly cost | 5,000 income tax + 5,000 social (turnover-tax regime) | director's salary with payroll taxes, or none if no salary |
| Bookkeeping | light; an accountant is optional at this size | proper accounting expected |
| Ameriabank vPOS | accepts IEs | accepts |

**Recommendation: register one family member as an IE.** Cheapest,
fastest, and every provider in the payments plan accepts it. An LLC only
earns its overhead if several people need to own the business or the
family wants liability separated from the person.

## 2. The steps, in order

Days are calendar days, typical → worst case, for someone who has the
documents ready. Steps 1 and 2 can start today; 4–9 run in parallel once
the IE exists; 10–12 run alongside the code phases P1–P3.

| # | Step | Where | Cost | Days |
|---|---|---|---|---|
| 1 | **ID card + electronic signature** for the person who will be the IE. Without them registration and every tax filing means a trip to an office. The e-signature is a yearly service from EKENG; newer ID cards can use mobile ID from the telecom operators instead. | Police passport office (ID card), [ekeng.am](https://www.ekeng.am/en/e_sign) | ID card: state fee; e-signature **10,000 AMD/year** (from 1 Feb 2026) | ID card 1–10 working days; e-signature same day → total **1–14** |
| 2 | **Register the IE** (ԱՁ պետական գրանցում). Online with ID card + e-signature, or in person at the State Register. You get the tax number (ՀՎՀՀ) with it. | [e-register.moj.am](https://e-register.moj.am/en/services/6d262855-3e61-417d-a601-bf4dcc673dab) | **3,000 AMD** state duty | **Same day** (1) |
| 3 | **Choose the tax regime** — the 20-day rule. Within **20 days of registration** file the application for the turnover-tax regime (or micro, see §3). Miss it and the IE sits in the general regime (20% VAT + 20% income tax) until next January. | SRC e-reporting portal, or the tax office | free | 1 day of work; **deadline day 20** |
| 4 | **Business bank account** at Ameriabank (the vPOS pick). IEs are served on the corporate tariff; remote opening is not offered to IEs, so one branch visit. Bring the IE certificate, ID, tax number. | Ameriabank branch | plain account opening **free**; package with cards 40,000; maintenance **free** if ≥300k AMD flows or ≥100k average balance per half-year, otherwise 10,000/half-year | **1–3 business days** |
| 5 | **Register as a food-chain operator** (սննդի շղթայի օպերատոր) with the Food Safety Inspection Body. Online self-registration with the tax number; "Honey production and packaging" is a listed activity; also tick sale/distribution. Changes are reported to 118 / snund@snund.am. | [snund.am/hy/business-registration](https://snund.am/hy/business-registration) | free | **Same day** |
| 6 | **Sanitary (medical) booklet** (սանիտարական գրքույկ) for everyone who touches the honey — harvest, jarring, packing, delivery. Exam on start, then every 6 months. FSIB has enforced this on delivery staff since 1 Aug 2026. | Polyclinic | small fee ⚠ verify | **1–3** |
| 7 | **Apiary and honey veterinary papers.** The Law on Veterinary Medicine requires animal owners to register and keep passports/records, and forbids removing hive marks; honey is subject to a **veterinary-sanitary examination** (sensory + lab: water, diastase, acidity, sugars, antibiotics, radioactivity; result within ~2 hours; honey from diseased hives or with antibiotics is unsaleable). Ask the community veterinarian what the apiary needs on file and where the batch examination is done. | Community / regional veterinary service, FSIB lab | lab fee per batch ⚠ verify | apiary papers **1–7**; each batch exam **1** |
| 8 | **Labels in Armenian** (EAEU TR CU 022/2011 + consumer law): product name, composition, net weight, producer name and address, harvest/packing date, shelf life, storage conditions, batch. ⚠ verify with FSIB whether a **declaration of conformity** (TR CU 021/2011) is required for honey sold domestically at this scale. | Designer + FSIB | printing | **3–7** design; conformity question 1–2 weeks |
| 9 | **Website legal pages, in Armenian** (Russian/English optional): full contact details (email, postal address, phone), public offer / terms, total prices incl. delivery, payment methods, return policy, dispute route, privacy notice. Consumer-law amendments in force **since 1 July 2026**. Details in §5. | the site | — | **2–5** |
| 10 | **Electronic cash register** (էլեկտրոնային ՀԴՄ). Every card or cash sale to a person needs a fiscal receipt with a QR code — card via vPOS included; bank transfer does not. Apply through the SRC e-reporting system, receive the certificate, then either integrate the SRC e-HDM web service into the API (our own code) or use a provider. | SRC portal; provider e.g. [ehdm.am](https://ehdm.am/) | SRC certificate free ⚠ verify; provider 30,000 one-off + 3,000/month + 12/SMS | certificate **1–5**; own integration 1–2 weeks of dev |
| 11 | **Ameriabank vPOS**: open an Ameria Business account → the vPOS team emails **test credentials** → make **5 successful test payments** → sign the Application-Agreement → bank registers the site URL in the processing centre **within 5 banking days** → live credentials. | Ameriabank e-commerce desk | commission **not published**, per agreement (regional benchmark ≈2–3%); ask for USD acquiring and the settlement term in writing | test creds **1–5**; live **5–10** after the tests |
| 12 | *(optional)* **Trademark** "Mountain Breath" at the IP agency. 75% discount for IEs. | AIPA | 7,500 filing + 12,500 registration = **20,000 AMD** | **~4 months**, up to 12 |

**Critical path to a live card payment:** 1 → 2 → 3 (same week) → 4 and 11
(test credentials) — about **2–3 weeks** of paperwork if the ID card and
e-signature already exist, 4–5 if not. Steps 5–10 fit inside that window.
Live credentials arrive only after the integration passes the five test
payments, i.e. after code phase P2.

## 3. Taxes: which regime

Honey from one's own hives is **production** (and agriculture), not trade.
Two regimes are on the table; the third is a question for the accountant.

### 3.1 Turnover tax (շրջանառության հարկ) — the working assumption

- Available while last year's turnover is **≤ 115 million AMD**. Replaces
  VAT and profit/income tax on the activity.
- **Production: 7%** of turnover, reduced by **5% of documented costs**
  (jars, lids, labels, sugar feed, fuel, packaging — with invoices), never
  below a **3.5% floor**.
- If the shop ever resells other beekeepers' honey, that part is **trade:
  10%**, reduced by 9.5% of documented purchase costs, floor 1%. Buying
  from a private beekeeper without documents costs 20% income tax on the
  purchase unless the community head certifies the producer — so keep
  resale out, or paper it.
- **Fixed monthly**: **5,000 AMD income tax + 5,000 AMD social payment**
  = 10,000/month (social only for people born 1 Jan 1974 or later).
- **Report + pay quarterly**, by the **20th of the month after the
  quarter** (20 Jan / Apr / Jul / Oct).

### 3.2 Micro-entrepreneurship (միկրոձեռնարկատիրություն) — 0%, probably closed to us

- **≤ 24 million AMD** turnover, **0%** on all state taxes for the
  activity, one annual turnover report by **1 February**.
- Since 1 Jan 2025 it excludes trade inside Yerevan, trade in shopping
  centres, and — per the SME agency, taxinfo.am and accountant.am —
  **e-commerce / online sales**. Agricultural production itself is
  allowed. ⚠ verify with the SRC call centre whether *a producer selling
  its own honey through its own website* falls under the e-commerce
  exclusion. If it does not, micro is the cheapest regime by far and is
  chosen in the same 20-day window as step 3.

### 3.3 General regime + agricultural exemption — ask the accountant

Income from agricultural production is exempt from profit tax until
**31 December 2026** (Tax Code, may be extended). But the general regime
also means being a **VAT payer at 20%** on every sale, which for a
consumer shop is worse than 7%. Only worth asking if the accountant
knows an agricultural VAT relief that applies to own-produced honey. ⚠

### 3.4 Everything else the IE pays regardless of regime

| Payment | Amount | When |
|---|---|---|
| Stamp duty (դրոշմանիշային վճար, military fund) | **12,000 AMD/year** if turnover ≤ 12M; **120,000** above | annually, by 20 April of the following year ⚠ verify the date |
| Mandatory health insurance (from 2026) | **129,600 AMD/year** once the *previous* year's turnover is ≥ 2.4M — so at the earliest for the year after the shop crosses 2.4M | by 20 April |
| VAT | none below 115M turnover | — |
| Employees, if any | 20% income tax withheld, social payments, per-employee stamp duty | monthly |

### 3.5 Worked example — 6,000,000 AMD/year (500k/month), turnover tax, no staff

| Item | AMD/year |
|---|---|
| Turnover tax 7% of 6.0M | 420,000 (floor 210,000 if costs are well documented) |
| Fixed income tax + social 10,000 × 12 | 120,000 |
| Stamp duty | 12,000 |
| Health insurance (from the second year) | 129,600 |
| Card acquiring ≈2.5% on, say, 4M of card sales | ~100,000 |
| Bank account maintenance | 0–20,000 |
| E-signature | 10,000 |
| e-HDM provider (0 if we integrate the SRC API ourselves) | 0–66,000 |
| **Total** | **≈ 700–880k, i.e. 12–15% of turnover** (≈ 8–10% once costs are documented and before health insurance kicks in) |

Under micro the same year would cost ≈ 60,000 social + 12,000 stamp
duty + bank/e-signature — which is why §3.2's ⚠ question is worth a
phone call.

## 4. The recurring calendar

| When | What |
|---|---|
| Every month, by the 20th | 5,000 income tax + 5,000 social payment (turnover-tax IE) |
| 20 Jan, 20 Apr, 20 Jul, 20 Oct | Turnover-tax report and payment for the previous quarter |
| 1 February | Micro annual turnover report (only if on micro) |
| 20 April | Stamp duty; health insurance when due; IE annual income declaration ⚠ verify |
| Every 6 months | Sanitary booklet medical exam |
| Each honey batch | Veterinary-sanitary examination before sale |
| Before 20 Feb each year | Re-confirm the tax regime if changing it |
| Each 7 business days before | Tell the bank about any change of site URL or business scope (vPOS T&C §19) |

## 5. What the bank and the consumer law require of the **site** — code items

These come out of the paperwork and land in the backlog, most in P2:

- **Fiscal e-receipt per paid order** (step 10): after `payments.state`
  flips to `paid`, issue the receipt through the SRC e-HDM web service
  and store its number on the order; email the receipt with the order
  confirmation. Cash-on-delivery needs one too; bank transfer does not.
- **"Order with obligation to pay"** — the consumer amendments require the
  final checkout button to say so explicitly (the canvas's "Place order"
  copy gets a legally required rewording, in all three languages).
- **Pre-contract information** on the checkout: total price including
  delivery, payment methods, delivery time, the return policy link, seller
  identity and contact details — most already there, audit against the
  list.
- **Return / withdrawal policy page**: the amendments introduce a 14-day
  no-reason withdrawal for distance sales; the law is modelled on the EU
  directive, which exempts perishable and sealed food once opened — ⚠
  verify the Armenian exception list before writing the policy. Refunds
  for card orders go back through the provider (P3), never in cash
  (vPOS T&C §16).
- **Privacy notice** and consent wording; the 2026 data-protection
  amendments add a **72-hour breach notification** to the Personal Data
  Protection Agency — the alerting road from #105 is where that would
  start. No registration of the shop with the agency is required.
- **Armenian first**: every consumer-facing text must exist in Armenian;
  the other two languages are optional extras — which our i18n already
  satisfies as long as `hy` is never the missing translation.
- **Delivery proof** (vPOS T&C §10): for card orders keep goods
  description, delivery address, date/time, recipient name and signature,
  last 4 card digits, retained 3 years — the courier's signed slip plus
  the order record cover it; do not shorten order retention below 3 years.
- **No card surcharge** (T&C §15): card price = cash price. The shop has
  one price per market, so this holds by construction.

## 6. Questions to take to people

**Accountant (one paid hour is enough):**
1. Does own-honey-via-own-website count as the e-commerce exclusion from
   micro? (§3.2)
2. Turnover tax at 7% production vs any agricultural relief under the
   general regime. (§3.3)
3. Exact stamp duty / health insurance timing for a first-year IE.
4. Which documented costs count toward the 5% reduction, and what an
   invoice from a private supplier must look like.

**Ameriabank e-commerce desk:**
1. Acquiring commission and settlement term (T+1?) in writing.
2. USD acquiring on the same contract, or AMD only (decides whether the
   USD market gets card at all — [PLAN_PAYMENTS §5](PLAN_PAYMENTS.md)).
3. The current "minimum requirements to the website" document (the linked
   PDF is Armenian-only and from 2022).
4. Whether the test order-id range must be honoured by offsetting our ids.

**Food Safety Inspection Body / community vet:**
1. Apiary registration and what a batch examination costs and where.
2. Declaration of conformity for honey at this scale, yes or no.
3. Label review before printing.

## 7. The alternative: stay a natural person (asked 2026-09-07)

Selling honey from one's own hives **without any registration is
lawful**: the Tax Code recognises "natural persons, not individual
entrepreneurs, engaged in agricultural production" as a status (buyers
document purchases from them with a purchase act and a community-head
certificate, art. 55), and a natural person's income from own
agricultural produce is exempt from income tax — currently until
31 Dec 2026, extension ⚠ verify.

What a natural person **cannot** do is take card payments online: Ameriabank
vPOS and ArCa need a legal entity or an IE, Idram's merchant contract
needs a legal entity, Stripe and PayPal do not serve Armenia. What
remains is exactly what the shop already has — `bank_transfer` to a
personal account or personal Idram wallet with the admin confirming, and
`cash_on_delivery`.

Risks of running the site that way: a personal bank account receiving
regular "for honey" transfers will be questioned under AML rules and can
be restricted; a site with a cart, prices, delivery and advertising looks
like systematic entrepreneurship — the exemption protects the *income*
from own honey, not the *activity*, and vanishes the moment anything not
self-produced is sold (fine ≈100,000 AMD, repeat up to 500,000, criminal
liability at large turnover); no fiscal receipts, since a natural person
has no cash register.

Still required regardless of status: the veterinary-sanitary honey
examination, Armenian labels, the consumer-law site obligations, the
privacy notice.

**The hybrid:** run on transfer + cash as a natural person now, register
the IE the day card payments are wanted. The IE's price for that is
3,000 AMD once, 10,000 AMD/month fixed, 7% of turnover (floor 3.5%). The
Ameriabank **sandbox** also needs the business account, so of the
payments plan only P4 waits on the IE; P1–P2 run against the scripted
bank and the stub provider.

## 8. Sources read on 2026-09-04

- IE registration, fee, same-day: [e-register.moj.am](https://e-register.moj.am/en/services/6d262855-3e61-417d-a601-bf4dcc673dab), [mblegal.am](https://mblegal.am/register-as-individual-entrepreneur/), [armenia.eregulations.org](https://armenia.eregulations.org/procedure/print/5/step/0?showRecourses=true&showCertification=false&l=en)
- LLC registration free, 2 working days: [armenian-lawyer.com](https://armenian-lawyer.com/immigration/registering-a-company-online-in-armenia-2026-step-by-step-plus-tax-and-social-security-setup-for-residency-applicants/), [gritarres.com](https://www.gritarres.com/blog/company-registration/)
- E-signature 10,000 AMD/year from 1 Feb 2026, mobile ID: [ekeng.am](https://www.ekeng.am/en/e_sign), [ekeng.am/mid](https://www.ekeng.am/en/mid_auth)
- 20-day regime rule: [easytaxes.am](https://easytaxes.am/en/article/tax-mode-20-days), [armenian-lawyer.com](https://armenian-lawyer.com/taxation-2/february-20-deadline-choosing-tax-regime-armenia/)
- Turnover tax rates/deductions/floors from 2025: [fincore.am](https://fincore.am/turnover-tax-changes-2025-part-1/), [alphaaccounting.am](https://alphaaccounting.am/news/changes-to-tax-code-related-turnover-tax/), [PwC](https://taxsummaries.pwc.com/armenia/corporate/other-taxes), [hartak.am](https://hartak.am/news/e5015ab2-3a5f-44ce-86e5-105dc36434ab/)
- Micro regime, exclusions incl. e-commerce and Yerevan trade: [smednc.am](https://smednc.am/hy/inner/635), [taxinfo.am](https://taxinfo.am/?page_id=1346), [accountant.am](https://www.accountant.am/tag/միկրոձեռնարկատիրություն/), [retrieve.am](https://www.retrieve.am/en/blog/taxes-in-armenia-the-ultimate-2026-guide-for-businesses-and-individuals)
- Agricultural purchases without documents / community-head certificate: [how2b.am](https://how2b.am/harkayin-popoxutyunner-2025-shrjanarutyan-hark/); agricultural profit-tax exemption to 31 Dec 2026: [PwC](https://taxsummaries.pwc.com/armenia/corporate/tax-credits-and-incentives)
- IE fixed 5,000 + 5,000 monthly, pension birth-year rule, quarterly deadline: [armenian-lawyer.com/taxes-armenia](https://armenian-lawyer.com/taxes-armenia/), [PwC individual](https://taxsummaries.pwc.com/armenia/individual/other-taxes)
- Stamp duty 12,000 / 120,000 and health insurance 129,600 from 2026: [easylife.am](https://www.easylife.am/en/post/tax-changes-for-individual-entrepreneurs-in-armenia-starting-2026-health-insurance-and-stamp-duty), [accountant.am](https://www.accountant.am/աձ-դրոշմանիշային-վճար-2026/)
- Ameriabank: [Internet Acquiring Terms and Conditions (ed. 4, 2021)](https://ameriabank.am/userfiles/file/Retail/Internet_Acquiring_Terms_and_Conditions_ed4_eng.pdf), [corporate tariffs ed. 34](https://ameriabank.am/Portals/0/files/Business/green/Corporate_tariffs_ed34_eng.pdf), onboarding via [Ucraft's guide](https://support.ucraft.com/hc/ucraft-knowledge-base/articles/how-to-setup-ameriabank-vpos)
- Food-chain operator registration: [snund.am/hy/business-registration](https://snund.am/hy/business-registration); sanitary booklets and 1 Aug 2026 enforcement: [arka.am](https://arka.am/en/news/economy/from-august-1-health-certificates-and-periodic-medical-examinations-will-become-mandatory-for-food-d/), [news.am](https://news.am/en/news/1052101)
- Honey veterinary-sanitary examination order: [arlis.am/hy/acts/73627](https://www.arlis.am/hy/acts/73627); Law on Veterinary Medicine (animal registration, hive marks): [arlis.am](https://www.arlis.am/hy/acts/193641)
- Electronic cash register for online shops: [finport.am](https://finport.am/full_news.php?id=44278&lang=3), [ehdm.am](https://ehdm.am/), [taxinfo.am ՀԴՄ](https://taxinfo.am/?page_id=1331)
- Consumer-protection amendments from 1 July 2026: [arka.am](https://arka.am/en/news/economy/armenia-introduces-a-14-day-cancellation-period-for-remote-purchases-regulator-on-new-rules/), [aravot.am](https://www.aravot.am/2026/06/19/1564546/), [armenian-lawyer.com](https://armenian-lawyer.com/business-immigration/armenias-consumer-protection-law-critical-requirements-for-foreign-companies-selling-to-armenian-consumers/)
- Personal data 2026 amendments: [vlolawfirm.com](https://vlolawfirm.com/legal-updates/armenia-2026-q3-data-protection)
- Trademark fees and timeline: [retrieve.am](https://www.retrieve.am/en/blog/trademark-registration-in-armeniaprocess-timeline-fees-and-common-refusals)

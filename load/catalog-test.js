// Load test for the Mountain Breath stack. Run against the CONTAINERIZED
// stack (realistic topology: nginx → api → postgres):
//
//   k6 run load/catalog-test.js
//   k6 run -e BASE_URL=http://localhost load/catalog-test.js
//
// Before a run with buyers, give the shop stock to sell:
//   docker compose -f deploy/docker-compose.yml exec postgres \
//     psql -U mb -d mountain_breath -c "UPDATE product_variants SET stock_qty = 1000000;"
//
// RE-BASELINED IN E10 (the plan's own instruction): the Era I script shopped
// a catalog that no longer exists (herbal-tea, armenian-coffee), and the
// interesting queries have all changed shape since —
//   - /catalog/facets (E2) does three FILTER-aggregates over a CTE per hit,
//     and EVERY shop view calls it: the suspected first bottleneck.
//   - the list query joins translations (E1.5) and per-market prices (E5).
//   - checkout (E6/E7) locks rows, prices via domain.Price, writes the
//     discount split, sends mail (LogSink here) — a real transaction.
// The browse mix below mirrors what the SHOP PAGE actually fires, in a
// spread of locales and currencies, so the cache-hostile paths get hit.

import http from 'k6/http'
import { check, sleep } from 'k6'
import exec from 'k6/execution'

const BASE = __ENV.BASE_URL || 'http://localhost'

// Registration requires a password, but nothing ever logs back in with it —
// each VU reuses the session cookie from its own register response. The value
// is therefore write-only, so a random default keeps a credential literal out
// of the repository (and out of secret scanners) with no loss of realism.
const PASSWORD =
  __ENV.LOAD_USER_PASSWORD || `k6-${Math.random().toString(36).slice(2)}-${Date.now()}`

export const options = {
  scenarios: {
    // The crowd: window-shoppers ramping 0 → 20 concurrent users.
    browsers: {
      executor: 'ramping-vus',
      exec: 'browse',
      startVUs: 0,
      stages: [
        { duration: '30s', target: 20 }, // ramp up
        { duration: '60s', target: 20 }, // hold
        { duration: '15s', target: 0 },  // ramp down
      ],
    },
    // The minority that actually buys (each VU = one logged-in customer).
    buyers: {
      executor: 'constant-vus',
      exec: 'buy',
      vus: 5,
      duration: '105s',
    },
  },

  // SLOs as code: k6 exits non-zero if these break — CI-able later.
  thresholds: {
    http_req_duration: ['p(95)<200'], // 95% of requests under 200ms
    http_req_failed: ['rate<0.01'],   // <1% errors
    checks: ['rate>0.99'],
  },
}

// The shop's real vocabulary (E2's seed). Cycling views per iteration keeps
// Postgres from serving one warm plan the whole run.
const LOCALES = ['', '?lang=hy', '?lang=ru']
const CATEGORIES = ['honey', 'propolis', 'bee-pollen']
const BENEFITS = ['energy', 'immunity']

export function browse() {
  const i = exec.scenario.iterationInTest
  const lang = LOCALES[i % LOCALES.length]

  // What one shop-page view actually costs: the grid AND the sidebar.
  let res = http.get(`${BASE}/api/v1/products${lang || '?'}&per_page=12`)
  check(res, { 'products list 200': (r) => r.status === 200 })

  res = http.get(`${BASE}/api/v1/catalog/facets${lang}`)
  check(res, { 'facets 200': (r) => r.status === 200 })

  // A filter click = both queries again, narrowed (the facet counts must
  // respect the other active filters — that is the expensive part).
  const cat = CATEGORIES[i % CATEGORIES.length]
  const benefit = BENEFITS[i % BENEFITS.length]
  res = http.get(`${BASE}/api/v1/products?category=${cat}&benefit=${benefit}&currency=AMD`)
  check(res, { 'filtered list 200': (r) => r.status === 200 })
  res = http.get(`${BASE}/api/v1/catalog/facets?category=${cat}&benefit=${benefit}&currency=AMD`)
  check(res, { 'filtered facets 200': (r) => r.status === 200 })

  // The search path (FTS + trigram), with a deliberate typo half the time.
  res = http.get(`${BASE}/api/v1/products?q=${i % 2 ? 'honey' : 'hony'}`)
  check(res, { 'search 200': (r) => r.status === 200 })

  // A product page: the detail read plus its related panel.
  res = http.get(`${BASE}/api/v1/products/mountain-wildflower-honey${lang}`)
  check(res, { 'product detail 200': (r) => r.status === 200 })
  res = http.get(`${BASE}/api/v1/products/mountain-wildflower-honey/related${lang}`)
  check(res, { 'related 200': (r) => r.status === 200 })

  sleep(Math.random() * 2 + 0.5) // humans pause between clicks
}

// Module-level state is per-VU and SURVIVES iterations (unlike k6 v2's
// cookie jar, which resets each iteration) — so each VU logs in once and
// carries its session manually.
let sessionCookie = null

export function buy() {
  if (!sessionCookie) {
    const email = `k6-vu${exec.vu.idInTest}-${Date.now()}@load.test`
    const res = http.post(
      `${BASE}/api/v1/auth/register`,
      JSON.stringify({ email, password: PASSWORD }),
      { headers: { 'Content-Type': 'application/json' } },
    )
    check(res, { registered: (r) => r.status === 201 })
    const c = res.cookies['mb_session']
    if (c && c.length > 0) {
      sessionCookie = c[0].value
    } else {
      sleep(1)
      return
    }
  }

  const authed = {
    headers: {
      'Content-Type': 'application/json',
      Cookie: `mb_session=${sessionCookie}`,
    },
  }

  // Pick a variant id dynamically — never hardcode DB ids.
  const product = http.get(`${BASE}/api/v1/products/mountain-wildflower-honey`)
  const ok = check(product, { 'product for purchase 200': (r) => r.status === 200 })
  if (!ok) {
    sleep(1)
    return
  }
  const variantId = product.json().variants[0].id

  let res = http.put(
    `${BASE}/api/v1/cart/items`,
    JSON.stringify({ variant_id: variantId, qty: 1 }),
    authed,
  )
  check(res, { 'cart updated': (r) => r.status === 200 })

  // The E7 calculator — what the cart and checkout screens render from,
  // called once per screen in real life.
  res = http.post(`${BASE}/api/v1/checkout/preview`, null, authed)
  check(res, { 'preview 200': (r) => r.status === 200 })

  // E6's real checkout: the body carries choices, never money.
  res = http.post(
    `${BASE}/api/v1/orders`,
    JSON.stringify({
      address: {
        first_name: 'Load', last_name: 'Test', phone: '+374 91 000000',
        street: '1 Bench St', city: 'Yerevan', postal_code: '0001', country: 'AM',
      },
      payment_method: 'bank_transfer',
      delivery_note: '',
      leave_with_neighbour: false,
    }),
    authed,
  )
  check(res, { 'order created': (r) => r.status === 201 })

  sleep(2)
}

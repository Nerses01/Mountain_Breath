// Load test for the Mountain Breath stack. Run against the CONTAINERIZED
// stack (realistic topology: nginx → api → postgres):
//
//   k6 run load/catalog-test.js
//   k6 run -e BASE_URL=http://localhost load/catalog-test.js
//
// Before a run with buyers, give the shop stock to sell:
//   docker compose -f deploy/docker-compose.yml exec postgres \
//     psql -U mb -d mountain_breath -c "UPDATE product_variants SET stock_qty = 1000000;"

import http from 'k6/http'
import { check, sleep } from 'k6'
import exec from 'k6/execution'

const BASE = __ENV.BASE_URL || 'http://localhost'

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

export function browse() {
  let res = http.get(`${BASE}/api/v1/products?per_page=20`)
  check(res, { 'products list 200': (r) => r.status === 200 })

  res = http.get(`${BASE}/api/v1/products?category=herbal-tea`)
  check(res, { 'filtered list 200': (r) => r.status === 200 })

  res = http.get(`${BASE}/api/v1/products/armenian-coffee`)
  check(res, { 'product detail 200': (r) => r.status === 200 })

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
      JSON.stringify({ email, password: 'k6-load-pass-123' }),
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
  const product = http.get(`${BASE}/api/v1/products/wild-thyme-tea`)
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

  res = http.post(`${BASE}/api/v1/orders`, null, authed)
  check(res, { 'order created': (r) => r.status === 201 })

  sleep(2)
}

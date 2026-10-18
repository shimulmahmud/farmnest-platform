// full-funnel.js — k6 load test: combined full-funnel scenario.
//
// Simulates realistic user behavior mixing all major endpoint groups:
//   - Browse products (list + detail)     35%
//   - Search products                       20%
//   - Auth (login)                          10%
//   - Create order (checkout)               15%
//   - List orders (my orders)               10%
//   - Health/readiness                       5%
//   - Categories                            5%
//
// Usage: k6 run tests/load/full-funnel.js
//        K6_SCENARIO=high k6 run tests/load/full-funnel.js

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';
import { BASE_URL, PRODUCT_SLUGS, DEFAULT_THRESHOLDS, CUSTOMER_COUNT } from './lib/config.js';
import { getCustomerToken, customerHeaders, makeCreateOrderPayload, randInt } from './lib/helpers.js';

// --- Custom metrics ---
const funnelErrorRate = new Rate('funnel_errors');
const browseLatency = new Trend('funnel_browse_latency');
const searchLatency = new Trend('funnel_search_latency');
const authLatency = new Trend('funnel_auth_latency');
const checkoutLatency = new Trend('funnel_checkout_latency');
const myOrdersLatency = new Trend('funnel_my_orders_latency');
const ordersCreated = new Counter('funnel_orders_created');

const PRODUCT_ID = __ENV.SEED_PRODUCT_ID || '000000000000000000000001';
const VARIANT_ID = __ENV.SEED_VARIANT_ID || '000000000000000000000002';
const SKU = __ENV.SEED_SKU || 'LT-SKU-0001';

// --- Scenarios ---
const scenarioName = __ENV.K6_SCENARIO || 'normal';

const SCENARIOS = {
  normal: {
    executor: 'constant-vus',
    vus: 20,
    duration: '3m',
  },
  high: {
    executor: 'constant-vus',
    vus: 100,
    duration: '5m',
  },
  spike: {
    executor: 'ramping-vus',
    startVUs: 5,
    stages: [
      { duration: '30s', target: 10 },
      { duration: '15s', target: 200 },   // traffic spike
      { duration: '2m', target: 200 },
      { duration: '30s', target: 10 },
      { duration: '15s', target: 5 },
    ],
  },
};

export const options = {
  scenarios: {
    funnel: SCENARIOS[scenarioName] || SCENARIOS.normal,
  },
  thresholds: {
    ...DEFAULT_THRESHOLDS,
    funnel_errors: ['rate<0.05'],
    browseLatency: ['p(95)<400', 'p(99)<1200'],
    searchLatency: ['p(95)<500', 'p(99)<1500'],
    authLatency: ['p(95)<400', 'p(99)<1500'],
    checkoutLatency: ['p(95)<1000', 'p(99)<3000'],
    myOrdersLatency: ['p(95)<500', 'p(99)<1500'],
  },
};

const ACCEPT_JSON = { 'Accept': 'application/json' };
const SEARCH_TERMS = ['LoadTest', 'Product', 'TestBrand1', 'TestBrand2', 'tag1'];

export default function () {
  const r = Math.random();
  let passed = true;

  if (r < 0.35) {
    // --- 35%: Browse products ---
    if (r < 0.20) {
      // List
      const page = Math.floor(Math.random() * 3) + 1;
      const start = Date.now();
      const res = http.get(`${BASE_URL}/api/v1/products?page=${page}&per_page=20`,
        { headers: ACCEPT_JSON, tags: { name: 'funnel/browse_list' } });
      browseLatency.add(Date.now() - start);
      passed = check(res, { 'browse list 200': (r) => r.status === 200 });
    } else {
      // Detail
      const slug = PRODUCT_SLUGS[Math.floor(Math.random() * PRODUCT_SLUGS.length)];
      const start = Date.now();
      const res = http.get(`${BASE_URL}/api/v1/products/${slug}`,
        { headers: ACCEPT_JSON, tags: { name: 'funnel/browse_detail' } });
      browseLatency.add(Date.now() - start);
      passed = check(res, { 'browse detail 200': (r) => r.status === 200 });
    }
    sleep(Math.random() * 0.5 + 0.2);

  } else if (r < 0.55) {
    // --- 20%: Search ---
    const q = SEARCH_TERMS[Math.floor(Math.random() * SEARCH_TERMS.length)];
    const start = Date.now();
    const res = http.get(`${BASE_URL}/api/v1/products/search?q=${encodeURIComponent(q)}&per_page=20`,
      { headers: ACCEPT_JSON, tags: { name: 'funnel/search' } });
    searchLatency.add(Date.now() - start);
    passed = check(res, { 'search 200': (r) => r.status === 200 });
    sleep(Math.random() * 0.3 + 0.1);

  } else if (r < 0.65) {
    // --- 10%: Auth (login) ---
    const idx = Math.floor(Math.random() * CUSTOMER_COUNT) + 1;
    const start = Date.now();
    const res = http.post(`${BASE_URL}/api/v1/auth/login`, JSON.stringify({
      email: `loadtest-customer-${idx}@ecommerce.test`,
      password: 'LoadTest1234!',
    }), {
      headers: { 'Content-Type': 'application/json' },
      tags: { name: 'funnel/auth' },
    });
    authLatency.add(Date.now() - start);
    passed = check(res, { 'auth login 200': (r) => r.status === 200 });
    sleep(Math.random() * 0.2 + 0.1);

  } else if (r < 0.80) {
    // --- 15%: Create order (checkout) ---
    const vuIdx = (__VU - 1) % CUSTOMER_COUNT;
    const token = getCustomerToken(vuIdx);
    if (!token) { funnelErrorRate.add(1); return; }

    const items = [{
      product_id: PRODUCT_ID,
      variant_id: VARIANT_ID,
      name: `Funnel Product ${__ITER}`,
      sku: SKU,
      quantity: randInt(1, 2),
      unit_price: 10000 + vuIdx * 100,
      discount: 0,
    }];

    const start = Date.now();
    const res = http.post(`${BASE_URL}/api/v1/orders`, JSON.stringify(makeCreateOrderPayload(items)),
      { headers: customerHeaders(vuIdx), tags: { name: 'funnel/checkout' } });
    checkoutLatency.add(Date.now() - start);
    passed = check(res, { 'checkout 201': (r) => r.status === 201 });
    if (passed) ordersCreated.add(1);
    sleep(Math.random() * 1 + 0.5);

  } else if (r < 0.90) {
    // --- 10%: My orders ---
    const vuIdx = (__VU - 1) % CUSTOMER_COUNT;
    const token = getCustomerToken(vuIdx);
    if (!token) { funnelErrorRate.add(1); return; }

    const start = Date.now();
    const res = http.get(`${BASE_URL}/api/v1/orders?page=1&per_page=20`,
      { headers: customerHeaders(vuIdx), tags: { name: 'funnel/my_orders' } });
    myOrdersLatency.add(Date.now() - start);
    passed = check(res, { 'my orders 200': (r) => r.status === 200 });
    sleep(Math.random() * 0.3 + 0.1);

  } else if (r < 0.95) {
    // --- 5%: Categories ---
    http.get(`${BASE_URL}/api/v1/categories`,
      { headers: ACCEPT_JSON, tags: { name: 'funnel/categories' } });
    sleep(Math.random() * 0.2 + 0.1);

  } else {
    // --- 5%: Health ---
    http.get(`${BASE_URL}/health`, { tags: { name: 'funnel/health' } });
    sleep(0.1);
  }

  funnelErrorRate.add(!passed);
}

export function handleSummary(data) {
  return {
    'stdout': `
=== Full Funnel Load Test (${scenarioName}) ===

Browse Latency:
  p50: ${data.metrics.funnel_browse_latency?.values?.p50 || 'N/A'} ms
  p95: ${data.metrics.funnel_browse_latency?.values?.p95 || 'N/A'} ms
  p99: ${data.metrics.funnel_browse_latency?.values?.p99 || 'N/A'} ms

Search Latency:
  p50: ${data.metrics.funnel_search_latency?.values?.p50 || 'N/A'} ms
  p95: ${data.metrics.funnel_search_latency?.values?.p95 || 'N/A'} ms
  p99: ${data.metrics.funnel_search_latency?.values?.p99 || 'N/A'} ms

Auth Latency:
  p50: ${data.metrics.funnel_auth_latency?.values?.p50 || 'N/A'} ms
  p95: ${data.metrics.funnel_auth_latency?.values?.p95 || 'N/A'} ms
  p99: ${data.metrics.funnel_auth_latency?.values?.p99 || 'N/A'} ms

Checkout Latency:
  p50: ${data.metrics.funnel_checkout_latency?.values?.p50 || 'N/A'} ms
  p95: ${data.metrics.funnel_checkout_latency?.values?.p95 || 'N/A'} ms
  p99: ${data.metrics.funnel_checkout_latency?.values?.p99 || 'N/A'} ms

My Orders Latency:
  p50: ${data.metrics.funnel_my_orders_latency?.values?.p50 || 'N/A'} ms
  p95: ${data.metrics.funnel_my_orders_latency?.values?.p95 || 'N/A'} ms

Overall Error Rate: ${((data.metrics.funnel_errors?.values?.rate || 0) * 100).toFixed(2)}%
Orders Created: ${data.metrics.funnel_orders_created?.values?.count || 0}

Total Requests: ${data.metrics.http_reqs?.values?.count || 0}
Throughput (RPS): ${data.metrics.http_reqs?.values?.rate || 'N/A'}
`,
  };
}

// order-operations.js — k6 load test: order creation and order reads.
//
// Tests the checkout-like flow:
//   1. Login as customer (token cached)
//   2. POST /api/v1/orders              (create order)
//   3. GET  /api/v1/orders              (list my orders)
//   4. GET  /api/v1/orders/:id          (get order detail)
//
// Also tests admin order reads:
//   GET /api/v1/admin/orders           (list all orders)
//
// Usage: k6 run tests/load/order-operations.js
//        K6_SCENARIO=checkout-spike k6 run tests/load/order-operations.js

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';
import { BASE_URL, DEFAULT_THRESHOLDS, CUSTOMER_COUNT } from './lib/config.js';
import { getCustomerToken, getAdminToken, adminHeaders, customerHeaders, randInt, makeCreateOrderPayload } from './lib/helpers.js';

// --- Custom metrics ---
const orderCreateErrorRate = new Rate('order_create_errors');
const orderReadErrorRate = new Rate('order_read_errors');
const orderCreateLatency = new Trend('order_create_latency');
const orderListLatency = new Trend('order_list_latency');
const orderDetailLatency = new Trend('order_detail_latency');
const adminOrderListLatency = new Trend('admin_order_list_latency');
const ordersCreated = new Counter('orders_created');

// --- Scenarios ---
const scenarioName = __ENV.K6_SCENARIO || 'normal';

const SCENARIOS = {
  normal: {
    executor: 'constant-vus',
    vus: 5,
    duration: '2m',
  },
  high: {
    executor: 'constant-vus',
    vus: 30,
    duration: '5m',
  },
  'checkout-spike': {
    executor: 'ramping-vus',
    startVUs: 2,
    stages: [
      { duration: '20s', target: 2 },
      { duration: '10s', target: 80 },  // checkout spike
      { duration: '3m', target: 80 },   // hold at peak
      { duration: '30s', target: 2 },   // ramp down
    ],
  },
};

export const options = {
  scenarios: {
    orders: SCENARIOS[scenarioName] || SCENARIOS.normal,
  },
  thresholds: {
    ...DEFAULT_THRESHOLDS,
    order_create_errors: ['rate<0.1'],     // allow slightly higher for write ops
    order_read_errors: ['rate<0.05'],
    order_create_latency: ['p(95)<800', 'p(99)<3000'],
    order_list_latency: ['p(95)<500', 'p(99)<1500'],
    order_detail_latency: ['p(95)<400', 'p(99)<1200'],
    admin_order_list_latency: ['p(95)<600', 'p(99)<2000'],
  },
};

// Mock product/variant IDs — replace after seeding (or use env vars).
// In a real run, these come from the seed output.
const PRODUCT_ID = __ENV.SEED_PRODUCT_ID || '000000000000000000000001';
const VARIANT_ID = __ENV.SEED_VARIANT_ID || '000000000000000000000002';
const SKU = __ENV.SEED_SKU || 'LT-SKU-0001';

export default function () {
  const vuIdx = (__VU - 1) % CUSTOMER_COUNT;
  const token = getCustomerToken(vuIdx);
  if (!token) {
    // Customer login failed — skip this iteration.
    orderCreateErrorRate.add(1);
    return;
  }

  const r = Math.random();
  const authH = customerHeaders(vuIdx);

  if (r < 0.5) {
    // --- 50%: Create order (checkout-like) ---
    const items = [{
      product_id: PRODUCT_ID,
      variant_id: VARIANT_ID,
      name: `LoadTest Product ${vuIdx}`,
      sku: SKU,
      quantity: randInt(1, 3),
      unit_price: 10000 + vuIdx * 100,
      discount: 0,
    }];
    const payload = makeCreateOrderPayload(items);

    const start = Date.now();
    const res = http.post(
      `${BASE_URL}/api/v1/orders`,
      JSON.stringify(payload),
      { headers: authH, tags: { name: 'orders/create' } },
    );
    orderCreateLatency.add(Date.now() - start);

    const passed = check(res, {
      'order create status 201': (r) => r.status === 201,
      'order create has id': (r) => {
        const body = r.json('data') || r.json();
        return body && body.id;
      },
      'order create has order_number': (r) => {
        const body = r.json('data') || r.json();
        return body && body.order_number;
      },
    });
    orderCreateErrorRate.add(!passed);
    if (passed) ordersCreated.add(1);

    sleep(Math.random() * 1 + 0.5); // 500-1500ms think time (checkout is slower)

  } else if (r < 0.75) {
    // --- 25%: List my orders ---
    const start = Date.now();
    const res = http.get(
      `${BASE_URL}/api/v1/orders?page=1&per_page=20`,
      { headers: authH, tags: { name: 'orders/list' } },
    );
    orderListLatency.add(Date.now() - start);

    const passed = check(res, {
      'order list status 200': (r) => r.status === 200,
    });
    orderReadErrorRate.add(!passed);

    sleep(Math.random() * 0.3 + 0.1);

  } else {
    // --- 25%: Admin list all orders ---
    const aH = adminHeaders();
    const start = Date.now();
    const res = http.get(
      `${BASE_URL}/api/v1/admin/orders?page=1&per_page=50`,
      { headers: aH, tags: { name: 'admin/orders/list' } },
    );
    adminOrderListLatency.add(Date.now() - start);

    const passed = check(res, {
      'admin order list status 200': (r) => r.status === 200,
    });
    orderReadErrorRate.add(!passed);

    sleep(Math.random() * 0.3 + 0.1);
  }
}

export function handleSummary(data) {
  return {
    'stdout': `
=== Order Operations Load Test (${scenarioName}) ===

Order Create Latency:
  p50: ${data.metrics.order_create_latency?.values?.p50 || 'N/A'} ms
  p95: ${data.metrics.order_create_latency?.values?.p95 || 'N/A'} ms
  p99: ${data.metrics.order_create_latency?.values?.p99 || 'N/A'} ms

Order List Latency:
  p50: ${data.metrics.order_list_latency?.values?.p50 || 'N/A'} ms
  p95: ${data.metrics.order_list_latency?.values?.p95 || 'N/A'} ms
  p99: ${data.metrics.order_list_latency?.values?.p99 || 'N/A'} ms

Order Detail Latency:
  p50: ${data.metrics.order_detail_latency?.values?.p50 || 'N/A'} ms
  p95: ${data.metrics.order_detail_latency?.values?.p95 || 'N/A'} ms

Admin Order List Latency:
  p50: ${data.metrics.admin_order_list_latency?.values?.p50 || 'N/A'} ms
  p95: ${data.metrics.admin_order_list_latency?.values?.p95 || 'N/A'} ms

Order Create Error Rate: ${((data.metrics.order_create_errors?.values?.rate || 0) * 100).toFixed(2)}%
Order Read Error Rate: ${((data.metrics.order_read_errors?.values?.rate || 0) * 100).toFixed(2)}%
Orders Created: ${data.metrics.orders_created?.values?.count || 0}

Total Requests: ${data.metrics.http_reqs?.values?.count || 0}
Throughput (RPS): ${data.metrics.http_reqs?.values?.rate || 'N/A'}
`,
  };
}

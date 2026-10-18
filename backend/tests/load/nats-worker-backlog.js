// nats-worker-backlog.js — k6 load test: generate events to stress NATS + order worker.
//
// Strategy: burst-create orders, then monitor via admin endpoints.
// The order worker processes outbox messages via NATS JetStream.
// By creating many orders rapidly, we build up the NATS consumer backlog.
//
// Tests:
//   - Burst order creation (generates outbox + NATS events)
//   - GET /api/v1/admin/orders (check processing status)
//   - Health/readiness to verify NATS is still responsive under load
//
// Prerequisites: seed.js must have been run; worker process must be running.
//
// Usage: k6 run tests/load/nats-worker-backlog.js
//        K6_BURST_SIZE=500 k6 run tests/load/nats-worker-backlog.js

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';
import { BASE_URL, DEFAULT_THRESHOLDS, CUSTOMER_COUNT } from './lib/config.js';
import { getCustomerToken, customerHeaders, adminHeaders, makeCreateOrderPayload, randInt } from './lib/helpers.js';

// --- Custom metrics ---
const burstErrorRate = new Rate('burst_order_errors');
const burstLatency = new Trend('burst_order_latency');
const adminPollLatency = new Trend('admin_poll_latency');
const natsHealthLatency = new Trend('nats_health_latency');
const totalBurstOrders = new Counter('burst_orders_created');

const BURST_SIZE = parseInt(__ENV.K6_BURST_SIZE || '200', 10);
const POLL_INTERVAL = 3; // seconds between admin polls
const POLL_COUNT = 10;  // number of polls after burst

const PRODUCT_ID = __ENV.SEED_PRODUCT_ID || '000000000000000000000001';
const VARIANT_ID = __ENV.SEED_VARIANT_ID || '000000000000000000000002';
const SKU = __ENV.SEED_SKU || 'LT-SKU-0001';

export const options = {
  scenarios: {
    // Phase 1: Burst order creation
    burst: {
      executor: 'constant-vus',
      vus: BURST_SIZE,
      duration: '30s',
      exec: 'burstOrders',
      startTime: '0s',
    },
    // Phase 2: Admin polling while worker catches up
    poll: {
      executor: 'constant-vus',
      vus: 3,
      duration: `${POLL_COUNT * POLL_INTERVAL + 5}s`,
      exec: 'pollAdmin',
      startTime: '25s', // overlap slightly
    },
  },
  thresholds: {
    ...DEFAULT_THRESHOLDS,
    burst_order_errors: ['rate<0.15'],  // tolerate more during burst
    burst_order_latency: ['p(95)<1500', 'p(99)<5000'],
    admin_poll_latency: ['p(95)<600', 'p(99)<2000'],
    nats_health_latency: ['p(95)<200', 'p(99)<500'],
  },
};

// --- Burst: create orders as fast as possible ---
export function burstOrders() {
  const vuIdx = (__VU - 1) % CUSTOMER_COUNT;
  const token = getCustomerToken(vuIdx);
  if (!token) {
    burstErrorRate.add(1);
    return;
  }

  const items = [{
    product_id: PRODUCT_ID,
    variant_id: VARIANT_ID,
    name: `Burst Product ${__VU}-${__ITER}`, 
    sku: SKU,
    quantity: randInt(1, 2),
    unit_price: 5000 + vuIdx * 50,
    discount: 0,
  }];
  const payload = makeCreateOrderPayload(items);

  const start = Date.now();
  const res = http.post(
    `${BASE_URL}/api/v1/orders`,
    JSON.stringify(payload),
    { headers: customerHeaders(vuIdx), tags: { name: 'orders/burst_create' } },
  );
  burstLatency.add(Date.now() - start);

  const passed = check(res, {
    'burst order status 201': (r) => r.status === 201,
    'burst order has id': (r) => {
      const body = r.json('data') || r.json();
      return body && body.id;
    },
  });
  burstErrorRate.add(!passed);
  if (passed) totalBurstOrders.add(1);
}

// --- Poll: admin monitors while worker processes backlog ---
export function pollAdmin() {
  const r = Math.random();

  if (r < 0.4) {
    // Poll admin order list
    const aH = adminHeaders();
    const start = Date.now();
    const res = http.get(
      `${BASE_URL}/api/v1/admin/orders?page=1&per_page=50`,
      { headers: aH, tags: { name: 'admin/orders/poll' } },
    );
    adminPollLatency.add(Date.now() - start);

    check(res, {
      'admin poll status 200': (r) => r.status === 200,
    });

  } else if (r < 0.7) {
    // Readiness check (validates NATS connection)
    const start = Date.now();
    http.get(`${BASE_URL}/ready`, { tags: { name: 'ready/nats' } });
    natsHealthLatency.add(Date.now() - start);

  } else {
    // Health check
    http.get(`${BASE_URL}/health`, { tags: { name: 'health/nats' } });
  }

  sleep(POLL_INTERVAL);
}

export function handleSummary(data) {
  return {
    'stdout': `
=== NATS Worker Backlog Test ===

Burst Configuration:
  VUs (concurrent orders): ${BURST_SIZE}
  Burst duration: 30s

Burst Order Latency:
  p50: ${data.metrics.burst_order_latency?.values?.p50 || 'N/A'} ms
  p95: ${data.metrics.burst_order_latency?.values?.p95 || 'N/A'} ms
  p99: ${data.metrics.burst_order_latency?.values?.p99 || 'N/A'} ms
  avg: ${data.metrics.burst_order_latency?.values?.avg || 'N/A'} ms

Admin Poll Latency (during backlog processing):
  p50: ${data.metrics.admin_poll_latency?.values?.p50 || 'N/A'} ms
  p95: ${data.metrics.admin_poll_latency?.values?.p95 || 'N/A'} ms

NATS Health (readiness check):
  p50: ${data.metrics.nats_health_latency?.values?.p50 || 'N/A'} ms
  p95: ${data.metrics.nats_health_latency?.values?.p95 || 'N/A'} ms

Burst Error Rate: ${((data.metrics.burst_order_errors?.values?.rate || 0) * 100).toFixed(2)}%
Orders Created in Burst: ${data.metrics.burst_orders_created?.values?.count || 0}

Total Requests: ${data.metrics.http_reqs?.values?.count || 0}
Throughput (RPS): ${data.metrics.http_reqs?.values?.rate || 'N/A'}

Note: After the test, query the database to check how many orders
reached PAID status vs still PENDING to measure worker catch-up time.
`,
  };
}

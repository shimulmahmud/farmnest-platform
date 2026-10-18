// product-reads.js — k6 load test: product listing and detail page reads.
//
// Tests the most common read path:
//   GET /api/v1/products          (list with pagination)
//   GET /api/v1/products/:slug    (detail by slug)
//   GET /api/v1/categories        (category listing)
//
// Usage: k6 run tests/load/product-reads.js
//        K6_SCENARIO=high k6 run tests/load/product-reads.js

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';
import { BASE_URL, PRODUCT_SLUGS, DEFAULT_THRESHOLDS } from './lib/config.js';

// --- Custom metrics ---
const productReadErrorRate = new Rate('product_read_errors');
const productListLatency = new Trend('product_list_latency');
const productDetailLatency = new Trend('product_detail_latency');
const categoryListLatency = new Trend('category_list_latency');

// --- Scenario selection via environment variable ---
const scenarioName = __ENV.K6_SCENARIO || 'normal';

const SCENARIOS = {
  normal: {
    executor: 'constant-vus',
    vus: 10,
    duration: '2m',
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
      { duration: '30s', target: 200 },  // sudden spike
      { duration: '2m', target: 200 },   // hold
      { duration: '30s', target: 5 },    // ramp down
    ],
  },
};

export const options = {
  scenarios: {
    product_reads: SCENARIOS[scenarioName] || SCENARIOS.normal,
  },
  thresholds: {
    ...DEFAULT_THRESHOLDS,
    product_read_errors: ['rate<0.05'],
    product_list_latency: ['p(95)<300', 'p(99)<1000'],
    product_detail_latency: ['p(95)<200', 'p(99)<800'],
    category_list_latency: ['p(95)<250', 'p(99)<900'],
  },
};

const ACCEPT_JSON = { 'Accept': 'application/json' };

export default function () {
  const r = Math.random();

  if (r < 0.5) {
    // --- 50%: List products with pagination ---
    const page = Math.floor(Math.random() * 5) + 1;
    const perPage = [10, 20, 50][Math.floor(Math.random() * 3)];

    const start = Date.now();
    const res = http.get(
      `${BASE_URL}/api/v1/products?page=${page}&per_page=${perPage}`,
      { headers: ACCEPT_JSON, tags: { name: 'products/list' } },
    );
    productListLatency.add(Date.now() - start);

    const passed = check(res, {
      'product list status 200': (r) => r.status === 200,
      'product list has data': (r) => {
        const body = r.json('data') || r.json();
        return Array.isArray(body) || (body && body.items);
      },
    });
    productReadErrorRate.add(!passed);

  } else if (r < 0.85) {
    // --- 35%: Get product by slug ---
    const slug = PRODUCT_SLUGS[Math.floor(Math.random() * PRODUCT_SLUGS.length)];

    const start = Date.now();
    const res = http.get(
      `${BASE_URL}/api/v1/products/${slug}`,
      { headers: ACCEPT_JSON, tags: { name: 'products/detail' } },
    );
    productDetailLatency.add(Date.now() - start);

    const passed = check(res, {
      'product detail status 200': (r) => r.status === 200,
      'product detail has id': (r) => {
        const body = r.json('data') || r.json();
        return body && body.id && body.slug;
      },
    });
    productReadErrorRate.add(!passed);

  } else {
    // --- 15%: List categories ---
    const start = Date.now();
    const res = http.get(
      `${BASE_URL}/api/v1/categories`,
      { headers: ACCEPT_JSON, tags: { name: 'categories/list' } },
    );
    categoryListLatency.add(Date.now() - start);

    const passed = check(res, {
      'category list status 200': (r) => r.status === 200,
    });
    productReadErrorRate.add(!passed);
  }

  sleep(Math.random() * 0.5 + 0.1); // 100-600ms think time
}

export function handleSummary(data) {
  const summary = {
    'stdout': `
=== Product Reads Load Test (${scenarioName}) ===

Product List Latency:
  p50: ${data.metrics.product_list_latency?.values?.p50 || 'N/A'} ms
  p95: ${data.metrics.product_list_latency?.values?.p95 || 'N/A'} ms
  p99: ${data.metrics.product_list_latency?.values?.p99 || 'N/A'} ms

Product Detail Latency:
  p50: ${data.metrics.product_detail_latency?.values?.p50 || 'N/A'} ms
  p95: ${data.metrics.product_detail_latency?.values?.p95 || 'N/A'} ms
  p99: ${data.metrics.product_detail_latency?.values?.p99 || 'N/A'} ms

Category List Latency:
  p50: ${data.metrics.category_list_latency?.values?.p50 || 'N/A'} ms
  p95: ${data.metrics.category_list_latency?.values?.p95 || 'N/A'} ms
  p99: ${data.metrics.category_list_latency?.values?.p99 || 'N/A'} ms

Error Rate: ${((data.metrics.product_read_errors?.values?.rate || 0) * 100).toFixed(2)}%

Total Requests: ${data.metrics.http_reqs?.values?.count || 0}
Total Iterations: ${data.metrics.iterations?.values?.count || 0}
Throughput (RPS): ${data.metrics.http_reqs?.values?.rate || 'N/A'}
`,
  };

  if (__ENV.K6_SUMMARY_OUT) {
    summary[__ENV.K6_SUMMARY_OUT] = data;
  }
  return summary;
}

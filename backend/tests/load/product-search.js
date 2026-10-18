// product-search.js — k6 load test: product search with various query patterns.
//
// Tests:
//   GET /api/v1/products/search?q=...    (text search)
//   GET /api/v1/products/search?brand=... (brand filter)
//   GET /api/v1/products/search?category_id=... (category filter)
//   GET /api/v1/products/search?min_price=...&max_price=... (price range)
//   Combined filters
//
// Usage: k6 run tests/load/product-search.js
//        K6_SCENARIO=high k6 run tests/load/product-search.js

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';
import { BASE_URL, DEFAULT_THRESHOLDS } from './lib/config.js';

// --- Custom metrics ---
const searchErrorRate = new Rate('search_errors');
const searchLatency = new Trend('search_latency');
const filterLatency = new Trend('filter_latency');
const combinedSearchLatency = new Trend('combined_search_latency');

// --- Scenarios ---
const scenarioName = __ENV.K6_SCENARIO || 'normal';

const SCENARIOS = {
  normal: {
    executor: 'constant-vus',
    vus: 10,
    duration: '2m',
  },
  high: {
    executor: 'constant-vus',
    vus: 80,
    duration: '5m',
  },
  spike: {
    executor: 'ramping-vus',
    startVUs: 5,
    stages: [
      { duration: '20s', target: 5 },
      { duration: '10s', target: 150 },   // sudden spike to 150 VUs
      { duration: '2m', target: 150 },
      { duration: '20s', target: 5 },
    ],
  },
};

export const options = {
  scenarios: {
    search: SCENARIOS[scenarioName] || SCENARIOS.normal,
  },
  thresholds: {
    ...DEFAULT_THRESHOLDS,
    search_errors: ['rate<0.05'],
    search_latency: ['p(95)<500', 'p(99)<1500'],
    filter_latency: ['p(95)<400', 'p(99)<1200'],
    combined_search_latency: ['p(95)<600', 'p(99)<2000'],
  },
};

const ACCEPT_JSON = { 'Accept': 'application/json' };

// Search terms that should match seeded products.
const SEARCH_TERMS = [
  'LoadTest', 'Product', 'product', 'test',
  'TestBrand1', 'TestBrand2', 'TestBrand3',
  'tag1', 'tag2', 'tag3',
];

const BRANDS = ['TestBrand1', 'TestBrand2', 'TestBrand3', 'TestBrand4', 'TestBrand5'];

export default function () {
  const r = Math.random();
  let url;
  let metric;

  if (r < 0.35) {
    // --- 35%: Text search ---
    const q = SEARCH_TERMS[Math.floor(Math.random() * SEARCH_TERMS.length)];
    const page = Math.floor(Math.random() * 3) + 1;
    url = `${BASE_URL}/api/v1/products/search?q=${encodeURIComponent(q)}&page=${page}&per_page=20`;
    metric = searchLatency;

  } else if (r < 0.55) {
    // --- 20%: Brand filter ---
    const brand = BRANDS[Math.floor(Math.random() * BRANDS.length)];
    url = `${BASE_URL}/api/v1/products/search?brand=${encodeURIComponent(brand)}&per_page=20`;
    metric = filterLatency;

  } else if (r < 0.70) {
    // --- 15%: Price range filter ---
    const min = [0, 1000, 3000, 5000][Math.floor(Math.random() * 4)];
    const max = [5000, 10000, 15000, 20000][Math.floor(Math.random() * 4)];
    url = `${BASE_URL}/api/v1/products/search?min_price=${min}&max_price=${max}&per_page=20`;
    metric = filterLatency;

  } else if (r < 0.85) {
    // --- 15%: Tag search ---
    const tag = [`tag1`, `tag2`, `tag3`, `loadtest`][Math.floor(Math.random() * 4)];
    url = `${BASE_URL}/api/v1/products/search?tags=${encodeURIComponent(tag)}&per_page=20`;
    metric = searchLatency;

  } else {
    // --- 15%: Combined filters (text + price + brand) ---
    const q = SEARCH_TERMS[Math.floor(Math.random() * 3)];
    const brand = BRANDS[Math.floor(Math.random() * BRANDS.length)];
    url = `${BASE_URL}/api/v1/products/search?q=${encodeURIComponent(q)}&brand=${encodeURIComponent(brand)}&min_price=1000&max_price=15000&per_page=20`;
    metric = combinedSearchLatency;
  }

  const start = Date.now();
  const res = http.get(url, { headers: ACCEPT_JSON, tags: { name: 'products/search' } });
  metric.add(Date.now() - start);

  const passed = check(res, {
    'search status 200': (r) => r.status === 200,
    'search response is object or array': (r) => {
      try {
        const body = r.json('data') || r.json();
        return body !== null && body !== undefined;
      } catch (e) {
        return false;
      }
    },
  });
  searchErrorRate.add(!passed);

  sleep(Math.random() * 0.3 + 0.1); // 100-400ms think time
}

export function handleSummary(data) {
  return {
    'stdout': `
=== Product Search Load Test (${scenarioName}) ===

Search Latency (text):
  p50: ${data.metrics.search_latency?.values?.p50 || 'N/A'} ms
  p95: ${data.metrics.search_latency?.values?.p95 || 'N/A'} ms
  p99: ${data.metrics.search_latency?.values?.p99 || 'N/A'} ms

Filter Latency (brand/price/tag):
  p50: ${data.metrics.filter_latency?.values?.p50 || 'N/A'} ms
  p95: ${data.metrics.filter_latency?.values?.p95 || 'N/A'} ms
  p99: ${data.metrics.filter_latency?.values?.p99 || 'N/A'} ms

Combined Search Latency:
  p50: ${data.metrics.combined_search_latency?.values?.p50 || 'N/A'} ms
  p95: ${data.metrics.combined_search_latency?.values?.p95 || 'N/A'} ms
  p99: ${data.metrics.combined_search_latency?.values?.p99 || 'N/A'} ms

Search Error Rate: ${((data.metrics.search_errors?.values?.rate || 0) * 100).toFixed(2)}%

Total Requests: ${data.metrics.http_reqs?.values?.count || 0}
Throughput (RPS): ${data.metrics.http_reqs?.values?.rate || 'N/A'}
`,
  };
}

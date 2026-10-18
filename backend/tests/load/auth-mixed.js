// auth-mixed.js — k6 load test: authentication endpoints under load.
//
// Tests:
//   POST /api/v1/auth/login          (most frequent)
//   POST /api/v1/auth/refresh         (token refresh)
//   GET  /health                      (liveness)
//   GET  /ready                       (readiness — checks MongoDB + NATS)
//
// Usage: k6 run tests/load/auth-mixed.js
//        K6_SCENARIO=high k6 run tests/load/auth-mixed.js

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';
import { BASE_URL, DEFAULT_THRESHOLDS, CUSTOMER_COUNT } from './lib/config.js';

// --- Custom metrics ---
const loginErrorRate = new Rate('login_errors');
const loginLatency = new Trend('login_latency');
const refreshLatency = new Trend('refresh_latency');
const healthLatency = new Trend('health_latency');
const readinessLatency = new Trend('readiness_latency');
const loginSuccesses = new Counter('login_successes');
let refreshTokens = []; // VU-local cache

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
    startVUs: 2,
    stages: [
      { duration: '15s', target: 2 },
      { duration: '5s', target: 150 },   // login spike
      { duration: '1m30s', target: 150 },
      { duration: '15s', target: 2 },
    ],
  },
};

export const options = {
  scenarios: {
    auth: SCENARIOS[scenarioName] || SCENARIOS.normal,
  },
  thresholds: {
    ...DEFAULT_THRESHOLDS,
    login_errors: ['rate<0.1'],
    login_latency: ['p(95)<400', 'p(99)<1500'],
    refresh_latency: ['p(95)<300', 'p(99)<1000'],
    health_latency: ['p(95)<50', 'p(99)<100'],
    readiness_latency: ['p(95)<200', 'p(99)<500'],
  },
};

export default function () {
  const r = Math.random();

  if (r < 0.60) {
    // --- 60%: Login as random customer ---
    const idx = Math.floor(Math.random() * CUSTOMER_COUNT) + 1;
    const email = `loadtest-customer-${idx}@ecommerce.test`;

    const start = Date.now();
    const res = http.post(`${BASE_URL}/api/v1/auth/login`, JSON.stringify({
      email: email,
      password: 'LoadTest1234!',
    }), {
      headers: { 'Content-Type': 'application/json' },
      tags: { name: 'auth/login' },
    });
    loginLatency.add(Date.now() - start);

    const passed = check(res, {
      'login status 200': (r) => r.status === 200,
      'login has tokens': (r) => {
        const body = r.json('data') || r.json();
        return body && body.tokens && body.tokens.access_token && body.tokens.refresh_token;
      },
    });
    loginErrorRate.add(!passed);
    if (passed) {
      loginSuccesses.add(1);
      const body = res.json('data') || res.json();
      if (body.tokens.refresh_token) {
        refreshTokens.push(body.tokens.refresh_token);
        if (refreshTokens.length > 5) refreshTokens.shift(); // keep bounded
      }
    }
    sleep(Math.random() * 0.3 + 0.1);

  } else if (r < 0.75) {
    // --- 15%: Refresh token ---
    if (refreshTokens.length > 0) {
      const rt = refreshTokens[Math.floor(Math.random() * refreshTokens.length)];
      const start = Date.now();
      const res = http.post(`${BASE_URL}/api/v1/auth/refresh`, JSON.stringify({
        refresh_token: rt,
      }), {
        headers: { 'Content-Type': 'application/json' },
        tags: { name: 'auth/refresh' },
      });
      refreshLatency.add(Date.now() - start);
    }
    sleep(Math.random() * 0.2 + 0.1);

  } else if (r < 0.90) {
    // --- 15%: Health check ---
    const start = Date.now();
    http.get(`${BASE_URL}/health`, { tags: { name: 'health' } });
    healthLatency.add(Date.now() - start);
    sleep(0.1);

  } else {
    // --- 10%: Readiness check (hits MongoDB + NATS) ---
    const start = Date.now();
    http.get(`${BASE_URL}/ready`, { tags: { name: 'ready' } });
    readinessLatency.add(Date.now() - start);
    sleep(0.2);
  }
}

export function handleSummary(data) {
  return {
    'stdout': `
=== Auth & Health Load Test (${scenarioName}) ===

Login Latency:
  p50: ${data.metrics.login_latency?.values?.p50 || 'N/A'} ms
  p95: ${data.metrics.login_latency?.values?.p95 || 'N/A'} ms
  p99: ${data.metrics.login_latency?.values?.p99 || 'N/A'} ms

Refresh Latency:
  p50: ${data.metrics.refresh_latency?.values?.p50 || 'N/A'} ms
  p95: ${data.metrics.refresh_latency?.values?.p95 || 'N/A'} ms

Health Latency:
  p50: ${data.metrics.health_latency?.values?.p50 || 'N/A'} ms
  p95: ${data.metrics.health_latency?.values?.p95 || 'N/A'} ms

Readiness Latency (MongoDB + NATS):
  p50: ${data.metrics.readiness_latency?.values?.p50 || 'N/A'} ms
  p95: ${data.metrics.readiness_latency?.values?.p95 || 'N/A'} ms

Login Error Rate: ${((data.metrics.login_errors?.values?.rate || 0) * 100).toFixed(2)}%
Successful Logins: ${data.metrics.login_successes?.values?.count || 0}

Total Requests: ${data.metrics.http_reqs?.values?.count || 0}
Throughput (RPS): ${data.metrics.http_reqs?.values?.rate || 'N/A'}
`,
  };
}

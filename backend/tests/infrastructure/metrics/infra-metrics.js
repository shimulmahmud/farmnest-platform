// infra-metrics.js — k6 load test: measure infrastructure metrics during load.
//
// This script runs alongside other tests and collects infrastructure-level metrics
// by querying the Prometheus endpoint (/metrics on port 9090) and the
// MongoDB/NATS stats available via readiness and admin endpoints.
//
// Usage: Run this in parallel with the main test scripts.
//   k6 run tests/infrastructure/metrics/infra-metrics.js &
//   k6 run tests/load/product-reads.js
//   wait

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Trend, Gauge } from 'k6/metrics';
import { BASE_URL } from '../../load/lib/config.js';

const PROMETHEUS_URL = __ENV.PROMETHEUS_URL || 'http://localhost:9090/metrics';

// --- Custom metrics for infrastructure ---
const mongoConnPoolGauge = new Gauge('infra_mongo_conn_pool');
const mongoConnCheckedOut = new Gauge('infra_mongo_conn_checked_out');
const mongoConnCheckoutMs = new Trend('infra_mongo_checkout_duration_ms');
const mongoCmdDuration = new Trend('infra_mongo_cmd_duration_ms');
const readinessCheckLatency = new Trend('infra_readiness_latency');

export const options = {
  vus: 1,
  iterations: 300,
  thresholds: {
    infra_readiness_latency: ['p(95)<500'],
  },
};

/**
 * Parse a Prometheus text-format metric line.
 * Returns an array of { name, value, labels } objects.
 */
function parsePrometheusMetrics(text) {
  const lines = text.split('\n');
  const results = [];
  for (const line of lines) {
    if (!line || line.startsWith('#')) continue;
    // Format: metric_name{labels} value
    const match = line.match(/^(\w+)(?:\{([^}]*)\})?\s+([\d.e+\-]+)/);
    if (match) {
      results.push({
        name: match[1],
        labels: match[2] || '',
        value: parseFloat(match[3]),
      });
    }
  }
  return results;
}

export default function () {
  // 1. Poll readiness endpoint (checks MongoDB + NATS connectivity)
  const start = Date.now();
  const readyRes = http.get(`${BASE_URL}/ready`, { tags: { name: 'infra/readiness' } });
  readinessCheckLatency.add(Date.now() - start);

  check(readyRes, {
    'readiness 200 or 503': (r) => r.status === 200 || r.status === 503,
  });

  if (readyRes.status === 200) {
    const body = readyRes.json();
    if (body.checks) {
      // Log infrastructure status periodically
      if (__ITER % 30 === 0) {
        console.log(`[${new Date().toISOString()}] Infrastructure status: ${JSON.stringify(body.checks)}`);
      }
    }
  }

  // 2. Poll Prometheus metrics endpoint (if available)
  if (__ITER % 10 === 0) {
    try {
      const promRes = http.get(PROMETHEUS_URL, { tags: { name: 'infra/prometheus' }, timeout: '3s' });
      if (promRes.status === 200) {
        const metrics = parsePrometheusMetrics(promRes.body);

        for (const m of metrics) {
          if (m.name.includes('mongo_pool') || m.name.includes('mongo_conn')) {
            if (m.labels.includes('max')) mongoConnPoolGauge.add(m.value);
            if (m.labels.includes('checked_out')) mongoConnCheckedOut.add(m.value);
          }
          if (m.name.includes('mongo_checkout')) {
            mongoConnCheckoutMs.add(m.value);
          }
          if (m.name.includes('mongo_cmd')) {
            mongoCmdDuration.add(m.value);
          }
        }

        if (__ITER % 30 === 0) {
          console.log(`[${new Date().toISOString()}] Prometheus metrics collected: ${metrics.length} entries`);
        }
      }
    } catch (e) {
      // Prometheus endpoint may not be available.
    }
  }

  sleep(1); // poll every second
}

export function handleSummary(data) {
  return {
    'stdout': `
=== Infrastructure Metrics ===

MongoDB Connection Pool (max observed): ${data.metrics.infra_mongo_conn_pool?.values?.max || 'N/A'}
MongoDB Connections Checked Out (max): ${data.metrics.infra_mongo_conn_checked_out?.values?.max || 'N/A'}
MongoDB Checkout Duration:
  avg: ${data.metrics.infra_mongo_checkout_duration_ms?.values?.avg || 'N/A'} ms
  p95: ${data.metrics.infra_mongo_checkout_duration_ms?.values?.p95 || 'N/A'} ms
MongoDB Command Duration:
  avg: ${data.metrics.infra_mongo_cmd_duration_ms?.values?.avg || 'N/A'} ms
  p95: ${data.metrics.infra_mongo_cmd_duration_ms?.values?.p95 || 'N/A'} ms

Readiness Check Latency:
  p50: ${data.metrics.infra_readiness_latency?.values?.p50 || 'N/A'} ms
  p95: ${data.metrics.infra_readiness_latency?.values?.p95 || 'N/A'} ms

Note: If Prometheus metrics show N/A, the /metrics endpoint may not be
exposed or the metric names differ. Check the observability configuration.
`,
  };
}

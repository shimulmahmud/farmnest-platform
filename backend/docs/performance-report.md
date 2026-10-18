# Performance Report — Ecommerce Backend

## 1. Test Configuration

### 1.1 Test Tool

- **k6** (https://k6.io/) — open-source load testing tool by Grafana Labs
- Version: latest stable (install via `k6 install` or `go install go.k6.io/k6/cmd/k6@latest`)

### 1.2 Test Infrastructure

| Component | Version | Configuration |
|-----------|---------|---------------|
| API Server | Go (Fiber v2) | 1 instance, `0.0.0.0:8080` |
| MongoDB | 7.x | Single node, `max_pool_size: 100`, `min_pool_size: 5` |
| NATS | 2.10+ | JetStream enabled, single node |
| Order Worker | Go | 1 instance |
| OS | Linux | k6 and server on same host or adjacent hosts |

### 1.3 Seed Data

Run `k6 run tests/load/seed.js` before any load test to create:

- 1 admin user (`loadtest-admin@ecommerce.test`)
- 50 customer users (`loadtest-customer-{1..50}@ecommerce.test`)
- 5 categories
- 20 products (each with 1 variant)
- 1 warehouse with stock entries

### 1.4 Scenarios

Each test script supports multiple scenarios selected via `K6_SCENARIO`:

| Scenario | VUs | Duration | Purpose |
|----------|-----|----------|---------|
| `normal` | 5–20 | 2–3 min | Baseline steady-state traffic |
| `high` | 30–100 | 5 min | Sustained high traffic |
| `spike` | 5→150–200 | ~3–4 min | Sudden traffic burst (staged ramp) |
| `checkout-spike` | 2→80 | ~4 min | Sudden checkout burst |
| NATS backlog | 200 (burst) | 30s burst + poll | Worker backlog stress test |

### 1.5 Test Scripts

| Script | Endpoint Group | Key Metrics |
|--------|---------------|-------------|
| `product-reads.js` | `GET /products`, `GET /products/:slug`, `GET /categories` | Product list/detail latency, error rate |
| `product-search.js` | `GET /products/search` (text, brand, price, tag, combined) | Search latency by filter type |
| `order-operations.js` | `POST /orders`, `GET /orders`, `GET /admin/orders` | Order create/list latency, error rate |
| `auth-mixed.js` | `POST /auth/login`, `POST /auth/refresh`, `/health`, `/ready` | Auth latency, health/readiness latency |
| `nats-worker-backlog.js` | Burst `POST /orders` + admin polling | Burst throughput, worker catch-up |
| `full-funnel.js` | All endpoints mixed by realistic ratios | End-to-end funnel performance |
| `infra-metrics.js` | `/ready`, `/metrics` (Prometheus) | MongoDB pool, conn checkout, cmd duration |

### 1.6 Metrics Collected

Per k6 test run, the following are measured:

- **RPS** — requests per second (throughput)
- **p50, p95, p99** — response time percentiles
- **Error rate** — percentage of failed requests (non-2xx or failed checks)
- **MongoDB latency** — connection checkout duration and command duration (via Prometheus metrics)
- **NATS latency** — readiness check includes NATS connectivity validation
- **CPU** — external monitoring (not collected by k6 directly; use `docker stats` or `pidstat`)
- **Memory** — external monitoring (same as CPU)
- **Consumer lag** — inferred from order status distribution after NATS backlog test
- **MongoDB connections** — pool size and checked-out count (via Prometheus)

---

## 2. Running the Tests

### 2.1 Prerequisites

```bash
# 1. Start infrastructure
make docker-up

# 2. Build and start API + Worker
go run ./cmd/api &
go run ./cmd/worker &

# 3. Seed test data
k6 run tests/load/seed.js
```

### 2.2 Run Individual Tests

```bash
# Product reads (normal traffic)
k6 run tests/load/product-reads.js

# Product reads (high traffic)
K6_SCENARIO=high k6 run tests/load/product-reads.js

# Product reads (traffic spike)
K6_SCENARIO=spike k6 run tests/load/product-reads.js

# Product search
K6_SCENARIO=high k6 run tests/load/product-search.js

# Order operations (checkout spike)
K6_SCENARIO=checkout-spike k6 run tests/load/order-operations.js

# NATS worker backlog
K6_BURST_SIZE=500 k6 run tests/load/nats-worker-backlog.js

# Full funnel
K6_SCENARIO=high k6 run tests/load/full-funnel.js

# Infrastructure metrics (run in parallel with any test)
k6 run tests/infrastructure/metrics/infra-metrics.js &
K6_SCENARIO=high k6 run tests/load/product-reads.js
wait
```

### 2.3 Export Results

```bash
# JSON output for post-processing
k6 run --out json=results/product-reads.json tests/load/product-reads.js

# InfluxDB for Grafana dashboards
k6 run --out influxdb=http://localhost:8086/k6 tests/load/product-reads.js

# Custom summary file
K6_SUMMARY_OUT=results/summary.json k6 run tests/load/product-reads.js
```

### 2.4 External Monitoring (CPU/Memory)

k6 does not collect server-side CPU/memory. Use one of:

```bash
# Docker container stats
docker stats --no-stream

# Process-level stats (install sysstat)
pidstat -p $(pgrep -f "cmd/api") 1 60

# Go pprof (built into the binary)
curl http://localhost:8080/debug/pprof/profile?seconds=30 > cpu.prof
go tool pprof -http=:8081 cpu.prof
```

---

## 3. Test Results

> **IMPORTANT**: This section must be populated with actual measurements.
> Do not claim performance without running the tests.
> The tables below are templates — replace placeholder values with real data.

### 3.1 Product Reads (Normal Traffic)

| Metric | Value |
|--------|-------|
| VUs | 10 |
| Duration | 2 min |
| Total Requests | _(from k6 output)_ |
| Throughput (RPS) | _(from k6 output)_ |
| Error Rate | _(from k6 output)_ |

| Endpoint | p50 (ms) | p95 (ms) | p99 (ms) |
|----------|----------|----------|----------|
| GET /products (list) | _ | _ | _ |
| GET /products/:slug (detail) | _ | _ | _ |
| GET /categories | _ | _ | _ |

### 3.2 Product Reads (High Traffic)

| Metric | Value |
|--------|-------|
| VUs | 100 |
| Duration | 5 min |
| Total Requests | _ |
| Throughput (RPS) | _ |
| Error Rate | _ |

| Endpoint | p50 (ms) | p95 (ms) | p99 (ms) |
|----------|----------|----------|----------|
| GET /products (list) | _ | _ | _ |
| GET /products/:slug (detail) | _ | _ | _ |
| GET /categories | _ | _ | _ |

### 3.3 Product Search (High Traffic)

| Metric | Value |
|--------|-------|
| VUs | 80 |
| Duration | 5 min |
| Total Requests | _ |
| Throughput (RPS) | _ |
| Error Rate | _ |

| Search Type | p50 (ms) | p95 (ms) | p99 (ms) |
|-------------|----------|----------|----------|
| Text search | _ | _ | _ |
| Brand filter | _ | _ | _ |
| Price range | _ | _ | _ |
| Tag search | _ | _ | _ |
| Combined | _ | _ | _ |

### 3.4 Traffic Spike (Product Reads)

| Stage | VUs | Duration | RPS | p95 (ms) | Error Rate |
|-------|-----|----------|-----|----------|------------|
| Baseline | 5 | 30s | _ | _ | _ |
| Spike | 200 | 2min | _ | _ | _ |
| Recovery | 5 | 30s | _ | _ | _ |

### 3.5 Order Operations (Checkout Spike)

| Metric | Value |
|--------|-------|
| Peak VUs | 80 |
| Duration | ~4 min |
| Orders Created | _ |
| Order Create p50 | _ ms |
| Order Create p95 | _ ms |
| Order Create p99 | _ ms |
| Error Rate (create) | _ |
| Error Rate (reads) | _ |

### 3.6 NATS Worker Backlog

| Metric | Value |
|--------|-------|
| Burst VUs | _ (default 200) |
| Burst Duration | 30s |
| Orders Created | _ |
| Burst p50 | _ ms |
| Burst p95 | _ ms |
| Orders reaching PAID | _ (query DB post-test) |
| Orders still PENDING | _ (query DB post-test) |
| Worker catch-up time | _ (estimate from timestamps) |
| Readiness check p95 (during backlog) | _ ms |

### 3.7 Full Funnel (High Traffic)

| Metric | Value |
|--------|-------|
| VUs | 100 |
| Duration | 5 min |
| Total RPS | _ |
| Overall Error Rate | _ |

| Flow Step | p50 (ms) | p95 (ms) | p99 (ms) |
|-----------|----------|----------|----------|
| Browse (list) | _ | _ | _ |
| Browse (detail) | _ | _ | _ |
| Search | _ | _ | _ |
| Auth (login) | _ | _ | _ |
| Checkout (create order) | _ | _ | _ |
| My Orders | _ | _ | _ |

### 3.8 Infrastructure Metrics

| Metric | Value |
|--------|-------|
| MongoDB connection pool (max) | _ |
| MongoDB connections checked out (max) | _ |
| MongoDB checkout duration avg | _ ms |
| MongoDB checkout duration p95 | _ ms |
| MongoDB command duration avg | _ ms |
| Readiness check p95 | _ ms |
| NATS status during peak load | _ (available/degraded) |

### 3.9 Resource Usage (External Monitoring)

| Metric | Idle | Normal Load | High Load | Spike |
|--------|------|-------------|-----------|-------|
| API CPU (%) | _ | _ | _ | _ |
| API Memory (MB) | _ | _ | _ | _ |
| Worker CPU (%) | _ | _ | _ | _ |
| Worker Memory (MB) | _ | _ | _ | _ |
| MongoDB CPU (%) | _ | _ | _ | _ |
| MongoDB Memory (MB) | _ | _ | _ | _ |
| NATS CPU (%) | _ | _ | _ | _ |
| NATS Memory (MB) | _ | _ | _ | _ |

---

## 4. Bottleneck Analysis

> **IMPORTANT**: Fill this section after analyzing real test results.

### 4.1 Identified Bottlenecks

_(Describe each bottleneck found during testing. Examples:)_

- **MongoDB query latency under high load**: _(if p95 exceeds threshold, investigate slow queries, missing indexes)_
- **Connection pool exhaustion**: _(if checked-out connections approach max_pool_size)_
- **NATS consumer lag**: _(if orders remain PENDING after burst, worker is the bottleneck)_
- **Auth rate limiter**: _(login endpoint has 10 req/min per IP limit — may cause false errors at high VUs)_
- **JWT signing CPU**: _(bcrypt/password hashing on login is CPU-bound)_

### 4.2 Latency Breakdown

_(For the most critical path — e.g., product list — break down where time is spent:)_

| Phase | Normal (ms) | High Load (ms) | Notes |
|-------|------------|-----------------|-------|
| Request parsing | _ | _ | _ |
| MongoDB query | _ | _ | _ |
| Response serialization | _ | _ | _ |
| Total | _ | _ | _ |

---

## 5. Recommendations

> **IMPORTANT**: Base recommendations on measured data, not assumptions.

### 5.1 Immediate (Low Effort)

- _(e.g., Add MongoDB compound indexes for search queries)_
- _(e.g., Increase connection pool if checkout frequently approaches max)_
- _(e.g., Tune auth rate limiter for load test IPs)_

### 5.2 Short-Term (Medium Effort)

- _(e.g., Implement response caching for product list/detail via NATS KV)_
- _(e.g., Add read replicas for MongoDB to distribute read load)_
- _(e.g., Batch outbox processing to reduce NATS message count)_

### 5.3 Long-Term (High Effort)

- _(e.g., Implement a proper checkout service with cart + idempotency)_
- _(e.g., Add a CQRS read model for product search)_
- _(e.g., Horizontal API scaling behind a load balancer)_

---

## 6. Maximum Observed Sustainable Throughput

> **This is the key output of this report. Fill it with real measurements only.**

| Endpoint Group | Sustained RPS | Conditions |
|---------------|---------------|------------|
| Product reads (list + detail) | _ | p95 < 300ms, error rate < 5% |
| Product search | _ | p95 < 500ms, error rate < 5% |
| Order creation | _ | p95 < 800ms, error rate < 10% |
| Auth (login) | _ | p95 < 400ms, error rate < 10% |
| Full funnel (mixed) | _ | overall error rate < 5% |

**Maximum observed sustainable throughput (all endpoints mixed):** _ RPS

**Conditions:** _(e.g., single API instance, single MongoDB node, 100 max pool size)_

---

## Appendix A: Threshold Definitions

| Metric | Threshold | Rationale |
|--------|-----------|-----------|
| http_req_duration p95 | < 500ms | User-perceived "fast" response |
| http_req_duration p99 | < 2000ms | Tolerable worst-case |
| http_req_failed | < 5% | SLA-grade reliability for reads |
| Order create error rate | < 10% | Writes are more error-prone under load |
| Health endpoint p95 | < 50ms | Must be near-instant |
| Readiness endpoint p95 | < 200ms | Includes dependency checks |

## Appendix B: Test Execution Commands (Quick Reference)

```bash
# Full test suite (run each and collect results)
make loadtest-seed
make loadtest-product-reads
make loadtest-product-reads-spike
make loadtest-search
make loadtest-orders
make loadtest-checkout-spike
make loadtest-nats-backlog
make loadtest-full-funnel
make loadtest-auth
```

## Appendix C: Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `BASE_URL` | `http://localhost:8080` | API base URL |
| `K6_SCENARIO` | `normal` | Scenario to run (normal/high/spike/checkout-spike) |
| `K6_BURST_SIZE` | `200` | Concurrent VUs for NATS backlog burst |
| `K6_SUMMARY_OUT` | _(none)_ | Path for JSON summary output |
| `PROMETHEUS_URL` | `http://localhost:9090/metrics` | Prometheus metrics endpoint |
| `SEED_PRODUCT_ID` | _(mock)_ | Override product ID for order tests |
| `SEED_VARIANT_ID` | _(mock)_ | Override variant ID for order tests |
| `SEED_SKU` | _(mock)_ | Override SKU for order tests |

# Final Architecture Review

**Date:** 2026-08-22
**Scope:** Full system architecture, code, and operational readiness
**Rule:** No code modifications. Findings only.

---

## 1. Architecture Overview

The ecommerce backend is a Go 1.22 monorepo using the Fiber v2 HTTP framework, MongoDB 7.x for persistence, and NATS 2.10+ JetStream for event-driven messaging. It follows a domain-driven design with 13 business modules (auth, users, products, categories, inventory, orders, payments, reviews, shipping, wishlist, marketing, analytics, admin), each structured as application/domain/infrastructure/transport layers. An outbox pattern ensures atomic event publishing. The system runs as two stateless binaries (API + Worker) deployed via Kubernetes with CI/CD, load testing, and a documented DR plan.

---

## 2. Answers to Review Questions

### Q1: Is MongoDB always the source of truth?

**Yes.**

All persistent business data (orders, products, users, inventory, payments, shipments, reviews, wishlists, promotions, campaigns, audit logs, outbox events) resides in MongoDB across 24 collections. NATS KV is used only as a read-through cache for products and categories. The authoritative data is always in MongoDB, and KV entries are populated from MongoDB reads and invalidated on writes. The outbox pattern writes events to MongoDB before publishing to NATS, making MongoDB the source of truth for the event stream as well.

---

### Q2: Is NATS KV misused as a database?

**No.**

Only 2 of 7 configured KV buckets are actually used: `PRODUCT_CACHE` and `CATEGORY_CACHE`. Both implement a read-through caching pattern where the service checks KV first, falls back to MongoDB on miss, and populates the cache. History is set to 1. MongoDB remains the authoritative data source in all cases.

However, 5 KV buckets (`SESSION_STATE`, `IDEMPOTENCY`, `RATE_LIMIT`, `CHECKOUT_STATE`, `DISTRIBUTED_LOCKS`) are defined in `pkg/config/config.go` and created at startup in `cmd/api/main.go` but never referenced by any application code. These consume NATS server resources for no benefit and should be removed.

---

### Q3: Are important events reliable?

**Mostly yes, with a gap in the outbox write.**

The outbox pattern (`pkg/outbox/`) provides at-least-once delivery: events are written to MongoDB's `outbox_events` collection, then a background worker polls, claims, and publishes them to NATS JetStream. Stale events (PUBLISHING state > 2 minutes) are recovered on startup. This prevents event loss when NATS is temporarily unavailable.

The gap is that the business write (e.g., inserting an order) and the outbox event insert are separate MongoDB operations, not wrapped in a shared session or transaction. The outbox insert uses fire-and-forget semantics: `_ = s.events.Publish(ctx, "order.created", ...)`. If the outbox insert fails (duplicate key, connection error), the business write succeeds but the event is lost with no indication to the caller.

---

### Q4: Can duplicate events corrupt state?

**No, but some are silently lost.**

All active JetStream consumers have idempotency defenses:
- **Order worker** (`PaymentCompletedHandler`, `PaymentFailedHandler`): state guard checks (e.g., `if order.Status != OrderPaymentPending { return nil }`) backed by atomic `FindOneAndUpdate`.
- **Notification workers**: DB lookup via `FindByEventIDAndChannel()` with unique index `(event_id, channel)`.
- **Marketing workers**: DB lookup via `ExistsByEventID()`.
- **Analytics worker**: DB lookup via `ExistsByEventID()`.

**Data loss concern:** When analytics or marketing consumers encounter a MongoDB store failure, they return `nil` (not an error), causing the message to be ACK'd. The event is consumed from JetStream but never stored. This means analytics data can be silently lost under MongoDB pressure.

---

### Q5: Can duplicate payment webhooks corrupt state?

**No.**

Three layers of defense exist:
1. **Application-level check:** `HasBeenProcessed` queries `webhook_events` collection before processing.
2. **Unique index:** `idx_webhook_id_unique` on `webhook_id` prevents concurrent duplicate inserts. `DuplicateKeyError` is caught gracefully.
3. **Payment status guard:** `UpdateStatus` uses `FindOneAndUpdate` with `filter: {status: expectedStatus}`, so only one state transition can succeed.

Additionally, `isValidWebhookTransition` prevents invalid state jumps (e.g., REFUNDED to PAID). The `constant-time HMAC comparison (`hmac.Equal`) prevents timing attacks on webhook verification. Integration tests validate this with 100 concurrent identical webhook deliveries.

---

### Q6: Can inventory oversell?

**No, but inventory accuracy can drift.**

The `ReserveStock` operation in `internal/inventory/infrastructure/repository.go` uses an atomic `FindOneAndUpdate` with `$expr`: `{$gte: [{$subtract: ["$total_stock", "$reserved_stock"]}, quantity]}`. The availability check and the `reserved_stock` increment happen in a single atomic operation. MongoDB's document-level locking serializes concurrent reservations for the same SKU+warehouse. This is the gold-standard pattern for preventing overselling and is validated by a 100-goroutine integration test.

**Drift risk:** The `Reserve` method performs two separate operations: (1) atomic `ReserveStock` (increments `reserved_stock`), then (2) `CreateReservation` (inserts a reservation document). If step 2 fails after step 1 succeeds, the rollback attempts to release the stock, but if the process crashes before the rollback, `reserved_stock` is permanently inflated. Similarly, `ReleaseByReference` can mark a reservation as RELEASED but fail to release the actual stock. And `ExpireReservations` has a TOCTOU race between `Find` and `UpdateMany` that could attempt to release stock for already-consumed reservations.

These drift bugs do not cause overselling (the `$expr` guard prevents it), but they cause available stock to be under-reported over time. A periodic reconciliation job comparing `SUM(active reservations)` vs `stock.reserved_stock` is recommended.

---

### Q7: Can checkout create duplicate orders?

**Yes.**

There is no idempotency mechanism on order creation. The `CreateOrder` method in `internal/orders/application/service.go` generates a new UUID and order number on every call. No `idempotency_key` field exists on `CreateOrderInput`. The `IDEMPOTENCY` NATS KV bucket is configured but never used.

If a client sends the same checkout request twice (due to a network timeout, browser retry, or load balancer retry), two distinct orders will be created with different UUIDs, different order numbers, and both reserving inventory.

This is the single most critical finding in this review. A production checkout flow must accept a client-provided idempotency key (e.g., `Idempotency-Key` header) and atomically check-create the order to prevent duplicates.

---

### Q8: Are APIs stateless?

**Yes.**

The API server holds no in-memory state between requests. No in-memory session store, no local cache, no in-memory rate limiter holding business state. All state is external:
- Authentication: JWT tokens (stateless, no server-side session).
- Caching: NATS KV (external) for product/category caches.
- Data: MongoDB (external).
- Events: NATS JetStream (external).
- Rate limiting: Fiber middleware uses in-process state, but this is acceptable for rate limiting — it resets on restart without data loss.

The only `var` maps in the codebase are read-only constants (e.g., `admin/domain/rbac.go:29`). The API is safe to run behind a load balancer with multiple replicas. Any pod can handle any request.

---

### Q9: Can APIs scale horizontally?

**Yes.**

The API is stateless (Q8) and has no single points of contention. Horizontal scaling is supported by:
- Kubernetes Deployment with 3 replicas, HPA (3-20 pods at 80% CPU).
- Rolling updates with `maxSurge: 1, maxUnavailable: 0` for zero-downtime deployments.
- Topology spread constraints (zone + hostname) and pod anti-affinity for HA.
- PodDisruptionBudget (minAvailable: 2).

**Limitation:** The product/category NATS KV caches are not synchronized across pods. When a product is updated, the cache is invalidated only in the pod that handled the write. Other pods serve stale data until their cache entries are independently refreshed on the next cache-miss read. This is acceptable for short TTL scenarios but should be documented.

---

### Q10: Can workers scale horizontally?

**No.**

JetStream consumers are durable with unique names, but **no consumers use queue groups**. Each consumer has a unique name (e.g., `order-worker-payment-completed`). In JetStream, a consumer with a unique name is assigned to exactly one connection. If you deploy 3 worker replicas, only one replica's consumer will be active for each consumer name. The other two replicas' consumers will be inactive.

To enable horizontal worker scaling, each consumer group should use a shared queue group name (same `QueueGroup` value across replicas) so JetStream distributes messages among all replicas.

---

### Q11: Are MongoDB indexes sufficient?

**Mostly yes, with gaps.**

~75 indexes across 24 collections provide good coverage. Unique indexes prevent duplicates on business keys (email, slug, SKU, order number, webhook ID). Compound indexes support common query patterns (customer + created_at, status + created_at). Text indexes with weights support product search. TTL indexes auto-expire analytics (30d), notifications (24h), webhook events (7d), and audit logs (90d).

**Gaps identified:**
- No index on `orders.items.product_id` — the `OrderVerifiedPurchaseChecker` queries `{customer_id, status: "DELIVERED", items.product_id}`. This performs a collection scan.
- No compound index `{provider: 1, provider_reference: 1}` on `payments` — `FindByProviderRef` queries both fields but only `{provider_reference: 1}` is indexed.
- Duplicate audit index definitions — `pkg/audit/mongodb.go` and `internal/admin/infrastructure/repository.go` define identical indexes. The admin version is dead code.
- Redundant `idx_inventory_sku` — the compound `idx_inventory_sku_warehouse_unique` on `{sku: 1, warehouse_id: 1}` already serves as a prefix index for `{sku: 1}` queries.
- No MongoDB schema validation (`$jsonSchema`) on any collection. All validation is application-layer only.
- No TTL index on `refresh_tokens.expires_at` or `verification_tokens.expires_at` — stale tokens accumulate forever.
- Index creation is synchronous at startup (`CreateMany` without `Background: true`). For large collections in production, this could delay startup.

---

### Q12: Are MongoDB pools configured correctly?

**Adequate but minimal.**

Pool configuration: `MaxPoolSize=100`, `MinPoolSize=5`, `ConnectTimeout=10s`. Both API and Worker use the same defaults.

**Missing configurations:**
- No `ServerSelectionTimeout` — if MongoDB is unavailable, the driver retries for 30s (default) before failing, which exceeds the API's 15-second write timeout.
- No `SocketTimeout` — a slow query can hold a connection indefinitely.
- No `MaxConnecting` — without this, a traffic spike causes a connection storm.
- No explicit `WriteConcern` — uses driver default (`w:1` for standalone, `w:majority` for replica sets). The outbox and orders collections would benefit from explicit `WriteConcern: majority`.
- No explicit `ReadConcern` or `ReadPreference`.
- Same pool size for both API and Worker. Worker does fewer concurrent DB operations and could use a smaller pool (20-30) to reduce total connection count under HPA.

---

### Q13: Are NATS consumers durable?

**Yes.**

All JetStream consumers are created with `Durable: opts.ConsumerName` in `pkg/eventstore/subscriber.go`. Consumer state (position, pending messages, ack state) is persisted by the NATS server. If a consumer disconnects and reconnects, it resumes from its last acknowledged position.

Additional durability features:
- Ack policy: `AckExplicit` — messages must be explicitly acknowledged.
- Max deliveries: 10 with exponential backoff (1s, 2s, 4s, 8s, 16s, 32s, 64s, 128s, 256s, 5m).
- Per-message timeout: 30 seconds.

**Limitation:** All 7 streams use `Replicas: 1` — no data replication within the NATS cluster. A NATS server crash loses unprocessed messages. Production requires at least `Replicas: 3`.

---

### Q14: Is backpressure handled?

**No.**

There is no explicit backpressure handling anywhere in the system:
- No `MaxAckPending` configured on JetStream consumers — NATS defaults to 1024, but this is not tuned or observable.
- No rate limiting on consumer message processing.
- No batch processing with concurrency limits.
- No circuit breaker pattern between consumers, the outbox worker, and NATS.
- The 30-second per-message context timeout causes a NAK and retry, not actual backpressure control.
- The outbox worker processes events sequentially; a single slow publish with 5-minute backoff blocks the entire batch of up to 50 events.

---

### Q15: Are failures retried correctly?

**Partially.**

**What works:**
- Outbox events: up to 10 retries with exponential backoff. Stale PUBLISHING events recovered on startup (2-minute threshold). Failed events marked FAILED with last error message.
- JetStream consumers: up to 10 redeliveries with exponential backoff (1s to 5m).
- MongoDB driver: automatic retry for network errors (driver default for replica sets).
- NATS connection: automatic reconnect with `ReconnectWait=2s`, `MaxReconnects=5`.

**What does not work:**
- Outbox publish calls use fire-and-forget (`_ = s.events.Publish(...)`). If the MongoDB outbox insert fails, the event is silently lost.
- No automated replay for outbox events in FAILED state (queryable but no remediation tooling).
- No retry for failed notification delivery (marked FAILED, no retry).
- The stale recovery threshold uses `created_at` instead of a dedicated `claimed_at` timestamp. A long-running first publish attempt (> 2 minutes from creation) could be incorrectly reset.

---

### Q16: Are failed events handled?

**Insufficiently.**

**JetStream messages** that exceed `MaxDeliver` (10) are effectively dropped. They remain in the stream until retention expires, but there is no dead-letter queue consumer, no alerting, and no manual recovery mechanism. An operator would need to manually query the JetStream stream for unprocessed messages.

**Outbox events** that exceed max attempts are marked `FAILED` in MongoDB with the last error message. The `CountByStatus` method enables querying failed events, but there is no dashboard, no alerting, and no automated remediation.

**Notifications** that fail delivery are marked `FAILED` in MongoDB. Same gap: queryable but not observable or automatable.

---

### Q17: Are secrets protected?

**Yes at the infrastructure level, with gaps at the application and config level.**

**Infrastructure protections:**
- Secrets stored in Kubernetes `Secret` objects, not in ConfigMaps or images.
- CI/CD: TruffleHog secret scanning (full git history, verified-only, `--fail`). SAST via gosec.
- Docker images: multi-stage builds, no secrets baked in. Distroless runtime.
- K8s: `automountServiceAccountToken: false`, NetworkPolicies default-deny all, `drop: ALL` capabilities, `readOnlyRootFilesystem: true`, `seccompProfile: RuntimeDefault`.
- DR plan documents secret backup with age encryption and rotation procedures.
- JWT secret minimum 32 characters enforced at startup.
- Webhook secrets loaded from `WEBHOOK_SECRET_<PROVIDER>` env vars.

**Gaps:**
- `configs/config.yaml:59` contains `secret: "change-me-in-production"` — a hardcoded JWT signing secret in version control. If copied to production, any attacker could forge JWTs.
- `k8s/secret.yaml` is listed in `kustomization.yaml` resources with placeholder `CHANGE-ME` values. Running `kustomize build | kubectl apply` would deploy known secrets.
- `configs/config.yaml:46,55` has wide-open CORS (`cors_allowed_origins: ["*"]`). The K8s ConfigMap correctly restricts this, but the YAML template is dangerous if copied for local/dev use.

---

### Q18: Are sensitive values excluded from logs?

**Partially.**

| Component | Sanitization | Status |
|---|---|---|
| Audit entries (`pkg/audit`) | Regex-based redaction of password, secret, token, api_key, private_key, card_number, cvv, Bearer tokens | Adequate |
| Trace spans (`pkg/telemetry`) | Allowlist of safe attribute keys + error message sanitization for secret patterns | Good |
| Application logger (`pkg/logger`) | **None** — raw zap.SugaredLogger wrapper with no field filtering | **Gap** |
| HTTP responses | Internal errors never leaked (`json:"-"` on `Err` field, generic fallback messages) | Good |

The logger gap is the primary risk. A developer adding `logger.Infow("login", "password", pw)` or `logger.Debugw("request", "headers", c.Request().Header.String())` would leak secrets into stdout/JSON logs.

---

### Q19: Are critical workflows concurrency-tested?

**Partially.**

**Well-tested (integration tests with testcontainers):**
- Inventory reservation: 100 concurrent goroutines, exact limit enforcement verified.
- Webhook idempotency: 100 concurrent identical webhook deliveries, exactly 1 state transition.
- Order creation: 50 concurrent requests, all unique IDs/order numbers.
- Coupon redemption: usage_limit enforcement under concurrency.
- Full E2E flow test: register -> create product -> create order -> payment -> webhook -> verify.

**Not tested:**
- Order creation under concurrent identical requests (no idempotency mechanism to test).
- Product update conflicts (two admins updating the same product — uses plain `$set`, no optimistic locking).
- Inventory drift scenarios (reserve succeeds but CreateReservation fails).
- ExpireReservations TOCTOU race.
- NATS KV cache consistency across multiple pods.

---

### Q20: Is the system load-tested?

**Test suites exist but have not been executed.**

8 k6 test scripts cover: product reads, product search, order operations, auth mixed, NATS worker backlog, full funnel, and infrastructure metrics. Scenarios include normal traffic, high traffic, and spike patterns. The performance report (`docs/performance-report.md`) has placeholder values with no actual measurements.

The CI pipeline runs unit tests with `-short` flag and a race detector. Load tests are defined as Make targets but require a running environment with Docker, k6, and the API server.

---

## 3. Findings by Severity

### CRITICAL

| # | Finding | Component | Evidence |
|---|---|---|---|
| C1 | **No order creation idempotency** — duplicate orders can be created on client retry. No idempotency key is accepted. | `internal/orders/application/service.go` | `CreateOrder` generates a new UUID on every call. No `Idempotency-Key` header processing. |
| C2 | **Workers cannot scale horizontally** — JetStream consumers have unique durable names but no queue groups. Multiple worker replicas do not share message load. | `pkg/eventstore/subscriber.go:72`, `cmd/worker/main.go` | All 13 consumers set `Durable: opts.ConsumerName` with unique names. Zero `QueueGroup` references in the codebase. |
| C3 | **No liveness/startup/readiness probes on worker** — a hung worker (deadlocked on MongoDB write, stuck in backoff loop) will never be restarted by Kubernetes. | `k8s/worker-deployment.yaml:57-63` | Comment acknowledges the gap. No probe configuration. |
| C4 | **No automated rollback in deploy workflow** — a bad deployment that passes readiness checks but has runtime bugs requires manual intervention. No post-deploy smoke test against the live endpoint. | `.github/workflows/deploy.yml` | No `rollout undo` step. Smoke test is just re-running unit tests, not hitting the deployed service. |
| C5 | **Deploy workflow smoke-test gate is bypassed** — `always()` condition means the deploy job runs even if smoke-test fails. | `.github/workflows/deploy.yml:99` | `if: always() && needs.prepare.result == 'success'` — smoke test failure does not block deployment. |
| C6 | **Hardcoded JWT secret in checked-in config** — `configs/config.yaml:59` contains `secret: "change-me-in-production"`. If deployed as-is, attackers can forge any JWT. | `configs/config.yaml:59` | Plaintext secret in version control. |
| C7 | **Placeholder secret.yaml in kustomization resources** — `k8s/secret.yaml` with `CHANGE-ME` values is listed in `kustomization.yaml:22`. `kustomize build | kubectl apply` deploys known secrets. | `k8s/kustomization.yaml:22`, `k8s/secret.yaml:24-32` | Placeholder values deployed to cluster. |

### HIGH

| # | Finding | Component | Evidence |
|---|---|---|---|
| H1 | **No dead letter queue** — JetStream messages exceeding MaxDeliver (10) are silently dropped. No DLQ consumer, no alerting, no recovery mechanism. | `pkg/eventstore/subscriber.go:84,209` | After max deliveries, message is terminated. No DLQ stream configured. |
| H2 | **Analytics/marketing consumers ACK on MongoDB store failure** — events are consumed from JetStream but never stored. Silent data loss. | `internal/analytics/worker/processor.go:107`, `internal/marketing/worker/processor.go:129` | Handler returns nil on store failure, triggering ACK. |
| H3 | **Outbox is not truly transactional** — business writes and outbox inserts are separate MongoDB operations. Fire-and-forget on outbox publish. | `pkg/outbox/adapters.go`, all application services | `_ = s.events.Publish(ctx, "order.created", ...)` swallows errors. |
| H4 | **Trace exporter is stdout only** — no OTLP exporter. Traces visible only in individual pod logs, not aggregated in a tracing backend. | `pkg/observability/observability.go:54` | `stdouttrace.New()` with comment "replace with OTLP in production." Not configurable. |
| H5 | **Rate limiting is in-memory and auth-only** — Fiber limiter stores counters in-process. With 3 API replicas, an attacker gets 30 req/min. The `RATE_LIMIT` KV bucket is unused. | `internal/auth/transport/routes.go:10-16`, `cmd/api/main.go:352-364` | Hardcoded `Max: 10` on auth routes. No global rate limiter. |
| H6 | **HS256 JWT signing** — symmetric algorithm requires all token-validating services to share the secret. Limits microservices evolution path. | `internal/auth/infrastructure/jwt.go:123` | `jwt.SigningMethodHS256`. Algorithm enforced but not configurable. |
| H7 | **API NetworkPolicy ingress allows all namespaces** — empty `namespaceSelector` lets any pod in any namespace bypass the Ingress controller. | `k8s/networkpolicy.yaml:58` | `namespaceSelector: {}` matches all namespaces. |
| H8 | **CI pipeline uses unpinned tool versions** — `gosec@latest`, `govulncheck@latest`, `trufflehog@main` break reproducibility and create supply chain risk. | `.github/workflows/ci.yml:212,241,261` | `@latest` and `@main` tags used for security tools. |
| H9 | **Static KUBECONFIG in deploy workflow** — base64 secret without rotation. Should use OIDC federation. | `.github/workflows/deploy.yml:111` | `echo "$KUBECONFIG" | base64 -d > kubeconfig`. No rotation. |
| H10 | **kustomization.yaml references all-comment file** — `scheduler-deployment.yaml` is entirely commented out. `kustomize build` will fail. | `k8s/kustomization.yaml:25` | File in resources list contains only comments (invalid YAML). |
| H11 | **imagePullPolicy: IfNotPresent with :latest tag** — cached images on nodes will never be updated. | `k8s/api-deployment.yaml:99`, `k8s/worker-deployment.yaml:68` | `imagePullPolicy: IfNotPresent`. |

### MEDIUM

| # | Finding | Component | Evidence |
|---|---|---|---|
| M1 | **Logger has no field-level sanitization** — sensitive values logged via `logger.Infow`/`CtxErrorw` appear verbatim. | `pkg/logger/logger.go` | Raw zap.SugaredLogger wrapper. No sanitization function. |
| M2 | **Five unused NATS KV buckets** created on every startup, consuming NATS resources. | `pkg/config/config.go:118-123`, `cmd/api/main.go:191-197` | `SESSION_STATE`, `IDEMPOTENCY`, `RATE_LIMIT`, `CHECKOUT_STATE`, `DISTRIBUTED_LOCKS` — zero application references. |
| M3 | **Outbox worker processes events sequentially** — a single slow publish with 5-minute backoff blocks up to 50 events. | `pkg/outbox/worker.go:190-209` | Events processed one-at-a-time in a loop. Backoff uses `time.Sleep` inside the poll loop. |
| M4 | **No failed event remediation tooling** — outbox FAILED events and failed notifications are queryable but have no dashboard, alerting, or automated replay. | `pkg/outbox/repository.go` (CountByStatus) | No remediation code or automation exists. |
| M5 | **Missing MongoDB pool options** — no `ServerSelectionTimeout`, `SocketTimeout`, `MaxConnecting`, `WriteConcern`, `ReadConcern`. | `pkg/mongodb/mongodb.go` | Only `MaxPoolSize`, `MinPoolSize`, `ConnectTimeout` are set. |
| M6 | **Missing MongoDB indexes** — no index on `orders.items.product_id`, no compound `{provider, provider_reference}` on `payments`. | `internal/orders/infrastructure/repository.go`, `internal/payments/infrastructure/repository.go` | Queries perform collection scans. |
| M7 | **No load test results** — k6 test scripts exist but have never been executed. Performance report has placeholder values only. | `tests/load/`, `docs/performance-report.md` | No actual measurements exist. |
| M8 | **No TTL index on refresh_tokens or verification_tokens** — stale tokens accumulate forever. | `internal/auth/infrastructure/repository.go` | `expires_at` indexes exist but lack `SetExpireAfterSeconds(0)`. |
| M9 | **Inventory reserve/createReservation non-atomic** — if CreateReservation fails after ReserveStock succeeds, reserved_stock drifts up permanently. | `internal/inventory/application/service.go:182-195` | Two-step operation with rollback that can be skipped on crash. |
| M10 | **Inventory ReleaseByReference non-atomic** — reservation marked RELEASED but ReleaseStock can fail, leaving reserved_stock inflated. | `internal/inventory/application/service.go:266-272` | Loop with `continue` on error silently skips stock release. |
| M11 | **ExpireReservations TOCTOU** — Find then UpdateMany race. May try to release stock for already-consumed reservations. | `internal/inventory/infrastructure/repository.go:293-320` | `Find` and `UpdateMany` are separate operations. |
| M12 | **Duplicate reservation TOCTOU** — `GetActiveReservation` check-then-act without unique index backup. | `internal/inventory/application/service.go:166-172` | No partial unique index on `(sku, reference)` where `status = "ACTIVE"`. |
| M13 | **No WriteConcern: majority on critical collections** — outbox, orders, payments use default w:1. Primary crash before replication loses data. | All repositories | No explicit WriteConcern configured at client or operation level. |
| M14 | **KV cache ignores request context** — `context.Background()` used for all KV operations. Request cancellation does not abort cache reads. | `internal/products/infrastructure/cache.go:24,42,50`, `internal/categories/infrastructure/cache.go` | `_ context.Context` parameter ignored in Get/Set/Delete. |
| M15 | **Swallowed errors on security operations** — `MarkVerificationTokenUsed` and `RevokeAllUserRefreshTokens` errors silently discarded after password reset. | `internal/auth/application/service.go:236-240,287-294` | `_ = err` on security-critical operations. |
| M16 | **PaymentReference not persisted** — set in-memory on order status update but never written to MongoDB. | `internal/orders/application/service.go:149-154` | `updated.PaymentReference = input.PaymentReference` modifies returned object only. |
| M17 | **Makefile lint target missing golangci-lint** — `make ci-check` does not catch all linter errors that CI would. | `Makefile:75` | `lint` target runs `fmt vet` but not `golangci-lint`. |
| M18 | **CI grants packages:write to fork PRs** — potential supply chain risk. | `.github/workflows/ci.yml:43` | Top-level `packages: write` permission. |
| M19 | **No MongoDB schema validation** — corrupted documents from bugs or direct DB access are not caught at the database level. | All collections | Zero `$jsonSchema` validators on any collection. |
| M20 | **Dockerfiles hardcoded GOARCH=amd64** — will not build on ARM nodes (AWS Graviton, Apple Silicon K8s). | `Dockerfile:17`, `Dockerfile.api:34`, `Dockerfile.worker:31` | `GOARCH=amd64` hardcoded instead of using `TARGETARCH`. |

### LOW

| # | Finding | Component | Evidence |
|---|---|---|---|
| L1 | **Duplicate audit_logs index definitions** — admin version is dead code. | `pkg/audit/mongodb.go`, `internal/admin/infrastructure/repository.go` | Identical index definitions. Admin version never called. |
| L2 | **Redundant `idx_inventory_sku`** — compound index already serves as prefix. | `internal/inventory/infrastructure/repository.go` | `{sku: 1}` redundant with `{sku: 1, warehouse_id: 1}`. |
| L3 | **Two separate analytics collections** — `analytics_data` and `analytics_events` with similar 30-day TTL. | `internal/analytics/`, `internal/marketing/` | Overlapping purpose may confuse operators. |
| L4 | **5 legacy analytics subjects in stream config** — no publishers, dead weight. | `pkg/eventstore/streams.go:101-105` | `product.viewed`, `cart.created`, etc. with no active publishers. |
| L5 | **3 legacy direct NATS publishers** — dead code with wrong subject format. | `internal/orders/infrastructure/event_publisher.go`, `internal/inventory/infrastructure/event_publisher.go`, `internal/payments/infrastructure/event_publisher.go` | Bare `conn.Publish()`. Not wired in main.go. |
| L6 | **9 published subjects have no active consumer** — events stored in streams but never processed. | Various outbox adapters | Products, inventory, shipping, payment.created/refunded — no subscriber. |
| L7 | **No per-domain metrics for reviews, wishlist, categories, shipping, auth.** | `pkg/telemetry/metrics.go` | Business metrics only cover orders, checkout, payments, inventory. |
| L8 | **No MongoDB connection pool metrics** — pool utilization not exposed to Prometheus. | `pkg/mongodb/mongodb.go` | No instrumentation of pool stats. |
| L9 | **InstrumentedCollection exists but is unused** — MongoDB tracing wrapper built but not wired into any repository. | `pkg/telemetry/mongowrap.go` | Defined but no repository uses it. |
| L10 | **No `Nats-Msg-Id` header** — no server-side JetStream dedup. | `pkg/eventstore/publisher.go` | All dedup is application-level after consumption. |
| L11 | **gosec disabled for test files** — test code can have security issues too. | `.golangci.yml:98` | `gosec` excluded for `_test.go`. |
| L12 | **Prometheus metrics endpoint has no auth** — plain HTTP, no TLS. | `pkg/observability/observability.go:99-108` | `http.ListenAndServe(addr, mux)`. |
| L13 | **Audit buffer drops not metered** — 256-entry buffer overflow drops entries silently. | `pkg/audit/mongodb.go:147-153` | Only `Warnw` log, no Prometheus counter. |
| L14 | **`RequestIDFromContext` always returns empty** — function body is `return ""`. | `pkg/telemetry/tracing.go:31-36` | Footgun for callers expecting a correlation ID. |
| L15 | **Integration tests lack build tags** — `go test ./...` attempts integration tests without Docker. | `tests/integration/` | No `//go:build integration` tags. |
| L16 | **Customers module has zero test coverage** — no unit, integration, or domain tests. | `internal/customers/` | No `_test.go` files. |
| L17 | **`mockReviewRepo` lacks mutex** — unsafe for concurrent tests unlike all other mocks. | `internal/reviews/` | All other mocks use `sync.Mutex`. |
| L18 | **GCR base image** — `gcr.io` is being deprecated in favor of Artifact Registry. | `Dockerfile.api:49` | `gcr.io/distroless/static-debian12:nonroot`. |

### INFORMATIONAL

| # | Finding | Component |
|---|---|---|
| I1 | Outbox cleanup deletes PUBLISHED events older than 24 hours. | `pkg/outbox/worker.go` |
| I2 | Webhook events auto-expire after 7 days, notifications after 24 hours, analytics after 30 days, audit logs after 90 days. | Various `EnsureIndexes` functions |
| I3 | Exponential backoff in outbox (1s to 5m) and JetStream consumers (1s to 5m) uses the same max delay. | `pkg/outbox/worker.go`, `pkg/eventstore/subscriber.go` |
| I4 | `scripts/fix_index.py` is a one-off tool that should be removed or documented. | `scripts/fix_index.py` |
| I5 | No `configMapGenerator` in kustomization — config changes don't trigger rolling updates automatically. | `k8s/kustomization.yaml` |
| I6 | No OTel collector egress rule in NetworkPolicy — blocked if OTel runs in a different namespace. | `k8s/networkpolicy.yaml` |
| I7 | No RBAC (Role/RoleBinding) defined — SAs have no K8s API permissions, which is correct for now. | `k8s/serviceaccount.yaml` |
| I8 | Worker PDB uses `minAvailable: 1` — consider `maxUnavailable: 1` for better HPA scaling. | `k8s/pdb.yaml:31` |
| I9 | Stale event recovery uses `created_at` instead of `claimed_at` timestamp. | `pkg/outbox/repository.go:126` |
| I10 | 55 test files: 37 unit, 10 integration, 8 pkg-level. Hand-rolled mocks with significant duplication. | All `*_test.go` files |

---

## 4. Architecture Strengths

This section notes what the system does well, to provide balanced context alongside the findings.

- **Clean domain-driven design** — 13 modules with consistent application/domain/infrastructure/transport layering. Domain models are framework-agnostic. Ports and adapters pattern with interfaces defined in domain, implemented in infrastructure.
- **Outbox pattern** — events are persisted before publishing. Stale event recovery on worker startup. Failed event tracking with error messages. Status machine (PENDING -> PUBLISHING -> PUBLISHED/FAILED).
- **Inventory concurrency** — atomic `$expr` MongoDB operations prevent overselling. No TOCTOU vulnerabilities on the critical reserve path. Validated by 100-goroutine integration tests.
- **Webhook idempotency** — three-layer defense (app check, unique index on webhook_id, payment status guard via FindOneAndUpdate). Constant-time HMAC comparison. Validated by 100-concurrent-webhook integration test.
- **Order/payment status transitions** — double-checked pattern (application layer + atomic FindOneAndUpdate). CONFLICT returned on race losers.
- **Security hardening** — distroless containers, non-root user, read-only filesystem, `drop: ALL` capabilities, `seccompProfile: RuntimeDefault`, NetworkPolicies (default-deny all), no SA tokens, SHA-pinned CI actions, least-privilege permissions.
- **Comprehensive CI/CD pipeline** — 13-stage pipeline with format, lint, vet, unit test, race detector, integration tests (testcontainers), SAST (gosec SARIF), secret scan (TruffleHog verified-only, full git history), dependency scan (govulncheck), Docker build (Buildx + GHA cache), container scan (Trivy SARIF), GHCR push, GitOps branch update.
- **Extensive indexing** — ~75 indexes across 24 collections with unique constraints, compound indexes, text search with weights, TTL indexes, partial filter expressions, and sparse indexes.
- **DR documentation** — detailed recovery procedures for MongoDB, NATS JetStream, application, Kubernetes, secrets, and DNS. RPO/RTO targets per component. 6 failure scenario runbooks. 27-item verification checklist.
- **Observability foundation** — OpenTelemetry integration, Prometheus metrics for HTTP/MongoDB/NATS/business operations, W3C TraceContext propagation, X-Request-ID, structured logging, audit trail with sanitization.
- **Integration test architecture** — gold-standard testcontainers setup with real MongoDB 7 + NATS 2.10, full application wiring, rich HTTP helper DSL, `WaitEventually` for async event propagation, and E2E flow tests covering the complete order lifecycle.
- **Password security** — bcrypt cost 12, no plaintext storage, account lockout (5 failures / 15-min lock), refresh token rotation, session revocation on password change.
- **Error handling** — internal errors never leaked to clients (`json:"-"` on Err field), generic fallback messages, W3C trace context in error responses.

---

## 5. Recommended Priority Actions

| Priority | Action | Addresses |
|---|---|---|
| **P0** | Implement order creation idempotency (accept `Idempotency-Key` header, atomic check-create with unique index) | C1 |
| **P0** | Add queue groups to JetStream consumers for horizontal worker scaling | C2 |
| **P0** | Add liveness probe to worker (TCP check to NATS, or add `/health` HTTP endpoint) | C3 |
| **P0** | Fix deploy workflow: remove `always()` from deploy gate, add post-deploy smoke test against live endpoint, add automatic `rollout undo` on failure | C4, C5 |
| **P0** | Remove `secret.yaml` from kustomization resources; replace hardcoded JWT secret with placeholder comment | C6, C7 |
| **P1** | Implement dead letter queue (JetStream max_ack_pull + DLQ consumer, or `failed_events` MongoDB collection with alerting) | H1 |
| **P1** | Fix analytics/marketing consumers to return error on store failure (not nil) | H2 |
| **P1** | Wrap business writes and outbox inserts in MongoDB sessions/transactions | H3, M13 |
| **P1** | Replace stdout trace exporter with configurable OTLP exporter | H4 |
| **P1** | Implement distributed rate limiting using existing `RATE_LIMIT` KV bucket or external store | H5 |
| **P1** | Restrict API NetworkPolicy ingress to Ingress controller namespace | H7 |
| **P1** | Pin gosec, govulncheck, and trufflehog to specific versions/tags | H8 |
| **P1** | Fix `kustomization.yaml` — remove or uncomment `scheduler-deployment.yaml` | H10 |
| **P1** | Change `imagePullPolicy` to `Always` or enforce SHA-based tags | H11 |
| **P2** | Add field-level sanitization to logger (or enforce telemetry-only logging policy) | M1 |
| **P2** | Remove unused KV bucket configuration | M2 |
| **P2** | Add missing MongoDB indexes (items.product_id, provider+provider_reference, TTL on tokens) | M6, M8 |
| **P2** | Add inventory reconciliation job (SUM reservations vs reserved_stock) | M9, M10, M11 |
| **P2** | Add partial unique index on `(sku, reference)` where `status = "ACTIVE"` | M12 |
| **P2** | Fix PaymentReference persistence in order status update | M16 |
| **P2** | Add `TARGETARCH` build arg to all Dockerfiles | M20 |
| **P2** | Add `//go:build integration` tags to integration tests | L15 |
| **P2** | Add customer module tests | L16 |
| **P3** | Execute load tests and populate the performance report | M7 |
| **P3** | Wire `InstrumentedCollection` into repositories for MongoDB tracing | L9 |
| **P3** | Add Prometheus counter for audit buffer drops | L13 |
| **P3** | Extract shared test utilities into `internal/testutil/` to reduce mock duplication | I10 |
| **P3** | Plan migration from HS256 to RS256 for microservices readiness | H6 |
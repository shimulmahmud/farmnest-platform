# Ecommerce Backend

Production-grade, scalable ecommerce backend built with Go.

## Architecture

**Modular Monolith** with event-driven processing.

- **Go + Fiber** — HTTP framework
- **MongoDB** — authoritative persistent datastore
- **NATS JetStream** — durable events, background jobs, async processing
- **NATS KV** — cache, temporary distributed state, idempotency/coordination
- **No Redis, no PostgreSQL**
- Stateless API instances, horizontally scalable

## Tech Stack

| Layer         | Technology                         |
| ------------- | ---------------------------------- |
| Language      | Go 1.22                            |
| HTTP          | Fiber v2                           |
| Database      | MongoDB 7                          |
| Messaging     | NATS JetStream + KV                |
| Observability | OpenTelemetry, Prometheus, Grafana |
| Testing       | Go testing, Testcontainers, k6     |
| Deployment    | Docker, Kubernetes                 |

## Project Structure

```
cmd/
  api/           # HTTP API server
  worker/         # Background job worker
  scheduler/     # Cron/scheduled tasks
internal/
  auth/           # Authentication & authorization
  users/          # User management
  customers/      # Customer profiles
  products/       # Product catalog
  categories/     # Category tree
  inventory/      # Stock management
  cart/           # Shopping cart
  pricing/        # Pricing engine
  coupons/        # Coupon & discounts
  checkout/       # Checkout flow
  orders/         # Order management
  payments/       # Payment processing
  shipping/       # Shipping management
  reviews/        # Product reviews
  wishlist/       # Wishlists
  notifications/  # Notification dispatch
  marketing/      # Marketing tools
  analytics/      # Analytics & reporting
  search/         # Search service
  admin/          # Admin operations
  audit/          # Audit logging
  health/         # Health checks
  middleware/     # HTTP middleware
pkg/
  config/         # Configuration
  logger/         # Structured logging
  mongodb/        # MongoDB client
  nats/           # NATS client
  errors/         # Error system
  response/       # API responses
  validator/      # Validation
  pagination/     # Pagination
  observability/  # OpenTelemetry
docs/
tests/
deployments/
```

## Quick Start

### Prerequisites

- Go 1.22+
- Docker & Docker Compose
- Make

### Local Development with Docker

```bash
# Copy environment file
cp .env.example .env

# Start all services (MongoDB, NATS, Mailpit, API, Worker)
make docker-up

# Open Mailpit to read verification and password-reset emails
# http://localhost:8025

# View logs
make logs

# Check health
curl http://localhost:8080/health
curl http://localhost:8080/health/ready
```

For Postman auth testing, register a new user, copy the verification token from
Mailpit into the `verification_token` environment variable, then run Verify Email.
Forgot Password sends a reset token to Mailpit; copy it into `reset_token` before
running Reset Password. Local SMTP uses `mailpit:1025`; production can use AWS SES
SMTP by setting the `SMTP_*` environment variables and enabling STARTTLS.

### Local Development (native Go)

```bash
# Start MongoDB and NATS only
make docker-up  # then scale down app: docker compose stop api worker

# Run API locally
make dev

# Run tests
make test

# Full check (fmt + vet + test + race)
make check
```

## Available Make Commands

| Command             | Description                    |
| ------------------- | ------------------------------ |
| `make dev`          | Run API locally                |
| `make build`        | Build all binaries             |
| `make test`         | Run all tests                  |
| `make race`         | Run tests with race detector   |
| `make cover`        | Generate coverage report       |
| `make lint`         | Format and vet                 |
| `make docker-build` | Build Docker images            |
| `make docker-up`    | Start all services             |
| `make docker-down`  | Stop all services              |
| `make logs`         | Tail all service logs          |
| `make check`        | Full check: lint + test + race |

## API Endpoints

| Method | Path            | Description                                           |
| ------ | --------------- | ----------------------------------------------------- |
| GET    | `/health`       | Basic health check                                    |
| GET    | `/health/live`  | Liveness probe                                        |
| GET    | `/health/ready` | Readiness probe (checks MongoDB, NATS, JetStream, KV) |

## Environment Variables

See [`.env.example`](.env.example) for all configurable values.

Key variables:

| Variable           | Default                     | Description                                      |
| ------------------ | --------------------------- | ------------------------------------------------ |
| `APP_ENV`          | —                           | Application environment (development/production) |
| `SERVER_PORT`      | `8080`                      | HTTP listen port                                 |
| `MONGODB_URI`      | `mongodb://localhost:27017` | MongoDB connection URI                           |
| `MONGODB_DATABASE` | `ecommerce`                 | Database name                                    |
| `NATS_URL`         | `nats://localhost:4222`     | NATS server URL                                  |
| `JWT_SECRET`       | —                           | JWT signing secret (must change in production)   |
| `LOG_LEVEL`        | `info`                      | Log level (debug/info/warn/error)                |

## NATS KV Buckets

| Bucket              | Purpose                     |
| ------------------- | --------------------------- |
| `PRODUCT_CACHE`     | Product read-through cache  |
| `CATEGORY_CACHE`    | Category read-through cache |
| `SESSION_STATE`     | Temporary session data      |
| `IDEMPOTENCY`       | Idempotency key tracking    |
| `RATE_LIMIT`        | Distributed rate limiting   |
| `CHECKOUT_STATE`    | Checkout flow state         |
| `DISTRIBUTED_LOCKS` | Distributed locking         |

## Design Principles

1. Business logic independent of Fiber
2. Domain code independent of MongoDB or NATS
3. Infrastructure implements interfaces
4. Important operations are idempotent
5. Inventory is concurrency-safe
6. Never trust client-supplied prices, totals, or payment status
7. MongoDB atomic operations/transactions where necessary
8. Outbox Pattern for reliable MongoDB-to-NATS events
9. External providers abstracted behind interfaces
10. Every feature has tests

## License

Private — All rights reserved.

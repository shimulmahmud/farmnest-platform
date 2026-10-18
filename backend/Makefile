# ============================================================
# Ecommerce Backend — Makefile
# Usage: make <target>
# ============================================================

.PHONY: dev build test lint vet fmt race cover \
        docker-build docker-build-prod docker-scan docker-push docker-up docker-down docker-logs docker-logs-api docker-restart \
        help \
        sast secret-scan dep-scan ci-check \
        loadtest-seed loadtest-product-reads loadtest-product-reads-spike \
        loadtest-search loadtest-search-spike loadtest-orders loadtest-checkout-spike \
        loadtest-nats-backlog loadtest-full-funnel loadtest-auth loadtest-auth-spike \
        loadtest-infra loadtest-all loadtest-clean

# Go toolchain
GO      := go
GOFLAGS := -v
GOOS    := $(shell $(GO) env GOOS)
GOARCH  := $(shell $(GO) env GOARCH)

# Docker
DOCKER_COMPOSE := docker compose
COMPOSE_FILE  := docker-compose.yml

# k6
K6         := k6
K6_DIR     := tests/load
K6_OUT_DIR := results/load
K6_COMMON  := BASE_URL=http://localhost:8080

# ----------------------------------------------------------
# Local Development
# ----------------------------------------------------------

dev: ## Run API locally (requires MongoDB + NATS running)
        $(GO) run $(GOFLAGS) ./cmd/api

# ----------------------------------------------------------
# Build
# ----------------------------------------------------------

build: ## Build all binaries
        $(GO) build $(GOFLAGS) -o bin/api ./cmd/api
        $(GO) build $(GOFLAGS) -o bin/worker ./cmd/worker

clean: ## Remove built binaries
        rm -rf bin/

# ----------------------------------------------------------
# Testing
# ----------------------------------------------------------

test: ## Run all tests
        $(GO) test ./... -count=1

race: ## Run tests with race detector
        $(GO) test -race ./... -count=1

cover: ## Run tests with coverage
        $(GO) test ./... -coverprofile=coverage.out -count=1
        $(GO) tool cover -html=coverage.out -o coverage.html
        @echo "Coverage report: coverage.html"

# ----------------------------------------------------------
# Linting & Formatting
# ----------------------------------------------------------

fmt: ## Format Go source files
        $(GO) fmt ./...

vet: ## Run go vet
        $(GO) vet ./...

lint: fmt vet ## Format and vet
        @echo "lint passed"

# ----------------------------------------------------------
# Docker
# ----------------------------------------------------------

docker-build: ## Build Docker images (dev, via docker-compose)
        $(DOCKER_COMPOSE) -f $(COMPOSE_FILE) build

docker-build-prod: ## Build hardened production images with version tags
        ./scripts/build.sh ${VERSION}

docker-scan: ## Build and scan production images for vulnerabilities
        ./scripts/build.sh ${VERSION} scan

docker-push: ## Push production images to registry
        ./scripts/build.sh ${VERSION}
        @echo "Pushing images..."
        docker push ecommerce-api:$(or ${VERSION},latest)
        docker push ecommerce-worker:$(or ${VERSION},latest)

docker-up: ## Start all services (MongoDB, NATS, API, Worker)
        $(DOCKER_COMPOSE) -f $(COMPOSE_FILE) up -d

docker-down: ## Stop and remove all containers
        $(DOCKER_COMPOSE) -f $(COMPOSE_FILE) down

docker-logs: ## Tail logs from all services
        $(DOCKER_COMPOSE) -f $(COMPOSE_FILE) logs -f

docker-logs-api: ## Tail API logs
        $(DOCKER_COMPOSE) -f $(COMPOSE_FILE) logs -f api

docker-restart: docker-down docker-up ## Restart all services

# ----------------------------------------------------------
# Utilities
# ----------------------------------------------------------

check: lint test race ## Run full check: lint + test + race
        @echo "All checks passed"

# ----------------------------------------------------------
# Security Scanning (mirrors CI pipeline)
# ----------------------------------------------------------

sast: ## Run SAST scan with gosec (outputs SARIF)
        @command -v gosec >/dev/null 2>&1 || go install github.com/securego/gosec/v2/cmd/gosec@latest
        gosec -fmt sarif -out gosec-results.sarif ./...
        @echo "SARIF report: gosec-results.sarif"

secret-scan: ## Scan for secrets with trufflehog (if installed)
        @command -v trufflehog >/dev/null 2>&1 || { echo "Install trufflehog: https://github.com/trufflesecurity/trufflehog"; exit 1; }
        trufflehog git file://. --only-verified


dep-scan: ## Run dependency vulnerability scan with govulncheck
        @command -v govulncheck >/dev/null 2>&1 || go install golang.org/x/vuln/cmd/govulncheck@latest
        govulncheck ./...

# ----------------------------------------------------------
# CI Local Check (run before pushing to catch issues fast)
# ----------------------------------------------------------

ci-check: fmt lint test race ## Run all CI checks locally (format, lint, test, race)
        @echo "All CI checks passed. Safe to push."

# ----------------------------------------------------------
# Load Testing (k6)
# Prerequisites: API + Worker running, seed data created
# ----------------------------------------------------------

loadtest-seed: ## Seed database with load test data (run once)
        @mkdir -p $(K6_OUT_DIR)
        $(K6) run $(K6_COMMON) $(K6_DIR)/seed.js

loadtest-product-reads: ## Load test: product reads (normal traffic, 10 VUs, 2m)
        @mkdir -p $(K6_OUT_DIR)
        $(K6) run $(K6_COMMON) --out json=$(K6_OUT_DIR)/product-reads.json $(K6_DIR)/product-reads.js

loadtest-product-reads-spike: ## Load test: product reads (spike: 5→200 VUs)
        @mkdir -p $(K6_OUT_DIR)
        K6_SCENARIO=spike $(K6) run $(K6_COMMON) --out json=$(K6_OUT_DIR)/product-reads-spike.json $(K6_DIR)/product-reads.js

loadtest-search: ## Load test: product search (high traffic, 80 VUs, 5m)
        @mkdir -p $(K6_OUT_DIR)
        K6_SCENARIO=high $(K6) run $(K6_COMMON) --out json=$(K6_OUT_DIR)/search.json $(K6_DIR)/product-search.js

loadtest-search-spike: ## Load test: product search (spike: 5→150 VUs)
        @mkdir -p $(K6_OUT_DIR)
        K6_SCENARIO=spike $(K6) run $(K6_COMMON) --out json=$(K6_OUT_DIR)/search-spike.json $(K6_DIR)/product-search.js

loadtest-orders: ## Load test: order operations (normal, 5 VUs, 2m)
        @mkdir -p $(K6_OUT_DIR)
        $(K6) run $(K6_COMMON) --out json=$(K6_OUT_DIR)/orders.json $(K6_DIR)/order-operations.js

loadtest-checkout-spike: ## Load test: checkout spike (2→80 VUs)
        @mkdir -p $(K6_OUT_DIR)
        K6_SCENARIO=checkout-spike $(K6) run $(K6_COMMON) --out json=$(K6_OUT_DIR)/checkout-spike.json $(K6_DIR)/order-operations.js

loadtest-nats-backlog: ## Load test: NATS worker backlog (200 VU burst)
        @mkdir -p $(K6_OUT_DIR)
        $(K6) run $(K6_COMMON) --out json=$(K6_OUT_DIR)/nats-backlog.json $(K6_DIR)/nats-worker-backlog.js

loadtest-full-funnel: ## Load test: full funnel (mixed, 100 VUs, 5m)
        @mkdir -p $(K6_OUT_DIR)
        K6_SCENARIO=high $(K6) run $(K6_COMMON) --out json=$(K6_OUT_DIR)/full-funnel.json $(K6_DIR)/full-funnel.js

loadtest-auth: ## Load test: auth + health (high, 80 VUs, 5m)
        @mkdir -p $(K6_OUT_DIR)
        K6_SCENARIO=high $(K6) run $(K6_COMMON) --out json=$(K6_OUT_DIR)/auth.json $(K6_DIR)/auth-mixed.js

loadtest-auth-spike: ## Load test: auth spike (2→150 VUs)
        @mkdir -p $(K6_OUT_DIR)
        K6_SCENARIO=spike $(K6) run $(K6_COMMON) --out json=$(K6_OUT_DIR)/auth-spike.json $(K6_DIR)/auth-mixed.js

loadtest-infra: ## Collect infrastructure metrics (300s, 1 VU)
        @mkdir -p $(K6_OUT_DIR)
        $(K6) run $(K6_COMMON) --out json=$(K6_OUT_DIR)/infra.json tests/infrastructure/metrics/infra-metrics.js

loadtest-clean: ## Remove load test seed data from database
        $(K6) run $(K6_COMMON) $(K6_DIR)/cleanup.js

loadtest-all: ## Run complete load test suite (sequential)
        @echo "=== Running full load test suite ==="
        @echo "--- 1/8 Seeding ---"
        $(MAKE) --silent loadtest-seed
        @echo "--- 2/8 Product Reads (normal) ---"
        $(MAKE) --silent loadtest-product-reads
        @echo "--- 3/8 Product Reads (spike) ---"
        $(MAKE) --silent loadtest-product-reads-spike
        @echo "--- 4/8 Product Search (high) ---"
        $(MAKE) --silent loadtest-search
        @echo "--- 5/8 Order Operations (normal) ---"
        $(MAKE) --silent loadtest-orders
        @echo "--- 6/8 Checkout Spike ---"
        $(MAKE) --silent loadtest-checkout-spike
        @echo "--- 7/8 NATS Worker Backlog ---"
        $(MAKE) --silent loadtest-nats-backlog
        @echo "--- 8/8 Full Funnel (high) ---"
        $(MAKE) --silent loadtest-full-funnel
        @echo "=== Load test suite complete ==="
        @echo "Results saved to $(K6_OUT_DIR)/"
        @echo "Fill in docs/performance-report.md with measured values."

# ----------------------------------------------------------
# Help
# ----------------------------------------------------------

help: ## Show this help
        @grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
                awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-35s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help

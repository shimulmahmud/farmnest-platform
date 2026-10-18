#!/usr/bin/env bash
# ============================================================
# build.sh — Build production Docker images with version tagging
# ============================================================
# Usage:
#   ./scripts/build.sh              # build with version from git
#   ./scripts/build.sh 1.2.3        # build with explicit version
#   ./scripts/build.sh 1.2.3 scan   # build + vulnerability scan
# ============================================================
set -euo pipefail

# ---- Resolve version ----
VERSION="${1:-}"
if [ -z "$VERSION" ]; then
    # Derive from git: use most recent tag, or "v0.0.0-<sha>" if untagged
    TAG=$(git describe --tags --always --dirty 2>/dev/null || echo "v0.0.0-unknown")
    VERSION="$TAG"
fi

COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_TIME=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
REGISTRY="${DOCKER_REGISTRY:-}"  # e.g., ghcr.io/myorg or docker.io/myuser

# Export for docker compose
export IMAGE_VERSION="$VERSION"
export IMAGE_COMMIT="$COMMIT"
export IMAGE_BUILD_TIME="$BUILD_TIME"

if [ -n "$REGISTRY" ]; then
    API_IMAGE="${REGISTRY}/ecommerce-api:${VERSION}"
    WORKER_IMAGE="${REGISTRY}/ecommerce-worker:${VERSION}"
else
    API_IMAGE="ecommerce-api:${VERSION}"
    WORKER_IMAGE="ecommerce-worker:${VERSION}"
fi

echo "========================================"
echo " Building Production Images"
echo "========================================"
echo "Version:    ${VERSION}"
echo "Commit:     ${COMMIT}"
echo "Build Time: ${BUILD_TIME}"
echo "API Image:  ${API_IMAGE}"
echo "Worker:     ${WORKER_IMAGE}"
echo "========================================"
echo

# ---- Build ----
echo "--- Building API image ---"
docker build \
    -f Dockerfile.api \
    -t "$API_IMAGE" \
    -t "${API_IMAGE%%:*}:latest" \
    --build-arg VERSION="$VERSION" \
    --build-arg COMMIT="$COMMIT" \
    --build-arg BUILD_TIME="$BUILD_TIME" \
    .

echo "--- Building Worker image ---"
docker build \
    -f Dockerfile.worker \
    -t "$WORKER_IMAGE" \
    -t "${WORKER_IMAGE%%:*}:latest" \
    --build-arg VERSION="$VERSION" \
    --build-arg COMMIT="$COMMIT" \
    --build-arg BUILD_TIME="$BUILD_TIME" \
    .

echo
echo "=== Build complete ==="
echo "  API:    ${API_IMAGE}"
echo "  Worker: ${WORKER_IMAGE}"

# ---- Vulnerability scanning (if requested) ----
if [ "${2:-}" = "scan" ]; then
    echo
    echo "=== Scanning for vulnerabilities ==="

    if command -v trivy &> /dev/null; then
        echo "--- Trivy scan: API ---"
        trivy image --severity HIGH,CRITICAL "$API_IMAGE" || true
        echo "--- Trivy scan: Worker ---"
        trivy image --severity HIGH,CRITICAL "$WORKER_IMAGE" || true
    elif command -v docker scout &> /dev/null; then
        echo "--- Docker Scout scan: API ---"
        docker scout cves "$API_IMAGE" || true
        echo "--- Docker Scout scan: Worker ---"
        docker scout cves "$WORKER_IMAGE" || true
    else
        echo "WARNING: Neither trivy nor docker scout found."
        echo "Install trivy: https://aquasecurity.github.io/trivy/"
        echo "  curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sh -s -- -b /usr/local/bin"
        echo "Or use Docker Scout (included with Docker Desktop)."
    fi

    echo
echo "=== Scan complete ==="
fi

echo ""
echo "To push images:"
echo "  docker push ${API_IMAGE}"
echo "  docker push ${WORKER_IMAGE}"
echo "  docker push ${API_IMAGE%%:*}:latest"
echo "  docker push ${WORKER_IMAGE%%:*}:latest"

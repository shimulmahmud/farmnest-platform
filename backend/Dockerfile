# ============================================================
# Stage 1: Build
# ============================================================
FROM golang:1.26-alpine AS builder

# Build for the architecture of the machine running Docker (amd64, arm64 / Apple Silicon...)
ARG TARGETOS=linux
ARG TARGETARCH

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /src

# Cache module downloads
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build statically
COPY . .

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH:-amd64} go build \
    -ldflags="-w -s" \
    -o /bin/api ./cmd/api

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH:-amd64} go build \
    -ldflags="-w -s" \
    -o /bin/worker ./cmd/worker

# ============================================================
# Stage 2: Runtime (API)
# ============================================================
FROM alpine:3.19 AS api

RUN apk add --no-cache ca-certificates tzdata wget

COPY --from=builder /bin/api /bin/api

EXPOSE 8080

RUN adduser -D -u 10001 appuser
USER appuser

ENTRYPOINT ["/bin/api"]

# ============================================================
# Stage 3: Runtime (Worker)
# ============================================================
FROM alpine:3.19 AS worker

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /bin/worker /bin/worker

RUN adduser -D -u 10002 workeruser
USER workeruser

ENTRYPOINT ["/bin/worker"]

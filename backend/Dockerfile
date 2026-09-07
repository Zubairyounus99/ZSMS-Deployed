# ==============================================================================
# ZSMS Backend & Worker — Multi-Stage Dockerfile
# ==============================================================================

# Build Stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Cache Go modules
COPY go.mod go.sum* ./
RUN go mod download || true

# Copy source code
COPY . .

# Ensure go.sum is up to date and download any missing packages
RUN go mod tidy

# Build API and Worker binaries with optimization flags
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /bin/api ./api/main.go
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /bin/worker ./worker/main.go

# Runtime Stage
FROM alpine:3.19

WORKDIR /app

# Install runtime dependencies & create unprivileged user
RUN apk add --no-cache ca-certificates tzdata curl && \
    addgroup -g 10001 -S appuser && \
    adduser -u 10001 -S appuser -G appuser

# Copy binaries from builder
COPY --from=builder /bin/api /app/api
COPY --from=builder /bin/worker /app/worker

# Set ownership
RUN chown -R appuser:appuser /app

USER appuser

EXPOSE 8080

HEALTHCHECK --interval=15s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8080/health/live || exit 1

ENTRYPOINT ["/app/api"]

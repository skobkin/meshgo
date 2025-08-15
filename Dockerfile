# Multi-stage build for MeshGo
FROM golang:1.24-alpine AS builder

# Install dependencies for CGO (if needed)
RUN apk add --no-cache gcc musl-dev

WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
ARG VERSION=dev
ARG BUILD_TIME
ARG COMMIT
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.Commit=${COMMIT}" \
    -o meshgo ./cmd/meshgo

# Final stage - minimal runtime image
FROM alpine:latest

# Install necessary runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -S meshgo && adduser -S meshgo -G meshgo

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/meshgo .

# Create config directory
RUN mkdir -p /home/meshgo/.config/meshgo && \
    chown -R meshgo:meshgo /home/meshgo

USER meshgo

# Expose default Meshtastic port (if running as server)
EXPOSE 4403

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD pgrep meshgo || exit 1

ENTRYPOINT ["./meshgo"]
CMD ["--help"]
# Hatcher - Go Development Environment
FROM golang:1.22-alpine AS builder

# Install necessary packages
RUN apk add --no-cache \
    git \
    make \
    bash \
    curl \
    ca-certificates \
    tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN make build

# Test stage
FROM golang:1.22-alpine AS test

# Install test dependencies
RUN apk add --no-cache \
    git \
    make \
    bash \
    curl \
    ca-certificates \
    tzdata

# Install additional tools for testing (compatible with Go 1.22)
RUN go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.60.3
RUN go install github.com/securego/gosec/v2/cmd/gosec@v2.20.0

# Set working directory
WORKDIR /app

# Copy everything
COPY . .

# Download dependencies
RUN go mod download

# Run tests
CMD ["make", "test-all"]

# Production stage
FROM alpine:latest AS production

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1001 -S hatcher && \
    adduser -u 1001 -S hatcher -G hatcher

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/build/hatcher /usr/local/bin/hatcher

# Change ownership
RUN chown -R hatcher:hatcher /app

# Switch to non-root user
USER hatcher

# Expose port (if needed for future web interface)
EXPOSE 3000

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD hatcher doctor --format json || exit 1

# Default command
ENTRYPOINT ["hatcher"]
CMD ["--help"]

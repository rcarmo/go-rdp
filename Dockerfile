# Multi-stage Dockerfile for RDP HTML5 Client
FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-s -w" -o rdp-html5 cmd/server/main.go

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata && \
    adduser -D -s /bin/sh appuser

# Set working directory
WORKDIR /app

# Copy binary from builder
ARG TARGETOS
ARG TARGETARCH
COPY --from=builder /app/rdp-html5 ./rdp-html5

# Copy static files
COPY --from=builder /app/web ./web

# Set permissions
RUN chown -R appuser:appuser /app && \
    chmod +x ./rdp-html5

# Switch to non-root user
USER appuser

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/ || exit 1

# Expose port
EXPOSE 8080

# Run the binary
CMD ["./rdp-html5"]
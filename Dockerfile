# Multi-stage Dockerfile for RDP HTML5 Client
# Stage 1: Build frontend assets (WASM + JS)
FROM tinygo/tinygo:0.34.0 AS wasm-builder

WORKDIR /app

# Copy WASM source
COPY web/wasm ./web/wasm
COPY go.mod go.sum ./

# Download dependencies and build WASM
RUN go mod download
RUN tinygo build -o web/js/rle/rle.wasm -target wasm -opt=z ./web/wasm/
RUN cp "$(tinygo env TINYGOROOT)/targets/wasm_exec.js" web/js/rle/wasm_exec.js

# Stage 2: Build JavaScript bundle
FROM node:20-alpine AS js-builder

WORKDIR /app

# Copy JS source and package files
COPY web/js/src ./web/js/src

# Install dependencies and build
WORKDIR /app/web/js/src
RUN npm install --silent
RUN npm run build:min

# Stage 3: Build Go backend
FROM golang:1.22-alpine AS go-builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Copy built frontend assets from previous stages
COPY --from=wasm-builder /app/web/js/rle/rle.wasm ./web/js/rle/rle.wasm
COPY --from=wasm-builder /app/web/js/rle/wasm_exec.js ./web/js/rle/wasm_exec.js
COPY --from=js-builder /app/web/js/client.bundle.min.js ./web/js/client.bundle.min.js

# Build the binary
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-s -w" -o rdp-html5 cmd/server/main.go

# Final stage: Runtime image
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata && \
    adduser -D -s /bin/sh appuser

WORKDIR /app

# Copy binary from builder
COPY --from=go-builder /app/rdp-html5 ./rdp-html5

# Copy static files including built frontend assets
COPY --from=go-builder /app/web ./web

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

# Environment variables for configuration (can be overridden at runtime with -e)
# TLS_SKIP_VERIFY: Set to "true" to connect to RDP servers with self-signed certificates
# TLS_ALLOW_ANY_SERVER_NAME: Allow connecting without enforcing SNI (lab/testing)
# LOG_LEVEL: debug, info, warn, error
ENV TLS_SKIP_VERIFY=false \
    TLS_ALLOW_ANY_SERVER_NAME=true \
    LOG_LEVEL=info

# Run the binary
CMD ["./rdp-html5"]

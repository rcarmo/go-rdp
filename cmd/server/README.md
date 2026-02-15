# cmd/server

HTTP/WebSocket server entry point for the RDP HTML5 client.

## Overview

This package contains the main entry point for the RDP-to-WebSocket gateway server. It serves a web-based UI and handles WebSocket connections that bridge browser clients to remote RDP servers.

## Files

| File | Purpose |
|------|---------|
| `main.go` | Entry point, CLI flags, HTTP server setup |
| `main_test.go` | Unit tests for server components |

## Command-Line Flags

```
Server:
  -host <addr>               Server listen host (default: 0.0.0.0)
  -port <port>               Server listen port (default: 8080)

Logging:
  -log-level <level>         Log level: debug, info, warn, error (default: info)

TLS/Security:
  -tls-skip-verify           Skip TLS certificate validation for RDP connections
  -tls-server-name <name>    Override TLS server name for RDP connections (SNI)
  -tls-allow-any-server-name Allow connecting without enforcing SNI (lab/testing only)

RDP Protocol:
  -nla                       Enable Network Level Authentication (CredSSP)
  -no-rfx                    Disable RemoteFX codec support
  -udp                       Enable UDP transport (experimental)

Audio:
  -prefer-pcm-audio          Prefer PCM audio (best quality, ~1.4 Mbps)
                             Default: prefer AAC/MP3 (~128-192 kbps)

Info:
  -version                   Show version information
  -help                      Show help message
```

For detailed flag descriptions, see [docs/configuration.md](../../docs/configuration.md).

## Architecture

```
main()
  │
  ├── parseFlags()         Parse CLI arguments
  │
  └── run()
        │
        ├── config.LoadWithOverrides()   Load configuration
        ├── setupLogging()               Initialize logger
        └── startServer()
              │
              └── createServer()
                    │
                    ├── Route: /           → Static files (./web/dist)
                    └── Route: /connect    → WebSocket handler
```

## HTTP Routes

| Route | Handler | Description |
|-------|---------|-------------|
| `/` | `http.FileServer` | Serves static web files (HTML, JS, WASM) |
| `/connect` | `handler.Connect` | WebSocket endpoint for RDP connections |

## Middleware Stack

Applied in order to all requests:

1. **Rate Limiting** - Configurable request throttling
2. **CORS Validation** - Origin checking against allowlist
3. **Security Headers** - Sets protective HTTP headers
4. **Request Logging** - Logs all incoming requests

### Security Headers

```
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Strict-Transport-Security: max-age=31536000; includeSubDomains
Content-Security-Policy: default-src 'self'; script-src 'self' 'wasm-unsafe-eval'; ...
```

## Configuration

The server reads configuration from environment variables with CLI overrides taking precedence. See `internal/config` for full configuration options.

Key settings:
- `SERVER_HOST`, `SERVER_PORT` - Listen address
- `ENABLE_TLS`, `TLS_CERT_FILE`, `TLS_KEY_FILE` - HTTPS support
- `ALLOWED_ORIGINS` - CORS allowlist (currently permissive for reverse proxies/port mappings)
- `MAX_CONNECTIONS`, `ENABLE_RATE_LIMIT` - Connection limits

## Usage

```bash
# Development mode
go run ./cmd/server

# Production with custom port
go run ./cmd/server -port 443 -host 0.0.0.0

# With NLA authentication and self-signed cert
go run ./cmd/server -nla -tls-skip-verify

# With PCM audio for high-bandwidth LAN
go run ./cmd/server -prefer-pcm-audio

# Build and run
make build
./bin/go-rdp -port 8080
```

## Related Packages

- `internal/config` - Configuration loading and validation
- `internal/handler` - WebSocket connection handling
- `internal/rdp` - RDP client implementation
- `web/` - Static web assets served by the server

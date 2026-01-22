# RDP HTML5 Client

A browser-based Remote Desktop Protocol (RDP) client built with Go and WebAssembly.

> ⚠️ **Note**: While functional, this implementation has known limitations.

## Quick Start

### Using Docker (Recommended)

```bash
# Run with default settings (TLS validation enabled)
docker run -d -p 8080:8080 ghcr.io/rcarmo/rdp-html5:latest

# Run with TLS validation disabled (for self-signed certs)
docker run -d -p 8080:8080 -e TLS_SKIP_VERIFY=true ghcr.io/rcarmo/rdp-html5:latest

# Run with debug logging
docker run -d -p 8080:8080 -e LOG_LEVEL=debug ghcr.io/rcarmo/rdp-html5:latest
```

Then open http://localhost:8080 in your browser.

### Using Docker Compose

```bash
git clone https://github.com/rcarmo/rdp-html5.git
cd rdp-html5
docker-compose up -d
```

### Building from Source

**Prerequisites:**
- Go 1.21+
- TinyGo 0.34+ (for WASM)
- Node.js 18+ (for JS bundling)

```bash
# Clone and build
git clone https://github.com/rcarmo/rdp-html5.git
cd rdp-html5
make deps    # Install dependencies
make build   # Build everything (WASM + JS + binary)

# Run
./bin/rdp-html5
```

## Configuration

Environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_PORT` | `8080` | HTTP server port |
| `LOG_LEVEL` | `info` | Logging level: debug, info, warn, error |
| `TLS_SKIP_VERIFY` | `false` (Docker default: `false`) | Skip RDP server TLS certificate validation |
| `TLS_ALLOW_ANY_SERVER_NAME` | `false` (Docker default: `false`) | Allow connecting without enforcing SNI (lab/testing) |
| `ENABLE_TLS` | `false` | Enable HTTPS for the web interface |
| `TLS_CERT_FILE` | - | Path to TLS certificate |
| `TLS_KEY_FILE` | - | Path to TLS private key |
| `RDP_ENABLE_RFX` | `true` | Enable RemoteFX codec support |

Command-line flags:

| Flag | Description |
|------|-------------|
| `-host` | Server listen host (default: 0.0.0.0) |
| `-port` | Server listen port (default: 8080) |
| `-log-level` | Log level: debug, info, warn, error |
| `-tls-skip-verify` | Skip TLS certificate validation |
| `-tls-server-name` | Override TLS server name (SNI) |
| `-tls-allow-any-server-name` | Allow connecting without enforcing SNI (lab/testing only) |
| `-nla` | Enable Network Level Authentication |
| `-no-rfx` | Disable RemoteFX codec support |

See [docs/configuration.md](docs/configuration.md) for full configuration options.

## Documentation

- [Architecture](docs/ARCHITECTURE.md) - System design and data flow
- [Configuration](docs/configuration.md) - Configuration options (env vars + flags)
- [Debugging](docs/debugging.md) - Troubleshooting guide
- [NSCodec](docs/NSCODEC.md) - Bitmap codec implementation
- [RemoteFX](docs/REMOTEFX.md) - RemoteFX wavelet codec implementation

## Features

- **Secure Credentials** - Passwords sent via WebSocket, not URL
- **TLS Support** - TLS 1.2+ encryption for RDP connections
- **NLA Authentication** - Network Level Authentication (with limitations)
- **Clipboard** - Bidirectional text clipboard
- **Audio** - Basic audio redirection
- **WebAssembly** - RLE/NSCodec/RemoteFX decoding via WASM
- **Configurable** - Environment-based configuration

## Limitations

- **Windows Compatibility** - Primarily tested with XRDP; Windows servers may have issues
- **Graphics** - RemoteFX codec implemented; H.264 not yet supported
- **NLA** - Works with many configurations but not all (see limitations below)

## Development

```bash
make help       # Show all targets
make dev        # Run in development mode
make test       # Run tests
make lint       # Run linters
make build-all  # Build for all platforms
```

## License

MIT License - see [LICENSE](LICENSE)

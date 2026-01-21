# RDP HTML5 Client

A browser-based Remote Desktop Protocol (RDP) client built with Go and WebAssembly.

> ⚠️ **Note**: While functional, this implementation has known limitations.

## Quick Start

### Using Docker (Recommended)

```bash
# Run with default settings (TLS validation enabled)
docker run -d -p 8080:8080 ghcr.io/rcarmo/rdp-html5:latest

# Run with TLS validation disabled (for self-signed certs)
docker run -d -p 8080:8080 -e SKIP_TLS_VALIDATION=true ghcr.io/rcarmo/rdp-html5:latest

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
| `PORT` | `8080` | HTTP server port |
| `LOG_LEVEL` | `info` | Logging level: debug, info, warn, error |
| `SKIP_TLS_VALIDATION` | `false` | Skip RDP server TLS certificate validation |
| `ENABLE_TLS` | `false` | Enable HTTPS for the web interface |
| `TLS_CERT_FILE` | - | Path to TLS certificate |
| `TLS_KEY_FILE` | - | Path to TLS private key |

See [docs/configuration.md](docs/configuration.md) for full configuration options.

## Documentation

- [Architecture](docs/ARCHITECTURE.md) - System design and data flow
- [Configuration](docs/configuration.md) - All configuration options
- [Debugging](docs/debugging.md) - Troubleshooting guide
- [NSCodec](docs/NSCODEC.md) - Bitmap codec implementation
- [RemoteFX](docs/REMOTEFX.md) - RemoteFX wavelet codec implementation
- [Known Issues](KNOWN_ISSUES.md) - Current limitations
- [Security](SECURITY.md) - Security considerations
- [Changelog](CHANGELOG.md) - Version history

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
- **NLA** - Works with many configurations but not all (see [KNOWN_ISSUES.md](KNOWN_ISSUES.md))

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

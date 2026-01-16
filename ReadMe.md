# RDP HTML5 Client

> ⚠️ **EXPERIMENTAL SOFTWARE** ⚠️
>
> This project is a proof-of-concept and experimental implementation. It is **NOT** intended for production use. The RDP protocol implementation is incomplete, may contain bugs, and has not undergone security auditing. Use at your own risk and only in controlled development/testing environments.

A browser-based Remote Desktop Protocol (RDP) client built with Go and WebAssembly.

## Features

- **TLS Support**: TLS 1.2+ encryption for transport security
- **Basic RDP Protocol**: Core RDP functionality tested primarily against XRDP on Linux
- **Web Interface**: HTML5/JavaScript client with canvas rendering
- **WebAssembly**: RLE bitmap decompression via WASM module
- **Environment Configuration**: Configuration via environment variables

## Limitations

- **NLA/CredSSP**: Network Level Authentication support is incomplete
- **Windows Compatibility**: Primarily tested with XRDP; Windows RDP servers may not work
- **Graphics**: Only basic bitmap updates supported; no RemoteFX or H.264
- **Clipboard/Audio/Printing**: Not implemented
- **Virtual Channels**: Partial implementation only
- **Security**: Not audited; do not use with sensitive systems

## Architecture

### Backend (Go)

- **RDP Protocol Implementation**: Partial RDP protocol stack, tested primarily with XRDP
- **WebSocket Server**: Bridges browser to RDP server
- **Configuration System**: Environment-based configuration

### Frontend (HTML5/JavaScript)

- **Canvas-based Rendering**: 2D canvas for bitmap display
- **WebAssembly Integration**: RLE decompression module
- **Basic Responsive Design**: Adapts to window size

## Configuration

The application uses environment variables for configuration. See `internal/pkg/config/config.go` for all available options.

### Server Configuration

```bash
export SERVER_HOST=0.0.0.0
export SERVER_PORT=8080
export SERVER_READ_TIMEOUT=30s
export SERVER_WRITE_TIMEOUT=30s
export SERVER_IDLE_TIMEOUT=120s
```

### Security Configuration

```bash
export ALLOWED_ORIGINS="https://example.com,https://app.example.com"
export MAX_CONNECTIONS=100
export ENABLE_RATE_LIMIT=true
export RATE_LIMIT_PER_MINUTE=60
export ENABLE_TLS=false
export MIN_TLS_VERSION=1.2
```

### RDP Configuration

```bash
export RDP_DEFAULT_WIDTH=1024
export RDP_DEFAULT_HEIGHT=768
export RDP_MAX_WIDTH=3840
export RDP_MAX_HEIGHT=2160
export RDP_BUFFER_SIZE=65536
export RDP_TIMEOUT=10s
```
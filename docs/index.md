# Documentation

This folder contains the long-form documentation for the project.

## Contents

- [Architecture](ARCHITECTURE.md) - System design, data flow, and protocol details
- [Configuration](configuration.md) - Environment variables, command-line flags, and settings
- [Debugging](debugging.md) - Logging, capabilities, and troubleshooting guide
- [NSCodec](NSCODEC.md) - Bitmap codec implementation details
- [RemoteFX](REMOTEFX.md) - RemoteFX wavelet codec implementation

## Package Documentation

Each Go package has its own README with implementation details:

- `internal/auth/` - NTLM/CredSSP authentication
- `internal/codec/` - Bitmap compression (RLE, NSCodec)
- `internal/codec/rfx/` - RemoteFX wavelet codec
- `internal/config/` - Configuration management
- `internal/handler/` - WebSocket connection handling
- `internal/logging/` - Leveled logging system
- `internal/protocol/` - RDP protocol layers
- `internal/rdp/` - RDP client implementation
- `web/wasm/` - WebAssembly codecs (RLE, NSCodec, RemoteFX)

## JavaScript Modules

- `web/js/src/` - Browser client modules
  - `wasm.js` - WASM codec wrapper (WASMCodec, RFXDecoder)
  - `codec-fallback.js` - Pure JS fallback codecs
  - `graphics.js` - Canvas rendering
  - `audio.js` - Audio redirection
  - `session.js` - Connection management

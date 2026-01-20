# Documentation

This folder contains the long-form documentation for the project.

## Contents

- [Architecture](ARCHITECTURE.md) - System design, data flow, and protocol details
- [Configuration](configuration.md) - Environment variables and settings
- [Debugging](debugging.md) - Logging and troubleshooting guide
- [NSCodec](NSCODEC.md) - Bitmap codec implementation details
- [RemoteFX](REMOTEFX.md) - Future RemoteFX/GFX implementation notes

## Package Documentation

Each Go package has its own README with implementation details:

- `internal/auth/` - NTLM/CredSSP authentication
- `internal/codec/` - Bitmap compression (RLE, NSCodec)
- `internal/config/` - Configuration management
- `internal/handler/` - WebSocket connection handling
- `internal/logging/` - Leveled logging system
- `internal/protocol/` - RDP protocol layers
- `internal/rdp/` - RDP client implementation
- `web/wasm/` - WebAssembly RLE decoder

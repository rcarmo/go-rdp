# Architecture

This project is a browser-based Remote Desktop Protocol (RDP) client built with Go and WebAssembly.

## High-level

- The browser UI connects to the Go backend over WebSockets.
- The backend connects to the target RDP server and acts as a bridge.
- Bitmap updates received from the RDP server are rendered in the browser using a `<canvas>`.
- RLE bitmap decompression is accelerated via a WebAssembly module.

## Backend (Go)

- **RDP protocol implementation**: A partial RDP protocol stack, tested primarily with XRDP.
- **WebSocket server**: Bridges browser messages/events to the RDP session.
- **Configuration**: Environment-variable driven configuration.

## Frontend (HTML5/JavaScript)

- **Canvas-based rendering**: 2D canvas for bitmap display.
- **WebAssembly integration**: RLE decompression module.
- **Responsive layout**: Adapts to window size.

## Related source areas

- Go server entry point: `cmd/server`
- Config package: `internal/pkg/config`
- RDP implementation: `internal/pkg/rdp`
- Web assets: `web/`

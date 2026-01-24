# web

Web client assets for the Go RDP client.

## Overview

This directory contains all web-facing assets:
- HTML entry point
- JavaScript modules for RDP protocol handling
- TinyGo WASM codec for high-performance bitmap processing
- Test pages for development

## Structure

```
web/
├── src/                        # Source files
│   ├── index.html              # Main app HTML
│   ├── test*.html              # Test pages
│   ├── js/                     # JavaScript source modules
│   │   ├── index.js            # Entry point (exports RDP client)
│   │   ├── client.js           # WebSocket client
│   │   ├── binary.js           # Binary protocol utilities
│   │   ├── graphics.js         # Canvas rendering
│   │   ├── input.js            # Keyboard/mouse handling
│   │   ├── wasm.js             # WASM loader
│   │   └── ...                 # Other modules
│   └── wasm/                   # TinyGo WASM source
│       └── main.go             # Codec implementations
├── dist/                       # Build output (generated)
│   ├── index.html              # Copied from src/
│   ├── js/
│   │   ├── client.bundle.min.js  # Bundled JS
│   │   └── rle/
│   │       ├── rle.wasm          # Compiled WASM
│   │       └── wasm_exec.js      # TinyGo runtime
│   └── test*.html              # Copied from src/
├── embed.go                    # Go embed directive for dist/
└── web_integration_test.go     # Integration tests
```

## Building

```bash
# Build all frontend assets (HTML + JS + WASM)
make build-frontend

# Build individual components
make build-html      # Copy HTML to dist/
make build-js-min    # Bundle and minify JS
make build-wasm      # Compile WASM module

# Clean build artifacts
make clean-frontend
```

## Embedding

Static assets are embedded into the Go binary using `go:embed`. The `embed.go` file
provides access to the dist/ filesystem:

```go
import "github.com/rcarmo/go-rdp/web"

staticFS, _ := web.DistFS()
http.Handle("/", http.FileServerFS(staticFS))
```

## Testing

```bash
# Run web integration tests
go test ./web/... -v

# Run JavaScript tests
make test-js
```

## Related Packages

- `internal/handler` - WebSocket server endpoint
- `internal/codec` - Go codec implementation (mirrored in WASM)
- `cmd/server` - Serves embedded static files

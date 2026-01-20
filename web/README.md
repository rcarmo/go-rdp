# web

Web client assets for the RDP HTML5 client.

## Overview

This directory contains all web-facing assets:
- HTML entry point
- JavaScript modules for RDP protocol handling
- TinyGo WASM codec for high-performance bitmap processing
- Test pages for development

## Structure

```
web/
├── index.html              # Main application entry point
├── test.html               # Protocol testing page
├── test_styling.html       # UI/styling test page
├── integration_test.html   # Full integration tests
├── web_integration_test.go # Go integration test runner
├── js/                     # JavaScript modules
│   ├── client.bundle.js    # Bundled client code
│   ├── wasm_exec.js        # TinyGo WASM runtime
│   ├── binary.js           # Binary protocol parsing
│   ├── color.js            # Color space utilities
│   ├── connection-history.js # Connection persistence
│   ├── input/              # Input handlers
│   │   ├── keyboard.js     # Keyboard event handling
│   │   ├── mouse.js        # Mouse event handling
│   │   └── keymap.js       # Keyboard layout mapping
│   └── update/             # Display update handlers
│       ├── bitmap.js       # Bitmap decompression
│       ├── pointer.js      # Cursor rendering
│       └── header.js       # Protocol headers
└── wasm/                   # TinyGo WASM source
    └── main.go             # WASM codec functions
```

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                          Browser                                     │
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │                       index.html                                ││
│  │                                                                  ││
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  ││
│  │  │   Canvas    │  │  Controls   │  │  Connection Form        │  ││
│  │  │  (display)  │  │  (toolbar)  │  │  (host/user/pass)       │  ││
│  │  └──────┬──────┘  └─────────────┘  └─────────────────────────┘  ││
│  └─────────┼───────────────────────────────────────────────────────┘│
│            │                                                         │
│  ┌─────────▼───────────────────────────────────────────────────────┐│
│  │                    JavaScript Layer                              ││
│  │                                                                  ││
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  ││
│  │  │   Client    │  │   Binary    │  │   Input Handlers        │  ││
│  │  │  (WebSocket)│  │   Parser    │  │   (keyboard/mouse)      │  ││
│  │  └──────┬──────┘  └──────┬──────┘  └───────────┬─────────────┘  ││
│  │         │                │                     │                 ││
│  │         │    ┌───────────▼───────────┐         │                 ││
│  │         │    │    Update Handlers    │         │                 ││
│  │         │    │  (bitmap/pointer)     │         │                 ││
│  │         │    └───────────┬───────────┘         │                 ││
│  │         │                │                     │                 ││
│  └─────────┼────────────────┼─────────────────────┼─────────────────┘│
│            │                │                     │                  │
│  ┌─────────▼────────────────▼─────────────────────▼─────────────────┐│
│  │                      WASM Layer                                  ││
│  │                  (TinyGo compiled)                               ││
│  │                                                                  ││
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  ││
│  │  │    RLE      │  │   NSCodec   │  │   Color Conversion      │  ││
│  │  │ Decompress  │  │   Decode    │  │   (RGB565→RGBA, etc.)   │  ││
│  │  └─────────────┘  └─────────────┘  └─────────────────────────┘  ││
│  └─────────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    WebSocket Connection                              │
│                    ws://server/connect                               │
└─────────────────────────────────────────────────────────────────────┘
```

## Data Flow

### Screen Updates (Server → Browser)

```
1. WebSocket receives binary frame
2. BinaryReader parses FastPath header
3. Update handler identifies type (bitmap, pointer, etc.)
4. Bitmap handler extracts compressed data
5. WASM codec decompresses (RLE/NSCodec/Planar)
6. WASM converts to RGBA
7. Canvas 2D context draws ImageData
```

### Input Events (Browser → Server)

```
1. DOM event fires (keydown, mousemove, etc.)
2. Input handler converts to RDP format
3. BinaryWriter serializes event
4. WebSocket sends binary frame
```

## JavaScript Modules

### client.bundle.js

Main client class managing:
- WebSocket lifecycle
- Reconnection logic
- Session persistence (cookies)
- Event coordination

### binary.js

Binary protocol utilities:
- `BinaryReader` - Parse multi-byte values (little-endian)
- `BinaryWriter` - Serialize input events
- Buffer management

### input/keyboard.js

Keyboard handling:
- DOM key event → RDP scancode conversion
- Extended key detection
- Key repeat handling
- Modifier state tracking

### input/mouse.js

Mouse handling:
- Position tracking
- Button state management
- Wheel events
- Coordinate scaling (canvas vs desktop)

### update/bitmap.js

Bitmap processing:
- Parse `TS_BITMAP_DATA` structures
- Delegate to WASM for decompression
- Handle compressed/uncompressed bitmaps
- Coordinate canvas updates

### update/pointer.js

Cursor rendering:
- Parse pointer updates
- Render custom cursors
- Cache cursor images
- Handle pointer visibility

## WASM Module (wasm/)

The `wasm/main.go` file compiles to WebAssembly and exports:

```go
// Exposed via js.Global().Set("goRLE", ...)
goRLE.decompressRLE16(src, dst, width)
goRLE.rgb565ToRGBA(src, dst)
goRLE.bgr24ToRGBA(src, dst)
goRLE.bgra32ToRGBA(src, dst)
goRLE.flipVertical(data, width, height, bytesPerPixel)
goRLE.processBitmap(src, dst, width, height, bpp, compressed)
goRLE.decodeNSCodec(src, dst, width, height)
goRLE.setPalette(data, numColors)
```

### Building WASM

```bash
# Build with TinyGo (smaller binary)
make wasm

# Or directly
tinygo build -o web/wasm/rle.wasm -target wasm ./web/wasm/
```

### Loading WASM

```javascript
// Load TinyGo runtime
const go = new Go();

// Fetch and instantiate WASM
const result = await WebAssembly.instantiateStreaming(
    fetch('wasm/rle.wasm'),
    go.importObject
);

// Start Go runtime
go.run(result.instance);

// Now goRLE.* functions are available
```

## HTML Entry Point

`index.html` loads everything in order:

```html
<!-- 1. TinyGo WASM runtime -->
<script src="js/wasm_exec.js"></script>

<!-- 2. Protocol modules -->
<script src="js/binary.js"></script>
<script src="js/update/bitmap.js"></script>
<script src="js/input/keyboard.js"></script>
<script src="js/input/mouse.js"></script>

<!-- 3. Main client bundle -->
<script src="js/client.bundle.js"></script>

<!-- 4. Initialize WASM and connect -->
<script>
    // Load WASM
    const go = new Go();
    WebAssembly.instantiateStreaming(fetch('wasm/rle.wasm'), go.importObject)
        .then(result => go.run(result.instance));
</script>
```

## Connection Parameters

The client connects with URL query parameters:

```
ws://host:port/connect?
    host=rdp-server.example.com&
    user=username&
    password=secret&
    width=1920&
    height=1080&
    colorDepth=32&
    audio=true
```

## Testing

### test.html

Interactive testing page for:
- Protocol message inspection
- Manual connection testing
- Event logging

### integration_test.html

Automated tests for:
- Binary parsing
- Color conversion
- Input encoding
- WASM function correctness

Run with:
```bash
go test ./web/... -v
```

## Building

```bash
# Build WASM module
make wasm

# Bundle JavaScript (if using bundler)
make bundle

# Full build
make build
```

## Related Packages

- `internal/handler` - WebSocket server endpoint
- `internal/codec` - Go codec implementation (mirrored in WASM)
- `cmd/server` - Serves these static files

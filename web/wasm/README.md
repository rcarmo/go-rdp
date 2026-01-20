# web/wasm

TinyGo WebAssembly codec module for browser-side bitmap processing.

## Overview

This package contains Go code that compiles to WebAssembly using TinyGo. It provides high-performance bitmap decompression and color conversion functions that run in the browser.

## Files

| File | Purpose |
|------|---------|
| `main.go` | WASM entry point and exported functions |
| `rle.wasm` | Compiled WebAssembly binary (generated) |

## Why WASM?

JavaScript is not optimal for:
- Bit manipulation in RLE decompression
- Tight loops over large buffers
- Color space math (YCoCg conversion)

TinyGo WASM provides:
- Near-native performance
- Type-safe code sharing with server
- Smaller binary than standard Go WASM (~100KB vs 2MB+)

## Exported Functions

The module exports a global `goRLE` object with these methods:

### Bitmap Processing

```javascript
// Full pipeline: decompress → flip → convert to RGBA
goRLE.processBitmap(
    src,        // Uint8Array - compressed data
    dst,        // Uint8Array - RGBA output buffer
    width,      // number
    height,     // number
    bpp,        // number (8, 15, 16, 24, 32)
    compressed  // boolean
) → boolean
```

### RLE Decompression

```javascript
// 16-bit RLE decompression
goRLE.decompressRLE16(
    src,      // Uint8Array - compressed data
    dst,      // Uint8Array - output buffer
    rowDelta  // number - bytes per row
) → boolean

// Similar for 8, 15, 24 bit...
```

### NSCodec Decoding

```javascript
// NSCodec (YCoCg) decoding
goRLE.decodeNSCodec(
    src,    // Uint8Array - NSCodec stream
    dst,    // Uint8Array - RGBA output
    width,  // number
    height  // number
) → boolean
```

### Color Conversion

```javascript
// RGB565 to RGBA
goRLE.rgb565ToRGBA(src, dst)

// BGR24 to RGBA
goRLE.bgr24ToRGBA(src, dst)

// BGRA32 to RGBA
goRLE.bgra32ToRGBA(src, dst)

// 8-bit palette to RGBA
goRLE.palette8ToRGBA(src, dst)
```

### Utilities

```javascript
// Vertical flip (RDP sends bottom-up)
goRLE.flipVertical(
    data,           // Uint8Array - in-place modification
    width,          // number
    height,         // number
    bytesPerPixel   // number
)

// Set color palette for 8-bit mode
goRLE.setPalette(
    data,      // Uint8Array - RGB palette data
    numColors  // number (max 256)
)
```

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                      JavaScript (Browser)                            │
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │                 Bitmap Update Handler                           ││
│  │                                                                  ││
│  │  // Create typed arrays for data transfer                       ││
│  │  const src = new Uint8Array(compressedData);                    ││
│  │  const dst = new Uint8Array(width * height * 4);                ││
│  │                                                                  ││
│  │  // Call WASM function                                          ││
│  │  const success = goRLE.processBitmap(                           ││
│  │      src, dst, width, height, bpp, compressed                   ││
│  │  );                                                             ││
│  │                                                                  ││
│  │  // Use result                                                  ││
│  │  const imageData = new ImageData(                               ││
│  │      new Uint8ClampedArray(dst.buffer),                         ││
│  │      width, height                                              ││
│  │  );                                                             ││
│  │  ctx.putImageData(imageData, x, y);                             ││
│  └─────────────────────────────────────────────────────────────────┘│
│                                │                                     │
│                                │ js.CopyBytesToGo/JS                 │
│                                ▼                                     │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │                    WASM Module (main.go)                        ││
│  │                                                                  ││
│  │  func processBitmap(this js.Value, args []js.Value) any {       ││
│  │      // Copy JS Uint8Array to Go slice                          ││
│  │      src := make([]byte, args[0].Length())                      ││
│  │      js.CopyBytesToGo(src, args[0])                             ││
│  │                                                                  ││
│  │      // Process using Go codec functions                        ││
│  │      rgba := codec.ProcessBitmap(src, w, h, bpp, compressed)    ││
│  │                                                                  ││
│  │      // Copy result back to JS                                  ││
│  │      js.CopyBytesToJS(args[1], rgba)                            ││
│  │      return true                                                ││
│  │  }                                                              ││
│  └─────────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────────┘
```

## Building

### With Make

```bash
make wasm
```

### Directly with TinyGo

```bash
tinygo build -o web/wasm/rle.wasm -target wasm ./web/wasm/
```

### Build Options

```bash
# Optimized for size
tinygo build -o rle.wasm -target wasm -opt=z ./web/wasm/

# With debug info
tinygo build -o rle.wasm -target wasm -no-debug=false ./web/wasm/
```

## Loading in Browser

```html
<!-- TinyGo WASM runtime (provides Go class) -->
<script src="js/wasm_exec.js"></script>

<script>
    async function loadWasm() {
        const go = new Go();
        
        const result = await WebAssembly.instantiateStreaming(
            fetch('wasm/rle.wasm'),
            go.importObject
        );
        
        // Start Go runtime (registers goRLE global)
        go.run(result.instance);
        
        // Now goRLE.* functions are available
        console.log('WASM loaded:', typeof goRLE);
    }
    
    loadWasm();
</script>
```

## Memory Management

### JavaScript → Go

```go
func myFunc(this js.Value, args []js.Value) any {
    // Get Uint8Array from JS
    jsArray := args[0]
    length := jsArray.Length()
    
    // Allocate Go slice
    goSlice := make([]byte, length)
    
    // Copy data (this is the crossing point)
    js.CopyBytesToGo(goSlice, jsArray)
    
    // Process...
}
```

### Go → JavaScript

```go
func myFunc(this js.Value, args []js.Value) any {
    result := processData()
    
    // Get JS destination buffer
    dstArray := args[1]
    
    // Copy result to JS
    js.CopyBytesToJS(dstArray, result)
    
    return nil
}
```

### Buffer Reuse

For performance, reuse output buffers:

```javascript
// Pre-allocate once
const rgbaBuffer = new Uint8Array(maxWidth * maxHeight * 4);

// Reuse for each update
goRLE.processBitmap(compressed, rgbaBuffer, width, height, bpp, true);
```

## Performance Considerations

### Batch Operations

The `processBitmap` function combines:
1. RLE decompression
2. Vertical flip
3. Color conversion

This is faster than three separate calls because it reduces JS↔WASM boundary crossings.

### Typed Arrays

Always use `Uint8Array` for binary data:
```javascript
// Good
const data = new Uint8Array(buffer);
goRLE.processBitmap(data, ...);

// Bad (conversion overhead)
const data = Array.from(buffer);
```

### Avoid Allocations

TinyGo's GC is simpler than Go's. Minimize allocations in hot paths:
```go
// Good - reuse slice
func (c *Codec) Process(src []byte) {
    if cap(c.buffer) < len(src) {
        c.buffer = make([]byte, len(src))
    }
    c.buffer = c.buffer[:len(src)]
    // use c.buffer...
}
```

## Debugging

### Console Logging

```go
import "syscall/js"

func log(msg string) {
    js.Global().Get("console").Call("log", msg)
}
```

### Build with Debug Info

```bash
tinygo build -o rle.wasm -target wasm -no-debug=false ./web/wasm/
```

## Size Optimization

TinyGo produces smaller binaries than standard Go:

| Compiler | Size | Notes |
|----------|------|-------|
| Go 1.21 | ~2MB | Full runtime |
| TinyGo | ~100KB | Minimal runtime |
| TinyGo -opt=z | ~80KB | Size optimized |

## Related Files

- `web/js/wasm_exec.js` - TinyGo JavaScript runtime
- `web/js/update/bitmap.js` - JavaScript caller
- `internal/codec/` - Go codec implementation (source of truth)

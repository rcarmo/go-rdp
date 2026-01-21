# RemoteFX (RFX) Codec

This package implements the RemoteFX codec decoder as specified in [MS-RDPRFX](https://docs.microsoft.com/en-us/openspecs/windows_protocols/ms-rdprfx/).

## Overview

RemoteFX is Microsoft's tile-based wavelet codec for RDP, providing high-quality lossy compression optimized for remote desktop scenarios. This implementation has been audited against FreeRDP's reference implementation for correctness.

### Key Features

- **64×64 pixel tiles** - Fixed tile size per MS-RDPRFX spec
- **3-level DWT** - 5/3 LeGall wavelet using lifting implementation
- **RLGR entropy coding** - Run-Length Golomb-Rice with RLGR1/RLGR3 modes
- **YCbCr color space** - ICT (Irreversible Color Transform) conversion
- **10 quantization subbands** - Configurable quality per subband
- **TinyGo WASM** - Compiles to WebAssembly for browser-side decoding

## Package Structure

```
rfx/
├── rfx.go            # Types, constants, subband offsets
├── rlgr.go           # RLGR entropy decoder (BitStream + RLGR1/RLGR3)
├── dwt.go            # Inverse discrete wavelet transform (5/3 LeGall)
├── quant.go          # Dequantization with linear buffer layout
├── ycbcr.go          # YCbCr to RGBA conversion (11.5 fixed-point)
├── tile.go           # Tile decoder (main entry point)
├── differential.go   # LL3 differential decode
├── message.go        # RFX message/frame parser
├── AUDIT.md          # FreeRDP comparison audit
└── *_test.go         # Unit tests (84.6% coverage)
```

## Usage

### Decode a single tile

```go
import "github.com/rcarmo/rdp-html5/internal/codec/rfx"

// Get tile data from RFX message (CBT_TILE block starting at 0xCAC3)
tileData := []byte{...}

// Use default quantization
quant := rfx.DefaultQuant()

// Decode tile
tile, err := rfx.DecodeTile(tileData, quant, quant, quant)
if err != nil {
    return err
}

// tile.RGBA contains 64×64×4 = 16384 bytes of RGBA pixel data
// Pixel position: (tile.X * 64, tile.Y * 64)
```

### Decode with pre-allocated buffers (zero-allocation hot path)

```go
// Pre-allocate buffers for WASM/hot path usage
yCoeff := make([]int16, rfx.TilePixels)
cbCoeff := make([]int16, rfx.TilePixels)
crCoeff := make([]int16, rfx.TilePixels)
rgba := make([]byte, rfx.TileRGBASize)

x, y, err := rfx.DecodeTileWithBuffers(
    tileData,
    quantY, quantCb, quantCr,
    yCoeff, cbCoeff, crCoeff,
    rgba,
)
```

### Parse quantization from protocol

```go
// Parse packed 5-byte quant values from RFX_TILESET
quantBytes := data[offset:offset+5]
quant, err := rfx.ParseQuantValues(quantBytes)
```

### WASM Usage (from JavaScript)

```javascript
import { WASMCodec, RFXDecoder } from './wasm.js';

// Initialize WASM module
await WASMCodec.init('js/rle/rle.wasm');

// High-level API with RFXDecoder class
const rfx = new RFXDecoder();
rfx.setQuantRaw(quantData);  // 15 bytes (3 × 5 byte tables)

// Decode and render
const result = rfx.decodeTileToCanvas(tileData, ctx);

// Or low-level API
const outputBuffer = new Uint8Array(16384);
const result = WASMCodec.decodeRFXTile(tileData, outputBuffer);
if (result) {
    const imageData = new ImageData(
        new Uint8ClampedArray(outputBuffer.buffer),
        result.width,
        result.height
    );
    ctx.putImageData(imageData, result.x, result.y);
}
```

## Decoding Pipeline

```
Compressed Tile Data (CBT_TILE block)
        │
        ▼
┌─────────────────────┐
│ 1. RLGR Decode      │  Entropy decode to int16 coefficients
│    (rlgr.go)        │  RLGR1 for Y, RLGR3 for Cb/Cr
└─────────┬───────────┘
          ▼
┌─────────────────────┐
│ 2. Differential     │  Cumulative sum for LL3 subband
│    (differential.go)│  Applied at offset 4032 (DC coefficients)
└─────────┬───────────┘
          ▼
┌─────────────────────┐
│ 3. Dequantize       │  coefficients << (quant - 1)
│    (quant.go)       │  Per-subband, linear buffer layout
└─────────┬───────────┘
          ▼
┌─────────────────────┐
│ 4. Inverse DWT      │  3-level 5/3 LeGall wavelet
│    (dwt.go)         │  Process: L3(3840) → L2(3072) → L1(0)
└─────────┬───────────┘
          ▼
┌─────────────────────┐
│ 5. YCbCr → RGBA     │  11.5 fixed-point arithmetic
│    (ycbcr.go)       │  Y level shift +4096 (128 << 5)
└─────────┬───────────┘
          ▼
    64×64 RGBA Tile (16384 bytes)
```

## Subband Buffer Layout

Coefficients are stored in packed/linear format (not 2D grid):

| Subband | Offset | Size | Dimensions |
|---------|--------|------|------------|
| HL1 | 0 | 1024 | 32×32 |
| LH1 | 1024 | 1024 | 32×32 |
| HH1 | 2048 | 1024 | 32×32 |
| HL2 | 3072 | 256 | 16×16 |
| LH2 | 3328 | 256 | 16×16 |
| HH2 | 3584 | 256 | 16×16 |
| HL3 | 3840 | 64 | 8×8 |
| LH3 | 3904 | 64 | 8×8 |
| HH3 | 3968 | 64 | 8×8 |
| LL3 | 4032 | 64 | 8×8 |

## Constants

| Constant | Value | Description |
|----------|-------|-------------|
| TileSize | 64 | Tile dimension in pixels |
| TilePixels | 4096 | Total pixels per tile |
| TileRGBASize | 16384 | RGBA buffer size (4096 × 4) |
| RLGR1 | 1 | RLGR mode for Y component |
| RLGR3 | 3 | RLGR mode for Cb/Cr components |
| KPMAX | 80 | Maximum kp parameter |
| LSGR | 3 | Log2 scale factor |

## Test Coverage

- **84.6%** statement coverage
- 69 unit tests
- Tests against FreeRDP reference values

Run tests:
```bash
make test
go test -v -cover ./internal/codec/rfx/...
```

## References

- [MS-RDPRFX](https://docs.microsoft.com/en-us/openspecs/windows_protocols/ms-rdprfx/) - Protocol specification
- [FreeRDP codec](https://github.com/FreeRDP/FreeRDP/tree/master/libfreerdp/codec) - Reference C implementation
- [AUDIT.md](./AUDIT.md) - Detailed comparison with FreeRDP

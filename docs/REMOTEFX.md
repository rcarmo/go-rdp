# RemoteFX (RFX) Implementation

This document describes the RemoteFX codec implementation in rdp-html5 for browser-side tile decoding via TinyGo WebAssembly.

## Overview

RemoteFX (MS-RDPRFX) is Microsoft's tile-based wavelet codec for RDP, providing high-quality lossy compression. Our implementation:

- **Go package**: `internal/codec/rfx/` - Core codec algorithms
- **WASM exports**: `web/src/wasm/main.go` - Browser-callable functions
- **JS wrapper**: `web/src/js/wasm.js` - High-level `WASMCodec` and `RFXDecoder` APIs

### Key Characteristics

| Feature | Implementation |
|---------|---------------|
| Tile Size | 64×64 pixels |
| Color Space | YCbCr (ICT) |
| Transform | 3-level 5/3 LeGall DWT |
| Entropy Coding | RLGR1 (Y) / RLGR3 (Cb/Cr) |
| Fixed-Point | 11.5 format for YCbCr→RGB |
| WASM Size | ~381KB (TinyGo optimized) |
| Test Coverage | 84.6% |

---

## Architecture

### Decoding Pipeline

```
RFX_TILE (compressed bitstream)
        │
        ▼
┌─────────────────────┐
│ 1. RLGR Decode      │  Run-Length Golomb-Rice entropy decoding
│    (rlgr.go)        │  → 4096 int16 coefficients per component
└─────────┬───────────┘
          ▼
┌─────────────────────┐
│ 2. Differential     │  Cumulative sum for LL3 (DC) subband
│    (differential.go)│  Applied at offset 4032
└─────────┬───────────┘
          ▼
┌─────────────────────┐
│ 3. Dequantize       │  coefficient << (quantValue - 1)
│    (quant.go)       │  Per-subband, linear buffer layout
└─────────┬───────────┘
          ▼
┌─────────────────────┐
│ 4. Inverse DWT      │  3-level 5/3 LeGall wavelet reconstruction
│    (dwt.go)         │  Process order: L3(3840) → L2(3072) → L1(0)
└─────────┬───────────┘
          ▼
┌─────────────────────┐
│ 5. YCbCr → RGBA     │  Fixed-point conversion (11.5 format)
│    (ycbcr.go)       │  Y level shift: +4096 (128 << 5)
└─────────┬───────────┘
          ▼
    64×64 RGBA tile (16384 bytes)
```

### Subband Buffer Layout (Linear)

Coefficients stored in packed format, NOT 2D grid (matches FreeRDP):

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

---

## File Structure

```
internal/codec/rfx/
├── rfx.go            # Types, constants, SubbandQuant, ParseQuantValues
├── rlgr.go           # BitStream reader, RLGRDecode (RLGR1/RLGR3)
├── dwt.go            # InverseDWT2D, idwt2DBlock, idwt1DRow/Col
├── quant.go          # Dequantize with linear buffer layout
├── ycbcr.go          # YCbCrToRGBA (11.5 fixed-point)
├── tile.go           # DecodeTile, DecodeTileWithBuffers
├── differential.go   # DifferentialDecode for LL3 subband
├── message.go        # ParseRFXMessage, ParseTileset
├── README.md         # Package documentation
├── AUDIT.md          # FreeRDP comparison audit
└── *_test.go         # 69 unit tests

web/src/wasm/main.go      # WASM exports: decodeRFXTile, setRFXQuant
web/src/js/wasm.js    # WASMCodec module, RFXDecoder class
```

---

## Usage

### Go Package

```go
import "github.com/rcarmo/rdp-html5/internal/codec/rfx"

// Decode single tile
quant := rfx.DefaultQuant()
tile, err := rfx.DecodeTile(tileData, quant, quant, quant)
// tile.RGBA = 16384 bytes, tile.X/Y = tile indices

// Zero-allocation hot path
x, y, err := rfx.DecodeTileWithBuffers(
    data, quantY, quantCb, quantCr,
    yCoeff, cbCoeff, crCoeff, rgba,
)
```

### JavaScript (WASM)

```javascript
import { WASMCodec, RFXDecoder } from './wasm.js';

await WASMCodec.init('dist/js/rle/rle.wasm');

// High-level API
const rfx = new RFXDecoder();
rfx.setQuantRaw(quantData);  // 15 bytes
rfx.decodeTileToCanvas(tileData, ctx);

// Low-level API
const output = new Uint8Array(16384);
const result = WASMCodec.decodeRFXTile(tileData, output);
ctx.putImageData(new ImageData(
    new Uint8ClampedArray(output.buffer),
    result.width, result.height
), result.x, result.y);
```

---

## Technical Details

### RLGR Constants (MS-RDPRFX 3.1.8.1.7.1)

| Constant | Value | Description |
|----------|-------|-------------|
| KPMAX | 80 | Maximum kp parameter |
| LSGR | 3 | Log2 scale for GR parameter |
| UP_GR | 4 | Run-length increment |
| DN_GR | 6 | Non-zero decrement |
| UQ_GR | 3 | RLGR1 zero increment |
| DQ_GR | 3 | RLGR1 non-zero decrement |

Initial parameters: `k=1, kp=8, kr=1, krp=8`

### DWT Lifting Equations (FreeRDP-compatible)

```
// 1D inverse for size n
even[i] = low[i] - ((high[i-1] + high[i] + 1) >> 1)
odd[i]  = (high[i] << 1) + ((even[i] + even[i+1]) >> 1)

// Boundary: high[-1] mirrors to high[0], even[n] mirrors to even[n-1]
```

### YCbCr 11.5 Fixed-Point Conversion

```go
// Coefficients scaled by 65536
const (
    CrToR = 91916   // 1.402 * 65536
    CbToG = 22527   // 0.344 * 65536
    CrToG = 46819   // 0.714 * 65536
    CbToB = 115992  // 1.772 * 65536
)

// Y level shift: +4096 (128 << 5 for 11.5 format)
y := int32(yPixels[i]) + 4096
cb := int32(cbPixels[i])
cr := int32(crPixels[i])

// Convert with divisor=16, final shift=21
r := (y*16 + cr*CrToR) >> 21
g := (y*16 - cb*CbToG - cr*CrToG) >> 21
b := (y*16 + cb*CbToB) >> 21
```

---

## Testing

```bash
# Run all tests with coverage
make test

# RFX-specific tests
go test -v -cover ./internal/codec/rfx/...

# Coverage report
go test -coverprofile=coverage.out ./internal/codec/rfx/...
go tool cover -html=coverage.out
```

Current coverage: **84.6%** (69 tests)

---

## References

- [MS-RDPRFX](https://docs.microsoft.com/en-us/openspecs/windows_protocols/ms-rdprfx/) - Microsoft protocol specification
- [FreeRDP](https://github.com/FreeRDP/FreeRDP/tree/master/libfreerdp/codec) - Reference C implementation
- [internal/codec/rfx/AUDIT.md](../internal/codec/rfx/AUDIT.md) - Detailed FreeRDP comparison

---

## Status

✅ **Fully implemented and audited**

- Core codec: RLGR, DWT, dequantization, YCbCr conversion
- WASM exports working (381KB binary)
- JavaScript integration (WASMCodec, RFXDecoder)
- Audited against FreeRDP reference implementation
- 84.6% test coverage
- Server-side enable/disable via `--no-rfx` flag or `RDP_ENABLE_RFX=false`

### Disabling RFX for Testing

To disable RemoteFX and fall back to simpler codecs:

```bash
# Command line
./rdp-html5 -no-rfx

# Environment variable
export RDP_ENABLE_RFX=false
./rdp-html5

# Docker
docker run -e RDP_ENABLE_RFX=false -p 8080:8080 ghcr.io/rcarmo/rdp-html5:latest
```

### Future Work (Protocol Integration)

- RFX capability negotiation in RDP handshake
- Surface command handler for RFX frames
- Tile caching for incremental updates
- Worker thread offloading for parallel decode

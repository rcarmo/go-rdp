# internal/codec

Bitmap codec implementation for RDP graphics decompression.

## Specification References

- [MS-RDPNSC](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-rdpnsc/) - Remote Desktop Protocol: NSCodec Extension
- [MS-RDPEGDI Section 2.2.2.5](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-rdpegdi/8bb25532-9dcd-418f-8b5f-9a01f57d86cb) - Bitmap Compression
- [MS-RDPBCGR Section 2.2.9.1.1.3.1.2.2](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-rdpbcgr/a0674162-3ba7-486f-9c69-05e27e8d0197) - Fast-Path Bitmap Update

## Overview

This package implements RDP bitmap decompression and color conversion utilities. It supports multiple compression formats used by RDP servers:

- **NSCodec** - Microsoft's Network Screen Codec (AYCoCg color space)
- **RDP6 Planar** - Planar bitmap compression for 32bpp
- **Interleaved RLE** - Run-length encoding for 8/15/16/24 bpp
- **Color Conversion** - RGB555, RGB565, BGR24, BGRA32 to RGBA

For RemoteFX (RFX) wavelet codec, see the [`rfx/`](./rfx/) subpackage.

For detailed technical documentation:
- NSCodec: [docs/NSCODEC.md](/docs/NSCODEC.md)
- RemoteFX: [docs/REMOTEFX.md](/docs/REMOTEFX.md)

## Files

| File | Purpose |
|------|---------|
| **NSCodec** ||
| `decoder.go` | High-level NSCodec decoder API |
| `nscodec.go` | NSCodec utilities (RLE, chroma, color space) |
| `nscodec_test.go` | NSCodec unit tests |
| **Planar** ||
| `planar.go` | RDP6 Planar codec decompression |
| `planar_test.go` | Planar codec tests |
| **RLE** ||
| `rle_common.go` | RLE constants and utilities |
| `rle8.go` | 8-bit RLE decompression |
| `rle15.go` | 15-bit RLE decompression |
| `rle16.go` | 16-bit RLE decompression |
| `rle24.go` | 24-bit RLE decompression |
| `rle32.go` | 32-bit handling (delegates to planar) |
| `rle_test.go`, `rle8_test.go` | RLE tests |
| **Utilities** ||
| `bitmap.go` | Flip, palette, color conversion |
| `bitmap_test.go` | Bitmap utility tests |
| `encode.go` | UTF-16 encoding utility |
| `security.go` | Security flag wrapping |
| `security_test.go` | Security tests |

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                       ProcessBitmap()                                │
│                      (unified entry point)                           │
└───────────────────────────────┬─────────────────────────────────────┘
                                │
        ┌───────────────────────┼───────────────────────┐
        │                       │                       │
        ▼                       ▼                       ▼
┌───────────────┐     ┌─────────────────┐     ┌─────────────────────┐
│  NSCodec      │     │  Planar Codec   │     │  Interleaved RLE    │
│  Decode()     │     │  Decompress()   │     │  RLEDecompress*()   │
└───────┬───────┘     └────────┬────────┘     └──────────┬──────────┘
        │                      │                         │
        ▼                      │                         │
┌───────────────┐              │                         │
│ AYCoCg→RGBA   │              │                         │
│ Chroma upsamp │              │                         │
│ Color restore │              │                         │
└───────┬───────┘              │                         │
        │                      │                         │
        └──────────────────────┼─────────────────────────┘
                               │
                               ▼
                    ┌─────────────────────┐
                    │  Color Conversion   │
                    │  (to 32-bit RGBA)   │
                    └──────────┬──────────┘
                               │
                               ▼
                    ┌─────────────────────┐
                    │   FlipVertical()    │
                    │  (bottom-up fix)    │
                    └──────────┬──────────┘
                               │
                               ▼
                    ┌─────────────────────┐
                    │   RGBA Output       │
                    │  (4 bytes/pixel)    │
                    └─────────────────────┘
```

## NSCodec

Microsoft's proprietary codec using YCoCg color space for efficient screen content compression.

### Wire Format

```
NSCODEC_BITMAP_STREAM (20+ bytes header)
├── LumaPlaneByteCount         (4 bytes)
├── OrangeChromaPlaneByteCount (4 bytes)
├── GreenChromaPlaneByteCount  (4 bytes)
├── AlphaPlaneByteCount        (4 bytes)
├── ColorLossLevel             (1 byte, 1-7)
├── ChromaSubsamplingLevel     (1 byte)
├── Reserved                   (2 bytes)
└── Plane data...
```

### Color Space Conversion

```go
// YCoCg to RGB
t := Y - Cg
R := clamp(t + Co)
G := clamp(Y + Cg)
B := clamp(t - Co)
```

### Usage

```go
// High-level API
rgba, err := codec.Decode(nscodecData, width, height)

// Or parse first, then decode
stream, err := codec.ParseBitmapStream(data)
rgba, err := stream.Decode(width, height)
```

## Planar Codec

RDP6 codec that separates color channels into planes.

### Format

```
Header (1 byte):
├── Bits 7-6: Reserved (0)
├── Bit 5: NoAlpha (1 = no alpha plane)
└── Bit 4: RLE (1 = RLE compressed)

Planes (in order): Alpha, Red, Green, Blue
```

### Usage

```go
rgba := codec.DecompressPlanar(src, width, height)
```

## Interleaved RLE

Classic RDP bitmap compression with multiple order codes.

### Order Codes

| Code | Name | Description |
|------|------|-------------|
| 0x0 | BG_RUN | Background color run |
| 0x1 | FG_RUN | Foreground color run |
| 0x2 | FGBG_IMAGE | Bitmask-based image |
| 0x3 | COLOR_RUN | Single color run |
| 0x4 | COLOR_IMAGE | Raw pixel data |
| 0xF0-0xF8 | MEGA_MEGA | Extended 16-bit lengths |
| 0xFD | WHITE | White pixel |
| 0xFE | BLACK | Black pixel |

### Usage

```go
// Per bit-depth
ok := codec.RLEDecompress8(src, dst, rowDelta)
ok := codec.RLEDecompress16(src, dst, rowDelta)
ok := codec.RLEDecompress24(src, dst, rowDelta)
```

## Color Conversion

### Supported Formats

| Source | Function | Bits |
|--------|----------|------|
| Palette | `Palette8ToRGBA()` | 8 |
| RGB555 | `RGB555ToRGBA()` | 15 |
| RGB565 | `RGB565ToRGBA()` | 16 |
| BGR24 | `BGR24ToRGBA()` | 24 |
| BGRA32 | `BGRA32ToRGBA()` | 32 |

### Palette Management

```go
// Set palette from server
codec.SetPalette(paletteData, 256)

// Convert paletted to RGBA
codec.Palette8ToRGBA(src, dst)
```

## Unified Processing

```go
// ProcessBitmap handles all formats uniformly
rgba := codec.ProcessBitmap(
    src,          // Compressed data
    width,        // Bitmap width
    height,       // Bitmap height
    bpp,          // 8, 15, 16, 24, or 32
    isCompressed, // true if RLE/Planar compressed
    rowDelta,     // Row stride for uncompressed
)
```

## Key Design Decisions

### Why bottom-up flip?

RDP servers send bitmaps in bottom-up order (like Windows BMP format). We flip to top-down for standard display APIs that expect top-left origin.

### Why separate RLE files?

Each bit depth has different pixel read/write logic and special handling. Separating them improves code clarity and allows bit-depth-specific optimizations.

### Why AYCoCg color space?

NSCodec uses YCoCg because:
- Better decorrelation than RGB → higher compression
- Integer-only math → fast conversion
- Reversible → lossless mode supported

## Testing

```bash
go test ./internal/codec/...
go test -cover ./internal/codec/...
```

## Related Packages

- `internal/codec/rfx` - RemoteFX wavelet codec (64×64 tiles)
- `internal/rdp` - Uses codecs to process bitmap updates
- `internal/protocol/fastpath` - Delivers compressed bitmaps
- `web/src/wasm` - WASM version of codecs for browser

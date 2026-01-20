# NSCodec and Bitmap Codec Implementation

**Date:** January 20, 2026  
**Version:** 1.0  
**Status:** Implemented

## Table of Contents

1. [Overview](#overview)
2. [Architecture](#architecture)
3. [Supported Codecs](#supported-codecs)
4. [NSCodec Implementation](#nscodec-implementation)
5. [RDP6 Planar Codec](#rdp6-planar-codec)
6. [Interleaved RLE](#interleaved-rle)
7. [Color Space Conversions](#color-space-conversions)
8. [Capability Negotiation](#capability-negotiation)
9. [File Structure](#file-structure)
10. [API Reference](#api-reference)
11. [Testing](#testing)
12. [References](#references)

---

## Overview

This document describes the bitmap codec implementation in our RDP HTML5 client, focusing on NSCodec (Network Screen Codec) and related compression algorithms. The implementation follows Microsoft's RDP specifications and is designed to efficiently decode compressed bitmap data received from RDP servers.

### Goals

- Decode NSCodec bitmap streams from RDP servers
- Support multiple pixel depths (8, 15, 16, 24, 32 bpp)
- Implement RLE decompression for various bitmap formats
- Handle color space conversions (YCoCg, RGB555, RGB565, BGR, BGRA)
- Support chroma subsampling and color loss restoration

### Non-Goals (Current Scope)

- Encoding (client-to-server bitmap compression)
- RemoteFX/RFX codec (see REMOTEFX.md)
- ClearCodec implementation
- Hardware acceleration

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                         RDP Server                                   │
│                                                                      │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐ │
│  │   NSCodec   │  │   Planar    │  │     RLE     │  │    Raw      │ │
│  │  Encoding   │  │   Encoding  │  │  Encoding   │  │   Bitmap    │ │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘ │
└─────────┼────────────────┼────────────────┼────────────────┼────────┘
          │                │                │                │
          ▼                ▼                ▼                ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    Bitmap Update PDU (Wire)                          │
└─────────────────────────────────────────────────────────────────────┘
          │
          ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      Go Codec Package                                │
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │                    ProcessBitmap() Entry Point                  ││
│  └─────────────────────────────────────────────────────────────────┘│
│          │                                                           │
│          ├──────────────┬──────────────┬──────────────┐             │
│          ▼              ▼              ▼              ▼             │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐   │
│  │DecodeNSCodec│ │Decompress   │ │RLEDecompress│ │   Direct    │   │
│  │  ToRGBA()   │ │  Planar()   │ │  8/15/16/24 │ │    Copy     │   │
│  └──────┬──────┘ └──────┬──────┘ └──────┬──────┘ └──────┬──────┘   │
│         │               │               │               │           │
│         ▼               ▼               ▼               ▼           │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │                  Color Conversion Layer                         ││
│  │  AYCoCgToRGBA | BGR24ToRGBA | RGB565ToRGBA | Palette8ToRGBA     ││
│  └─────────────────────────────────────────────────────────────────┘│
│         │                                                           │
│         ▼                                                           │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │                    FlipVertical()                               ││
│  │               (RDP sends bottom-up bitmaps)                     ││
│  └─────────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────────┘
          │
          ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      RGBA Output Buffer                              │
│                    (4 bytes per pixel)                               │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Supported Codecs

| Codec | GUID | Status | Description |
|-------|------|--------|-------------|
| NSCodec | `CA8D1BB9-000F-154F-589F-AE2D1A87E2D6` | ✅ Implemented | Network Screen Codec with YCoCg color space |
| RDP6 Planar | N/A (bitmap flag) | ✅ Implemented | Planar bitmap compression for 32bpp |
| Interleaved RLE | N/A (bitmap flag) | ✅ Implemented | Run-length encoding for 8/15/16/24 bpp |
| RemoteFX | `76772F12-BD72-4463-AFB3-B73C9C6F7886` | ❌ Not Implemented | See REMOTEFX.md |
| ClearCodec | `E4D1C7A4-8C1D-4044-AB27-26AD5A6FE744` | ❌ Not Implemented | Lossless for small regions |

---

## NSCodec Implementation

NSCodec (Network Screen Codec) is Microsoft's proprietary codec for compressing screen content over RDP. It uses the YCoCg color space and supports both lossless and lossy compression.

### Wire Format

The NSCodec bitmap stream follows the `NSCODEC_BITMAP_STREAM` structure defined in MS-RDPNSC:

```
NSCODEC_BITMAP_STREAM
├── LumaPlaneByteCount         (4 bytes, uint32)
├── OrangeChromaPlaneByteCount (4 bytes, uint32)
├── GreenChromaPlaneByteCount  (4 bytes, uint32)
├── AlphaPlaneByteCount        (4 bytes, uint32)
├── ColorLossLevel             (1 byte, uint8, range 1-7)
├── ChromaSubsamplingLevel     (1 byte, uint8, 0=none, 1=enabled)
├── Reserved                   (2 bytes)
├── LumaPlane                  (variable, RLE or raw)
├── OrangeChromaPlane          (variable, RLE or raw)
├── GreenChromaPlane           (variable, RLE or raw)
└── AlphaPlane                 (variable, RLE or raw, optional)
```

### Color Space: AYCoCg

NSCodec uses the YCoCg (Luma, Chroma-Orange, Chroma-Green) color space, which provides:
- Better compression than RGB due to decorrelation
- Simple integer-based conversion (no floating point required)
- Reversible transformation for lossless mode

#### YCoCg to RGB Conversion

```go
// Y = Luma (brightness)
// Co = Orange Chroma (red-blue difference)
// Cg = Green Chroma (green difference)

// Values are stored as unsigned bytes, shifted by 128 for signed range
yVal := int(luma[idx])
coVal := int(co[idx]) - 128  // Shift to signed
cgVal := int(cg[idx]) - 128  // Shift to signed

// Conversion formulas
t := yVal - cgVal
R := clamp(t + coVal)
G := clamp(yVal + cgVal)
B := clamp(t - coVal)
```

### Chroma Subsampling

When enabled (`ChromaSubsamplingLevel != 0`), the chroma planes (Co, Cg) are stored at half resolution:

| Plane | Full Resolution | Subsampled Resolution |
|-------|-----------------|----------------------|
| Luma (Y) | `roundUp(width, 8) × height` | Same |
| Orange (Co) | `width × height` | `lumaWidth/2 × roundUp(height,2)/2` |
| Green (Cg) | `width × height` | `lumaWidth/2 × roundUp(height,2)/2` |
| Alpha (A) | `width × height` | Never subsampled |

Chroma planes are upsampled using nearest-neighbor interpolation during decoding.

### Color Loss Levels

The `ColorLossLevel` field (1-7) controls lossy compression:

| Level | Description | Bit Shift |
|-------|-------------|-----------|
| 1 | Lossless | 0 (no shift) |
| 2 | Minimal loss | 1 |
| 3 | Low loss | 2 |
| 4 | Medium loss | 3 |
| 5 | High loss | 4 |
| 6 | Higher loss | 5 |
| 7 | Maximum loss | 6 |

Restoration is performed by left-shifting the quantized values:
```go
restored := int(value) << (colorLossLevel - 1)
```

### NSCodec RLE Format

NSCodec uses a simple segment-based RLE format (different from Interleaved RLE):

```
Segment Header (1 byte):
├── Bit 7: 1 = Run segment, 0 = Literal segment
└── Bits 0-6: Length (0 = extended length in next byte)

Run Segment:
├── Header with bit 7 set
├── [Extended length if header & 0x7F == 0: next byte + 128]
└── Run value (1 byte, repeated)

Literal Segment:
├── Header with bit 7 clear
├── [Extended length if header == 0: next byte + 128]
└── Raw bytes (length count)

EndData: Last 4 bytes of original plane (appended raw)
```

---

## RDP6 Planar Codec

The RDP6 Planar codec compresses 32-bit RGBA bitmaps by separating color channels into planes, then optionally applying RLE compression to each plane.

### Format Header

```
Format Header (1 byte):
├── Bits 7-6: Reserved (must be 0)
├── Bit 5: NoAlpha (1 = no alpha plane)
├── Bit 4: RLE (1 = planes are RLE compressed)
└── Bits 3-0: Reserved
```

### Plane Order

Planes are stored in this order:
1. Alpha (if present)
2. Red
3. Green
4. Blue

### Planar RLE Encoding

Each scanline is encoded using delta compression relative to the previous scanline:

```
Control Byte:
├── Bits 7-4: cRawBytes (raw/literal byte count)
└── Bits 3-0: nRunLength (run length)

Special cases:
├── nRunLength == 1: Extended run, length = cRawBytes + 16, cRawBytes = 0
└── nRunLength == 2: Extended run, length = cRawBytes + 32, cRawBytes = 0

First scanline: Absolute values
Subsequent scanlines: Delta values with sign-magnitude encoding
├── Bit 0: Sign (1 = negative)
└── Bits 7-1: Magnitude (value >> 1)
```

---

## Interleaved RLE

MS-RDPBCGR defines Interleaved RLE for bitmap compression at 8, 15, 16, and 24 bits per pixel. This is the "classic" RDP bitmap compression.

### Order Codes

| Code | Name | Description |
|------|------|-------------|
| 0x0 | REGULAR_BG_RUN | Background run |
| 0x1 | REGULAR_FG_RUN | Foreground run |
| 0x2 | REGULAR_FGBG_IMAGE | Foreground/background image |
| 0x3 | REGULAR_COLOR_RUN | Single color run |
| 0x4 | REGULAR_COLOR_IMAGE | Color image (raw pixels) |
| 0xC | LITE_SET_FG_FG_RUN | Set foreground + run |
| 0xD | LITE_SET_FG_FGBG_IMAGE | Set foreground + FG/BG image |
| 0xE | LITE_DITHERED_RUN | Dithered run |
| 0xF0-0xF8 | MEGA_MEGA_* | Extended versions with 16-bit length |
| 0xF9 | SPECIAL_FGBG_1 | Special FG/BG (mask 0x03) |
| 0xFA | SPECIAL_FGBG_2 | Special FG/BG (mask 0x05) |
| 0xFD | WHITE | Insert white pixel |
| 0xFE | BLACK | Insert black pixel |

### Run Length Extraction

```go
// Regular codes: length in bits 0-4, 0 means extended (next byte + 32)
// Lite codes: length in bits 0-3, 0 means extended (next byte + 16)
// Mega-mega codes: 16-bit length in next 2 bytes
// FG/BG images: length × 8 (or extended byte + 1)
```

### Bit Depths

| BPP | Bytes per Pixel | Implementation |
|-----|-----------------|----------------|
| 8 | 1 | `rle8.go` |
| 15 | 2 | `rle15.go` |
| 16 | 2 | `rle16.go` |
| 24 | 3 | `rle24.go` |
| 32 | Uses Planar or 24-bit RLE | N/A |

**Note:** MS-RDPBCGR RLE only supports up to 24-bit encoding. For 32-bit color, servers typically use RDP6 Planar codec or send uncompressed data.

---

## Color Space Conversions

### Supported Conversions

| Source Format | Function | Notes |
|---------------|----------|-------|
| 8-bit Indexed | `Palette8ToRGBA()` | Uses 256-color palette |
| RGB555 (15-bit) | `RGB555ToRGBA()` | 5-5-5 bit expansion |
| RGB565 (16-bit) | `RGB565ToRGBA()` | 5-6-5 bit expansion |
| BGR24 (24-bit) | `BGR24ToRGBA()` | Byte swap only |
| BGRA32 (32-bit) | `BGRA32ToRGBA()` | Byte swap only |
| AYCoCg | `AYCoCgToRGBA()` | NSCodec color space |

### Palette Management

The codec package maintains a global 256-color palette initialized to the Windows default system palette:
- Colors 0-9: Windows system colors (black, dark red, etc.)
- Colors 10-225: 6×6×6 color cube (216 colors)
- Colors 226-245: Grayscale ramp
- Colors 246-255: Windows system colors (cream, red, green, etc.)

Servers can update the palette via `SetPalette()`.

### Vertical Flip

RDP servers send bitmaps in bottom-up order (like BMP files). The `FlipVertical()` function corrects this to top-down order for display:

```go
// In-place vertical flip
for i := 0; i < height/2; i++ {
    swap(row[i], row[height-1-i])
}
```

---

## Capability Negotiation

### Advertising NSCodec Support

During capability exchange, the client advertises NSCodec support via `TS_BITMAPCODECS_CAPABILITYSET`:

```go
// From cap_surface.go
NewBitmapCodecsCapabilitySet() CapabilitySet {
    nscodecProps := NSCodecCapabilitySet{
        FAllowDynamicFidelity: 1,  // Allow server to adjust quality
        FAllowSubsampling:     1,  // Allow chroma subsampling
        ColorLossLevel:        3,  // Moderate compression
    }
    return CapabilitySet{
        CapabilitySetType: CapabilitySetTypeBitmapCodecs,
        BitmapCodecsCapabilitySet: &BitmapCodecsCapabilitySet{
            BitmapCodecArray: []BitmapCodec{{
                CodecGUID:       NSCodecGUID,
                CodecID:         1,  // Assigned by server
                CodecProperties: nscodecProps.Serialize(),
            }},
        },
    }
}
```

### Surface Commands Capability

For surface-based rendering (used with codecs), the client advertises:

```go
NewSurfaceCommandsCapabilitySet() CapabilitySet {
    return CapabilitySet{
        CapabilitySetType: CapabilitySetTypeSurfaceCommands,
        SurfaceCommandsCapabilitySet: &SurfaceCommandsCapabilitySet{
            CmdFlags: SurfCmdSetSurfaceBits |    // Set Surface Bits
                      SurfCmdFrameMarker |        // Frame markers
                      SurfCmdStreamSurfBits,      // Stream Surface Bits
        },
    }
}
```

---

## File Structure

```
internal/codec/
├── decoder.go          # NSCodec decoder (structured API)
├── nscodec.go          # NSCodec utilities (RLE, color conversion)
├── nscodec_test.go     # NSCodec tests
├── planar.go           # RDP6 Planar codec
├── planar_test.go      # Planar codec tests
├── bitmap.go           # Bitmap utilities and color conversions
├── bitmap_test.go      # Bitmap tests
├── rle_common.go       # Interleaved RLE definitions
├── rle8.go             # 8-bit RLE decompression
├── rle8_test.go        # 8-bit RLE tests
├── rle15.go            # 15-bit RLE decompression
├── rle16.go            # 16-bit RLE decompression
├── rle24.go            # 24-bit RLE decompression
├── rle32.go            # 32-bit RLE (delegates to planar)
├── rle_test.go         # RLE integration tests
├── encode.go           # Bitmap encoding for client-to-server
├── security.go         # Security/encryption utilities
└── security_test.go    # Security tests

internal/protocol/pdu/
├── cap_surface.go      # Surface and codec capabilities
└── ...
```

---

## API Reference

### Primary Entry Point

```go
// ProcessBitmap handles all bitmap formats uniformly
func ProcessBitmap(
    src []byte,           // Compressed/raw bitmap data
    width, height int,    // Bitmap dimensions
    bpp int,              // Bits per pixel (8, 15, 16, 24, 32)
    isCompressed bool,    // True if data is RLE/Planar compressed
    rowDelta int,         // Row stride (for uncompressed)
) []byte                  // Returns RGBA output (4 bytes per pixel)
```

### NSCodec API

```go
// High-level decoder
func Decode(data []byte, width, height int) ([]byte, error)

// Structured parsing
func ParseBitmapStream(data []byte) (*BitmapStream, error)
func (s *BitmapStream) Decode(width, height int) ([]byte, error)

// Alternative direct decoder
func DecodeNSCodecToRGBA(data []byte, width, height int) []byte
```

### Planar Codec API

```go
func DecompressPlanar(src []byte, width, height int) []byte
```

### RLE API

```go
func RLEDecompress8(src, dst []byte, rowDelta int) bool
func RLEDecompress15(src, dst []byte, rowDelta int) bool
func RLEDecompress16(src, dst []byte, rowDelta int) bool
func RLEDecompress24(src, dst []byte, rowDelta int) bool
```

### Color Conversion API

```go
func Palette8ToRGBA(src, dst []byte)
func RGB555ToRGBA(src, dst []byte)
func RGB565ToRGBA(src, dst []byte)
func BGR24ToRGBA(src, dst []byte)
func BGRA32ToRGBA(src, dst []byte)
func AYCoCgToRGBA(luma, co, cg, alpha []byte, ...) []byte
```

### Utility Functions

```go
func FlipVertical(data []byte, width, height, bytesPerPixel int)
func SetPalette(data []byte, numColors int)
func ChromaSuperSample(plane []byte, srcW, srcH, dstW, dstH int) []byte
func RestoreColorLoss(plane []byte, colorLossLevel uint8) []byte
```

---

## Testing

### Test Coverage

| File | Coverage | Notes |
|------|----------|-------|
| decoder.go | ~80% | Structured NSCodec API |
| nscodec.go | ~85% | Core NSCodec functions |
| planar.go | ~90% | Planar codec |
| bitmap.go | ~85% | Color conversions |
| rle8.go | ~80% | 8-bit RLE |
| rle_common.go | ~75% | RLE utilities |

### Running Tests

```bash
# Run all codec tests
go test ./internal/codec/...

# Run with coverage
go test -cover ./internal/codec/...

# Verbose output
go test -v ./internal/codec/...
```

### Test Data

Tests use both synthetic test vectors and captured real-world data from RDP sessions. Key test scenarios:

1. **NSCodec decoding**: Various chroma subsampling and color loss levels
2. **Planar codec**: RLE and raw plane variants
3. **RLE edge cases**: Mega-mega codes, special codes, boundary conditions
4. **Color conversion**: All supported pixel formats
5. **Vertical flip**: Odd/even heights, various bytes per pixel

---

## References

### Microsoft Specifications

- **MS-RDPBCGR** - Remote Desktop Protocol: Basic Connectivity and Graphics Remoting
  - Section 2.2.9.1.1.3.1.2.4: Interleaved RLE Bitmap Compression
  - Section 2.2.9.2.2: Bitmap Capability Set
  - Section 2.2.7.1.6: Bitmap Codecs Capability Set

- **MS-RDPNSC** - Remote Desktop Protocol: NSCodec Extension
  - Section 2: Protocol Overview
  - Section 3: Protocol Details

- **MS-RDPEGDI** - Remote Desktop Protocol: Graphics Device Interface (GDI) Acceleration Extensions
  - Section 2.2.2.5: RDP6 Planar Bitmap Compression

### Related Implementation

- **FreeRDP** (open source RDP client)
  - `libfreerdp/codec/planar.c` - Planar codec reference
  - `libfreerdp/codec/nsc.c` - NSCodec reference
  - `libfreerdp/codec/interleaved.c` - RLE reference

### Color Space References

- "YCoCg-R: A Color Space with RGB Reversibility and Low Dynamic Range" - Malvar & Sullivan
- ITU-T H.264 Annex E - YCoCg color space definition

---

## Changelog

| Date | Version | Changes |
|------|---------|---------|
| 2026-01-20 | 1.0 | Initial documentation |

---

## Future Work

1. **RemoteFX Support** - See REMOTEFX.md for implementation plan
2. **ClearCodec** - Lossless codec for small UI elements
3. **Hardware Acceleration** - WebGL-based color conversion
4. **Encoding** - Client-to-server bitmap compression for RemoteApp


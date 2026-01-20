# RemoteFX (RFX) Implementation Guide

This document describes what would be needed to implement RemoteFX rendering in our RDP-to-HTML5 gateway using TinyGo compiled to WebAssembly.

## Table of Contents

1. [Overview](#overview)
2. [Protocol Specification](#protocol-specification)
3. [Codec Architecture](#codec-architecture)
4. [Implementation Components](#implementation-components)
5. [TinyGo WASM Considerations](#tinygo-wasm-considerations)
6. [Data Flow](#data-flow)
7. [File Structure](#file-structure)
8. [Performance Optimization](#performance-optimization)
9. [Testing Strategy](#testing-strategy)
10. [References](#references)

---

## Overview

RemoteFX (RFX) is Microsoft's progressive graphics codec for RDP, specified in **MS-RDPRFX**. It provides high-quality, lossy image compression optimized for remote desktop scenarios.

### Key Characteristics

| Feature | Description |
|---------|-------------|
| **Tile-based** | Images divided into 64×64 pixel tiles |
| **Color Space** | YCbCr (YCoCg variant in some modes) |
| **Transform** | Discrete Wavelet Transform (DWT) |
| **Entropy Coding** | RLGR (Run-Length Golomb-Rice) |
| **Progressive** | Supports progressive quality refinement |
| **Lossy** | Configurable quality via quantization |

### Why WASM?

- **CPU-intensive**: DWT and RLGR decoding are computationally expensive
- **Browser limitation**: JavaScript is single-threaded and slow for bitwise operations
- **Memory efficiency**: WASM provides direct memory access without GC pauses
- **TinyGo advantage**: Produces small WASM binaries (~300-500KB for codec)

---

## Protocol Specification

### Capability Negotiation

RemoteFX must be negotiated during RDP capability exchange:

```go
// GCC Conference Create Request
type RFXClientCapsContainer struct {
    Length         uint16
    CaptureFlags   uint32  // 0x00000001 = CARDP_CAPS_CAPTURE_NON_CAC
    CapsLength     uint32
    CapsData       []RFXCaps
}

type RFXCaps struct {
    BlockType      uint16  // CBY_CAPS (0xCBC0)
    BlockLen       uint32
    NumCapsets     uint16
    Capsets        []RFXCapset
}

type RFXCapset struct {
    BlockType      uint16  // CBT_CAPSET (0xCBC1)
    BlockLen       uint32
    CodecId        uint8   // 0x01
    CapsetType     uint16  // CLY_CAPSET (0xCFC0)
    NumIcaps       uint16
    IcapLen        uint16
    Icaps          []RFXIcap
}

type RFXIcap struct {
    Version        uint16  // CLW_VERSION_1_0 (0x0100)
    TileSize       uint16  // CT_TILE_64x64 (0x0040)
    Flags          uint8   // CODEC_MODE flags
    ColConvBits    uint8   // CLW_COL_CONV_ICT (0x01)
    TransformBits  uint8   // CLW_XFORM_DWT_53_A (0x01)
    EntropyBits    uint8   // CLW_ENTROPY_RLGR1 (0x01) or RLGR3 (0x04)
}
```

### Message Structure

RFX data arrives via Surface Commands in fastpath updates:

```
┌─────────────────────────────────────────────────────────┐
│ Fast-Path Update PDU                                    │
├─────────────────────────────────────────────────────────┤
│ ├── updateHeader                                        │
│ ├── compressionFlags                                    │
│ └── updateData                                          │
│     └── Surface Command (CMDTYPE_STREAM_SURFACE_BITS)   │
│         ├── destLeft, destTop, destRight, destBottom    │
│         ├── bpp, codecID, width, height                 │
│         └── bitmapData (RFX encoded)                    │
│             ├── RFX_SYNC                                │
│             ├── RFX_CODEC_VERSIONS                      │
│             ├── RFX_CHANNELS                            │
│             ├── RFX_CONTEXT                             │
│             ├── RFX_FRAME_BEGIN                         │
│             ├── RFX_REGION                              │
│             ├── RFX_TILESET                             │
│             │   └── RFX_TILE[] (encoded tiles)          │
│             └── RFX_FRAME_END                           │
└─────────────────────────────────────────────────────────┘
```

### Block Types

```go
const (
    // Sync and context blocks
    WBT_SYNC           = 0xCCC0
    WBT_CODEC_VERSIONS = 0xCCC1
    WBT_CHANNELS       = 0xCCC2
    WBT_CONTEXT        = 0xCCC3
    
    // Frame blocks
    WBT_FRAME_BEGIN    = 0xCCC4
    WBT_FRAME_END      = 0xCCC5
    WBT_REGION         = 0xCCC6
    WBT_EXTENSION      = 0xCCC7
    
    // Tile blocks
    WBT_TILESET        = 0xCAC2
    CBT_TILE           = 0xCAC3
)
```

---

## Codec Architecture

### Decoding Pipeline

```
┌──────────────────────────────────────────────────────────────────┐
│                     RFX Decoding Pipeline                        │
└──────────────────────────────────────────────────────────────────┘

Input: RFX_TILE (compressed bitstream)
        │
        ▼
┌───────────────────┐
│ 1. RLGR Decode    │  Entropy decoding of quantized coefficients
│    (per component)│  Y, Cb, Cr components decoded separately
└─────────┬─────────┘
          │ Quantized DWT coefficients (3 arrays)
          ▼
┌───────────────────┐
│ 2. Dequantize     │  Apply inverse quantization
│    (per subband)  │  Different quant values for each subband
└─────────┬─────────┘
          │ DWT coefficients
          ▼
┌───────────────────┐
│ 3. Inverse DWT    │  2D inverse discrete wavelet transform
│    (2 levels)     │  Reconstruct spatial domain data
└─────────┬─────────┘
          │ YCbCr pixel data (64×64)
          ▼
┌───────────────────┐
│ 4. Color Convert  │  YCbCr to RGB conversion
│    (ICT inverse)  │  Clamp to [0, 255]
└─────────┬─────────┘
          │ RGB pixel data (64×64)
          ▼
Output: 64×64 RGBA tile → Canvas
```

### Subband Structure

The DWT produces a hierarchical structure of subbands:

```
Level 2 DWT (16×16 coefficients each):
┌────────┬────────┐
│   LL2  │   HL2  │
│ (DC)   │(Horiz) │
├────────┼────────┤
│   LH2  │   HH2  │
│(Vert)  │(Diag)  │
└────────┴────────┘

Level 1 DWT (32×32 coefficients each):
┌────────────────┬────────────────┐
│                │                │
│   (from L2)    │      HL1       │
│                │                │
├────────────────┼────────────────┤
│                │                │
│      LH1       │      HH1       │
│                │                │
└────────────────┴────────────────┘

Full tile (64×64 pixels):
┌────────────────────────────────┐
│                                │
│     Reconstructed from         │
│     inverse DWT                │
│                                │
└────────────────────────────────┘
```

### Quantization Table

Each subband has separate quantization factors:

```go
type QuantizationValues struct {
    LL3 uint8  // Approximation (DC)
    LH3 uint8  // Level 3 subbands
    HL3 uint8
    HH3 uint8
    LH2 uint8  // Level 2 subbands
    HL2 uint8
    HH2 uint8
    LH1 uint8  // Level 1 subbands
    HL1 uint8
    HH1 uint8
}

// Quantization: coefficient >> quantValue
// Dequantization: coefficient << quantValue
```

---

## Implementation Components

### 1. RLGR Decoder

RLGR (Run-Length Golomb-Rice) is the entropy coding used by RemoteFX.

```go
// internal/codec/rfx/rlgr.go

package rfx

// RLGR mode constants
const (
    RLGR1 = 1  // Mode used for Y component
    RLGR3 = 3  // Mode used for Cb, Cr components
)

// RLGRDecoder decodes RLGR-encoded data
type RLGRDecoder struct {
    data    []byte
    bitPos  int
    bytePos int
    mode    int
}

// NewRLGRDecoder creates a decoder for the given data
func NewRLGRDecoder(data []byte, mode int) *RLGRDecoder {
    return &RLGRDecoder{
        data: data,
        mode: mode,
    }
}

// Decode decodes the bitstream into coefficients
func (d *RLGRDecoder) Decode(output []int16, count int) error {
    k := 1  // Golomb-Rice parameter
    kp := 0 // Adaptive parameter (scaled by 8)
    kr := 0 // Run length parameter
    krp := 0
    
    idx := 0
    for idx < count {
        // Check for run of zeros
        if d.mode == RLGR1 {
            // RLGR1: Separate run-length and value coding
            runLength := d.decodeRun(&kr, &krp)
            for i := 0; i < runLength && idx < count; i++ {
                output[idx] = 0
                idx++
            }
            if idx >= count {
                break
            }
        }
        
        // Decode Golomb-Rice value
        value := d.decodeGR(k)
        
        // Update adaptive parameter
        if value == 0 {
            kp = max(0, kp-2)
        } else {
            kp = min(kp+value, 80) // Cap at 80/8 = 10
        }
        k = kp >> 3
        
        // Sign extension
        if value != 0 {
            if d.readBit() == 1 {
                value = -value
            }
        }
        
        output[idx] = int16(value)
        idx++
    }
    
    return nil
}

// decodeGR decodes a Golomb-Rice coded value
func (d *RLGRDecoder) decodeGR(k int) int {
    // Unary prefix (quotient)
    q := 0
    for d.readBit() == 0 {
        q++
    }
    
    // Binary suffix (remainder)
    r := 0
    for i := 0; i < k; i++ {
        r = (r << 1) | d.readBit()
    }
    
    return (q << k) + r
}

// decodeRun decodes a run length
func (d *RLGRDecoder) decodeRun(kr, krp *int) int {
    // Similar to GR coding but for run lengths
    nIdx := 0
    for d.readBit() == 0 {
        nIdx++
    }
    
    runLength := 0
    if nIdx < 32 {
        runLength = (1 << *kr) * nIdx
        // Read remainder bits
        for i := 0; i < *kr; i++ {
            runLength += d.readBit() << i
        }
    }
    
    // Update adaptive run-length parameter
    if nIdx == 0 {
        *krp = max(0, *krp-2)
    } else {
        *krp = min(*krp+nIdx, 80)
    }
    *kr = *krp >> 3
    
    return runLength
}

// readBit reads a single bit from the stream
func (d *RLGRDecoder) readBit() int {
    if d.bytePos >= len(d.data) {
        return 0
    }
    
    bit := (d.data[d.bytePos] >> (7 - d.bitPos)) & 1
    d.bitPos++
    if d.bitPos >= 8 {
        d.bitPos = 0
        d.bytePos++
    }
    return int(bit)
}
```

### 2. Discrete Wavelet Transform (Inverse)

```go
// internal/codec/rfx/dwt.go

package rfx

const TileSize = 64

// InverseDWT2D performs 2-level inverse DWT on a 64×64 tile
func InverseDWT2D(coefficients []int16) []int16 {
    // Working buffer
    temp := make([]int16, TileSize*TileSize)
    copy(temp, coefficients)
    
    // Level 2 inverse (16×16 -> 32×32)
    inverseDWTLevel(temp, 16)
    
    // Level 1 inverse (32×32 -> 64×64)
    inverseDWTLevel(temp, 32)
    
    return temp
}

// inverseDWTLevel performs one level of inverse DWT
func inverseDWTLevel(data []int16, size int) {
    // Horizontal pass
    row := make([]int16, size*2)
    for y := 0; y < size*2; y++ {
        // Extract row
        for x := 0; x < size*2; x++ {
            row[x] = data[y*TileSize+x]
        }
        // Inverse transform
        idwtRow(row, size)
        // Store back
        for x := 0; x < size*2; x++ {
            data[y*TileSize+x] = row[x]
        }
    }
    
    // Vertical pass
    col := make([]int16, size*2)
    for x := 0; x < size*2; x++ {
        // Extract column
        for y := 0; y < size*2; y++ {
            col[y] = data[y*TileSize+x]
        }
        // Inverse transform
        idwtRow(col, size)
        // Store back
        for y := 0; y < size*2; y++ {
            data[y*TileSize+x] = col[y]
        }
    }
}

// idwtRow performs 1D inverse DWT (Le Gall 5/3 wavelet)
func idwtRow(data []int16, halfSize int) {
    // Split into low and high frequency
    low := make([]int16, halfSize)
    high := make([]int16, halfSize)
    
    copy(low, data[:halfSize])
    copy(high, data[halfSize:halfSize*2])
    
    // Undo predict step
    // even[n] = low[n] + floor((high[n-1] + high[n] + 2) / 4)
    for n := 0; n < halfSize; n++ {
        h0 := high[max(0, n-1)]
        h1 := high[n]
        data[n*2] = low[n] + (h0+h1+2)>>2
    }
    
    // Undo update step
    // odd[n] = high[n] + floor((even[n] + even[n+1]) / 2)
    for n := 0; n < halfSize; n++ {
        e0 := data[n*2]
        e1 := data[min((n+1)*2, halfSize*2-2)]
        data[n*2+1] = high[n] + (e0+e1)>>1
    }
}
```

### 3. Dequantization

```go
// internal/codec/rfx/quant.go

package rfx

// SubbandQuant holds quantization values for all subbands
type SubbandQuant struct {
    LL3, LH3, HL3, HH3 uint8
    LH2, HL2, HH2      uint8
    LH1, HL1, HH1      uint8
}

// Dequantize applies inverse quantization to coefficients
func Dequantize(coefficients []int16, quant *SubbandQuant) {
    // LL3 subband (0-15, 0-15)
    dequantSubband(coefficients, 0, 0, 16, 16, quant.LL3)
    
    // Level 3 subbands
    dequantSubband(coefficients, 16, 0, 16, 16, quant.HL3)
    dequantSubband(coefficients, 0, 16, 16, 16, quant.LH3)
    dequantSubband(coefficients, 16, 16, 16, 16, quant.HH3)
    
    // Level 2 subbands
    dequantSubband(coefficients, 32, 0, 32, 32, quant.HL2)
    dequantSubband(coefficients, 0, 32, 32, 32, quant.LH2)
    dequantSubband(coefficients, 32, 32, 32, 32, quant.HH2)
    
    // Level 1 subbands (in expanded area after inverse L2 DWT)
    // These are applied during progressive decode
}

func dequantSubband(data []int16, x, y, w, h int, shift uint8) {
    for dy := 0; dy < h; dy++ {
        for dx := 0; dx < w; dx++ {
            idx := (y+dy)*TileSize + (x + dx)
            data[idx] = data[idx] << shift
        }
    }
}
```

### 4. Color Conversion

```go
// internal/codec/rfx/color.go

package rfx

// YCbCrToRGB converts YCbCr tile data to RGB
// Uses ICT (Irreversible Color Transform) from JPEG 2000
func YCbCrToRGB(y, cb, cr []int16, output []byte) {
    for i := 0; i < TileSize*TileSize; i++ {
        // ICT inverse transform
        // R = Y + 1.402 * Cr
        // G = Y - 0.344136 * Cb - 0.714136 * Cr  
        // B = Y + 1.772 * Cb
        
        // Fixed-point arithmetic (scaled by 4096)
        yVal := int(y[i]) + 128  // DC level shift
        cbVal := int(cb[i])
        crVal := int(cr[i])
        
        r := yVal + (5765*crVal+2048)>>12
        g := yVal - (1410*cbVal+2048)>>12 - (2925*crVal+2048)>>12
        b := yVal + (7258*cbVal+2048)>>12
        
        // Clamp to [0, 255]
        output[i*4+0] = clampByte(r)
        output[i*4+1] = clampByte(g)
        output[i*4+2] = clampByte(b)
        output[i*4+3] = 255 // Alpha
    }
}

func clampByte(v int) byte {
    if v < 0 {
        return 0
    }
    if v > 255 {
        return 255
    }
    return byte(v)
}
```

### 5. Tile Decoder (Main Entry Point)

```go
// internal/codec/rfx/tile.go

package rfx

import (
    "encoding/binary"
    "errors"
)

var (
    ErrInvalidTile   = errors.New("invalid RFX tile data")
    ErrInvalidQuant  = errors.New("invalid quantization values")
)

// Tile represents a 64×64 decoded tile
type Tile struct {
    X, Y   int
    Width  int
    Height int
    RGBA   []byte // 64*64*4 bytes
}

// DecodeTile decodes a single RFX tile
func DecodeTile(data []byte, quantY, quantCb, quantCr *SubbandQuant) (*Tile, error) {
    if len(data) < 6 {
        return nil, ErrInvalidTile
    }
    
    // Parse tile header
    blockType := binary.LittleEndian.Uint16(data[0:2])
    if blockType != CBT_TILE {
        return nil, ErrInvalidTile
    }
    
    blockLen := binary.LittleEndian.Uint32(data[2:6])
    if int(blockLen) > len(data) {
        return nil, ErrInvalidTile
    }
    
    // Tile coordinates and component sizes
    offset := 6
    // quantIdxY := data[offset]
    // quantIdxCb := data[offset+1]
    // quantIdxCr := data[offset+2]
    offset += 3
    
    xIdx := binary.LittleEndian.Uint16(data[offset:])
    offset += 2
    yIdx := binary.LittleEndian.Uint16(data[offset:])
    offset += 2
    
    yLen := binary.LittleEndian.Uint16(data[offset:])
    offset += 2
    cbLen := binary.LittleEndian.Uint16(data[offset:])
    offset += 2
    crLen := binary.LittleEndian.Uint16(data[offset:])
    offset += 2
    
    // Extract component data
    yData := data[offset : offset+int(yLen)]
    offset += int(yLen)
    cbData := data[offset : offset+int(cbLen)]
    offset += int(cbLen)
    crData := data[offset : offset+int(crLen)]
    
    // Decode each component
    yCoeff := make([]int16, TileSize*TileSize)
    cbCoeff := make([]int16, TileSize*TileSize)
    crCoeff := make([]int16, TileSize*TileSize)
    
    // RLGR decode
    NewRLGRDecoder(yData, RLGR1).Decode(yCoeff, TileSize*TileSize)
    NewRLGRDecoder(cbData, RLGR3).Decode(cbCoeff, TileSize*TileSize)
    NewRLGRDecoder(crData, RLGR3).Decode(crCoeff, TileSize*TileSize)
    
    // Dequantize
    Dequantize(yCoeff, quantY)
    Dequantize(cbCoeff, quantCb)
    Dequantize(crCoeff, quantCr)
    
    // Inverse DWT
    yPixels := InverseDWT2D(yCoeff)
    cbPixels := InverseDWT2D(cbCoeff)
    crPixels := InverseDWT2D(crCoeff)
    
    // Color convert
    rgba := make([]byte, TileSize*TileSize*4)
    YCbCrToRGB(yPixels, cbPixels, crPixels, rgba)
    
    return &Tile{
        X:      int(xIdx) * TileSize,
        Y:      int(yIdx) * TileSize,
        Width:  TileSize,
        Height: TileSize,
        RGBA:   rgba,
    }, nil
}
```

---

## TinyGo WASM Considerations

### Build Configuration

```makefile
# Makefile additions for RFX WASM

build-rfx-wasm:
	@echo "Building RemoteFX WASM module..."
	cd internal/codec/rfx && tinygo build \
		-o ../../../web/js/rfx/rfx.wasm \
		-target wasm \
		-no-debug \
		-opt=2 \
		-gc=leaking \
		-scheduler=none \
		./wasm/main.go
	@cp $$(tinygo env TINYGOROOT)/targets/wasm_exec.js web/js/rfx/
	@ls -lh web/js/rfx/rfx.wasm
```

### WASM Entry Points

```go
// internal/codec/rfx/wasm/main.go

//go:build wasm

package main

import (
    "unsafe"
    
    "github.com/rcarmo/rdp-html5/internal/codec/rfx"
)

// Memory for tile decoding
var (
    inputBuffer  [65536]byte  // Input compressed data
    outputBuffer [16384]byte  // Output RGBA (64*64*4)
    quantBuffer  [30]byte     // Quantization values
)

//export getInputBuffer
func getInputBuffer() *byte {
    return &inputBuffer[0]
}

//export getOutputBuffer  
func getOutputBuffer() *byte {
    return &outputBuffer[0]
}

//export getQuantBuffer
func getQuantBuffer() *byte {
    return &quantBuffer[0]
}

//export decodeTile
func decodeTile(inputLen int) int {
    // Parse quantization from buffer
    quantY := parseQuant(quantBuffer[0:10])
    quantCb := parseQuant(quantBuffer[10:20])
    quantCr := parseQuant(quantBuffer[20:30])
    
    // Decode tile
    tile, err := rfx.DecodeTile(
        inputBuffer[:inputLen],
        quantY, quantCb, quantCr,
    )
    if err != nil {
        return -1
    }
    
    // Copy to output buffer
    copy(outputBuffer[:], tile.RGBA)
    
    return len(tile.RGBA)
}

func parseQuant(data []byte) *rfx.SubbandQuant {
    return &rfx.SubbandQuant{
        LL3: data[0], LH3: data[1], HL3: data[2], HH3: data[3],
        LH2: data[4], HL2: data[5], HH2: data[6],
        LH1: data[7], HL1: data[8], HH1: data[9],
    }
}

func main() {}
```

### JavaScript Integration

```javascript
// web/js/rfx/rfx.js

class RFXDecoder {
    constructor() {
        this.wasm = null;
        this.memory = null;
        this.inputPtr = null;
        this.outputPtr = null;
        this.quantPtr = null;
    }

    async init() {
        const go = new Go();
        const result = await WebAssembly.instantiateStreaming(
            fetch('js/rfx/rfx.wasm'),
            go.importObject
        );
        
        this.wasm = result.instance;
        this.memory = this.wasm.exports.memory;
        
        // Get buffer pointers
        this.inputPtr = this.wasm.exports.getInputBuffer();
        this.outputPtr = this.wasm.exports.getOutputBuffer();
        this.quantPtr = this.wasm.exports.getQuantBuffer();
        
        // Don't call go.run() for TinyGo - it's not needed
    }

    decodeTile(compressedData, quantY, quantCb, quantCr) {
        // Copy quantization values
        const quantView = new Uint8Array(this.memory.buffer, this.quantPtr, 30);
        quantView.set(this.packQuant(quantY), 0);
        quantView.set(this.packQuant(quantCb), 10);
        quantView.set(this.packQuant(quantCr), 20);
        
        // Copy compressed data
        const inputView = new Uint8Array(this.memory.buffer, this.inputPtr, compressedData.length);
        inputView.set(compressedData);
        
        // Decode
        const resultLen = this.wasm.exports.decodeTile(compressedData.length);
        if (resultLen < 0) {
            throw new Error('RFX decode failed');
        }
        
        // Copy result
        const outputView = new Uint8Array(this.memory.buffer, this.outputPtr, resultLen);
        return new Uint8Array(outputView);
    }

    packQuant(quant) {
        return new Uint8Array([
            quant.LL3, quant.LH3, quant.HL3, quant.HH3,
            quant.LH2, quant.HL2, quant.HH2,
            quant.LH1, quant.HL1, quant.HH1
        ]);
    }

    decodeFrame(rfxData, canvas) {
        const ctx = canvas.getContext('2d');
        const tiles = this.parseRFXMessage(rfxData);
        
        for (const tile of tiles) {
            const rgba = this.decodeTile(
                tile.data,
                tile.quantY,
                tile.quantCb, 
                tile.quantCr
            );
            
            // Create ImageData and draw
            const imageData = new ImageData(
                new Uint8ClampedArray(rgba),
                64, 64
            );
            ctx.putImageData(imageData, tile.x, tile.y);
        }
    }

    parseRFXMessage(data) {
        // Parse RFX message structure
        // Returns array of tile objects
        const tiles = [];
        let offset = 0;
        
        while (offset < data.length) {
            const blockType = data[offset] | (data[offset + 1] << 8);
            const blockLen = data[offset + 2] | (data[offset + 3] << 8) |
                           (data[offset + 4] << 16) | (data[offset + 5] << 24);
            
            switch (blockType) {
                case 0xCAC3: // CBT_TILE
                    tiles.push(this.parseTile(data.slice(offset, offset + blockLen)));
                    break;
                // Handle other block types...
            }
            
            offset += blockLen;
        }
        
        return tiles;
    }
}

// Export for use
window.RFXDecoder = RFXDecoder;
```

### TinyGo Limitations

| Limitation | Workaround |
|------------|------------|
| No `reflect` | Use concrete types, no JSON marshal |
| Limited `math` | Implement needed functions manually |
| No goroutines in WASM | Single-threaded decoding |
| GC can be slow | Use `-gc=leaking` for short-lived allocations |
| No `fmt.Sprintf` | Use simple error types |
| Binary size | Use `-opt=2 -no-debug` |

### Memory Management

```go
// Use sync.Pool for tile buffers to reduce allocations
var tilePool = sync.Pool{
    New: func() interface{} {
        return &TileBuffer{
            YCoeff:  make([]int16, TileSize*TileSize),
            CbCoeff: make([]int16, TileSize*TileSize),
            CrCoeff: make([]int16, TileSize*TileSize),
            RGBA:    make([]byte, TileSize*TileSize*4),
        }
    },
}

type TileBuffer struct {
    YCoeff, CbCoeff, CrCoeff []int16
    RGBA                      []byte
}

func (b *TileBuffer) Reset() {
    // Clear buffers for reuse
    for i := range b.YCoeff {
        b.YCoeff[i] = 0
        b.CbCoeff[i] = 0
        b.CrCoeff[i] = 0
    }
}
```

---

## Data Flow

### Server to Client

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Data Flow Diagram                            │
└─────────────────────────────────────────────────────────────────────┘

RDP Server
    │
    │ RFX encoded frame (Surface Command)
    ▼
┌─────────────────────┐
│   Go Backend        │
│   (internal/rdp)    │
│   - Parse fastpath  │
│   - Extract surface │
│     commands        │
└──────────┬──────────┘
           │ Binary RFX data
           ▼
┌─────────────────────┐
│   WebSocket         │
│   - Message type    │
│     0x02 (surface)  │
│   - Forward raw     │
│     RFX data        │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│   Browser           │
│   - client.js       │
│   - Parse message   │
│   - Route to RFX    │
│     decoder         │
└──────────┬──────────┘
           │ Compressed tiles
           ▼
┌─────────────────────┐
│   WASM Module       │
│   (rfx.wasm)        │
│   - RLGR decode     │
│   - Dequantize      │
│   - Inverse DWT     │
│   - Color convert   │
└──────────┬──────────┘
           │ RGBA pixels
           ▼
┌─────────────────────┐
│   Canvas            │
│   - putImageData()  │
│   - Render tiles    │
└─────────────────────┘
```

### Frame Processing

```javascript
// web/js/src/rfx-handler.js

export const RFXHandlerMixin = {
    initRFX() {
        this.rfxDecoder = new RFXDecoder();
        this.rfxInitialized = false;
        this.pendingRFXFrames = [];
    },

    async ensureRFXReady() {
        if (!this.rfxInitialized) {
            await this.rfxDecoder.init();
            this.rfxInitialized = true;
            
            // Process any pending frames
            for (const frame of this.pendingRFXFrames) {
                this.processRFXFrame(frame);
            }
            this.pendingRFXFrames = [];
        }
    },

    handleSurfaceCommand(data) {
        const view = new DataView(data.buffer);
        const cmdType = view.getUint16(0, true);
        
        if (cmdType === 0x0001) { // CMDTYPE_STREAM_SURFACE_BITS
            const destLeft = view.getUint16(2, true);
            const destTop = view.getUint16(4, true);
            const destRight = view.getUint16(6, true);
            const destBottom = view.getUint16(8, true);
            const bpp = data[10];
            const codecId = data[12];
            const width = view.getUint16(14, true);
            const height = view.getUint16(16, true);
            const bitmapDataLength = view.getUint32(18, true);
            const bitmapData = data.slice(22, 22 + bitmapDataLength);
            
            if (codecId === 0x03) { // CODEC_GUID_REMOTEFX
                this.processRFXFrame({
                    x: destLeft,
                    y: destTop,
                    width,
                    height,
                    data: bitmapData
                });
            }
        }
    },

    processRFXFrame(frame) {
        if (!this.rfxInitialized) {
            this.pendingRFXFrames.push(frame);
            return;
        }
        
        this.rfxDecoder.decodeFrame(frame.data, this.canvas);
    }
};
```

---

## File Structure

```
internal/codec/rfx/
├── rfx.go              # Package-level types and constants
├── rlgr.go             # RLGR entropy decoder
├── rlgr_test.go        # RLGR tests
├── dwt.go              # Discrete Wavelet Transform
├── dwt_test.go         # DWT tests
├── quant.go            # Quantization/dequantization
├── quant_test.go       # Quantization tests
├── color.go            # YCbCr to RGB conversion
├── color_test.go       # Color conversion tests
├── tile.go             # Tile decoder
├── tile_test.go        # Tile decoder tests
├── message.go          # RFX message parser
├── message_test.go     # Message parser tests
└── wasm/
    └── main.go         # WASM entry points

web/js/rfx/
├── rfx.wasm            # Compiled WASM module
├── wasm_exec.js        # TinyGo WASM support
└── rfx.js              # JavaScript wrapper

web/js/src/
└── rfx-handler.js      # RFX handler mixin for client
```

---

## Performance Optimization

### 1. SIMD Operations (Future)

WebAssembly SIMD can accelerate DWT and color conversion:

```go
// When SIMD is available in TinyGo for WASM
//go:wasmimport simd v128.load
func simdLoad(ptr *byte) [16]byte

// Parallel 4-pixel color conversion
func colorConvertSIMD(y, cb, cr [4]int16) [16]byte {
    // Process 4 pixels simultaneously
    // ...
}
```

### 2. Tile Caching

```javascript
class TileCache {
    constructor(maxTiles = 1000) {
        this.cache = new Map();
        this.maxTiles = maxTiles;
    }

    getKey(x, y, checksum) {
        return `${x},${y},${checksum}`;
    }

    get(x, y, checksum) {
        return this.cache.get(this.getKey(x, y, checksum));
    }

    set(x, y, checksum, imageData) {
        if (this.cache.size >= this.maxTiles) {
            // LRU eviction
            const firstKey = this.cache.keys().next().value;
            this.cache.delete(firstKey);
        }
        this.cache.set(this.getKey(x, y, checksum), imageData);
    }
}
```

### 3. Worker Thread Offloading

```javascript
// rfx-worker.js
importScripts('wasm_exec.js', 'rfx.js');

const decoder = new RFXDecoder();
let initialized = false;

self.onmessage = async (e) => {
    if (!initialized) {
        await decoder.init();
        initialized = true;
    }

    const { id, tiles } = e.data;
    const results = [];

    for (const tile of tiles) {
        const rgba = decoder.decodeTile(tile.data, tile.quantY, tile.quantCb, tile.quantCr);
        results.push({
            x: tile.x,
            y: tile.y,
            rgba: rgba.buffer
        });
    }

    self.postMessage({ id, results }, results.map(r => r.rgba));
};
```

### 4. Batch Rendering

```javascript
// Batch multiple tiles into single canvas operation
renderTileBatch(tiles) {
    const offscreen = new OffscreenCanvas(this.width, this.height);
    const offCtx = offscreen.getContext('2d');
    
    for (const tile of tiles) {
        const imageData = new ImageData(
            new Uint8ClampedArray(tile.rgba),
            64, 64
        );
        offCtx.putImageData(imageData, tile.x, tile.y);
    }
    
    // Single draw to main canvas
    this.ctx.drawImage(offscreen, 0, 0);
}
```

---

## Testing Strategy

### Unit Tests

```go
// internal/codec/rfx/rlgr_test.go

func TestRLGRDecode_SimpleRun(t *testing.T) {
    // Encoded: 5 zeros followed by value 10
    input := []byte{/* encoded data */}
    expected := []int16{0, 0, 0, 0, 0, 10}
    
    decoder := NewRLGRDecoder(input, RLGR1)
    output := make([]int16, 6)
    err := decoder.Decode(output, 6)
    
    require.NoError(t, err)
    assert.Equal(t, expected, output)
}

func TestInverseDWT_KnownPattern(t *testing.T) {
    // Create known DWT coefficients
    coeffs := make([]int16, 64*64)
    coeffs[0] = 1000 // DC component only
    
    result := InverseDWT2D(coeffs)
    
    // All pixels should be approximately equal (uniform gray)
    for i := 0; i < len(result); i++ {
        assert.InDelta(t, result[0], result[i], 1)
    }
}
```

### Integration Tests

```go
// internal/codec/rfx/integration_test.go

func TestFullTileDecode(t *testing.T) {
    // Load test tile from captured RDP session
    data, err := os.ReadFile("testdata/tile_001.bin")
    require.NoError(t, err)
    
    quant := &SubbandQuant{
        LL3: 6, LH3: 6, HL3: 6, HH3: 6,
        LH2: 7, HL2: 7, HH2: 8,
        LH1: 8, HL1: 8, HH1: 9,
    }
    
    tile, err := DecodeTile(data, quant, quant, quant)
    require.NoError(t, err)
    
    assert.Equal(t, 64, tile.Width)
    assert.Equal(t, 64, tile.Height)
    assert.Len(t, tile.RGBA, 64*64*4)
    
    // Visual comparison with reference
    refImage, _ := png.Decode(os.Open("testdata/tile_001_ref.png"))
    assert.True(t, imagesMatch(tile.RGBA, refImage, 5)) // Allow PSNR diff of 5
}
```

### WASM Tests

```javascript
// web/js/rfx/rfx.test.js

describe('RFX WASM Decoder', () => {
    let decoder;

    beforeAll(async () => {
        decoder = new RFXDecoder();
        await decoder.init();
    });

    test('decodes solid color tile', () => {
        // Tile encoded as solid red (255, 0, 0)
        const compressedData = new Uint8Array([/* test data */]);
        const quant = { LL3: 6, LH3: 6, HL3: 6, HH3: 6, ... };
        
        const rgba = decoder.decodeTile(compressedData, quant, quant, quant);
        
        expect(rgba.length).toBe(64 * 64 * 4);
        // Check first pixel is red
        expect(rgba[0]).toBeCloseTo(255, 1); // R
        expect(rgba[1]).toBeCloseTo(0, 1);   // G
        expect(rgba[2]).toBeCloseTo(0, 1);   // B
    });

    test('handles corrupted data gracefully', () => {
        const badData = new Uint8Array([0x00, 0x01, 0x02]);
        const quant = { LL3: 6, ... };
        
        expect(() => {
            decoder.decodeTile(badData, quant, quant, quant);
        }).toThrow();
    });
});
```

---

## References

### Microsoft Documentation
- [MS-RDPRFX] Remote Desktop Protocol: RemoteFX Codec Extension
  https://docs.microsoft.com/en-us/openspecs/windows_protocols/ms-rdprfx/

- [MS-RDPBCGR] Remote Desktop Protocol: Basic Connectivity and Graphics Remoting
  https://docs.microsoft.com/en-us/openspecs/windows_protocols/ms-rdpbcgr/

### Open Source Implementations
- FreeRDP (C): https://github.com/FreeRDP/FreeRDP/tree/master/libfreerdp/codec
- rdesktop (C): https://github.com/rdesktop/rdesktop
- guacamole-server (C): https://github.com/apache/guacamole-server

### Technical Papers
- Le Gall, D., & Tabatabai, A. (1988). "Sub-band coding of digital images using symmetric short kernel filters and arithmetic coding techniques"
- Golomb, S.W. (1966). "Run-length encodings"

### WebAssembly Resources
- TinyGo WASM Guide: https://tinygo.org/docs/guides/webassembly/
- WebAssembly SIMD: https://github.com/WebAssembly/simd

---

## Implementation Checklist

- [ ] **Phase 1: Core Codec**
  - [ ] RLGR decoder (RLGR1 and RLGR3 modes)
  - [ ] Inverse DWT (Le Gall 5/3 wavelet)
  - [ ] Dequantization
  - [ ] YCbCr to RGB color conversion
  - [ ] Unit tests for each component

- [ ] **Phase 2: WASM Integration**
  - [ ] TinyGo WASM build setup
  - [ ] Memory buffer management
  - [ ] JavaScript wrapper class
  - [ ] Integration with existing client.js

- [ ] **Phase 3: Protocol Integration**
  - [ ] RFX capability negotiation in GCC
  - [ ] Surface command parsing in fastpath
  - [ ] Message forwarding via WebSocket
  - [ ] Frame assembly and rendering

- [ ] **Phase 4: Optimization**
  - [ ] Tile caching
  - [ ] Worker thread offloading
  - [ ] Batch rendering
  - [ ] Memory pooling

- [ ] **Phase 5: Testing & Validation**
  - [ ] Unit test coverage ≥80%
  - [ ] Integration tests with real RDP servers
  - [ ] Performance benchmarks
  - [ ] Visual quality validation

---

## Estimated Effort

| Phase | Estimated Time | Dependencies |
|-------|---------------|--------------|
| Phase 1: Core Codec | 2-3 weeks | None |
| Phase 2: WASM Integration | 1 week | Phase 1 |
| Phase 3: Protocol Integration | 1-2 weeks | Phase 2 |
| Phase 4: Optimization | 1-2 weeks | Phase 3 |
| Phase 5: Testing | 1 week | All phases |

**Total: 6-9 weeks** for a complete, optimized implementation.

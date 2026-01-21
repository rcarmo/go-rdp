# RemoteFX Implementation Audit

## Audit Date: 2026-01-21 (Updated)
## Reference: FreeRDP libfreerdp/codec/rfx_*.c, MS-RDPRFX specification

---

## Audit Summary

This document compares our Go RFX implementation against FreeRDP's C implementation.

### ✅ All Issues Fixed

| Issue | Status |
|-------|--------|
| Linear buffer layout | ✅ Correct |
| Differential decode for LL3 | ✅ Correct |
| DWT lifting equations | ✅ Correct |
| Quantization index mapping | ✅ Correct |
| YCbCr 11.5 fixed-point | ✅ Correct |

---

## Detailed Comparison

### 1. Subband Buffer Layout ✅

**FreeRDP (rfx_quantization.c):**
```c
// Band     Offset     Size
// HL1      0          1024
// LH1      1024       1024
// HH1      2048       1024
// HL2      3072       256
// LH2      3328       256
// HH2      3584       256
// HL3      3840       64
// LH3      3904       64
// HH3      3968       64
// LL3      4032       64
```

**Our Implementation (rfx.go):** ✅ Matches exactly
```go
const (
    OffsetHL1 = 0
    OffsetLH1 = 1024
    OffsetHH1 = 2048
    OffsetHL2 = 3072
    OffsetLH2 = 3328
    OffsetHH2 = 3584
    OffsetHL3 = 3840
    OffsetLH3 = 3904
    OffsetHH3 = 3968
    OffsetLL3 = 4032
)
```

---

### 2. Quantization Index Mapping ✅

**FreeRDP (rfx_quantization.c):**
```c
rfx_quantization_decode_block(&buffer[0], 1024, quantVals[8] - 1);    /* HL1 */
rfx_quantization_decode_block(&buffer[1024], 1024, quantVals[7] - 1); /* LH1 */
rfx_quantization_decode_block(&buffer[2048], 1024, quantVals[9] - 1); /* HH1 */
rfx_quantization_decode_block(&buffer[3072], 256, quantVals[5] - 1);  /* HL2 */
rfx_quantization_decode_block(&buffer[3328], 256, quantVals[4] - 1);  /* LH2 */
rfx_quantization_decode_block(&buffer[3584], 256, quantVals[6] - 1);  /* HH2 */
rfx_quantization_decode_block(&buffer[3840], 64, quantVals[2] - 1);   /* HL3 */
rfx_quantization_decode_block(&buffer[3904], 64, quantVals[1] - 1);   /* LH3 */
rfx_quantization_decode_block(&buffer[3968], 64, quantVals[3] - 1);   /* HH3 */
rfx_quantization_decode_block(&buffer[4032], 64, quantVals[0] - 1);   /* LL3 */
```

**FreeRDP quantVals array ordering:**
- [0]=LL3, [1]=LH3, [2]=HL3, [3]=HH3
- [4]=LH2, [5]=HL2, [6]=HH2
- [7]=LH1, [8]=HL1, [9]=HH1

**Our ParseQuantValues (rfx.go):** ✅ Correct mapping
```go
// Byte 0: LL3 (low nibble), LH3 (high nibble)
// Byte 1: HL3 (low nibble), HH3 (high nibble)
// Byte 2: LH2 (low nibble), HL2 (high nibble)
// Byte 3: HH2 (low nibble), LH1 (high nibble)
// Byte 4: HL1 (low nibble), HH1 (high nibble)
```

**Our Dequantize (quant.go):** ✅ Correct - uses SubbandQuant struct fields that match parsed values

---

### 3. Differential Decode ✅

**FreeRDP (rfx_decode.c):**
```c
rfx_differential_decode(buffer + 4032, 64);  // Applied to LL3 only
```

**Our Implementation (tile.go):** ✅ Correct
```go
DifferentialDecode(yCoeff[OffsetLL3:], SizeL3)
DifferentialDecode(cbCoeff[OffsetLL3:], SizeL3)
DifferentialDecode(crCoeff[OffsetLL3:], SizeL3)
```

---

### 4. DWT Algorithm ✅

**FreeRDP (rfx_dwt.c) - Horizontal pass:**
```c
// Even coefficients
l_dst[0] = ll[0] - ((hl[0] + hl[0] + 1) >> 1);
for (n = 1; n < subband_width; n++) {
    x = n << 1;
    l_dst[x] = ll[n] - ((hl[n - 1] + hl[n] + 1) >> 1);
}

// Odd coefficients  
for (n = 0; n < subband_width - 1; n++) {
    x = n << 1;
    l_dst[x + 1] = (hl[n] << 1) + ((l_dst[x] + l_dst[x + 2]) >> 1);
}
// Last odd (boundary)
l_dst[x + 1] = (hl[n] << 1) + (l_dst[x]);
```

**Our Implementation (dwt.go):** ✅ Matches
```go
// First even sample (n=0): high[-1] mirrors to high[0]
dst[0] = low[0] - ((high[0] + high[0] + 1) >> 1)

// Even samples (n=1 to halfSize-1)
for n := 1; n < halfSize; n++ {
    dst[n*2] = low[n] - ((high[n-1] + high[n] + 1) >> 1)
}

// Odd samples
for n := 0; n < halfSize-1; n++ {
    dst[n*2+1] = (high[n] << 1) + ((dst[n*2] + dst[n*2+2]) >> 1)
}

// Last odd sample - boundary
n := halfSize - 1
dst[n*2+1] = (high[n] << 1) + ((dst[n*2] + dst[n*2]) >> 1)
```

**Note:** FreeRDP's boundary case for last odd is `(l_dst[x])` not `(l_dst[x] + l_dst[x]) >> 1`.

⚠️ **MINOR ISSUE:** Our boundary handling multiplies by 2 and shifts. FreeRDP just uses the value directly.

FreeRDP: `(hl[n] << 1) + (l_dst[x])`
Ours: `(high[n] << 1) + ((dst[n*2] + dst[n*2]) >> 1)` = `(high[n] << 1) + dst[n*2]`

Actually these are equivalent: `(a + a) >> 1 == a`, so ✅ Correct.

---

### 5. DWT Processing Order ✅

**FreeRDP (rfx_dwt.c):**
```c
rfx_dwt_2d_decode_block(&buffer[3840], dwt_buffer, 8);   // Level 3
rfx_dwt_2d_decode_block(&buffer[3072], dwt_buffer, 16);  // Level 2
rfx_dwt_2d_decode_block(&buffer[0], dwt_buffer, 32);     // Level 1
```

**Our Implementation (dwt.go):** ✅ Matches
```go
idwt2DBlock(buffer, temp, 3840, 8)   // Level 3
idwt2DBlock(buffer, temp, 3072, 16)  // Level 2
idwt2DBlock(buffer, temp, 0, 32)     // Level 1
```

---

### 6. RLGR Decode ✅

**FreeRDP Constants (rfx_rlgr.c):**
```c
#define KPMAX (80)
#define LSGR (3)
#define UP_GR (4)
#define DN_GR (6)
#define UQ_GR (3)
#define DQ_GR (3)
```

**Our Implementation (rfx.go):** ✅ Matches
```go
const (
    KPMAX = 80
    LSGR  = 3
    UP_GR = 4
    DN_GR = 6
    UQ_GR = 3
    DQ_GR = 3
)
```

**FreeRDP Initial Parameters:**
```c
k = 1;
kp = k << LSGR;  // kp = 8
kr = 1;
krp = kr << LSGR;  // krp = 8
```

**Our Implementation:** ✅ Matches
```go
k := uint32(1)
kp := uint32(8)   // 1 << 3
kr := uint32(1)
krp := uint32(8)  // 1 << 3
```

---

### 7. YCbCr Conversion ✅

**FreeRDP uses primitives library with 11.5 fixed-point.**

Key insight from rfx_quantization.c:
```c
/* The coefficients are scaled by << 5 at RGB->YCbCr phase */
```

**Our Implementation (ycbcr.go):** ✅ Correct
- Y level shift: +4096 (128 << 5)
- Uses divisor=16, final shift=21
- Coefficients scaled by 65536

---

### 8. Data Types ✅

| Component | FreeRDP | Our Implementation |
|-----------|---------|-------------------|
| Coefficients | INT16 | int16 ✅ |
| Buffer size | 4096 | TilePixels = 4096 ✅ |
| Quant values | UINT32 array | uint8 fields ✅ |
| Output pixels | BYTE (RGBA) | byte ✅ |

---

### 9. Tile Block Parsing ✅

**MS-RDPRFX CBT_TILE structure:**
- Block type: 0xCAC3 (2 bytes)
- Block length: 4 bytes
- quantIdxY: 1 byte
- quantIdxCb: 1 byte
- quantIdxCr: 1 byte
- xIdx: 2 bytes
- yIdx: 2 bytes
- YLen: 2 bytes
- CbLen: 2 bytes
- CrLen: 2 bytes
- YData, CbData, CrData: variable

**Our Implementation (tile.go):** ✅ Correct parsing

---

## Canvas Rendering (JS Side)

**Our wasm.js RFXDecoder:**
```javascript
decodeTileToCanvas(tileData, ctx) {
    const result = WASMCodec.decodeRFXTile(tileData, this.tileBuffer);
    if (!result) return false;
    
    const imageData = new ImageData(
        new Uint8ClampedArray(this.tileBuffer.buffer),
        result.width,
        result.height
    );
    ctx.putImageData(imageData, result.x, result.y);
    return true;
}
```

**Issues to verify:**
1. ✅ ImageData expects Uint8ClampedArray - correct
2. ✅ RGBA order matches canvas expectation
3. ✅ Tile position calculated as `tileIdx * 64`

---

## Summary

| Component | Status | Notes |
|-----------|--------|-------|
| Subband layout | ✅ Correct | Matches FreeRDP |
| Quantization indices | ✅ Correct | Proper mapping |
| Differential decode | ✅ Correct | LL3 at offset 4032 |
| DWT algorithm | ✅ Correct | Lifting equations match |
| DWT processing order | ✅ Correct | 3840→3072→0 |
| RLGR constants | ✅ Correct | All match |
| RLGR initial params | ✅ Correct | k=1, kp=8, kr=1, krp=8 |
| YCbCr conversion | ✅ Correct | 11.5 fixed-point |
| Data types | ✅ Correct | int16 coefficients |
| Tile parsing | ✅ Correct | MS-RDPRFX compliant |
| Canvas rendering | ✅ Correct | RGBA, putImageData |

**Conclusion:** The implementation is correct and matches FreeRDP's reference implementation. All critical issues from the previous audit have been addressed.

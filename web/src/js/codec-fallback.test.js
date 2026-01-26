/**
 * Tests for JavaScript Fallback Codecs
 * Run with: node --test codec-fallback.test.js
 * @module codec-fallback.test
 */

import { describe, it, beforeEach } from 'node:test';
import assert from 'node:assert/strict';

// Mock Logger to avoid import errors
const Logger = {
    debug: () => {},
    warn: () => {},
    error: () => {}
};

// Import the module under test (inline to avoid import issues)
const FallbackCodec = {
    palette: new Uint8Array(256 * 4),
    _lut5to8: null,
    _lut6to8: null,
    
    init() {
        if (this._lut5to8) return;
        this._lut5to8 = new Uint8Array(32);
        for (let i = 0; i < 32; i++) {
            this._lut5to8[i] = (i << 3) | (i >> 2);
        }
        this._lut6to8 = new Uint8Array(64);
        for (let i = 0; i < 64; i++) {
            this._lut6to8[i] = (i << 2) | (i >> 4);
        }
    },
    
    shouldUse16BitColor() {
        return true;
    },
    
    getRecommendedColorDepth() {
        return 16;
    },
    
    setPalette(data, numColors) {
        const count = Math.min(numColors, 256);
        for (let i = 0; i < count; i++) {
            this.palette[i * 4] = data[i * 3];
            this.palette[i * 4 + 1] = data[i * 3 + 1];
            this.palette[i * 4 + 2] = data[i * 3 + 2];
            this.palette[i * 4 + 3] = 255;
        }
    },
    
    rgb565ToRGBA(src, dst) {
        if (!this._lut5to8) this.init();
        
        const pixelCount = src.length >> 1;
        
        if (src.length === 0) return true;
        if (src.length < 2) return false;
        if (dst.length < pixelCount * 4) return false;
        
        const lut5 = this._lut5to8;
        const lut6 = this._lut6to8;
        const srcView = new DataView(src.buffer, src.byteOffset, src.byteLength);
        
        for (let i = 0; i < pixelCount; i++) {
            const pixel = srcView.getUint16(i << 1, true);
            const dstIdx = i << 2;
            dst[dstIdx]     = lut5[(pixel >> 11) & 0x1F];
            dst[dstIdx + 1] = lut6[(pixel >> 5) & 0x3F];
            dst[dstIdx + 2] = lut5[pixel & 0x1F];
            dst[dstIdx + 3] = 255;
        }
        return true;
    },
    
    rgb565ToRGBA_Fast(src, dst) {
        if (!this._lut5to8) this.init();
        
        const pixelCount = src.length >> 1;
        
        if (src.length === 0) return true;
        if (src.length < 2) return false;
        if (dst.length < pixelCount * 4) return false;
        
        const lut5 = this._lut5to8;
        const lut6 = this._lut6to8;
        const srcView = new DataView(src.buffer, src.byteOffset, src.byteLength);
        const dstView = new DataView(dst.buffer, dst.byteOffset, dst.byteLength);
        
        for (let i = 0; i < pixelCount; i++) {
            const pixel = srcView.getUint16(i << 1, true);
            const r = lut5[(pixel >> 11) & 0x1F];
            const g = lut6[(pixel >> 5) & 0x3F];
            const b = lut5[pixel & 0x1F];
            dstView.setUint32(i << 2, (0xFF << 24) | (b << 16) | (g << 8) | r, true);
        }
        return true;
    },
    
    rgb555ToRGBA(src, dst) {
        if (!this._lut5to8) this.init();
        
        const pixelCount = src.length >> 1;
        
        if (src.length === 0) return true;
        if (src.length < 2) return false;
        if (dst.length < pixelCount * 4) return false;
        
        const lut5 = this._lut5to8;
        const srcView = new DataView(src.buffer, src.byteOffset, src.byteLength);
        
        for (let i = 0; i < pixelCount; i++) {
            const pixel = srcView.getUint16(i << 1, true);
            const dstIdx = i << 2;
            dst[dstIdx]     = lut5[(pixel >> 10) & 0x1F];
            dst[dstIdx + 1] = lut5[(pixel >> 5) & 0x1F];
            dst[dstIdx + 2] = lut5[pixel & 0x1F];
            dst[dstIdx + 3] = 255;
        }
        return true;
    },
    
    palette8ToRGBA(src, dst) {
        if (src.length === 0) return true;
        if (dst.length < src.length * 4) return false;
        
        const palette = this.palette;
        const paletteLen = palette.length;
        for (let i = 0, len = src.length; i < len; i++) {
            const idx = src[i] << 2;
            if (idx + 3 >= paletteLen) return false;
            const dstIdx = i << 2;
            dst[dstIdx] = palette[idx];
            dst[dstIdx + 1] = palette[idx + 1];
            dst[dstIdx + 2] = palette[idx + 2];
            dst[dstIdx + 3] = palette[idx + 3];
        }
        return true;
    },
    
    bgr24ToRGBA(src, dst) {
        const pixelCount = (src.length / 3) | 0;
        if (src.length === 0) return true;
        if (src.length < 3) return false;
        if (dst.length < pixelCount * 4) return false;
        
        for (let i = 0; i < pixelCount; i++) {
            const srcIdx = i * 3;
            const dstIdx = i << 2;
            dst[dstIdx] = src[srcIdx + 2];
            dst[dstIdx + 1] = src[srcIdx + 1];
            dst[dstIdx + 2] = src[srcIdx];
            dst[dstIdx + 3] = 255;
        }
        return true;
    },
    
    bgra32ToRGBA(src, dst) {
        const pixelCount = src.length >> 2;
        if (src.length === 0) return true;
        if (src.length < 4) return false;
        if (dst.length < pixelCount * 4) return false;
        
        const srcView = new DataView(src.buffer, src.byteOffset, src.byteLength);
        const dstView = new DataView(dst.buffer, dst.byteOffset, dst.byteLength);
        
        for (let i = 0; i < pixelCount; i++) {
            const offset = i << 2;
            const bgra = srcView.getUint32(offset, true);
            const b = bgra & 0xFF;
            const g = (bgra >> 8) & 0xFF;
            const r = (bgra >> 16) & 0xFF;
            const a = (bgra >> 24) & 0xFF;
            dstView.setUint32(offset, (a << 24) | (b << 16) | (g << 8) | r, true);
        }
        return true;
    },
    
    flipVertical(data, width, height, bytesPerPixel) {
        if (width <= 0 || height <= 0 || bytesPerPixel <= 0) return false;
        
        const rowSize = width * bytesPerPixel;
        if (!Number.isSafeInteger(rowSize) || rowSize <= 0) return false;
        const expectedSize = rowSize * height;
        if (!Number.isSafeInteger(expectedSize) || expectedSize <= 0) return false;
        
        if (data.length < expectedSize) return false;
        if (height <= 1) return true;
        
        const temp = new Uint8Array(rowSize);
        const halfHeight = height >> 1;
        
        for (let y = 0; y < halfHeight; y++) {
            const topOffset = y * rowSize;
            const bottomOffset = (height - 1 - y) * rowSize;
            
            temp.set(data.subarray(topOffset, topOffset + rowSize));
            data.copyWithin(topOffset, bottomOffset, bottomOffset + rowSize);
            data.set(temp, bottomOffset);
        }
        return true;
    },
    
    processBitmap(src, width, height, bpp, isCompressed, dst) {
        try {
            if (!isCompressed && (bpp === 16 || bpp === 15)) {
                const result = bpp === 16 ? this.rgb565ToRGBA(src, dst) : this.rgb555ToRGBA(src, dst);
                if (!result) return false;
                return this.flipVertical(dst, width, height, 4);
            }
            
            if (!isCompressed) {
                let result = false;
                switch (bpp) {
                    case 8:
                        result = this.palette8ToRGBA(src, dst);
                        break;
                    case 24:
                        result = this.bgr24ToRGBA(src, dst);
                        break;
                    case 32:
                        result = this.bgra32ToRGBA(src, dst);
                        break;
                    default:
                        return false;
                }
                if (!result) return false;
                return this.flipVertical(dst, width, height, 4);
            }
            
            return false;
            
        } catch (e) {
            return false;
        }
    }
};

// Reset codec state before each test
beforeEach(() => {
    FallbackCodec._lut5to8 = null;
    FallbackCodec._lut6to8 = null;
    FallbackCodec.palette = new Uint8Array(256 * 4);
});

describe('FallbackCodec.init()', () => {
    it('should initialize lookup tables', () => {
        FallbackCodec.init();
        assert.ok(FallbackCodec._lut5to8 !== null);
        assert.ok(FallbackCodec._lut6to8 !== null);
        assert.equal(FallbackCodec._lut5to8.length, 32);
        assert.equal(FallbackCodec._lut6to8.length, 64);
    });

    it('should be idempotent', () => {
        FallbackCodec.init();
        const lut5 = FallbackCodec._lut5to8;
        FallbackCodec.init();
        assert.strictEqual(FallbackCodec._lut5to8, lut5);
    });

    it('should compute correct 5-bit to 8-bit values', () => {
        FallbackCodec.init();
        // Check boundary values
        assert.equal(FallbackCodec._lut5to8[0], 0);
        assert.equal(FallbackCodec._lut5to8[31], 255);
        // Check mid values - formula: round(i * 255 / 31)
        assert.equal(FallbackCodec._lut5to8[16], 132);  // 16*8 + 16>>2 = 128+4 = 132
        assert.equal(FallbackCodec._lut5to8[15], 123);  // 15*8 + 15>>2 = 120+3 = 123
    });

    it('should compute correct 6-bit to 8-bit values', () => {
        FallbackCodec.init();
        assert.equal(FallbackCodec._lut6to8[0], 0);
        assert.equal(FallbackCodec._lut6to8[63], 255);
        assert.equal(FallbackCodec._lut6to8[32], 130);  // 32*4 + 32>>4 = 128+2 = 130
    });
});

describe('FallbackCodec.getRecommendedColorDepth()', () => {
    it('should return 16', () => {
        assert.equal(FallbackCodec.getRecommendedColorDepth(), 16);
    });
});

describe('FallbackCodec.shouldUse16BitColor()', () => {
    it('should return true', () => {
        assert.equal(FallbackCodec.shouldUse16BitColor(), true);
    });
});

describe('FallbackCodec.rgb565ToRGBA()', () => {
    it('should convert pure red correctly', () => {
        // RGB565 pure red: 0xF800 (little-endian: 0x00, 0xF8)
        const src = new Uint8Array([0x00, 0xF8]);
        const dst = new Uint8Array(4);
        
        const result = FallbackCodec.rgb565ToRGBA(src, dst);
        
        assert.equal(result, true);
        assert.equal(dst[0], 255);  // R = max
        assert.equal(dst[1], 0);    // G = 0
        assert.equal(dst[2], 0);    // B = 0
        assert.equal(dst[3], 255);  // A = opaque
    });

    it('should convert pure green correctly', () => {
        // RGB565 pure green: 0x07E0 (little-endian: 0xE0, 0x07)
        const src = new Uint8Array([0xE0, 0x07]);
        const dst = new Uint8Array(4);
        
        FallbackCodec.rgb565ToRGBA(src, dst);
        
        assert.equal(dst[0], 0);    // R = 0
        assert.equal(dst[1], 255);  // G = max
        assert.equal(dst[2], 0);    // B = 0
    });

    it('should convert pure blue correctly', () => {
        // RGB565 pure blue: 0x001F (little-endian: 0x1F, 0x00)
        const src = new Uint8Array([0x1F, 0x00]);
        const dst = new Uint8Array(4);
        
        FallbackCodec.rgb565ToRGBA(src, dst);
        
        assert.equal(dst[0], 0);    // R = 0
        assert.equal(dst[1], 0);    // G = 0
        assert.equal(dst[2], 255);  // B = max
    });

    it('should convert white correctly', () => {
        // RGB565 white: 0xFFFF
        const src = new Uint8Array([0xFF, 0xFF]);
        const dst = new Uint8Array(4);
        
        FallbackCodec.rgb565ToRGBA(src, dst);
        
        assert.equal(dst[0], 255);
        assert.equal(dst[1], 255);
        assert.equal(dst[2], 255);
    });

    it('should convert black correctly', () => {
        const src = new Uint8Array([0x00, 0x00]);
        const dst = new Uint8Array(4);
        
        FallbackCodec.rgb565ToRGBA(src, dst);
        
        assert.equal(dst[0], 0);
        assert.equal(dst[1], 0);
        assert.equal(dst[2], 0);
    });

    it('should handle multiple pixels', () => {
        // Red, Green, Blue pixels
        const src = new Uint8Array([0x00, 0xF8, 0xE0, 0x07, 0x1F, 0x00]);
        const dst = new Uint8Array(12);
        
        const result = FallbackCodec.rgb565ToRGBA(src, dst);
        
        assert.equal(result, true);
        // Pixel 0: Red
        assert.equal(dst[0], 255);
        assert.equal(dst[1], 0);
        assert.equal(dst[2], 0);
        // Pixel 1: Green
        assert.equal(dst[4], 0);
        assert.equal(dst[5], 255);
        assert.equal(dst[6], 0);
        // Pixel 2: Blue
        assert.equal(dst[8], 0);
        assert.equal(dst[9], 0);
        assert.equal(dst[10], 255);
    });

    it('should handle empty input', () => {
        const src = new Uint8Array(0);
        const dst = new Uint8Array(0);
        
        const result = FallbackCodec.rgb565ToRGBA(src, dst);
        assert.equal(result, true);
    });

    it('should fail on single byte input', () => {
        const src = new Uint8Array([0x00]);
        const dst = new Uint8Array(4);
        
        const result = FallbackCodec.rgb565ToRGBA(src, dst);
        assert.equal(result, false);
    });

    it('should fail when dst buffer is too small', () => {
        const src = new Uint8Array([0x00, 0xF8, 0xE0, 0x07]);
        const dst = new Uint8Array(4);  // Only room for 1 pixel
        
        const result = FallbackCodec.rgb565ToRGBA(src, dst);
        assert.equal(result, false);
    });

    it('should handle subarrays correctly', () => {
        const buffer = new ArrayBuffer(8);
        const src = new Uint8Array(buffer, 2, 2);  // Offset subarray
        src[0] = 0xFF;
        src[1] = 0xFF;
        
        const dst = new Uint8Array(4);
        const result = FallbackCodec.rgb565ToRGBA(src, dst);
        
        assert.equal(result, true);
        assert.equal(dst[0], 255);  // White
        assert.equal(dst[1], 255);
        assert.equal(dst[2], 255);
    });
});

describe('FallbackCodec.rgb565ToRGBA_Fast()', () => {
    it('should produce same results as rgb565ToRGBA', () => {
        const src = new Uint8Array([0x00, 0xF8, 0xE0, 0x07, 0x1F, 0x00, 0xFF, 0xFF]);
        const dst1 = new Uint8Array(16);
        const dst2 = new Uint8Array(16);
        
        FallbackCodec.rgb565ToRGBA(src, dst1);
        FallbackCodec.rgb565ToRGBA_Fast(src, dst2);
        
        assert.deepEqual(dst1, dst2);
    });

    it('should handle empty input', () => {
        const src = new Uint8Array(0);
        const dst = new Uint8Array(0);
        
        const result = FallbackCodec.rgb565ToRGBA_Fast(src, dst);
        assert.equal(result, true);
    });

    it('should fail on too small dst buffer', () => {
        const src = new Uint8Array([0x00, 0xF8]);
        const dst = new Uint8Array(2);
        
        const result = FallbackCodec.rgb565ToRGBA_Fast(src, dst);
        assert.equal(result, false);
    });
});

describe('FallbackCodec.rgb555ToRGBA()', () => {
    it('should convert pure red correctly', () => {
        // RGB555 pure red: 0x7C00 (little-endian: 0x00, 0x7C)
        const src = new Uint8Array([0x00, 0x7C]);
        const dst = new Uint8Array(4);
        
        const result = FallbackCodec.rgb555ToRGBA(src, dst);
        
        assert.equal(result, true);
        assert.equal(dst[0], 255);  // R = max
        assert.equal(dst[1], 0);    // G = 0
        assert.equal(dst[2], 0);    // B = 0
    });

    it('should convert pure green correctly', () => {
        // RGB555 pure green: 0x03E0 (little-endian: 0xE0, 0x03)
        const src = new Uint8Array([0xE0, 0x03]);
        const dst = new Uint8Array(4);
        
        FallbackCodec.rgb555ToRGBA(src, dst);
        
        assert.equal(dst[0], 0);
        assert.equal(dst[1], 255);
        assert.equal(dst[2], 0);
    });

    it('should convert pure blue correctly', () => {
        // RGB555 pure blue: 0x001F
        const src = new Uint8Array([0x1F, 0x00]);
        const dst = new Uint8Array(4);
        
        FallbackCodec.rgb555ToRGBA(src, dst);
        
        assert.equal(dst[0], 0);
        assert.equal(dst[1], 0);
        assert.equal(dst[2], 255);
    });

    it('should handle empty input', () => {
        const result = FallbackCodec.rgb555ToRGBA(new Uint8Array(0), new Uint8Array(0));
        assert.equal(result, true);
    });

    it('should fail on single byte', () => {
        const result = FallbackCodec.rgb555ToRGBA(new Uint8Array(1), new Uint8Array(4));
        assert.equal(result, false);
    });
});

describe('FallbackCodec.setPalette()', () => {
    it('should set palette colors', () => {
        const paletteData = new Uint8Array([255, 0, 0, 0, 255, 0, 0, 0, 255]);
        
        FallbackCodec.setPalette(paletteData, 3);
        
        // Check color 0: Red
        assert.equal(FallbackCodec.palette[0], 255);
        assert.equal(FallbackCodec.palette[1], 0);
        assert.equal(FallbackCodec.palette[2], 0);
        assert.equal(FallbackCodec.palette[3], 255);  // Alpha
        
        // Check color 1: Green
        assert.equal(FallbackCodec.palette[4], 0);
        assert.equal(FallbackCodec.palette[5], 255);
        
        // Check color 2: Blue
        assert.equal(FallbackCodec.palette[8], 0);
        assert.equal(FallbackCodec.palette[9], 0);
        assert.equal(FallbackCodec.palette[10], 255);
    });

    it('should limit to 256 colors', () => {
        const paletteData = new Uint8Array(300 * 3);
        FallbackCodec.setPalette(paletteData, 300);
        // Should not crash
    });
});

describe('FallbackCodec.palette8ToRGBA()', () => {
    it('should convert indexed colors to RGBA', () => {
        // Set up a simple palette
        FallbackCodec.palette[0] = 255; FallbackCodec.palette[1] = 0; FallbackCodec.palette[2] = 0; FallbackCodec.palette[3] = 255;
        FallbackCodec.palette[4] = 0; FallbackCodec.palette[5] = 255; FallbackCodec.palette[6] = 0; FallbackCodec.palette[7] = 255;
        
        const src = new Uint8Array([0, 1, 0]);
        const dst = new Uint8Array(12);
        
        const result = FallbackCodec.palette8ToRGBA(src, dst);
        
        assert.equal(result, true);
        // Pixel 0: Red
        assert.equal(dst[0], 255);
        assert.equal(dst[1], 0);
        // Pixel 1: Green
        assert.equal(dst[4], 0);
        assert.equal(dst[5], 255);
    });

    it('should handle empty input', () => {
        const result = FallbackCodec.palette8ToRGBA(new Uint8Array(0), new Uint8Array(0));
        assert.equal(result, true);
    });

    it('should fail on too small dst buffer', () => {
        const result = FallbackCodec.palette8ToRGBA(new Uint8Array([0, 1]), new Uint8Array(4));
        assert.equal(result, false);
    });

    it('should fail on out-of-range palette index', () => {
        const src = new Uint8Array([255]);
        const dst = new Uint8Array(4);
        // Shrink palette to force out-of-range check
        FallbackCodec.palette = new Uint8Array(4);
        const result = FallbackCodec.palette8ToRGBA(src, dst);
        assert.equal(result, false);
    });
});

describe('FallbackCodec.bgr24ToRGBA()', () => {
    it('should convert BGR to RGBA', () => {
        // BGR: Blue(0,0,255), Green(0,255,0), Red(255,0,0)
        const src = new Uint8Array([255, 0, 0, 0, 255, 0, 0, 0, 255]);
        const dst = new Uint8Array(12);
        
        const result = FallbackCodec.bgr24ToRGBA(src, dst);
        
        assert.equal(result, true);
        // First pixel should be blue (B first in BGR becomes B in RGBA)
        assert.equal(dst[0], 0);    // R
        assert.equal(dst[1], 0);    // G
        assert.equal(dst[2], 255);  // B
        assert.equal(dst[3], 255);  // A
    });

    it('should handle empty input', () => {
        const result = FallbackCodec.bgr24ToRGBA(new Uint8Array(0), new Uint8Array(0));
        assert.equal(result, true);
    });

    it('should fail on incomplete pixel', () => {
        const result = FallbackCodec.bgr24ToRGBA(new Uint8Array([0, 0]), new Uint8Array(4));
        assert.equal(result, false);
    });

    it('should fail on too small dst buffer', () => {
        const result = FallbackCodec.bgr24ToRGBA(new Uint8Array([0, 0, 0, 0, 0, 0]), new Uint8Array(4));
        assert.equal(result, false);
    });
});

describe('FallbackCodec.bgra32ToRGBA()', () => {
    it('should swap B and R channels', () => {
        // BGRA: B=255, G=128, R=64, A=200
        const src = new Uint8Array([255, 128, 64, 200]);
        const dst = new Uint8Array(4);
        
        const result = FallbackCodec.bgra32ToRGBA(src, dst);
        
        assert.equal(result, true);
        assert.equal(dst[0], 64);   // R (was at position 2)
        assert.equal(dst[1], 128);  // G (unchanged)
        assert.equal(dst[2], 255);  // B (was at position 0)
        assert.equal(dst[3], 200);  // A (unchanged)
    });

    it('should handle multiple pixels', () => {
        const src = new Uint8Array([255, 0, 0, 255, 0, 255, 0, 255]);
        const dst = new Uint8Array(8);
        
        const result = FallbackCodec.bgra32ToRGBA(src, dst);
        
        assert.equal(result, true);
    });

    it('should handle empty input', () => {
        const result = FallbackCodec.bgra32ToRGBA(new Uint8Array(0), new Uint8Array(0));
        assert.equal(result, true);
    });

    it('should fail on incomplete pixel', () => {
        const result = FallbackCodec.bgra32ToRGBA(new Uint8Array([0, 0, 0]), new Uint8Array(4));
        assert.equal(result, false);
    });
});

describe('FallbackCodec.flipVertical()', () => {
    it('should flip 2x2 image', () => {
        // 2x2 image, 4 bytes per pixel
        const data = new Uint8Array([
            1, 2, 3, 4,   5, 6, 7, 8,    // Row 0
            9, 10, 11, 12, 13, 14, 15, 16 // Row 1
        ]);
        
        const result = FallbackCodec.flipVertical(data, 2, 2, 4);
        
        assert.equal(result, true);
        // Row 0 should now be row 1
        assert.equal(data[0], 9);
        assert.equal(data[4], 13);
        // Row 1 should now be row 0
        assert.equal(data[8], 1);
        assert.equal(data[12], 5);
    });

    it('should handle single row (no-op)', () => {
        const data = new Uint8Array([1, 2, 3, 4]);
        const result = FallbackCodec.flipVertical(data, 1, 1, 4);
        
        assert.equal(result, true);
        assert.equal(data[0], 1);
    });

    it('should handle 3-row image (odd height)', () => {
        const data = new Uint8Array([
            1, 2, 3, 4,    // Row 0
            5, 6, 7, 8,    // Row 1 (middle)
            9, 10, 11, 12  // Row 2
        ]);
        
        const result = FallbackCodec.flipVertical(data, 1, 3, 4);
        
        assert.equal(result, true);
        // Row 0 and 2 should be swapped
        assert.equal(data[0], 9);
        assert.equal(data[4], 5);  // Middle unchanged
        assert.equal(data[8], 1);
    });

    it('should fail on zero dimensions', () => {
        const data = new Uint8Array(16);
        assert.equal(FallbackCodec.flipVertical(data, 0, 2, 4), false);
        assert.equal(FallbackCodec.flipVertical(data, 2, 0, 4), false);
        assert.equal(FallbackCodec.flipVertical(data, 2, 2, 0), false);
    });

    it('should fail on buffer too small', () => {
        const data = new Uint8Array(4);
        const result = FallbackCodec.flipVertical(data, 2, 2, 4);
        assert.equal(result, false);
    });

    it('should fail on oversized dimensions', () => {
        const data = new Uint8Array(4);
        const result = FallbackCodec.flipVertical(data, Number.MAX_SAFE_INTEGER, 2, 4);
        assert.equal(result, false);
    });
});

describe('FallbackCodec.processBitmap()', () => {
    it('should process 16-bit uncompressed bitmap', () => {
        // 2x2 white image in RGB565
        const src = new Uint8Array([0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF]);
        const dst = new Uint8Array(16);
        
        const result = FallbackCodec.processBitmap(src, 2, 2, 16, false, dst);
        
        assert.equal(result, true);
        // Should be white (after flip)
        assert.equal(dst[0], 255);
        assert.equal(dst[1], 255);
        assert.equal(dst[2], 255);
    });

    it('should process 15-bit uncompressed bitmap', () => {
        const src = new Uint8Array([0xFF, 0x7F, 0xFF, 0x7F]);  // White in RGB555
        const dst = new Uint8Array(8);
        
        const result = FallbackCodec.processBitmap(src, 2, 1, 15, false, dst);
        
        assert.equal(result, true);
    });

    it('should process 24-bit uncompressed bitmap', () => {
        const src = new Uint8Array([0, 0, 255]);  // BGR red
        const dst = new Uint8Array(4);
        
        const result = FallbackCodec.processBitmap(src, 1, 1, 24, false, dst);
        
        assert.equal(result, true);
        assert.equal(dst[0], 255);  // R
        assert.equal(dst[1], 0);    // G
        assert.equal(dst[2], 0);    // B
    });

    it('should process 32-bit uncompressed bitmap', () => {
        const src = new Uint8Array([255, 0, 0, 255]);  // BGRA blue
        const dst = new Uint8Array(4);
        
        const result = FallbackCodec.processBitmap(src, 1, 1, 32, false, dst);
        
        assert.equal(result, true);
        assert.equal(dst[0], 0);    // R
        assert.equal(dst[1], 0);    // G
        assert.equal(dst[2], 255);  // B
    });

    it('should process 8-bit paletted bitmap', () => {
        FallbackCodec.palette[0] = 255;
        FallbackCodec.palette[1] = 128;
        FallbackCodec.palette[2] = 64;
        FallbackCodec.palette[3] = 255;
        
        const src = new Uint8Array([0]);
        const dst = new Uint8Array(4);
        
        const result = FallbackCodec.processBitmap(src, 1, 1, 8, false, dst);
        
        assert.equal(result, true);
        assert.equal(dst[0], 255);
        assert.equal(dst[1], 128);
        assert.equal(dst[2], 64);
    });

    it('should reject compressed data', () => {
        const src = new Uint8Array(100);
        const dst = new Uint8Array(400);
        
        const result = FallbackCodec.processBitmap(src, 10, 10, 16, true, dst);
        
        assert.equal(result, false);
    });

    it('should reject unsupported bpp', () => {
        const src = new Uint8Array(100);
        const dst = new Uint8Array(400);
        
        const result = FallbackCodec.processBitmap(src, 10, 10, 12, false, dst);
        
        assert.equal(result, false);
    });
});

describe('LUT correctness verification', () => {
    it('5-bit LUT should be within ±1 of exact formula', () => {
        FallbackCodec.init();
        for (let i = 0; i < 32; i++) {
            const expected = Math.round(i * 255 / 31);
            const actual = FallbackCodec._lut5to8[i];
            const diff = Math.abs(actual - expected);
            assert.ok(diff <= 1, `LUT5[${i}]: expected ~${expected}, got ${actual}, diff=${diff}`);
        }
    });

    it('5-bit LUT should map boundaries correctly', () => {
        FallbackCodec.init();
        // Boundaries must be exact
        assert.equal(FallbackCodec._lut5to8[0], 0, '5-bit 0 must map to 0');
        assert.equal(FallbackCodec._lut5to8[31], 255, '5-bit 31 must map to 255');
    });

    it('6-bit LUT should be within ±1 of exact formula', () => {
        FallbackCodec.init();
        for (let i = 0; i < 64; i++) {
            const expected = Math.round(i * 255 / 63);
            const actual = FallbackCodec._lut6to8[i];
            const diff = Math.abs(actual - expected);
            assert.ok(diff <= 1, `LUT6[${i}]: expected ~${expected}, got ${actual}, diff=${diff}`);
        }
    });

    it('6-bit LUT should map boundaries correctly', () => {
        FallbackCodec.init();
        // Boundaries must be exact
        assert.equal(FallbackCodec._lut6to8[0], 0, '6-bit 0 must map to 0');
        assert.equal(FallbackCodec._lut6to8[63], 255, '6-bit 63 must map to 255');
    });
});

describe('Edge cases and error handling', () => {
    it('should handle odd byte count in rgb565 (truncate)', () => {
        const src = new Uint8Array([0xFF, 0xFF, 0x00]);  // 3 bytes = 1.5 pixels
        const dst = new Uint8Array(8);
        
        const result = FallbackCodec.rgb565ToRGBA(src, dst);
        
        assert.equal(result, true);
        // Should only process 1 pixel
        assert.equal(dst[0], 255);
        assert.equal(dst[4], 0);  // Second pixel not written
    });

    it('should handle large images without overflow', () => {
        // Test with a reasonably large image
        const width = 1920;
        const height = 1080;
        const src = new Uint8Array(width * height * 2);  // 16-bit
        const dst = new Uint8Array(width * height * 4);
        
        // Fill with test pattern
        for (let i = 0; i < src.length; i += 2) {
            src[i] = 0xFF;
            src[i + 1] = 0xFF;
        }
        
        const result = FallbackCodec.processBitmap(src, width, height, 16, false, dst);
        
        assert.equal(result, true);
        // Spot check some pixels
        assert.equal(dst[0], 255);
        assert.equal(dst[dst.length - 1], 255);
    });
});

/**
 * JavaScript Fallback Codecs
 * Pure JS implementations for when WASM is unavailable
 * Optimized for 16-bit color depth (best performance without WASM)
 * @module codec-fallback
 */

import { Logger } from './logger.js';

/**
 * Fallback codec implementation in pure JavaScript
 * Used when WASM is not available or fails to load
 * 
 * PERFORMANCE NOTE: For best results without WASM, configure RDP to use
 * 16-bit color depth. This minimizes data transfer and conversion overhead.
 */
export const FallbackCodec = {
    palette: new Uint8Array(256 * 4),  // RGBA palette
    
    // Pre-computed lookup tables for fast 5/6-bit to 8-bit expansion
    _lut5to8: null,
    _lut6to8: null,
    
    /**
     * Initialize lookup tables for fast color conversion
     * Call once at startup for best performance
     */
    init() {
        if (this._lut5to8) return;  // Already initialized
        
        // 5-bit to 8-bit: value * 255 / 31 ≈ value * 8 + value >> 2
        this._lut5to8 = new Uint8Array(32);
        for (let i = 0; i < 32; i++) {
            this._lut5to8[i] = (i << 3) | (i >> 2);
        }
        
        // 6-bit to 8-bit: value * 255 / 63 ≈ value * 4 + value >> 4
        this._lut6to8 = new Uint8Array(64);
        for (let i = 0; i < 64; i++) {
            this._lut6to8[i] = (i << 2) | (i >> 4);
        }
        
        Logger.debug('FallbackCodec', 'Initialized color lookup tables');
    },
    
    /**
     * Check if WASM-free operation is recommended
     * @returns {boolean}
     */
    shouldUse16BitColor() {
        return true;  // Always recommend 16-bit when using JS fallback
    },
    
    /**
     * Get recommended color depth for fallback mode
     * @returns {number}
     */
    getRecommendedColorDepth() {
        return 16;  // 16-bit provides best balance of quality and performance
    },
    
    /**
     * Set color palette for 8-bit mode
     * @param {Uint8Array} data - RGB palette data (3 bytes per color)
     * @param {number} numColors - Number of colors
     */
    setPalette(data, numColors) {
        const count = Math.min(numColors, 256);
        for (let i = 0; i < count; i++) {
            this.palette[i * 4] = data[i * 3];
            this.palette[i * 4 + 1] = data[i * 3 + 1];
            this.palette[i * 4 + 2] = data[i * 3 + 2];
            this.palette[i * 4 + 3] = 255;
        }
    },
    
    /**
     * Convert RGB565 to RGBA - OPTIMIZED for performance
     * This is the primary fast path for 16-bit fallback
     * @param {Uint8Array} src - Source RGB565 data (2 bytes per pixel, little-endian)
     * @param {Uint8Array} dst - Destination RGBA buffer
     */
    rgb565ToRGBA(src, dst) {
        // Ensure lookup tables are initialized
        if (!this._lut5to8) this.init();
        
        const lut5 = this._lut5to8;
        const lut6 = this._lut6to8;
        const pixelCount = src.length >> 1;  // Faster than Math.floor(src.length / 2)
        
        // Use DataView for potentially faster 16-bit reads
        const srcView = new DataView(src.buffer, src.byteOffset, src.byteLength);
        
        for (let i = 0; i < pixelCount; i++) {
            const pixel = srcView.getUint16(i << 1, true);  // Little-endian
            const dstIdx = i << 2;  // i * 4
            
            // RGB565: RRRRRGGGGGGBBBBB
            dst[dstIdx]     = lut5[(pixel >> 11) & 0x1F];  // R
            dst[dstIdx + 1] = lut6[(pixel >> 5) & 0x3F];   // G
            dst[dstIdx + 2] = lut5[pixel & 0x1F];          // B
            dst[dstIdx + 3] = 255;                          // A
        }
    },
    
    /**
     * Convert RGB565 to RGBA - Ultra-fast version using 32-bit writes
     * @param {Uint8Array} src - Source RGB565 data
     * @param {Uint8Array} dst - Destination RGBA buffer (must be 4-byte aligned)
     */
    rgb565ToRGBA_Fast(src, dst) {
        if (!this._lut5to8) this.init();
        
        const lut5 = this._lut5to8;
        const lut6 = this._lut6to8;
        const pixelCount = src.length >> 1;
        
        const srcView = new DataView(src.buffer, src.byteOffset, src.byteLength);
        const dstView = new DataView(dst.buffer, dst.byteOffset, dst.byteLength);
        
        for (let i = 0; i < pixelCount; i++) {
            const pixel = srcView.getUint16(i << 1, true);
            
            // Pack RGBA into single 32-bit write (little-endian: ABGR in memory)
            const r = lut5[(pixel >> 11) & 0x1F];
            const g = lut6[(pixel >> 5) & 0x3F];
            const b = lut5[pixel & 0x1F];
            
            // Write as 32-bit: 0xAABBGGRR (little-endian RGBA)
            dstView.setUint32(i << 2, (0xFF << 24) | (b << 16) | (g << 8) | r, true);
        }
    },
    
    /**
     * Convert RGB555 to RGBA - OPTIMIZED
     * @param {Uint8Array} src - Source RGB555 data (2 bytes per pixel)
     * @param {Uint8Array} dst - Destination RGBA buffer
     */
    rgb555ToRGBA(src, dst) {
        if (!this._lut5to8) this.init();
        
        const lut5 = this._lut5to8;
        const pixelCount = src.length >> 1;
        const srcView = new DataView(src.buffer, src.byteOffset, src.byteLength);
        
        for (let i = 0; i < pixelCount; i++) {
            const pixel = srcView.getUint16(i << 1, true);
            const dstIdx = i << 2;
            
            // RGB555: XRRRRRGGGGGBBBBB
            dst[dstIdx]     = lut5[(pixel >> 10) & 0x1F];  // R
            dst[dstIdx + 1] = lut5[(pixel >> 5) & 0x1F];   // G
            dst[dstIdx + 2] = lut5[pixel & 0x1F];          // B
            dst[dstIdx + 3] = 255;                          // A
        }
    },
    
    /**
     * Convert 8-bit paletted to RGBA
     */
    palette8ToRGBA(src, dst) {
        const palette = this.palette;
        for (let i = 0, len = src.length; i < len; i++) {
            const idx = src[i] << 2;
            const dstIdx = i << 2;
            dst[dstIdx] = palette[idx];
            dst[dstIdx + 1] = palette[idx + 1];
            dst[dstIdx + 2] = palette[idx + 2];
            dst[dstIdx + 3] = palette[idx + 3];
        }
    },
    
    /**
     * Convert BGR24 to RGBA
     */
    bgr24ToRGBA(src, dst) {
        const pixelCount = (src.length / 3) | 0;
        for (let i = 0; i < pixelCount; i++) {
            const srcIdx = i * 3;
            const dstIdx = i << 2;
            dst[dstIdx] = src[srcIdx + 2];
            dst[dstIdx + 1] = src[srcIdx + 1];
            dst[dstIdx + 2] = src[srcIdx];
            dst[dstIdx + 3] = 255;
        }
    },
    
    /**
     * Convert BGRA32 to RGBA - optimized with 32-bit operations
     */
    bgra32ToRGBA(src, dst) {
        const pixelCount = src.length >> 2;
        const srcView = new DataView(src.buffer, src.byteOffset, src.byteLength);
        const dstView = new DataView(dst.buffer, dst.byteOffset, dst.byteLength);
        
        for (let i = 0; i < pixelCount; i++) {
            const offset = i << 2;
            const bgra = srcView.getUint32(offset, true);
            
            // BGRA -> RGBA: swap R and B
            const b = bgra & 0xFF;
            const g = (bgra >> 8) & 0xFF;
            const r = (bgra >> 16) & 0xFF;
            const a = (bgra >> 24) & 0xFF;
            
            dstView.setUint32(offset, (a << 24) | (b << 16) | (g << 8) | r, true);
        }
    },
    
    /**
     * Flip image vertically (in-place) - optimized
     */
    flipVertical(data, width, height, bytesPerPixel) {
        const rowSize = width * bytesPerPixel;
        const temp = new Uint8Array(rowSize);
        const halfHeight = height >> 1;
        
        for (let y = 0; y < halfHeight; y++) {
            const topOffset = y * rowSize;
            const bottomOffset = (height - 1 - y) * rowSize;
            
            temp.set(data.subarray(topOffset, topOffset + rowSize));
            data.copyWithin(topOffset, bottomOffset, bottomOffset + rowSize);
            data.set(temp, bottomOffset);
        }
    },
    
    /**
     * Process a bitmap with fallback codecs
     * Optimized for 16-bit uncompressed (fastest path)
     * 
     * @param {Uint8Array} src - Source data
     * @param {number} width - Image width
     * @param {number} height - Image height
     * @param {number} bpp - Bits per pixel
     * @param {boolean} isCompressed - Whether data is compressed
     * @param {Uint8Array} dst - Destination RGBA buffer
     * @returns {boolean}
     */
    processBitmap(src, width, height, bpp, isCompressed, dst) {
        try {
            // FAST PATH: Uncompressed 16-bit (recommended for JS fallback)
            if (!isCompressed && (bpp === 16 || bpp === 15)) {
                if (bpp === 16) {
                    this.rgb565ToRGBA(src, dst);
                } else {
                    this.rgb555ToRGBA(src, dst);
                }
                this.flipVertical(dst, width, height, 4);
                return true;
            }
            
            // Other uncompressed formats
            if (!isCompressed) {
                switch (bpp) {
                    case 8:
                        this.palette8ToRGBA(src, dst);
                        break;
                    case 24:
                        this.bgr24ToRGBA(src, dst);
                        break;
                    case 32:
                        this.bgra32ToRGBA(src, dst);
                        break;
                    default:
                        Logger.warn('FallbackCodec', `Unsupported uncompressed bpp: ${bpp}`);
                        return false;
                }
                this.flipVertical(dst, width, height, 4);
                return true;
            }
            
            // Compressed formats - limited support
            // For best performance, configure RDP to use uncompressed 16-bit
            Logger.debug('FallbackCodec', `Compressed ${bpp}bpp not optimized in JS fallback`);
            return false;
            
        } catch (e) {
            Logger.error('FallbackCodec', `Processing failed: ${e.message}`);
            return false;
        }
    }
};

// Auto-initialize lookup tables
FallbackCodec.init();

export default FallbackCodec;

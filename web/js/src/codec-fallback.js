/**
 * JavaScript Fallback Codecs
 * Pure JS implementations for when WASM is unavailable
 * These are slower but provide basic functionality
 * @module codec-fallback
 */

import { Logger } from './logger.js';

/**
 * Fallback codec implementation in pure JavaScript
 * Used when WASM is not available or fails to load
 */
export const FallbackCodec = {
    palette: new Uint8Array(256 * 4),  // RGBA palette
    
    /**
     * Set color palette for 8-bit mode
     * @param {Uint8Array} data - RGB palette data (3 bytes per color)
     * @param {number} numColors - Number of colors
     */
    setPalette(data, numColors) {
        const count = Math.min(numColors, 256);
        for (let i = 0; i < count; i++) {
            this.palette[i * 4] = data[i * 3];       // R
            this.palette[i * 4 + 1] = data[i * 3 + 1]; // G
            this.palette[i * 4 + 2] = data[i * 3 + 2]; // B
            this.palette[i * 4 + 3] = 255;            // A
        }
    },
    
    /**
     * Convert 8-bit paletted to RGBA
     * @param {Uint8Array} src - Source paletted data
     * @param {Uint8Array} dst - Destination RGBA buffer
     */
    palette8ToRGBA(src, dst) {
        for (let i = 0; i < src.length; i++) {
            const idx = src[i] * 4;
            const dstIdx = i * 4;
            dst[dstIdx] = this.palette[idx];
            dst[dstIdx + 1] = this.palette[idx + 1];
            dst[dstIdx + 2] = this.palette[idx + 2];
            dst[dstIdx + 3] = this.palette[idx + 3];
        }
    },
    
    /**
     * Convert RGB555 to RGBA
     * @param {Uint8Array} src - Source RGB555 data (2 bytes per pixel)
     * @param {Uint8Array} dst - Destination RGBA buffer
     */
    rgb555ToRGBA(src, dst) {
        const pixelCount = Math.floor(src.length / 2);
        for (let i = 0; i < pixelCount; i++) {
            const pixel = src[i * 2] | (src[i * 2 + 1] << 8);
            const dstIdx = i * 4;
            // RGB555: XRRRRRGGGGGBBBBB
            dst[dstIdx] = ((pixel >> 10) & 0x1F) << 3;     // R
            dst[dstIdx + 1] = ((pixel >> 5) & 0x1F) << 3;  // G
            dst[dstIdx + 2] = (pixel & 0x1F) << 3;          // B
            dst[dstIdx + 3] = 255;                          // A
        }
    },
    
    /**
     * Convert RGB565 to RGBA
     * @param {Uint8Array} src - Source RGB565 data (2 bytes per pixel)
     * @param {Uint8Array} dst - Destination RGBA buffer
     */
    rgb565ToRGBA(src, dst) {
        const pixelCount = Math.floor(src.length / 2);
        for (let i = 0; i < pixelCount; i++) {
            const pixel = src[i * 2] | (src[i * 2 + 1] << 8);
            const dstIdx = i * 4;
            // RGB565: RRRRRGGGGGGBBBBB
            dst[dstIdx] = ((pixel >> 11) & 0x1F) << 3;     // R
            dst[dstIdx + 1] = ((pixel >> 5) & 0x3F) << 2;  // G
            dst[dstIdx + 2] = (pixel & 0x1F) << 3;          // B
            dst[dstIdx + 3] = 255;                          // A
        }
    },
    
    /**
     * Convert BGR24 to RGBA
     * @param {Uint8Array} src - Source BGR24 data (3 bytes per pixel)
     * @param {Uint8Array} dst - Destination RGBA buffer
     */
    bgr24ToRGBA(src, dst) {
        const pixelCount = Math.floor(src.length / 3);
        for (let i = 0; i < pixelCount; i++) {
            const srcIdx = i * 3;
            const dstIdx = i * 4;
            dst[dstIdx] = src[srcIdx + 2];     // R <- B position
            dst[dstIdx + 1] = src[srcIdx + 1]; // G
            dst[dstIdx + 2] = src[srcIdx];     // B <- R position
            dst[dstIdx + 3] = 255;              // A
        }
    },
    
    /**
     * Convert BGRA32 to RGBA
     * @param {Uint8Array} src - Source BGRA32 data (4 bytes per pixel)
     * @param {Uint8Array} dst - Destination RGBA buffer
     */
    bgra32ToRGBA(src, dst) {
        const pixelCount = Math.floor(src.length / 4);
        for (let i = 0; i < pixelCount; i++) {
            const srcIdx = i * 4;
            const dstIdx = i * 4;
            dst[dstIdx] = src[srcIdx + 2];     // R <- B position
            dst[dstIdx + 1] = src[srcIdx + 1]; // G
            dst[dstIdx + 2] = src[srcIdx];     // B <- R position
            dst[dstIdx + 3] = src[srcIdx + 3]; // A
        }
    },
    
    /**
     * Flip image vertically (in-place)
     * @param {Uint8Array} data - Image data
     * @param {number} width - Image width
     * @param {number} height - Image height
     * @param {number} bytesPerPixel - Bytes per pixel
     */
    flipVertical(data, width, height, bytesPerPixel) {
        const rowSize = width * bytesPerPixel;
        const temp = new Uint8Array(rowSize);
        const halfHeight = Math.floor(height / 2);
        
        for (let y = 0; y < halfHeight; y++) {
            const topOffset = y * rowSize;
            const bottomOffset = (height - 1 - y) * rowSize;
            
            // Swap rows
            temp.set(data.subarray(topOffset, topOffset + rowSize));
            data.set(data.subarray(bottomOffset, bottomOffset + rowSize), topOffset);
            data.set(temp, bottomOffset);
        }
    },
    
    /**
     * Decompress RLE8 data (basic implementation)
     * @param {Uint8Array} src - Compressed data
     * @param {Uint8Array} dst - Output buffer
     * @param {number} width - Image width
     * @param {number} height - Image height
     * @returns {boolean}
     */
    decompressRLE8(src, dst, width, height) {
        let srcIdx = 0;
        let dstIdx = 0;
        const dstSize = width * height;
        
        while (srcIdx < src.length && dstIdx < dstSize) {
            const code = src[srcIdx++];
            
            if (code === 0) {
                // Escape sequence
                if (srcIdx >= src.length) break;
                const escape = src[srcIdx++];
                
                if (escape === 0) {
                    // End of line - pad to next row
                    const rowPos = dstIdx % width;
                    if (rowPos > 0) {
                        dstIdx += (width - rowPos);
                    }
                } else if (escape === 1) {
                    // End of bitmap
                    break;
                } else if (escape === 2) {
                    // Delta - skip pixels
                    if (srcIdx + 1 >= src.length) break;
                    const dx = src[srcIdx++];
                    const dy = src[srcIdx++];
                    dstIdx += dx + dy * width;
                } else {
                    // Absolute mode - copy literal bytes
                    const count = escape;
                    for (let i = 0; i < count && srcIdx < src.length && dstIdx < dstSize; i++) {
                        dst[dstIdx++] = src[srcIdx++];
                    }
                    // Padding for word alignment
                    if (count & 1) srcIdx++;
                }
            } else {
                // Run of pixels
                if (srcIdx >= src.length) break;
                const pixel = src[srcIdx++];
                for (let i = 0; i < code && dstIdx < dstSize; i++) {
                    dst[dstIdx++] = pixel;
                }
            }
        }
        
        return true;
    },
    
    /**
     * Process a bitmap with fallback codecs
     * @param {Uint8Array} src - Source data
     * @param {number} width - Image width
     * @param {number} height - Image height
     * @param {number} bpp - Bits per pixel
     * @param {boolean} isCompressed - Whether data is compressed
     * @param {Uint8Array} dst - Destination RGBA buffer
     * @returns {boolean}
     */
    processBitmap(src, width, height, bpp, isCompressed, dst) {
        const pixelCount = width * height;
        
        try {
            if (isCompressed) {
                // Only basic RLE8 decompression supported
                if (bpp === 8) {
                    const temp = new Uint8Array(pixelCount);
                    if (this.decompressRLE8(src, temp, width, height)) {
                        this.palette8ToRGBA(temp, dst);
                        this.flipVertical(dst, width, height, 4);
                        return true;
                    }
                }
                // Other compressed formats not supported in fallback
                Logger.warn('FallbackCodec', `Compressed ${bpp}bpp not supported in JS fallback`);
                return false;
            }
            
            // Uncompressed data
            switch (bpp) {
                case 8:
                    this.palette8ToRGBA(src, dst);
                    break;
                case 15:
                    this.rgb555ToRGBA(src, dst);
                    break;
                case 16:
                    this.rgb565ToRGBA(src, dst);
                    break;
                case 24:
                    this.bgr24ToRGBA(src, dst);
                    break;
                case 32:
                    this.bgra32ToRGBA(src, dst);
                    break;
                default:
                    Logger.warn('FallbackCodec', `Unsupported bpp: ${bpp}`);
                    return false;
            }
            
            // Flip vertically (RDP sends bottom-up)
            this.flipVertical(dst, width, height, 4);
            return true;
            
        } catch (e) {
            Logger.error('FallbackCodec', `Processing failed: ${e.message}`);
            return false;
        }
    }
};

export default FallbackCodec;

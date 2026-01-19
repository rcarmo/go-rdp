/**
 * NSCodec Decoder - JavaScript implementation of the MS-RDPNSC codec
 * 
 * NSCodec compresses bitmaps using:
 * 1. ARGB to AYCoCg color space conversion
 * 2. Optional chroma subsampling
 * 3. Optional color loss reduction
 * 4. RLE compression per plane
 */

// Export to window for global access
window.NSCodec = (function() {
    'use strict';

    /**
     * Parse NSCODEC_BITMAP_STREAM structure
     * @param {Uint8Array} data - Raw bitmap stream data
     * @returns {Object} Parsed stream structure
     */
    function parseBitmapStream(data) {
        if (data.length < 20) {
            throw new Error('NSCodec: Invalid bitmap stream (too short)');
        }

        const view = new DataView(data.buffer, data.byteOffset, data.byteLength);
        
        const stream = {
            lumaPlaneByteCount: view.getUint32(0, true),
            orangeChromaPlaneByteCount: view.getUint32(4, true),
            greenChromaPlaneByteCount: view.getUint32(8, true),
            alphaPlaneByteCount: view.getUint32(12, true),
            colorLossLevel: data[16],
            chromaSubsamplingLevel: data[17],
            // Reserved: data[18:20]
        };

        if (stream.colorLossLevel < 1 || stream.colorLossLevel > 7) {
            throw new Error('NSCodec: Invalid color loss level: ' + stream.colorLossLevel);
        }

        let offset = 20;

        // Parse luma plane
        if (stream.lumaPlaneByteCount > 0) {
            if (data.length < offset + stream.lumaPlaneByteCount) {
                throw new Error('NSCodec: Invalid luma plane size');
            }
            stream.lumaPlane = data.subarray(offset, offset + stream.lumaPlaneByteCount);
            offset += stream.lumaPlaneByteCount;
        }

        // Parse orange chroma plane
        if (stream.orangeChromaPlaneByteCount > 0) {
            if (data.length < offset + stream.orangeChromaPlaneByteCount) {
                throw new Error('NSCodec: Invalid orange chroma plane size');
            }
            stream.orangeChromaPlane = data.subarray(offset, offset + stream.orangeChromaPlaneByteCount);
            offset += stream.orangeChromaPlaneByteCount;
        }

        // Parse green chroma plane
        if (stream.greenChromaPlaneByteCount > 0) {
            if (data.length < offset + stream.greenChromaPlaneByteCount) {
                throw new Error('NSCodec: Invalid green chroma plane size');
            }
            stream.greenChromaPlane = data.subarray(offset, offset + stream.greenChromaPlaneByteCount);
            offset += stream.greenChromaPlaneByteCount;
        }

        // Parse alpha plane (optional)
        if (stream.alphaPlaneByteCount > 0) {
            if (data.length < offset + stream.alphaPlaneByteCount) {
                throw new Error('NSCodec: Invalid alpha plane size');
            }
            stream.alphaPlane = data.subarray(offset, offset + stream.alphaPlaneByteCount);
        }

        return stream;
    }

    /**
     * Decompress RLE-encoded plane or return raw data
     * @param {Uint8Array} data - Compressed or raw plane data
     * @param {number} expectedSize - Expected decompressed size
     * @returns {Uint8Array} Decompressed plane data
     */
    function decompressPlane(data, expectedSize) {
        if (data.length === expectedSize) {
            // Raw data, no decompression needed
            return data;
        }

        if (data.length > expectedSize) {
            throw new Error('NSCodec: Plane data larger than expected');
        }

        // RLE compressed - decompress
        return rleDecompress(data, expectedSize);
    }

    /**
     * Decompress NSCodec RLE data
     * Format: segments followed by 4-byte EndData (last 4 raw bytes)
     * @param {Uint8Array} data - RLE compressed data
     * @param {number} expectedSize - Expected decompressed size
     * @returns {Uint8Array} Decompressed data
     */
    function rleDecompress(data, expectedSize) {
        if (data.length < 4) {
            throw new Error('NSCodec: RLE data too short');
        }

        const result = new Uint8Array(expectedSize);
        let resultOffset = 0;
        let offset = 0;
        const dataLen = data.length - 4; // Exclude EndData

        while (offset < dataLen && resultOffset < expectedSize - 4) {
            if (offset >= dataLen) break;

            const header = data[offset++];

            if (header & 0x80) {
                // Run segment: repeat single byte
                let runLength = header & 0x7F;
                if (runLength === 0) {
                    // Extended run length
                    if (offset >= dataLen) break;
                    runLength = data[offset++] + 128;
                }
                if (offset >= dataLen) break;
                const runValue = data[offset++];

                for (let i = 0; i < runLength && resultOffset < expectedSize - 4; i++) {
                    result[resultOffset++] = runValue;
                }
            } else {
                // Literal segment: copy raw bytes
                let literalLength = header;
                if (literalLength === 0) {
                    // Extended literal length
                    if (offset >= dataLen) break;
                    literalLength = data[offset++] + 128;
                }

                if (offset + literalLength > dataLen) break;

                for (let i = 0; i < literalLength && resultOffset < expectedSize - 4; i++) {
                    result[resultOffset++] = data[offset++];
                }
            }
        }

        // Append EndData (last 4 bytes of original plane)
        const endData = data.subarray(data.length - 4);
        for (let i = 0; i < 4 && resultOffset < expectedSize; i++) {
            result[resultOffset++] = endData[i];
        }

        return result;
    }

    /**
     * Round up n to the nearest multiple of m
     */
    function roundUpToMultiple(n, m) {
        if (m === 0) return n;
        const remainder = n % m;
        if (remainder === 0) return n;
        return n + m - remainder;
    }

    /**
     * Upsample chroma planes from subsampled to full resolution
     */
    function chromaSuperSample(plane, srcWidth, srcHeight, dstWidth, dstHeight) {
        const result = new Uint8Array(dstWidth * dstHeight);

        for (let y = 0; y < dstHeight; y++) {
            let srcY = Math.floor(y / 2);
            if (srcY >= srcHeight) srcY = srcHeight - 1;

            for (let x = 0; x < dstWidth; x++) {
                let srcX = Math.floor(x / 2);
                if (srcX >= srcWidth) srcX = srcWidth - 1;

                const srcIdx = srcY * srcWidth + srcX;
                const dstIdx = y * dstWidth + x;

                if (srcIdx < plane.length) {
                    result[dstIdx] = plane[srcIdx];
                }
            }
        }

        return result;
    }

    /**
     * Restore color values that were quantized during compression
     */
    function restoreColorLoss(plane, colorLossLevel) {
        if (colorLossLevel <= 1) return plane;

        const shift = colorLossLevel - 1;
        const result = new Uint8Array(plane.length);

        for (let i = 0; i < plane.length; i++) {
            let restored = plane[i] << shift;
            if (restored > 255) restored = 255;
            result[i] = restored;
        }

        return result;
    }

    /**
     * Clamp value to 0-255 range
     */
    function clamp(v) {
        if (v < 0) return 0;
        if (v > 255) return 255;
        return v;
    }

    /**
     * Convert AYCoCg color space to RGBA
     * @param {Uint8Array} luma - Luma (Y) plane
     * @param {Uint8Array} co - Orange chroma (Co) plane
     * @param {Uint8Array} cg - Green chroma (Cg) plane
     * @param {Uint8Array|null} alpha - Alpha plane (optional)
     * @param {number} planeWidth - Width of the planes
     * @param {number} planeHeight - Height of the planes
     * @param {number} imgWidth - Output image width
     * @param {number} imgHeight - Output image height
     * @returns {Uint8Array} RGBA pixel data
     */
    function aycoCgToRGBA(luma, co, cg, alpha, planeWidth, planeHeight, imgWidth, imgHeight) {
        const rgba = new Uint8Array(imgWidth * imgHeight * 4);

        for (let y = 0; y < imgHeight; y++) {
            for (let x = 0; x < imgWidth; x++) {
                const planeIdx = y * planeWidth + x;
                const rgbaIdx = (y * imgWidth + x) * 4;

                if (planeIdx >= luma.length || planeIdx >= co.length || planeIdx >= cg.length) {
                    continue;
                }

                // Get YCoCg values (shifted to signed range)
                const yVal = luma[planeIdx];
                const coVal = co[planeIdx] - 128;
                const cgVal = cg[planeIdx] - 128;

                // YCoCg to RGB conversion
                // t = Y - Cg
                // R = t + Co
                // G = Y + Cg
                // B = t - Co
                const t = yVal - cgVal;
                rgba[rgbaIdx + 0] = clamp(t + coVal);  // R
                rgba[rgbaIdx + 1] = clamp(yVal + cgVal); // G
                rgba[rgbaIdx + 2] = clamp(t - coVal);  // B

                // Alpha
                if (alpha && planeIdx < alpha.length) {
                    rgba[rgbaIdx + 3] = alpha[planeIdx];
                } else {
                    rgba[rgbaIdx + 3] = 255;
                }
            }
        }

        return rgba;
    }

    /**
     * Decode NSCodec bitmap stream to RGBA pixels
     * @param {Uint8Array} data - Raw NSCodec bitmap stream
     * @param {number} width - Image width
     * @param {number} height - Image height
     * @returns {Uint8Array} RGBA pixel data (4 bytes per pixel)
     */
    function decode(data, width, height) {
        const stream = parseBitmapStream(data);
        const chromaSubsampling = stream.chromaSubsamplingLevel !== 0;

        // Calculate expected plane sizes
        let lumaWidth, lumaHeight;
        let chromaWidth, chromaHeight;

        if (chromaSubsampling) {
            lumaWidth = roundUpToMultiple(width, 8);
            lumaHeight = height;
            chromaWidth = lumaWidth / 2;
            chromaHeight = roundUpToMultiple(height, 2) / 2;
        } else {
            lumaWidth = width;
            lumaHeight = height;
            chromaWidth = width;
            chromaHeight = height;
        }

        const lumaExpectedSize = lumaWidth * lumaHeight;
        const chromaExpectedSize = chromaWidth * chromaHeight;

        // Decompress or use raw planes
        let lumaPlane = decompressPlane(stream.lumaPlane, lumaExpectedSize);
        let orangePlane = decompressPlane(stream.orangeChromaPlane, chromaExpectedSize);
        let greenPlane = decompressPlane(stream.greenChromaPlane, chromaExpectedSize);

        // Decompress alpha plane if present
        let alphaPlane = null;
        if (stream.alphaPlaneByteCount > 0) {
            const alphaExpectedSize = width * height;
            alphaPlane = decompressPlane(stream.alphaPlane, alphaExpectedSize);
        }

        // Apply chroma super-sampling if needed
        if (chromaSubsampling) {
            orangePlane = chromaSuperSample(orangePlane, chromaWidth, chromaHeight, lumaWidth, lumaHeight);
            greenPlane = chromaSuperSample(greenPlane, chromaWidth, chromaHeight, lumaWidth, lumaHeight);
        }

        // Apply color loss restoration
        if (stream.colorLossLevel > 1) {
            orangePlane = restoreColorLoss(orangePlane, stream.colorLossLevel);
            greenPlane = restoreColorLoss(greenPlane, stream.colorLossLevel);
        }

        // Convert AYCoCg to RGBA
        return aycoCgToRGBA(lumaPlane, orangePlane, greenPlane, alphaPlane, lumaWidth, lumaHeight, width, height);
    }

    // Public API
    return {
        decode: decode,
        parseBitmapStream: parseBitmapStream
    };
})();

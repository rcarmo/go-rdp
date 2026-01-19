// Flip bitmap vertically (RDP sends bottom-up, canvas expects top-down)
window.flipV = function flipV(inA, width, height, bytesPerPixel) {
    // Default to 2 bytes per pixel for backward compatibility
    if (bytesPerPixel === undefined) {
        bytesPerPixel = 2;
    }
    var rowDelta = width * bytesPerPixel;
    var half = Math.floor(height / 2);
    var bottomLine = rowDelta * (height - 1);
    var topLine = 0;
    var tmp = new Uint8Array(rowDelta);

    for (var i = 0; i < half; ++i) {
        tmp.set(inA.subarray(topLine, topLine + rowDelta));
        inA.set(inA.subarray(bottomLine, bottomLine + rowDelta), topLine);
        inA.set(tmp, bottomLine);

        topLine += rowDelta;
        bottomLine -= rowDelta;
    }
};

// Convert 16-bit RGB565 to 32-bit RGBA
window.rgb565toRGBA = function rgb565toRGBA(inA, inLength, outA) {
    var inI = 0;
    var outI = 0;
    while (inI < inLength) {
        var pel = inA[inI] | (inA[inI + 1] << 8);
        var pelR = (pel & 0xF800) >> 11;
        var pelG = (pel & 0x7E0) >> 5;
        var pelB = pel & 0x1F;
        // 565 -> 888
        pelR = (pelR << 3 & ~0x7) | (pelR >> 2);
        pelG = (pelG << 2 & ~0x3) | (pelG >> 4);
        pelB = (pelB << 3 & ~0x7) | (pelB >> 2);

        outA[outI++] = pelR;
        outA[outI++] = pelG;
        outA[outI++] = pelB;
        outA[outI++] = 255; // alpha
        inI += 2;
    }
};

// Convert 24-bit BGR to 32-bit RGBA
window.bgr24toRGBA = function bgr24toRGBA(inA, inLength, outA) {
    var inI = 0;
    var outI = 0;
    while (inI < inLength) {
        outA[outI++] = inA[inI + 2]; // R (from B position)
        outA[outI++] = inA[inI + 1]; // G
        outA[outI++] = inA[inI];     // B (from R position)
        outA[outI++] = 255;          // alpha
        inI += 3;
    }
};

// Convert 32-bit BGRA to 32-bit RGBA (just swap R and B)
window.bgra32toRGBA = function bgra32toRGBA(inA, outA) {
    for (var i = 0; i < inA.length; i += 4) {
        outA[i] = inA[i + 2];     // R (from B position)
        outA[i + 1] = inA[i + 1]; // G
        outA[i + 2] = inA[i];     // B (from R position)
        outA[i + 3] = 255;        // alpha (ignore source alpha, always opaque)
    }
};

// Legacy aliases for backward compatibility
window.rgb2rgba = function rgb2rgba(inA, inLength, outA) {
    rgb565toRGBA(inA, inLength, outA);
};

window.buf2RGBA = function buf2RGBA(inA, inI, outA, outI) {
    var pel = inA[inI] | (inA[inI + 1] << 8);
    var pelR = (pel & 0xF800) >> 11;
    var pelG = (pel & 0x7E0) >> 5;
    var pelB = pel & 0x1F;
    pelR = (pelR << 3 & ~0x7) | (pelR >> 2);
    pelG = (pelG << 2 & ~0x3) | (pelG >> 4);
    pelB = (pelB << 3 & ~0x7) | (pelB >> 2);

    outA[outI++] = pelR;
    outA[outI++] = pelG;
    outA[outI++] = pelB;
    outA[outI] = 255;
};

// 16-bit RLE decompression (MS-RDPBCGR spec)
// JavaScript fallback when WASM is unavailable
window.rleDecompress16 = function rleDecompress16(src, dest, rowDelta) {
    // RLE order codes
    var REGULAR_BG_RUN = 0x0;
    var MEGA_MEGA_BG_RUN = 0xF0;
    var REGULAR_FG_RUN = 0x1;
    var MEGA_MEGA_FG_RUN = 0xF1;
    var LITE_SET_FG_FG_RUN = 0xC;
    var MEGA_MEGA_SET_FG_RUN = 0xF6;
    var LITE_DITHERED_RUN = 0xE;
    var MEGA_MEGA_DITHERED_RUN = 0xF8;
    var REGULAR_COLOR_RUN = 0x3;
    var MEGA_MEGA_COLOR_RUN = 0xF3;
    var REGULAR_FGBG_IMAGE = 0x2;
    var MEGA_MEGA_FGBG_IMAGE = 0xF2;
    var LITE_SET_FG_FGBG_IMAGE = 0xD;
    var MEGA_MEGA_SET_FGBG_IMAGE = 0xF7;
    var REGULAR_COLOR_IMAGE = 0x4;
    var MEGA_MEGA_COLOR_IMAGE = 0xF4;
    var SPECIAL_FGBG_1 = 0xF9;
    var SPECIAL_FGBG_2 = 0xFA;
    var WHITE = 0xFD;
    var BLACK = 0xFE;

    var srcIdx = 0;
    var destIdx = 0;
    var fgPel = 0xFFFF;
    var insertFgPel = false;
    var firstLine = true;

    function readPixel(arr, idx) {
        if (idx + 1 >= arr.length) return 0;
        return arr[idx] | (arr[idx + 1] << 8);
    }

    function writePixel(arr, idx, val) {
        if (idx + 1 >= arr.length) return;
        arr[idx] = val & 0xFF;
        arr[idx + 1] = (val >> 8) & 0xFF;
    }

    function extractCode(b) {
        if ((b & 0xC0) === 0xC0) return b; // LITE/MEGA codes
        return (b >> 4) & 0x0F;
    }

    function extractRunLength(code, src, idx) {
        var runLength = 0;
        var advance = 1;

        if (code === REGULAR_BG_RUN || code === REGULAR_FG_RUN || 
            code === REGULAR_COLOR_RUN || code === REGULAR_FGBG_IMAGE || 
            code === REGULAR_COLOR_IMAGE) {
            runLength = src[idx] & 0x1F;
            if (runLength === 0) {
                runLength = src[idx + 1] + 1;
                advance = 2;
            } else {
                runLength = runLength * 8;
            }
        } else if (code === LITE_SET_FG_FG_RUN || code === LITE_DITHERED_RUN || 
                   code === LITE_SET_FG_FGBG_IMAGE) {
            runLength = src[idx] & 0x0F;
            if (runLength === 0) {
                runLength = src[idx + 1] + 1;
                advance = 2;
            } else {
                runLength = runLength * 8;
            }
        } else if (code >= 0xF0 && code <= 0xF8) {
            runLength = src[idx + 1] | (src[idx + 2] << 8);
            advance = 3;
        }
        
        return { runLength: runLength, nextIdx: idx + advance };
    }

    while (srcIdx < src.length && destIdx < dest.length) {
        if (firstLine && destIdx >= rowDelta) {
            firstLine = false;
            insertFgPel = false;
        }

        var code = extractCode(src[srcIdx]);

        // Background Run
        if (code === REGULAR_BG_RUN || code === MEGA_MEGA_BG_RUN) {
            var result = extractRunLength(code, src, srcIdx);
            var runLength = result.runLength;
            srcIdx = result.nextIdx;

            if (firstLine) {
                if (insertFgPel) {
                    writePixel(dest, destIdx, fgPel);
                    destIdx += 2;
                    runLength--;
                }
                while (runLength > 0 && destIdx < dest.length) {
                    writePixel(dest, destIdx, 0);
                    destIdx += 2;
                    runLength--;
                }
            } else {
                if (insertFgPel) {
                    var prev = readPixel(dest, destIdx - rowDelta);
                    writePixel(dest, destIdx, prev ^ fgPel);
                    destIdx += 2;
                    runLength--;
                }
                while (runLength > 0 && destIdx < dest.length) {
                    var prev = readPixel(dest, destIdx - rowDelta);
                    writePixel(dest, destIdx, prev);
                    destIdx += 2;
                    runLength--;
                }
            }
            insertFgPel = true;
            continue;
        }

        insertFgPel = false;

        // Foreground Run
        if (code === REGULAR_FG_RUN || code === MEGA_MEGA_FG_RUN ||
            code === LITE_SET_FG_FG_RUN || code === MEGA_MEGA_SET_FG_RUN) {
            var result = extractRunLength(code, src, srcIdx);
            var runLength = result.runLength;
            srcIdx = result.nextIdx;

            if (code === LITE_SET_FG_FG_RUN || code === MEGA_MEGA_SET_FG_RUN) {
                fgPel = readPixel(src, srcIdx);
                srcIdx += 2;
            }

            while (runLength > 0 && destIdx < dest.length) {
                if (firstLine) {
                    writePixel(dest, destIdx, fgPel);
                } else {
                    var prev = readPixel(dest, destIdx - rowDelta);
                    writePixel(dest, destIdx, prev ^ fgPel);
                }
                destIdx += 2;
                runLength--;
            }
            continue;
        }

        // Dithered Run
        if (code === LITE_DITHERED_RUN || code === MEGA_MEGA_DITHERED_RUN) {
            var result = extractRunLength(code, src, srcIdx);
            var runLength = result.runLength;
            srcIdx = result.nextIdx;

            var pixelA = readPixel(src, srcIdx);
            srcIdx += 2;
            var pixelB = readPixel(src, srcIdx);
            srcIdx += 2;

            while (runLength > 0 && destIdx + 4 <= dest.length) {
                writePixel(dest, destIdx, pixelA);
                destIdx += 2;
                writePixel(dest, destIdx, pixelB);
                destIdx += 2;
                runLength--;
            }
            continue;
        }

        // Color Run
        if (code === REGULAR_COLOR_RUN || code === MEGA_MEGA_COLOR_RUN) {
            var result = extractRunLength(code, src, srcIdx);
            var runLength = result.runLength;
            srcIdx = result.nextIdx;

            var pixel = readPixel(src, srcIdx);
            srcIdx += 2;

            while (runLength > 0 && destIdx < dest.length) {
                writePixel(dest, destIdx, pixel);
                destIdx += 2;
                runLength--;
            }
            continue;
        }

        // FgBg Image
        if (code === REGULAR_FGBG_IMAGE || code === MEGA_MEGA_FGBG_IMAGE ||
            code === LITE_SET_FG_FGBG_IMAGE || code === MEGA_MEGA_SET_FGBG_IMAGE) {
            var result = extractRunLength(code, src, srcIdx);
            var runLength = result.runLength;
            srcIdx = result.nextIdx;

            if (code === LITE_SET_FG_FGBG_IMAGE || code === MEGA_MEGA_SET_FGBG_IMAGE) {
                fgPel = readPixel(src, srcIdx);
                srcIdx += 2;
            }

            while (runLength > 0 && srcIdx < src.length) {
                var bitmask = src[srcIdx++];
                var cBits = Math.min(runLength, 8);
                
                for (var i = 0; i < cBits; i++) {
                    if (firstLine) {
                        if (bitmask & (1 << i)) {
                            writePixel(dest, destIdx, fgPel);
                        } else {
                            writePixel(dest, destIdx, 0);
                        }
                    } else {
                        var prev = readPixel(dest, destIdx - rowDelta);
                        if (bitmask & (1 << i)) {
                            writePixel(dest, destIdx, prev ^ fgPel);
                        } else {
                            writePixel(dest, destIdx, prev);
                        }
                    }
                    destIdx += 2;
                }
                runLength -= cBits;
            }
            continue;
        }

        // Color Image (raw pixels)
        if (code === REGULAR_COLOR_IMAGE || code === MEGA_MEGA_COLOR_IMAGE) {
            var result = extractRunLength(code, src, srcIdx);
            var runLength = result.runLength;
            srcIdx = result.nextIdx;

            var byteCount = runLength * 2;
            for (var i = 0; i < byteCount && srcIdx < src.length && destIdx < dest.length; i++) {
                dest[destIdx++] = src[srcIdx++];
            }
            continue;
        }

        // Special FgBg 1
        if (code === SPECIAL_FGBG_1) {
            srcIdx++;
            var bitmask = 0x03;
            for (var i = 0; i < 8; i++) {
                if (firstLine) {
                    writePixel(dest, destIdx, (bitmask & (1 << i)) ? fgPel : 0);
                } else {
                    var prev = readPixel(dest, destIdx - rowDelta);
                    writePixel(dest, destIdx, (bitmask & (1 << i)) ? prev ^ fgPel : prev);
                }
                destIdx += 2;
            }
            continue;
        }

        // Special FgBg 2
        if (code === SPECIAL_FGBG_2) {
            srcIdx++;
            var bitmask = 0x05;
            for (var i = 0; i < 8; i++) {
                if (firstLine) {
                    writePixel(dest, destIdx, (bitmask & (1 << i)) ? fgPel : 0);
                } else {
                    var prev = readPixel(dest, destIdx - rowDelta);
                    writePixel(dest, destIdx, (bitmask & (1 << i)) ? prev ^ fgPel : prev);
                }
                destIdx += 2;
            }
            continue;
        }

        // White pixel
        if (code === WHITE) {
            srcIdx++;
            writePixel(dest, destIdx, 0xFFFF);
            destIdx += 2;
            continue;
        }

        // Black pixel
        if (code === BLACK) {
            srcIdx++;
            writePixel(dest, destIdx, 0);
            destIdx += 2;
            continue;
        }

        // Unknown - skip to prevent infinite loop
        srcIdx++;
    }

    return true;
};

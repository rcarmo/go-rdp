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

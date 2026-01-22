/**
 * RDP Protocol Parsing Functions
 * Handles FastPath and pointer update parsing
 * @module protocol
 */

/**
 * FastPath update codes
 */
export const UpdateCode = {
    ORDERS: 0x00,
    BITMAP: 0x01,
    PALETTE: 0x02,
    SYNCHRONIZE: 0x03,
    SURFCMDS: 0x04,
    PTR_NULL: 0x05,
    PTR_DEFAULT: 0x06,
    PTR_POSITION: 0x08,
    PTR_COLOR: 0x09,
    PTR_CACHED: 0x0A,
    PTR_NEW: 0x0B
};

/**
 * Update header parsed from FastPath data
 */
export class UpdateHeader {
    constructor(updateCode, fragmentation, compression, size) {
        this.updateCode = updateCode;
        this.fragmentation = fragmentation;
        this.compression = compression;
        this.size = size;
    }

    isBitmap() {
        return this.updateCode === UpdateCode.BITMAP;
    }

    isPointer() {
        return this.updateCode >= UpdateCode.PTR_NULL && this.updateCode <= UpdateCode.PTR_NEW;
    }

    isCompressed() {
        return this.compression !== 0;
    }

    isPTRNull() {
        return this.updateCode === UpdateCode.PTR_NULL;
    }

    isPTRDefault() {
        return this.updateCode === UpdateCode.PTR_DEFAULT;
    }

    isPTRColor() {
        return this.updateCode === UpdateCode.PTR_COLOR;
    }

    isPTRNew() {
        return this.updateCode === UpdateCode.PTR_NEW;
    }

    isPTRCached() {
        return this.updateCode === UpdateCode.PTR_CACHED;
    }

    isPTRPosition() {
        return this.updateCode === UpdateCode.PTR_POSITION;
    }
}

/**
 * Parse FastPath update header from binary reader
 * @param {BinaryReader} r - Binary reader positioned at start of update
 * @returns {UpdateHeader}
 */
export function parseUpdateHeader(r) {
    // FastPath header: 1 byte
    // bits 0-3: updateCode
    // bits 4-5: fragmentation
    // bits 6-7: compression
    const headerByte = r.uint8();
    const updateCode = headerByte & 0x0f;
    const fragmentation = (headerByte >> 4) & 0x03;
    const compression = (headerByte >> 6) & 0x03;
    
    // Size: 1 or 2 bytes
    // If high bit of first byte is set, size is 2 bytes
    let size;
    const sizeByte1 = r.uint8();
    if (sizeByte1 & 0x80) {
        const sizeByte2 = r.uint8();
        size = ((sizeByte1 & 0x7f) << 8) | sizeByte2;
    } else {
        size = sizeByte1;
    }
    
    return new UpdateHeader(updateCode, fragmentation, compression, size);
}

/**
 * New pointer update structure
 */
export class NewPointerUpdate {
    constructor(cacheIndex, x, y, width, height, xorBpp, andMask, xorMask) {
        this.cacheIndex = cacheIndex;
        this.x = x;
        this.y = y;
        this.width = width;
        this.height = height;
        this.xorBpp = xorBpp;
        this.andMask = andMask;
        this.xorMask = xorMask;
    }

    getImageData(ctx) {
        const imageData = ctx.createImageData(this.width, this.height);
        const data = imageData.data;

        // Convert cursor data to RGBA
        // XOR mask contains the color data, AND mask is the transparency
        const bytesPerPixel = this.xorBpp / 8;
        const xorRowBytes = Math.ceil((this.width * this.xorBpp) / 8);
        const andRowBytes = Math.ceil(this.width / 8);

        for (let y = 0; y < this.height; y++) {
            // Cursor data is bottom-up
            const srcY = this.height - 1 - y;
            
            for (let x = 0; x < this.width; x++) {
                const dstIdx = (y * this.width + x) * 4;
                
                // Get AND mask bit (transparency)
                const andByteIdx = srcY * andRowBytes + Math.floor(x / 8);
                const andBit = (this.andMask[andByteIdx] >> (7 - (x % 8))) & 1;
                
                // Get XOR mask color
                const xorByteIdx = srcY * xorRowBytes + x * bytesPerPixel;
                
                if (bytesPerPixel >= 3) {
                    // 24/32 bpp: BGR(A) format
                    data[dstIdx] = this.xorMask[xorByteIdx + 2];     // R
                    data[dstIdx + 1] = this.xorMask[xorByteIdx + 1]; // G
                    data[dstIdx + 2] = this.xorMask[xorByteIdx];     // B
                    data[dstIdx + 3] = andBit ? 0 : 255;             // A
                } else if (bytesPerPixel === 2) {
                    // 16 bpp: RGB565
                    const pixel = this.xorMask[xorByteIdx] | (this.xorMask[xorByteIdx + 1] << 8);
                    data[dstIdx] = ((pixel >> 11) & 0x1f) << 3;     // R
                    data[dstIdx + 1] = ((pixel >> 5) & 0x3f) << 2;  // G
                    data[dstIdx + 2] = (pixel & 0x1f) << 3;         // B
                    data[dstIdx + 3] = andBit ? 0 : 255;            // A
                } else {
                    // 1 bpp: monochrome
                    const xorBit = (this.xorMask[xorByteIdx] >> (7 - (x % 8))) & 1;
                    const color = xorBit ? 255 : 0;
                    data[dstIdx] = color;
                    data[dstIdx + 1] = color;
                    data[dstIdx + 2] = color;
                    data[dstIdx + 3] = andBit ? 0 : 255;
                }
            }
        }

        return imageData;
    }
}

/**
 * Parse new pointer update from binary reader
 * @param {BinaryReader} r - Binary reader
 * @returns {NewPointerUpdate}
 */
export function parseNewPointerUpdate(r) {
    const xorBpp = r.uint16(true);
    const cacheIndex = r.uint16(true);
    const x = r.uint16(true);  // hotspot X
    const y = r.uint16(true);  // hotspot Y
    const width = r.uint16(true);
    const height = r.uint16(true);
    const andMaskLen = r.uint16(true);
    const xorMaskLen = r.uint16(true);
    
    const xorMask = r.blob(xorMaskLen);
    const andMask = r.blob(andMaskLen);
    
    // Skip padding (1 byte if present)
    if (r.remaining() > 0) {
        r.skip(1);
    }
    
    return new NewPointerUpdate(cacheIndex, x, y, width, height, xorBpp, andMask, xorMask);
}

/**
 * Cached pointer update
 */
export class CachedPointerUpdate {
    constructor(cacheIndex) {
        this.cacheIndex = cacheIndex;
    }
}

/**
 * Parse cached pointer update
 * @param {BinaryReader} r
 * @returns {CachedPointerUpdate}
 */
export function parseCachedPointerUpdate(r) {
    const cacheIndex = r.uint16(true);
    return new CachedPointerUpdate(cacheIndex);
}

/**
 * Pointer position update
 */
export class PointerPositionUpdate {
    constructor(x, y) {
        this.x = x;
        this.y = y;
    }
}

/**
 * Parse pointer position update
 * @param {BinaryReader} r
 * @returns {PointerPositionUpdate}
 */
export function parsePointerPositionUpdate(r) {
    const x = r.uint16(true);
    const y = r.uint16(true);
    return new PointerPositionUpdate(x, y);
}

/**
 * Bitmap update containing multiple rectangles
 */
export class BitmapUpdate {
    constructor(rectangles) {
        this.rectangles = rectangles;
    }
}

/**
 * Single bitmap rectangle data
 */
export class BitmapData {
    constructor(destLeft, destTop, destRight, destBottom, width, height, bitsPerPixel, flags, bitmapLength, bitmapDataStream) {
        this.destLeft = destLeft;
        this.destTop = destTop;
        this.destRight = destRight;
        this.destBottom = destBottom;
        this.width = width;
        this.height = height;
        this.bitsPerPixel = bitsPerPixel;
        this.flags = flags;
        this.bitmapLength = bitmapLength;
        this.bitmapDataStream = bitmapDataStream;
    }

    isCompressed() {
        return (this.flags & 0x0001) !== 0;
    }

    hasNoCompressionHdr() {
        return (this.flags & 0x0400) !== 0;
    }
}

/**
 * Parse bitmap update from binary reader
 * Format: TS_UPDATE_BITMAP_DATA (MS-RDPBCGR 2.2.9.1.1.3.1.2)
 * @param {BinaryReader} r - Binary reader
 * @returns {BitmapUpdate}
 */
export function parseBitmapUpdate(r) {
    // updateType already consumed by caller or injected
    const updateType = r.uint16(true);
    const numberRectangles = r.uint16(true);
    
    const rectangles = [];
    for (let i = 0; i < numberRectangles; i++) {
        const destLeft = r.uint16(true);
        const destTop = r.uint16(true);
        const destRight = r.uint16(true);
        const destBottom = r.uint16(true);
        const width = r.uint16(true);
        const height = r.uint16(true);
        const bitsPerPixel = r.uint16(true);
        const flags = r.uint16(true);
        const bitmapLength = r.uint16(true);
        
        let bitmapDataStream;
        if (flags & 0x0001) {
            // Compressed
            if (flags & 0x0400) {
                // NO_BITMAP_COMPRESSION_HDR - data follows directly
                bitmapDataStream = r.blob(bitmapLength);
            } else {
                // Has compression header
                const cbCompFirstRowSize = r.uint16(true);
                const cbCompMainBodySize = r.uint16(true);
                const cbScanWidth = r.uint16(true);
                const cbUncompressedSize = r.uint16(true);
                bitmapDataStream = r.blob(cbCompMainBodySize);
            }
        } else {
            // Uncompressed
            bitmapDataStream = r.blob(bitmapLength);
        }
        
        rectangles.push(new BitmapData(
            destLeft, destTop, destRight, destBottom,
            width, height, bitsPerPixel, flags, bitmapLength, bitmapDataStream
        ));
    }
    
    return new BitmapUpdate(rectangles);
}

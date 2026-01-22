/**
 * RDP Protocol Parsing Functions
 * Handles FastPath updates, input events, and pointer parsing
 * @module protocol
 */

// ============================================================================
// Keyboard Scancode Mapping (US keyboard layout)
// ============================================================================
const SCANCODE_MAP = {
    'Escape': 0x01, 'Digit1': 0x02, 'Digit2': 0x03, 'Digit3': 0x04, 'Digit4': 0x05,
    'Digit5': 0x06, 'Digit6': 0x07, 'Digit7': 0x08, 'Digit8': 0x09, 'Digit9': 0x0A,
    'Digit0': 0x0B, 'Minus': 0x0C, 'Equal': 0x0D, 'Backspace': 0x0E, 'Tab': 0x09,
    'KeyQ': 0x10, 'KeyW': 0x11, 'KeyE': 0x12, 'KeyR': 0x13, 'KeyT': 0x14,
    'KeyY': 0x15, 'KeyU': 0x16, 'KeyI': 0x17, 'KeyO': 0x18, 'KeyP': 0x19,
    'BracketLeft': 0x1A, 'BracketRight': 0x1B, 'Enter': 0x1C, 'ControlLeft': 0x1D,
    'KeyA': 0x1E, 'KeyS': 0x1F, 'KeyD': 0x20, 'KeyF': 0x21, 'KeyG': 0x22,
    'KeyH': 0x23, 'KeyJ': 0x24, 'KeyK': 0x25, 'KeyL': 0x26, 'Semicolon': 0x27,
    'Quote': 0x28, 'Backquote': 0x29, 'ShiftLeft': 0x2A, 'Backslash': 0x2B,
    'KeyZ': 0x2C, 'KeyX': 0x2D, 'KeyC': 0x2E, 'KeyV': 0x2F, 'KeyB': 0x30,
    'KeyN': 0x31, 'KeyM': 0x32, 'Comma': 0x33, 'Period': 0x34, 'Slash': 0x35,
    'ShiftRight': 0x36, 'NumpadMultiply': 0x37, 'AltLeft': 0x38, 'Space': 0x39,
    'CapsLock': 0x3A, 'F1': 0x3B, 'F2': 0x3C, 'F3': 0x3D, 'F4': 0x3E, 'F5': 0x3F,
    'F6': 0x40, 'F7': 0x41, 'F8': 0x42, 'F9': 0x43, 'F10': 0x44,
    'NumLock': 0x45, 'ScrollLock': 0x46,
    'Numpad7': 0x47, 'Numpad8': 0x48, 'Numpad9': 0x49, 'NumpadSubtract': 0x4A,
    'Numpad4': 0x4B, 'Numpad5': 0x4C, 'Numpad6': 0x4D, 'NumpadAdd': 0x4E,
    'Numpad1': 0x4F, 'Numpad2': 0x50, 'Numpad3': 0x51, 'Numpad0': 0x52,
    'NumpadDecimal': 0x53, 'IntlBackslash': 0x56, 'F11': 0x57, 'F12': 0x58,
    'NumpadEnter': 0x1C, 'ControlRight': 0x1D, 'NumpadDivide': 0x35,
    'PrintScreen': 0x37, 'AltRight': 0x38,
    'Home': 0x47, 'ArrowUp': 0x48, 'PageUp': 0x49,
    'ArrowLeft': 0x4B, 'ArrowRight': 0x4D,
    'End': 0x4F, 'ArrowDown': 0x50, 'PageDown': 0x51,
    'Insert': 0x52, 'Delete': 0x53,
    'MetaLeft': 0x5B, 'MetaRight': 0x5C, 'ContextMenu': 0x5D,
};

// Extended keys that need the 0xE0 prefix
const EXTENDED_KEYS = new Set([
    'NumpadEnter', 'ControlRight', 'NumpadDivide', 'PrintScreen', 'AltRight',
    'Home', 'ArrowUp', 'PageUp', 'ArrowLeft', 'ArrowRight', 'End', 'ArrowDown',
    'PageDown', 'Insert', 'Delete', 'MetaLeft', 'MetaRight', 'ContextMenu'
]);

// ============================================================================
// FastPath Input Event Classes
// ============================================================================

/**
 * FastPath keyboard event - key down
 */
export class KeyboardEventKeyDown {
    constructor(code) {
        this.code = code;
        this.keyCode = SCANCODE_MAP[code];
        this.extended = EXTENDED_KEYS.has(code);
    }
    
    serialize() {
        // FastPath keyboard event format:
        // eventHeader (1 byte): eventFlags (5 bits) + eventCode (3 bits)
        // keyCode (1 byte): scancode
        const flags = this.extended ? 0x02 : 0x00; // KBDFLAGS_EXTENDED
        const eventHeader = (flags << 5) | 0x00; // FASTPATH_INPUT_EVENT_SCANCODE
        
        const data = new ArrayBuffer(2);
        const view = new DataView(data);
        view.setUint8(0, eventHeader);
        view.setUint8(1, this.keyCode || 0);
        return data;
    }
}

/**
 * FastPath keyboard event - key up
 */
export class KeyboardEventKeyUp {
    constructor(code) {
        this.code = code;
        this.keyCode = SCANCODE_MAP[code];
        this.extended = EXTENDED_KEYS.has(code);
    }
    
    serialize() {
        // KBDFLAGS_RELEASE = 0x01, KBDFLAGS_EXTENDED = 0x02
        const flags = 0x01 | (this.extended ? 0x02 : 0x00);
        const eventHeader = (flags << 5) | 0x00; // FASTPATH_INPUT_EVENT_SCANCODE
        
        const data = new ArrayBuffer(2);
        const view = new DataView(data);
        view.setUint8(0, eventHeader);
        view.setUint8(1, this.keyCode || 0);
        return data;
    }
}

/**
 * FastPath mouse move event
 */
export class MouseMoveEvent {
    constructor(x, y) {
        this.x = x;
        this.y = y;
    }
    
    serialize() {
        // FastPath mouse event format:
        // eventHeader (1 byte): eventFlags (5 bits) + eventCode (3 bits)
        // pointerFlags (2 bytes): PTRFLAGS_MOVE
        // xPos (2 bytes)
        // yPos (2 bytes)
        const PTRFLAGS_MOVE = 0x0800;
        const eventHeader = 0x01; // FASTPATH_INPUT_EVENT_MOUSE
        
        const data = new ArrayBuffer(7);
        const view = new DataView(data);
        view.setUint8(0, eventHeader);
        view.setUint16(1, PTRFLAGS_MOVE, true);
        view.setUint16(3, Math.max(0, this.x), true);
        view.setUint16(5, Math.max(0, this.y), true);
        return data;
    }
}

/**
 * FastPath mouse button down event
 */
export class MouseDownEvent {
    constructor(x, y, button) {
        this.x = x;
        this.y = y;
        this.button = button;
    }
    
    serialize() {
        const PTRFLAGS_DOWN = 0x8000;
        const PTRFLAGS_BUTTON1 = 0x1000;
        const PTRFLAGS_BUTTON2 = 0x2000;
        const PTRFLAGS_BUTTON3 = 0x4000;
        
        let flags = PTRFLAGS_DOWN;
        switch (this.button) {
            case 1: flags |= PTRFLAGS_BUTTON1; break;
            case 2: flags |= PTRFLAGS_BUTTON2; break;
            case 3: flags |= PTRFLAGS_BUTTON3; break;
        }
        
        const eventHeader = 0x01; // FASTPATH_INPUT_EVENT_MOUSE
        const data = new ArrayBuffer(7);
        const view = new DataView(data);
        view.setUint8(0, eventHeader);
        view.setUint16(1, flags, true);
        view.setUint16(3, Math.max(0, this.x), true);
        view.setUint16(5, Math.max(0, this.y), true);
        return data;
    }
}

/**
 * FastPath mouse button up event
 */
export class MouseUpEvent {
    constructor(x, y, button) {
        this.x = x;
        this.y = y;
        this.button = button;
    }
    
    serialize() {
        const PTRFLAGS_BUTTON1 = 0x1000;
        const PTRFLAGS_BUTTON2 = 0x2000;
        const PTRFLAGS_BUTTON3 = 0x4000;
        
        let flags = 0; // No DOWN flag = button up
        switch (this.button) {
            case 1: flags |= PTRFLAGS_BUTTON1; break;
            case 2: flags |= PTRFLAGS_BUTTON2; break;
            case 3: flags |= PTRFLAGS_BUTTON3; break;
        }
        
        const eventHeader = 0x01; // FASTPATH_INPUT_EVENT_MOUSE
        const data = new ArrayBuffer(7);
        const view = new DataView(data);
        view.setUint8(0, eventHeader);
        view.setUint16(1, flags, true);
        view.setUint16(3, Math.max(0, this.x), true);
        view.setUint16(5, Math.max(0, this.y), true);
        return data;
    }
}

/**
 * FastPath mouse wheel event
 */
export class MouseWheelEvent {
    constructor(x, y, delta, isNegative, isHorizontal) {
        this.x = x;
        this.y = y;
        this.delta = delta;
        this.isNegative = isNegative;
        this.isHorizontal = isHorizontal;
    }
    
    serialize() {
        const PTRFLAGS_WHEEL = 0x0200;
        const PTRFLAGS_WHEEL_NEGATIVE = 0x0100;
        const PTRFLAGS_HWHEEL = 0x0400;
        
        let flags = this.isHorizontal ? PTRFLAGS_HWHEEL : PTRFLAGS_WHEEL;
        if (this.isNegative) flags |= PTRFLAGS_WHEEL_NEGATIVE;
        flags |= (this.delta & 0xFF);
        
        const eventHeader = 0x01; // FASTPATH_INPUT_EVENT_MOUSE
        const data = new ArrayBuffer(7);
        const view = new DataView(data);
        view.setUint8(0, eventHeader);
        view.setUint16(1, flags, true);
        view.setUint16(3, Math.max(0, this.x), true);
        view.setUint16(5, Math.max(0, this.y), true);
        return data;
    }
}

// ============================================================================
// FastPath Update Codes
// ============================================================================
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

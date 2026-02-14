/**
 * RDP Protocol Parsing Functions
 * Recreated from original web/js/input/* and web/js/update/* files
 * @module protocol
 */

// ============================================================================
// FastPath Constants (from input/header.js)
// ============================================================================
const FASTPATH_INPUT_EVENT_SCANCODE = 0x0;
const FASTPATH_INPUT_EVENT_MOUSE = 0x1;
const FASTPATH_INPUT_EVENT_MOUSEX = 0x2;
const FASTPATH_INPUT_EVENT_SYNC = 0x3;
const FASTPATH_INPUT_EVENT_UNICODE = 0x4;

const FASTPATH_INPUT_KBDFLAGS_RELEASE = 0x01;

// ============================================================================
// Mouse Constants (from input/mouse.js)
// ============================================================================
const PTRFLAGS_HWHEEL = 0x0400;
const PTRFLAGS_WHEEL = 0x0200;
const PTRFLAGS_WHEEL_NEGATIVE = 0x0100;
const PTRFLAGS_MOVE = 0x0800;
const PTRFLAGS_DOWN = 0x8000;
const PTRFLAGS_BUTTON1 = 0x1000;
const PTRFLAGS_BUTTON2 = 0x2000;
const PTRFLAGS_BUTTON3 = 0x4000;
const WheelRotationMask = 0x01FF;

// ============================================================================
// Keyboard Scancode Mapping (from input/keymap.js)
// ============================================================================
const KeyMap = {
    "Escape": 0x01,
    "Digit1": 0x02,
    "Digit2": 0x03,
    "Digit3": 0x04,
    "Digit4": 0x05,
    "Digit5": 0x06,
    "Digit6": 0x07,
    "Digit7": 0x08,
    "Digit8": 0x09,
    "Digit9": 0x0A,
    "Digit0": 0x0B,
    "Minus": 0x0C,
    "Equal": 0x0D,
    "Backspace": 0x0E,
    "Tab": 0x0F,
    "KeyQ": 0x10,
    "KeyW": 0x11,
    "KeyE": 0x12,
    "KeyR": 0x13,
    "KeyT": 0x14,
    "KeyY": 0x15,
    "KeyU": 0x16,
    "KeyI": 0x17,
    "KeyO": 0x18,
    "KeyP": 0x19,
    "BracketLeft": 0x1A,
    "BracketRight": 0x1B,
    "Enter": 0x1C,
    "ControlLeft": 0x1D,
    "KeyA": 0x1E,
    "KeyS": 0x1F,
    "KeyD": 0x20,
    "KeyF": 0x21,
    "KeyG": 0x22,
    "KeyH": 0x23,
    "KeyJ": 0x24,
    "KeyK": 0x25,
    "KeyL": 0x26,
    "Semicolon": 0x27,
    "Quote": 0x28,
    "Backquote": 0x29,
    "ShiftLeft": 0x2A,
    "Backslash": 0x2B,
    "KeyZ": 0x2C,
    "KeyX": 0x2D,
    "KeyC": 0x2E,
    "KeyV": 0x2F,
    "KeyB": 0x30,
    "KeyN": 0x31,
    "KeyM": 0x32,
    "Comma": 0x33,
    "Period": 0x34,
    "Slash": 0x35,
    "ShiftRight": 0x36,
    "NumpadMultiply": 0x37,
    "AltLeft": 0x38,
    "Space": 0x39,
    "CapsLock": 0x3A,
    "F1": 0x3B,
    "F2": 0x3C,
    "F3": 0x3D,
    "F4": 0x3E,
    "F5": 0x3F,
    "F6": 0x40,
    "F7": 0x41,
    "F8": 0x42,
    "F9": 0x43,
    "F10": 0x44,
    "NumLock": 0x45,
    "Pause": 0x46,
    "PrintScreen": 0x46,
    "Numpad7": 0x47,
    "Numpad8": 0x48,
    "Numpad9": 0x49,
    "NumpadSubtract": 0x4A,
    "Numpad4": 0x4B,
    "Numpad5": 0x4C,
    "Numpad6": 0x4D,
    "NumpadAdd": 0x4E,
    "Numpad1": 0x4F,
    "Numpad2": 0x50,
    "Numpad3": 0x51,
    "Numpad0": 0x52,
    "NumpadDecimal": 0x53,
    "F11": 0x57,
    "F12": 0x58,
    "ArrowUp": 0x48,
    "ArrowLeft": 0x4B,
    "ArrowRight": 0x4D,
    "ArrowDown": 0x50,
};

// ============================================================================
// Keyboard Events (from input/keyboard.js)
// ============================================================================

/**
 * Keyboard key down event
 */
export class KeyboardEventKeyDown {
    constructor(code) {
        this.keyCode = KeyMap[code];
    }

    serialize() {
        const data = new ArrayBuffer(2);
        const view = new DataView(data);

        const eventFlags = 0;
        const eventCode = (FASTPATH_INPUT_EVENT_SCANCODE & 0x7) << 5;
        const eventHeader = eventFlags | eventCode;

        view.setUint8(0, eventHeader);
        view.setUint8(1, this.keyCode || 0);

        return data;
    }
}

/**
 * Keyboard key up event
 */
export class KeyboardEventKeyUp {
    constructor(code) {
        this.keyCode = KeyMap[code];
    }

    serialize() {
        const data = new ArrayBuffer(2);
        const view = new DataView(data);

        const eventFlags = FASTPATH_INPUT_KBDFLAGS_RELEASE & 0x1f;
        const eventCode = (FASTPATH_INPUT_EVENT_SCANCODE & 0x7) << 5;
        const eventHeader = eventFlags | eventCode;

        view.setUint8(0, eventHeader);
        view.setUint8(1, this.keyCode || 0);

        return data;
    }
}

// ============================================================================
// Mouse Events (from input/mouse.js)
// ============================================================================

/**
 * Mouse move event
 */
export class MouseMoveEvent {
    constructor(xPos, yPos) {
        this.pointerFlags = PTRFLAGS_MOVE;
        this.xPos = xPos;
        this.yPos = yPos;
    }

    serialize() {
        const data = new ArrayBuffer(7);
        const view = new DataView(data);

        const eventHeader = FASTPATH_INPUT_EVENT_MOUSE << 5;

        view.setUint8(0, eventHeader);
        view.setUint16(1, this.pointerFlags, true);
        view.setUint16(3, this.xPos, true);
        view.setUint16(5, this.yPos, true);

        return data;
    }
}

/**
 * Mouse button down event
 */
export class MouseDownEvent {
    constructor(xPos, yPos, button) {
        this.pointerFlags = PTRFLAGS_DOWN;
        this.xPos = xPos;
        this.yPos = yPos;

        switch (button) {
            case 1:
                this.pointerFlags |= PTRFLAGS_BUTTON1;
                break;
            case 2:
                this.pointerFlags |= PTRFLAGS_BUTTON2;
                break;
            case 3:
                this.pointerFlags |= PTRFLAGS_BUTTON3;
                break;
        }
    }

    serialize() {
        const data = new ArrayBuffer(7);
        const view = new DataView(data);

        const eventHeader = FASTPATH_INPUT_EVENT_MOUSE << 5;

        view.setUint8(0, eventHeader);
        view.setUint16(1, this.pointerFlags, true);
        view.setUint16(3, this.xPos, true);
        view.setUint16(5, this.yPos, true);

        return data;
    }
}

/**
 * Mouse button up event
 */
export class MouseUpEvent {
    constructor(xPos, yPos, button) {
        this.pointerFlags = PTRFLAGS_MOVE;
        this.xPos = xPos;
        this.yPos = yPos;

        switch (button) {
            case 1:
                this.pointerFlags = PTRFLAGS_BUTTON1;
                break;
            case 2:
                this.pointerFlags = PTRFLAGS_BUTTON2;
                break;
            case 3:
                this.pointerFlags = PTRFLAGS_BUTTON3;
                break;
        }
    }

    serialize() {
        const data = new ArrayBuffer(7);
        const view = new DataView(data);

        const eventHeader = FASTPATH_INPUT_EVENT_MOUSE << 5;

        view.setUint8(0, eventHeader);
        view.setUint16(1, this.pointerFlags, true);
        view.setUint16(3, this.xPos, true);
        view.setUint16(5, this.yPos, true);

        return data;
    }
}

/**
 * Mouse wheel event
 */
export class MouseWheelEvent {
    constructor(xPos, yPos, step, isNegative, isHorizontal) {
        this.xPos = xPos;
        this.yPos = yPos;

        this.pointerFlags = PTRFLAGS_WHEEL;
        if (isHorizontal) {
            this.pointerFlags = PTRFLAGS_HWHEEL;
        }

        if (isNegative) {
            this.pointerFlags |= PTRFLAGS_WHEEL_NEGATIVE;
        }

        this.pointerFlags |= (step & WheelRotationMask);
    }

    serialize() {
        const data = new ArrayBuffer(7);
        const view = new DataView(data);

        const eventHeader = FASTPATH_INPUT_EVENT_MOUSE << 5;

        view.setUint8(0, eventHeader);
        view.setUint16(1, this.pointerFlags, true);
        view.setUint16(3, this.xPos, true);
        view.setUint16(5, this.yPos, true);

        return data;
    }
}

// ============================================================================
// FastPath Update Constants (from update/header.js)
// ============================================================================
const FASTPATH_UPDATETYPE_ORDERS = 0x0;
const FASTPATH_UPDATETYPE_BITMAP = 0x1;
const FASTPATH_UPDATETYPE_PALETTE = 0x2;
const FASTPATH_UPDATETYPE_SYNCHRONIZE = 0x3;
const FASTPATH_UPDATETYPE_SURFCMDS = 0x4;
const FASTPATH_UPDATETYPE_PTR_NULL = 0x5;
const FASTPATH_UPDATETYPE_PTR_DEFAULT = 0x6;
const FASTPATH_UPDATETYPE_PTR_POSITION = 0x8;
const FASTPATH_UPDATETYPE_COLOR = 0x9;
const FASTPATH_UPDATETYPE_CACHED = 0xa;
const FASTPATH_UPDATETYPE_POINTER = 0xb;
const FASTPATH_UPDATETYPE_LARGE_POINTER = 0xc;

const FASTPATH_FRAGMENT_SINGLE = 0x0;
const FASTPATH_FRAGMENT_LAST = 0x1;
const FASTPATH_FRAGMENT_FIRST = 0x2;
const FASTPATH_FRAGMENT_NEXT = 0x3;

const FASTPATH_OUTPUT_COMPRESSION_USED = 0x2;

// ============================================================================
// Update Header (from update/header.js)
// ============================================================================

/**
 * FastPath update header
 */
export class UpdateHeader {
    constructor() {
        this.updateCode = 0;
        this.fragmentation = 0;
        this.compression = 0;
        this.compressionFlags = 0;
        this.size = 0;
    }

    isOrders() {
        return this.updateCode === FASTPATH_UPDATETYPE_ORDERS;
    }

    isBitmap() {
        return this.updateCode === FASTPATH_UPDATETYPE_BITMAP;
    }

    isPalette() {
        return this.updateCode === FASTPATH_UPDATETYPE_PALETTE;
    }

    isSynchronize() {
        return this.updateCode === FASTPATH_UPDATETYPE_SYNCHRONIZE;
    }

    isSurfCMDs() {
        return this.updateCode === FASTPATH_UPDATETYPE_SURFCMDS;
    }

    isPTRNull() {
        return this.updateCode === FASTPATH_UPDATETYPE_PTR_NULL;
    }

    isPTRDefault() {
        return this.updateCode === FASTPATH_UPDATETYPE_PTR_DEFAULT;
    }

    isPTRPosition() {
        return this.updateCode === FASTPATH_UPDATETYPE_PTR_POSITION;
    }

    isPTRColor() {
        return this.updateCode === FASTPATH_UPDATETYPE_COLOR;
    }

    isPTRCached() {
        return this.updateCode === FASTPATH_UPDATETYPE_CACHED;
    }

    isPTRNew() {
        return this.updateCode === FASTPATH_UPDATETYPE_POINTER;
    }

    isPointer() {
        return this.updateCode >= FASTPATH_UPDATETYPE_PTR_NULL;
    }

    isLargePointer() {
        return this.updateCode === FASTPATH_UPDATETYPE_LARGE_POINTER;
    }

    isSingleFragment() {
        return this.fragmentation === FASTPATH_FRAGMENT_SINGLE;
    }

    isLastFragment() {
        return this.fragmentation === FASTPATH_FRAGMENT_LAST;
    }

    isFirstFragment() {
        return this.fragmentation === FASTPATH_FRAGMENT_FIRST;
    }

    isNextFragment() {
        return this.fragmentation === FASTPATH_FRAGMENT_NEXT;
    }

    isCompressed() {
        return this.compression === FASTPATH_OUTPUT_COMPRESSION_USED;
    }
}

/**
 * Parse FastPath update header
 * @param {BinaryReader} r
 * @returns {UpdateHeader}
 */
export function parseUpdateHeader(r) {
    const header = new UpdateHeader();
    const updateHeader = r.uint8();

    header.updateCode = updateHeader & 0xf;
    header.fragmentation = (updateHeader & 0x30) >> 4;
    header.compression = (updateHeader & 0xc0) >> 6;

    if (header.isCompressed()) {
        header.compressionFlags = r.uint16(true);
    }

    header.size = r.uint16(true);

    return header;
}

// ============================================================================
// Bitmap Update (from update/bitmap.js)
// ============================================================================

const BITMAP_COMPRESSION = 0x0001;
const NO_BITMAP_COMPRESSION_HDR = 0x0400;

/**
 * Bitmap data for a single rectangle
 */
export class BitmapData {
    constructor() {
        this.destLeft = 0;
        this.destTop = 0;
        this.destRight = 0;
        this.destBottom = 0;
        this.width = 0;
        this.height = 0;
        this.bitsPerPixel = 0;
        this.flags = 0;
        this.bitmapLength = 0;
        this.bitmapComprHdr = null;
        this.bitmapDataStream = null;
    }

    isCompressed() {
        return (this.flags & BITMAP_COMPRESSION) === BITMAP_COMPRESSION;
    }

    hasNoBitmapCompressionHDR() {
        return (this.flags & NO_BITMAP_COMPRESSION_HDR) === NO_BITMAP_COMPRESSION_HDR;
    }
}

/**
 * Compressed data header
 */
class CompressedDataHeader {
    constructor() {
        this.cbCompFirstRowSize = 0;
        this.cbCompMainBodySize = 0;
        this.cbScanWidth = 0;
        this.cbUncompressedSize = 0;
    }
}

function parseCompressedDataHeader(r) {
    const header = new CompressedDataHeader();

    header.cbCompFirstRowSize = r.uint16(true);
    header.cbCompMainBodySize = r.uint16(true);
    header.cbScanWidth = r.uint16(true);
    header.cbUncompressedSize = r.uint16(true);

    return header;
}

function parseBitmapData(r) {
    const bitmapData = new BitmapData();

    bitmapData.destLeft = r.uint16(true);
    bitmapData.destTop = r.uint16(true);
    bitmapData.destRight = r.uint16(true);
    bitmapData.destBottom = r.uint16(true);
    bitmapData.width = r.uint16(true);
    bitmapData.height = r.uint16(true);
    bitmapData.bitsPerPixel = r.uint16(true);
    bitmapData.flags = r.uint16(true);
    bitmapData.bitmapLength = r.uint16(true);

    // Calculate actual data length
    // bitmapLength includes compression header (8 bytes) when present
    let dataLength = bitmapData.bitmapLength;

    if (bitmapData.isCompressed() && !bitmapData.hasNoBitmapCompressionHDR()) {
        bitmapData.bitmapComprHdr = parseCompressedDataHeader(r);
        dataLength -= 8; // Compression header is 8 bytes
    }

    bitmapData.bitmapDataStream = r.blob(dataLength);

    return bitmapData;
}

/**
 * Bitmap update containing rectangles
 */
export class BitmapUpdate {
    constructor() {
        this.numberRectangles = 0;
        this.rectangles = [];
    }
}

/**
 * Parse bitmap update
 * @param {BinaryReader} r
 * @returns {BitmapUpdate}
 */
export function parseBitmapUpdate(r) {
    const update = new BitmapUpdate();

    // updateType
    r.uint16(true);

    update.numberRectangles = r.uint16(true);

    for (let i = 0; i < update.numberRectangles; i++) {
        const bitmapData = parseBitmapData(r);
        update.rectangles.push(bitmapData);
    }

    return update;
}

// ============================================================================
// Pointer Updates
// ============================================================================

/**
 * New pointer update
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

        const bytesPerPixel = this.xorBpp / 8;
        // Scanlines are padded to 2-byte boundaries per MS-RDPBCGR
        const xorRowBytesRaw = Math.ceil((this.width * this.xorBpp) / 8);
        const xorRowBytes = xorRowBytesRaw + (xorRowBytesRaw % 2);
        const andRowBytesRaw = Math.ceil(this.width / 8);
        const andRowBytes = andRowBytesRaw + (andRowBytesRaw % 2);

        // For 32-bit cursors, check if XOR mask contains per-pixel alpha
        let useXorAlpha = false;
        if (bytesPerPixel === 4) {
            for (let i = 3; i < this.xorMask.length; i += 4) {
                if (this.xorMask[i] !== 0) {
                    useXorAlpha = true;
                    break;
                }
            }
        }

        for (let y = 0; y < this.height; y++) {
            const srcY = this.height - 1 - y;

            for (let x = 0; x < this.width; x++) {
                const dstIdx = (y * this.width + x) * 4;

                const andByteIdx = srcY * andRowBytes + Math.floor(x / 8);
                const andBit = (this.andMask[andByteIdx] >> (7 - (x % 8))) & 1;

                const xorByteIdx = srcY * xorRowBytes + x * bytesPerPixel;

                if (bytesPerPixel === 4) {
                    data[dstIdx] = this.xorMask[xorByteIdx + 2];
                    data[dstIdx + 1] = this.xorMask[xorByteIdx + 1];
                    data[dstIdx + 2] = this.xorMask[xorByteIdx];
                    data[dstIdx + 3] = useXorAlpha
                        ? this.xorMask[xorByteIdx + 3]
                        : (andBit ? 0 : 255);
                } else if (bytesPerPixel === 3) {
                    data[dstIdx] = this.xorMask[xorByteIdx + 2];
                    data[dstIdx + 1] = this.xorMask[xorByteIdx + 1];
                    data[dstIdx + 2] = this.xorMask[xorByteIdx];
                    data[dstIdx + 3] = andBit ? 0 : 255;
                } else if (bytesPerPixel === 2) {
                    const pixel = this.xorMask[xorByteIdx] | (this.xorMask[xorByteIdx + 1] << 8);
                    data[dstIdx] = ((pixel >> 11) & 0x1f) << 3;
                    data[dstIdx + 1] = ((pixel >> 5) & 0x3f) << 2;
                    data[dstIdx + 2] = (pixel & 0x1f) << 3;
                    data[dstIdx + 3] = andBit ? 0 : 255;
                } else {
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
 * Parse new pointer update
 * @param {BinaryReader} r
 * @returns {NewPointerUpdate}
 */
export function parseNewPointerUpdate(r) {
    const xorBpp = r.uint16(true);
    const cacheIndex = r.uint16(true);
    const x = r.uint16(true);
    const y = r.uint16(true);
    const width = r.uint16(true);
    const height = r.uint16(true);
    const andMaskLen = r.uint16(true);
    const xorMaskLen = r.uint16(true);

    const xorMask = r.blob(xorMaskLen);
    const andMask = r.blob(andMaskLen);

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

// ============================================================================
// Surface Commands (MS-RDPBCGR 2.2.9.1.2.1.10)
// ============================================================================

const CMDTYPE_SET_SURFACE_BITS = 0x0001;
const CMDTYPE_FRAME_MARKER = 0x0004;
const CMDTYPE_STREAM_SURFACE_BITS = 0x0006;

const FRAME_START = 0x0000;
const FRAME_END = 0x0001;

/**
 * Parsed SetSurfaceBits / StreamSurfaceBits command
 */
export class SetSurfaceBitsCommand {
    constructor() {
        this.destLeft = 0;
        this.destTop = 0;
        this.destRight = 0;
        this.destBottom = 0;
        this.bpp = 0;
        this.codecID = 0;
        this.width = 0;
        this.height = 0;
        this.bitmapData = null;
    }
}

/**
 * Parse surface commands from a fastpath update data section.
 * @param {BinaryReader} r
 * @returns {Array} Array of { type, command } objects
 */
export function parseSurfaceCommands(r) {
    const commands = [];

    while (r.offset < r.buffer.byteLength) {
        const cmdType = r.uint16(true);

        if (cmdType === CMDTYPE_SET_SURFACE_BITS || cmdType === CMDTYPE_STREAM_SURFACE_BITS) {
            const cmd = new SetSurfaceBitsCommand();
            cmd.destLeft = r.uint16(true);
            cmd.destTop = r.uint16(true);
            cmd.destRight = r.uint16(true);
            cmd.destBottom = r.uint16(true);
            cmd.bpp = r.uint8();
            r.uint8(); // flags
            r.uint8(); // reserved
            cmd.codecID = r.uint8();
            cmd.width = r.uint16(true);
            cmd.height = r.uint16(true);
            const bitmapDataLength = r.uint32(true);
            cmd.bitmapData = new Uint8Array(r.blob(bitmapDataLength));
            commands.push({ type: 'surfaceBits', command: cmd });
        } else if (cmdType === CMDTYPE_FRAME_MARKER) {
            const frameAction = r.uint16(true);
            const frameID = r.uint32(true);
            commands.push({
                type: 'frameMarker',
                command: { action: frameAction, frameID, isStart: frameAction === FRAME_START }
            });
        } else {
            // Unknown command, consume remaining data
            break;
        }
    }

    return commands;
}

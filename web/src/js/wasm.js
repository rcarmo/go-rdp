/**
 * WASM Codec Module - Loader and wrapper for Go WASM codec functions
 * Includes RLE, NSCodec, color conversion, and RemoteFX support
 * @module wasm
 */

import { Logger } from './logger.js';

/**
 * Check if WebAssembly is supported in this environment
 * @returns {boolean}
 */
export function isWASMSupported() {
    try {
        if (typeof WebAssembly === 'object' &&
            typeof WebAssembly.instantiate === 'function') {
            // Test with minimal valid WASM module
            const module = new WebAssembly.Module(
                new Uint8Array([0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00])
            );
            return module instanceof WebAssembly.Module;
        }
    } catch (e) {
        // WebAssembly not available
    }
    return false;
}

/**
 * WASM Codec interface
 * Provides access to Go-implemented codec functions
 */
export const WASMCodec = {
    ready: false,
    goInstance: null,
    wasmInstance: null,
    supported: isWASMSupported(),
    initError: null,
    
    /**
     * Initialize WASM module
     * @param {string} wasmPath - Path to the WASM file
     * @returns {Promise<boolean>}
     */
    async init(wasmPath = 'js/rle/rle.wasm') {
        if (this.ready) {
            return true;
        }
        
        // Check WebAssembly support first
        if (!this.supported) {
            this.initError = 'WebAssembly not supported in this browser';
            Logger.error('WASM', this.initError);
            return false;
        }
        
        try {
            // Check if Go class is available (from wasm_exec.js)
            if (typeof Go === 'undefined') {
                this.initError = 'Go class not found. Include wasm_exec.js before initializing.';
                Logger.error('WASM', this.initError);
                return false;
            }
            
            this.goInstance = new Go();
            
            // Try streaming instantiation first, fall back to array buffer
            let result;
            if (typeof WebAssembly.instantiateStreaming === 'function') {
                try {
                    result = await WebAssembly.instantiateStreaming(
                        fetch(wasmPath),
                        this.goInstance.importObject
                    );
                } catch (e) {
                    Logger.warn('WASM', 'Streaming failed, using array buffer fallback');
                    const response = await fetch(wasmPath);
                    const bytes = await response.arrayBuffer();
                    result = await WebAssembly.instantiate(bytes, this.goInstance.importObject);
                }
            } else {
                const response = await fetch(wasmPath);
                const bytes = await response.arrayBuffer();
                result = await WebAssembly.instantiate(bytes, this.goInstance.importObject);
            }
            
            this.wasmInstance = result.instance;
            
            // Run the Go program (this sets up goRLE global)
            this.goInstance.run(this.wasmInstance);
            
            // Verify goRLE is available
            if (typeof goRLE === 'undefined') {
                this.initError = 'goRLE not initialized after running WASM';
                Logger.error('WASM', this.initError);
                return false;
            }
            
            this.ready = true;
            this.initError = null;
            Logger.debug('WASM', 'Codec module initialized (RLE + RFX)');
            return true;
            
        } catch (error) {
            this.initError = error.message;
            Logger.error('WASM', `Failed to initialize: ${error.message}`);
            return false;
        }
    },
    
    /**
     * Check if WASM is supported (before init)
     * @returns {boolean}
     */
    isSupported() {
        return this.supported;
    },
    
    /**
     * Get initialization error message
     * @returns {string|null}
     */
    getInitError() {
        return this.initError;
    },
    
    /**
     * Check if WASM is ready
     * @returns {boolean}
     */
    isReady() {
        return this.ready && typeof goRLE !== 'undefined';
    },
    
    // ========================================
    // RLE and Bitmap functions
    // ========================================
    
    /**
     * Decompress RLE16 data
     * @param {Uint8Array} src - Compressed data
     * @param {Uint8Array} dst - Output buffer
     * @param {number} width - Image width
     * @param {number} rowDelta - Row stride in bytes
     * @returns {boolean}
     */
    decompressRLE16(src, dst, width, rowDelta) {
        if (!this.isReady()) return false;
        return goRLE.decompressRLE16(src, dst, width, rowDelta);
    },
    
    /**
     * Flip image vertically
     * @param {Uint8Array} data - Image data (modified in place)
     * @param {number} width
     * @param {number} height
     * @param {number} bytesPerPixel
     */
    flipVertical(data, width, height, bytesPerPixel) {
        if (!this.isReady()) return;
        goRLE.flipVertical(data, width, height, bytesPerPixel);
    },
    
    /**
     * Convert RGB565 to RGBA
     * @param {Uint8Array} src
     * @param {Uint8Array} dst
     */
    rgb565toRGBA(src, dst) {
        if (!this.isReady()) return;
        goRLE.rgb565toRGBA(src, dst);
    },
    
    /**
     * Convert BGR24 to RGBA
     * @param {Uint8Array} src
     * @param {Uint8Array} dst
     */
    bgr24toRGBA(src, dst) {
        if (!this.isReady()) return;
        goRLE.bgr24toRGBA(src, dst);
    },
    
    /**
     * Convert BGRA32 to RGBA
     * @param {Uint8Array} src
     * @param {Uint8Array} dst
     */
    bgra32toRGBA(src, dst) {
        if (!this.isReady()) return;
        goRLE.bgra32toRGBA(src, dst);
    },
    
    /**
     * Process a complete bitmap (decompress + convert)
     * @param {Uint8Array} src - Source bitmap data
     * @param {number} width
     * @param {number} height
     * @param {number} bpp - Bits per pixel
     * @param {boolean} isCompressed
     * @param {Uint8Array} dst - Output RGBA buffer
     * @param {number} rowDelta
     * @returns {boolean}
     */
    processBitmap(src, width, height, bpp, isCompressed, dst, rowDelta) {
        if (!this.isReady()) return false;
        return goRLE.processBitmap(src, width, height, bpp, isCompressed, dst, rowDelta);
    },
    
    /**
     * Decode NSCodec data to RGBA
     * @param {Uint8Array} src
     * @param {number} width
     * @param {number} height
     * @param {Uint8Array} dst
     * @returns {boolean}
     */
    decodeNSCodec(src, width, height, dst) {
        if (!this.isReady()) return false;
        return goRLE.decodeNSCodec(src, width, height, dst);
    },
    
    /**
     * Set palette colors
     * @param {Uint8Array} data - Palette data (RGB triples)
     * @param {number} numColors
     * @returns {boolean}
     */
    setPalette(data, numColors) {
        if (!this.isReady()) return false;
        return goRLE.setPalette(data, numColors);
    },
    
    // ========================================
    // RemoteFX (RFX) functions
    // ========================================
    
    /**
     * Set RFX quantization values
     * @param {Uint8Array} quantData - 15 bytes (3 tables × 5 bytes)
     * @returns {boolean}
     */
    setRFXQuant(quantData) {
        if (!this.isReady()) return false;
        return goRLE.setRFXQuant(quantData);
    },
    
    /**
     * Decode a single RFX tile
     * @param {Uint8Array} tileData - Compressed tile data (CBT_TILE block)
     * @param {Uint8Array} outputBuffer - Output buffer (16384 bytes for 64x64 RGBA)
     * @returns {Object|null} { x, y, width, height } or null on error
     */
    decodeRFXTile(tileData, outputBuffer) {
        if (!this.isReady()) return null;
        
        const result = goRLE.decodeRFXTile(tileData, outputBuffer);
        
        // Result is [x, y, width, height] array or null
        if (result === null || result === undefined) {
            return null;
        }
        
        return {
            x: result[0],
            y: result[1],
            width: result[2],
            height: result[3]
        };
    },
    
    /**
     * RFX tile constants
     */
    RFX_TILE_SIZE: 64,
    RFX_TILE_PIXELS: 4096,
    RFX_TILE_RGBA_SIZE: 16384,
};

/**
 * RFX Decoder - High-level RemoteFX decoder
 */
export class RFXDecoder {
    constructor() {
        // Pre-allocated output buffer for single tile
        this.tileBuffer = new Uint8Array(WASMCodec.RFX_TILE_RGBA_SIZE);
        // Current quant values (15 bytes)
        this.quantBuffer = new Uint8Array(15);
        this.quantSet = false;
    }
    
    /**
     * Set quantization values for subsequent decodes
     * @param {Uint8Array} quantY - 5 bytes for Y component
     * @param {Uint8Array} quantCb - 5 bytes for Cb component  
     * @param {Uint8Array} quantCr - 5 bytes for Cr component
     */
    setQuant(quantY, quantCb, quantCr) {
        this.quantBuffer.set(quantY, 0);
        this.quantBuffer.set(quantCb, 5);
        this.quantBuffer.set(quantCr, 10);
        this.quantSet = WASMCodec.setRFXQuant(this.quantBuffer);
    }
    
    /**
     * Set quantization from raw buffer (15 bytes)
     * @param {Uint8Array} quantData
     */
    setQuantRaw(quantData) {
        this.quantBuffer.set(quantData.subarray(0, 15));
        this.quantSet = WASMCodec.setRFXQuant(this.quantBuffer);
    }
    
    /**
     * Decode a tile and return result
     * @param {Uint8Array} tileData - CBT_TILE block data
     * @returns {Object|null} { x, y, width, height, rgba }
     */
    decodeTile(tileData) {
        const result = WASMCodec.decodeRFXTile(tileData, this.tileBuffer);
        if (!result) {
            return null;
        }
        
        return {
            x: result.x,
            y: result.y,
            width: result.width,
            height: result.height,
            rgba: this.tileBuffer
        };
    }
    
    /**
     * Decode a tile and render directly to a Canvas 2D or WebGL renderer context.
     * @param {Uint8Array} tileData - CBT_TILE block data
     * @param {Object} ctx - CanvasRenderingContext2D or WebGLRenderer
     * @returns {boolean}
     */
    decodeTileToCanvas(tileData, ctx) {
        const result = WASMCodec.decodeRFXTile(tileData, this.tileBuffer);
        if (!result) {
            return false;
        }

        const rgba = new Uint8ClampedArray(this.tileBuffer);
        if (ctx && typeof ctx.drawRGBA === 'function') {
            return ctx.drawRGBA(result.x, result.y, result.width, result.height, rgba);
        }

        if (!ctx || typeof ctx.putImageData !== 'function') {
            return false;
        }

        // Copy buffer to avoid data corruption if called again before render
        const imageData = new ImageData(rgba, result.width, result.height);
        ctx.putImageData(imageData, result.x, result.y);
        return true;
    }
}

export default WASMCodec;

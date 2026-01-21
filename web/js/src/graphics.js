/**
 * Graphics handling for RDP client
 * Handles bitmap rendering, pointer/cursor updates, and canvas management
 * @module graphics
 */

import { Logger } from './logger.js';
import { WASMCodec, RFXDecoder } from './wasm.js';

/**
 * Graphics handling mixin - adds graphics functionality to Client
 */
export const GraphicsMixin = {
    /**
     * Initialize graphics subsystem
     */
    initGraphics() {
        this.canvasShown = false;
        this.pointerCache = {};
        
        // Bitmap cache
        this.bitmapCacheEnabled = true;
        this.bitmapCache = new Map();
        this.bitmapCacheMaxSize = 1000;
        this.bitmapCacheHits = 0;
        this.bitmapCacheMisses = 0;
        
        // Original desktop size
        this.originalWidth = 0;
        this.originalHeight = 0;
        this.resizeTimeout = null;
        
        // RFX decoder instance
        this.rfxDecoder = new RFXDecoder();
        
        // Bind resize handler
        this.handleResize = this.handleResize.bind(this);
    },
    
    /**
     * Handle palette update
     * @param {Reader} r - Binary reader
     */
    handlePalette(r) {
        const updateType = r.uint16(true);
        const pad = r.uint16(true);
        const numberColors = r.uint32(true);
        
        if (numberColors > 256 || numberColors === 0) {
            Logger.warn("Palette", `Invalid color count: ${numberColors}`);
            return;
        }
        
        Logger.info("Palette", `Received ${numberColors} colors`);
        
        const paletteData = r.blob(numberColors * 3);
        
        if (WASMCodec.isReady()) {
            WASMCodec.setPalette(new Uint8Array(paletteData), numberColors);
            Logger.debug("Palette", "Updated via WASM");
        } else {
            Logger.warn("Palette", "WASM not available");
        }
    },
    
    /**
     * Handle bitmap update
     * @param {Reader} r - Binary reader
     */
    handleBitmap(r) {
        const bitmap = parseBitmapUpdate(r);
        
        if (!this.canvasShown) {
            this.showCanvas();
            this.canvasShown = true;
        }
        
        if (this.multiMonitorMode && !this.multiMonitorMessageShown) {
            this.showUserSuccess('Multi-monitor environment detected');
            this.multiMonitorMessageShown = true;
        }

        bitmap.rectangles.forEach((bitmapData) => {
            this.processBitmapData(bitmapData);
        });
    },
    
    /**
     * Process a single bitmap rectangle
     * @param {Object} bitmapData
     */
    processBitmapData(bitmapData) {
        const width = bitmapData.width;
        const height = bitmapData.height;
        const bpp = bitmapData.bitsPerPixel;
        const size = width * height;
        const bytesPerPixel = bpp / 8;
        const rowDelta = width * bytesPerPixel;
        const isCompressed = bitmapData.isCompressed();
        
        let rgba = new Uint8ClampedArray(size * 4);
        
        if (WASMCodec.isReady()) {
            const srcData = new Uint8Array(bitmapData.bitmapDataStream);
            const result = WASMCodec.processBitmap(srcData, width, height, bpp, isCompressed, rgba, rowDelta);
            
            if (result) {
                this.ctx.putImageData(
                    new ImageData(rgba, width, height),
                    bitmapData.destLeft,
                    bitmapData.destTop
                );
                
                if (this.bitmapCacheEnabled) {
                    this.cacheBitmap(bitmapData, rgba);
                }
                return;
            }
            Logger.debug("Bitmap", `WASM processBitmap failed, bpp=${bpp}, compressed=${isCompressed}`);
        } else {
            // WASM not available - show error once and provide fallback for uncompressed data
            if (!this._wasmErrorShown) {
                this._wasmErrorShown = true;
                const errorMsg = WASMCodec.getInitError() || 'WASM not loaded';
                Logger.error("Bitmap", `WASM unavailable: ${errorMsg}`);
                this.showUserError('Graphics decoder not available. Some images may not display correctly.');
            }
            
            // Fallback for uncompressed 32-bit data only
            if (!isCompressed && bpp === 32) {
                const src = new Uint8Array(bitmapData.bitmapDataStream);
                // Convert BGRA to RGBA (simple swap)
                for (let i = 0; i < size; i++) {
                    const srcIdx = i * 4;
                    const dstIdx = i * 4;
                    rgba[dstIdx] = src[srcIdx + 2];     // R <- B
                    rgba[dstIdx + 1] = src[srcIdx + 1]; // G <- G
                    rgba[dstIdx + 2] = src[srcIdx];     // B <- R
                    rgba[dstIdx + 3] = src[srcIdx + 3]; // A <- A
                }
                this.ctx.putImageData(
                    new ImageData(rgba, width, height),
                    bitmapData.destLeft,
                    bitmapData.destTop
                );
                return;
            }
            
            Logger.warn("Bitmap", `Cannot decode: bpp=${bpp}, compressed=${isCompressed} (no WASM fallback)`);
        }
    },
    
    /**
     * Initialize bitmap cache
     */
    initBitmapCache() {
        this.bitmapCacheEnabled = true;
        this.bitmapCache = new Map();
        this.bitmapCacheMaxSize = 1000;
        this.bitmapCacheHits = 0;
        this.bitmapCacheMisses = 0;
    },
    
    /**
     * Cache a bitmap for future use
     * @param {Object} bitmapData
     * @param {Uint8ClampedArray} rgba
     */
    cacheBitmap(bitmapData, rgba) {
        const key = this.getBitmapCacheKey(bitmapData);
        
        if (this.bitmapCache.size >= this.bitmapCacheMaxSize) {
            const firstKey = this.bitmapCache.keys().next().value;
            this.bitmapCache.delete(firstKey);
        }
        
        this.bitmapCache.set(key, {
            imageData: new ImageData(new Uint8ClampedArray(rgba), bitmapData.width, bitmapData.height),
            width: bitmapData.width,
            height: bitmapData.height,
            timestamp: Date.now()
        });
    },
    
    /**
     * Get a cached bitmap
     * @param {Object} bitmapData
     * @returns {ImageData|null}
     */
    getCachedBitmap(bitmapData) {
        const key = this.getBitmapCacheKey(bitmapData);
        const cached = this.bitmapCache.get(key);
        
        if (cached) {
            this.bitmapCacheHits++;
            this.bitmapCache.delete(key);
            this.bitmapCache.set(key, cached);
            return cached.imageData;
        }
        
        this.bitmapCacheMisses++;
        return null;
    },
    
    /**
     * Generate cache key for bitmap
     * @param {Object} bitmapData
     * @returns {string}
     */
    getBitmapCacheKey(bitmapData) {
        const sample = bitmapData.bitmapDataStream.slice(0, Math.min(64, bitmapData.bitmapDataStream.length));
        let hash = 0;
        for (let i = 0; i < sample.length; i++) {
            hash = ((hash << 5) - hash) + sample[i];
            hash = hash & hash;
        }
        return `${bitmapData.width}x${bitmapData.height}x${bitmapData.bitsPerPixel}:${hash}`;
    },
    
    /**
     * Clear bitmap cache
     */
    clearBitmapCache() {
        if (this.bitmapCache) {
            this.bitmapCache.clear();
        }
        this.bitmapCacheHits = 0;
        this.bitmapCacheMisses = 0;
    },
    
    /**
     * Get bitmap cache statistics
     * @returns {Object}
     */
    getBitmapCacheStats() {
        return {
            size: this.bitmapCache ? this.bitmapCache.size : 0,
            hits: this.bitmapCacheHits || 0,
            misses: this.bitmapCacheMisses || 0,
            hitRate: this.bitmapCacheHits && this.bitmapCacheMisses 
                ? (this.bitmapCacheHits / (this.bitmapCacheHits + this.bitmapCacheMisses) * 100).toFixed(1) + '%'
                : 'N/A'
        };
    },
    
    /**
     * Handle pointer/cursor update
     * @param {Object} header
     * @param {Reader} r
     */
    handlePointer(header, r) {
        try {
            Logger.debug("Cursor", `Update type: ${header.updateCode}`);
            
            if (header.isPTRNull()) {
                Logger.debug("Cursor", "Hidden");
                this.canvas.className = 'pointer-cache-null';
                return;
            }

            if (header.isPTRDefault()) {
                Logger.debug("Cursor", "Default");
                this.canvas.className = 'pointer-cache-default';
                return;
            }

            if (header.isPTRColor()) {
                return;
            }

            if (header.isPTRNew()) {
                const newPointerUpdate = parseNewPointerUpdate(r);
                Logger.debug("Cursor", `New cursor: cache=${newPointerUpdate.cacheIndex}, hotspot=(${newPointerUpdate.x},${newPointerUpdate.y}), size=${newPointerUpdate.width}x${newPointerUpdate.height}`);
                this.pointerCacheCanvasCtx.putImageData(newPointerUpdate.getImageData(this.pointerCacheCanvasCtx), 0, 0);

                const url = this.pointerCacheCanvas.toDataURL('image/png');

                if (this.pointerCache.hasOwnProperty(newPointerUpdate.cacheIndex)) {
                    document.getElementsByTagName('head')[0].removeChild(this.pointerCache[newPointerUpdate.cacheIndex]);
                    delete this.pointerCache[newPointerUpdate.cacheIndex];
                }

                const style = document.createElement('style');
                const className = 'pointer-cache-' + newPointerUpdate.cacheIndex;
                style.innerHTML = '.' + className + ' {cursor:url("' + url + '") ' + newPointerUpdate.x + ' ' + newPointerUpdate.y + ', auto !important}';

                document.getElementsByTagName('head')[0].appendChild(style);
                this.pointerCache[newPointerUpdate.cacheIndex] = style;
                this.canvas.className = className;
                return;
            }

            if (header.isPTRCached()) {
                const cacheIndex = r.uint16(true);
                Logger.debug("Cursor", `Cached index: ${cacheIndex}`);
                const className = 'pointer-cache-' + cacheIndex;
                this.canvas.className = className;
                return;
            }

            if (header.isPTRPosition()) {
                Logger.debug("Cursor", "Position update (ignored)");
                return;
            }
            
            Logger.debug("Cursor", "Unknown pointer type");
        } catch (error) {
            Logger.error("Cursor", `Error: ${error.message}`);
        }
    },
    
    /**
     * Handle window resize
     */
    handleResize() {
        if (!this.connected) {
            return;
        }
        
        if (this.resizeTimeout) {
            clearTimeout(this.resizeTimeout);
        }
        
        this.resizeTimeout = setTimeout(() => {
            const newWidth = window.innerWidth;
            const newHeight = window.innerHeight;
            
            if (newWidth !== this.originalWidth || newHeight !== this.originalHeight) {
                Logger.info("Resize", `${newWidth}x${newHeight}, reconnecting...`);
                this.showUserInfo('Resizing desktop...');
                this.reconnectWithNewSize(newWidth, newHeight);
            }
        }, 500);
    },
    
    /**
     * Show the canvas (hide login form)
     */
    showCanvas() {
        Logger.info("Connection", "First bitmap received - session active");
        
        const loginForm = document.getElementById('login-form');
        const canvasContainer = document.getElementById('canvas-container');
        
        if (loginForm) {
            loginForm.style.display = 'none';
        }
        if (canvasContainer) {
            canvasContainer.style.display = 'block';
        }
        
        this.canvas.tabIndex = 1000;
        this.canvas.focus();
    },
    
    /**
     * Hide the canvas (show login form)
     */
    hideCanvas() {
        const loginForm = document.getElementById('login-form');
        const canvasContainer = document.getElementById('canvas-container');
        
        if (canvasContainer) {
            canvasContainer.style.display = 'none';
        }
        if (loginForm) {
            loginForm.style.display = 'block';
        }
    },
    
    /**
     * Clear the canvas
     */
    clearCanvas() {
        this.ctx.clearRect(0, 0, this.canvas.width, this.canvas.height);
        this.canvas.className = '';
    },
    
    // ========================================
    // RemoteFX (RFX) Support
    // ========================================
    
    /**
     * Handle RemoteFX surface command
     * @param {Uint8Array} data - Surface command data
     */
    handleRFXSurface(data) {
        if (!WASMCodec.isReady()) {
            Logger.error("RFX", "WASM not loaded");
            return;
        }
        
        // TODO: Parse surface command header and extract tiles
        // For now, this is a placeholder for future integration
        Logger.debug("RFX", `Surface command received, ${data.length} bytes`);
    },
    
    /**
     * Set RFX quantization values
     * @param {Uint8Array} quantY - 5 bytes
     * @param {Uint8Array} quantCb - 5 bytes
     * @param {Uint8Array} quantCr - 5 bytes
     */
    setRFXQuant(quantY, quantCb, quantCr) {
        this.rfxDecoder.setQuant(quantY, quantCb, quantCr);
    },
    
    /**
     * Decode and render an RFX tile
     * @param {Uint8Array} tileData - CBT_TILE block data
     * @returns {boolean}
     */
    decodeRFXTile(tileData) {
        return this.rfxDecoder.decodeTileToCanvas(tileData, this.ctx);
    },
    
    /**
     * Process multiple RFX tiles
     * @param {Array<Uint8Array>} tiles - Array of tile data
     */
    processRFXTiles(tiles) {
        let decoded = 0;
        let failed = 0;
        
        for (const tileData of tiles) {
            if (this.rfxDecoder.decodeTileToCanvas(tileData, this.ctx)) {
                decoded++;
            } else {
                failed++;
            }
        }
        
        if (failed > 0) {
            Logger.warn("RFX", `Decoded ${decoded} tiles, ${failed} failed`);
        } else {
            Logger.debug("RFX", `Decoded ${decoded} tiles`);
        }
    }
};

export default GraphicsMixin;

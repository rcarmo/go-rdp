/**
 * Graphics handling for RDP client
 * Handles bitmap rendering, pointer/cursor updates, and canvas management
 * @module graphics
 */

import { Logger } from './logger.js';
import { WASMCodec, RFXDecoder } from './wasm.js';
import { FallbackCodec } from './codec-fallback.js';
import { parseNewPointerUpdate, parseCachedPointerUpdate, parsePointerPositionUpdate, parseBitmapUpdate } from './protocol.js';

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
        
        // WASM status tracking
        this._wasmErrorShown = false;
        this._usingFallback = false;
        this._fallbackWarningShown = false;
        this._capabilitiesLogged = false;
        
        // Performance stats
        this._frameCount = 0;
        this._lastFpsTime = 0;
        this._fps = 0;
        this._bytesReceived = 0;
        this._statsInterval = null;
        this._activeDecoder = null;
        this._imageEncodingLogged = false;
        
        // Bind resize handler
        this.handleResize = this.handleResize.bind(this);
        
        // Check WASM availability (log only at debug level)
        if (!WASMCodec.isSupported()) {
            Logger.debug('Graphics', 'WebAssembly not available - using JS fallback');
        }
    },
    
    /**
     * Start performance stats tracking (only in debug mode)
     */
    startPerfStats() {
        // Only enable perf stats in debug mode
        if (!this.serverCapabilities?.logLevel || this.serverCapabilities.logLevel !== 'debug') {
            return;
        }
        
        this._frameCount = 0;
        this._lastFpsTime = performance.now();
        this._bytesReceived = 0;
        
        // Log stats every 10 seconds
        this._statsInterval = setInterval(() => {
            this.logPerfStats();
        }, 10000);
    },
    
    /**
     * Stop performance stats tracking
     */
    stopPerfStats() {
        if (this._statsInterval) {
            clearInterval(this._statsInterval);
            this._statsInterval = null;
        }
    },
    
    /**
     * Log performance statistics
     */
    logPerfStats() {
        const now = performance.now();
        const elapsed = (now - this._lastFpsTime) / 1000;
        
        if (elapsed > 0) {
            this._fps = this._frameCount / elapsed;
            const kbps = (this._bytesReceived * 8 / 1000) / elapsed;
            const cacheStats = this.getBitmapCacheStats();
            
            console.info(
                '%c[RDP Stats]',
                'color: #9C27B0; font-weight: bold',
                `FPS: ${this._fps.toFixed(1)}`,
                `| Bandwidth: ${kbps.toFixed(0)} kbps`,
                `| Cache: ${cacheStats.hitRate} hit rate (${cacheStats.size} items)`
            );
        }
        
        // Reset counters
        this._frameCount = 0;
        this._bytesReceived = 0;
        this._lastFpsTime = now;
    },
    
    /**
     * Track and log the active bitmap decoder (WASM vs JS fallback)
     * @param {string} decoderName
     */
    setActiveDecoder(decoderName) {
        if (this._activeDecoder === decoderName) {
            return;
        }
        this._activeDecoder = decoderName;
        console.info(
            '%c[RDP Client] Active decoder',
            'color: #FF9800; font-weight: bold',
            decoderName
        );
    },
    
    /**
     * Record a frame for FPS calculation
     * @param {number} bytes - Bytes received for this frame
     */
    recordFrame(bytes) {
        this._frameCount++;
        this._bytesReceived += bytes || 0;
    },
    
    /**
     * Log client capabilities to the console
     * Called once upon connection
     */
    logCapabilities() {
        if (this._capabilitiesLogged) return;
        this._capabilitiesLogged = true;
        
        const wasmSupported = WASMCodec.isSupported();
        const wasmReady = WASMCodec.isReady();
        const wasmError = WASMCodec.getInitError();
        
        const caps = {
            wasm: {
                supported: wasmSupported,
                loaded: wasmReady,
                error: wasmError
            },
            codecs: {
                rfx: wasmReady,
                rle: wasmReady,
                nscodec: wasmReady,
                fallback: !wasmReady
            },
            display: {
                width: this.originalWidth || window.innerWidth,
                height: this.originalHeight || window.innerHeight,
                colorDepth: this.getRecommendedColorDepth(),
                devicePixelRatio: window.devicePixelRatio || 1
            },
            browser: {
                userAgent: navigator.userAgent,
                platform: navigator.platform
            }
        };
        
        // Log summary to console (always visible)
        const codecList = [];
        if (caps.codecs.rfx) codecList.push('RemoteFX');
        if (caps.codecs.rle) codecList.push('RLE');
        if (caps.codecs.nscodec) codecList.push('NSCodec');
        if (caps.codecs.fallback) codecList.push('JS-Fallback');
        
        console.info(
            '%c[RDP Client] Decoder Capabilities',
            'color: #4CAF50; font-weight: bold',
            '\n  WASM:', wasmReady ? '✓ loaded' : (wasmSupported ? '✗ failed' : '✗ unsupported'),
            '\n  Can decode:', codecList.join(', '),
            '\n  Display:', `${caps.display.width}×${caps.display.height}`,
            '\n  Color:', `${caps.display.colorDepth}bpp`,
            wasmError ? `\n  Error: ${wasmError}` : ''
        );
        
        // Emit event for programmatic access
        if (this.emitEvent) {
            this.emitEvent('capabilities', caps);
        }
        
        return caps;
    },
    
    /**
     * Get recommended color depth based on codec availability
     * @returns {number} Recommended bits per pixel (16 or 32)
     */
    getRecommendedColorDepth() {
        // If WASM is available and working, 32-bit is fine
        if (WASMCodec.isReady()) {
            return 32;
        }
        // Without WASM, recommend 16-bit for best JS performance
        return FallbackCodec.getRecommendedColorDepth();
    },
    
    /**
     * Check if using fallback codecs
     * @returns {boolean}
     */
    isUsingFallback() {
        return this._usingFallback || !WASMCodec.isReady();
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
        
        Logger.debug("Palette", `Received ${numberColors} colors`);
        
        const paletteData = r.blob(numberColors * 3);
        const paletteArray = new Uint8Array(paletteData);
        
        // Set palette in WASM codec
        if (WASMCodec.isReady()) {
            WASMCodec.setPalette(paletteArray, numberColors);
            Logger.debug("Palette", "Updated via WASM");
        }
        
        // Also set in fallback codec (always, in case WASM fails later)
        FallbackCodec.setPalette(paletteArray, numberColors);
        Logger.debug("Palette", "Updated in fallback codec");
    },
    
    /**
     * Handle bitmap update
     * @param {Reader} r - Binary reader
     */
    handleBitmap(r) {
        const startOffset = r.offset;
        const bitmap = parseBitmapUpdate(r);
        const bytesRead = r.offset - startOffset;
        
        // Record frame for stats
        this.recordFrame(bytesRead);
        
        if (!this.canvasShown) {
            this.showCanvas();
            this.canvasShown = true;
            this.startPerfStats();
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
        const encoding = isCompressed ? 'RLE/Planar' : 'Raw Bitmap';
        
        let rgba = new Uint8ClampedArray(size * 4);
        const srcData = new Uint8Array(bitmapData.bitmapDataStream);
        
        // Try WASM first (fast path)
        if (WASMCodec.isReady()) {
            const result = WASMCodec.processBitmap(srcData, width, height, bpp, isCompressed, rgba, rowDelta);
            
            if (result) {
                if (!this._imageEncodingLogged) {
                    this._imageEncodingLogged = true;
                    console.info(
                        '%c[RDP Session] Image encoding',
                        'color: #03A9F4; font-weight: bold',
                        `${encoding} (${bpp}bpp)`
                    );
                }
                this.setActiveDecoder('WASM');
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
            Logger.debug("Bitmap", `WASM processBitmap failed, trying fallback`);
        }
        
        // WASM not available or failed - use JavaScript fallback
        if (!this._usingFallback) {
            this._usingFallback = true;
            const reason = WASMCodec.isReady() ? 'WASM decode failed' : (WASMCodec.getInitError() || 'WASM not loaded');
            Logger.warn("Bitmap", `Using JavaScript fallback codec (${reason})`);
        }
        
        const fallbackResult = FallbackCodec.processBitmap(srcData, width, height, bpp, isCompressed, rgba);
        
        if (fallbackResult) {
            if (!this._imageEncodingLogged) {
                this._imageEncodingLogged = true;
                console.info(
                    '%c[RDP Session] Image encoding',
                    'color: #03A9F4; font-weight: bold',
                    `${encoding} (${bpp}bpp)`
                );
            }
            this.setActiveDecoder('JS-Fallback');
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
        
        // Both WASM and fallback failed
        if (!this._wasmErrorShown) {
            this._wasmErrorShown = true;
            Logger.error("Bitmap", `Cannot decode: bpp=${bpp}, compressed=${isCompressed}`);
            if (isCompressed && bpp !== 8) {
                this.showUserError('Some compressed graphics cannot be displayed. Try reducing color depth.');
            }
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
        this.stopPerfStats();
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
                Logger.debug("Resize", `${newWidth}x${newHeight}, reconnecting...`);
                this.showUserInfo('Resizing desktop...');
                this.reconnectWithNewSize(newWidth, newHeight);
            }
        }, 500);
    },
    
    /**
     * Show the canvas (hide login form)
     */
    showCanvas() {
        Logger.debug("Connection", "First bitmap received - session active");
        
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

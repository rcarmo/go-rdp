/**
 * Graphics handling for RDP client
 * Handles bitmap rendering, pointer/cursor updates, and canvas management
 * @module graphics
 */

import { Logger } from './logger.js';
import { WASMCodec, RFXDecoder } from './wasm.js';
import { FallbackCodec } from './codec-fallback.js';
import { parseNewPointerUpdate, parseCachedPointerUpdate, parsePointerPositionUpdate, parseBitmapUpdate, parseSurfaceCommands } from './protocol.js';
import { CanvasRenderer } from './renderer.js';
import { WebGLRenderer } from './webgl-renderer.js';

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
        this._activeRenderer = null;
        
        // Bind resize handler
        this.handleResize = this.handleResize.bind(this);
        
        // Check WASM availability (log only at debug level)
        if (!WASMCodec.isSupported()) {
            Logger.debug('Graphics', 'WebAssembly not available - using JS fallback');
        }

        this.initRenderer();
    },

    /**
     * Initialize renderer (WebGL preferred, Canvas fallback)
     */
    initRenderer() {
        const webglRenderer = new WebGLRenderer(this.canvas);
        if (webglRenderer.init()) {
            this.renderer = webglRenderer;
            this._activeRenderer = `WebGL${webglRenderer.webglVersion}`;
            // Set up context loss fallback
            webglRenderer.onContextLostCallback = () => {
                Logger.warn('Graphics', 'WebGL context lost, switching to Canvas fallback');
                this._switchToCanvasRenderer();
            };
            console.info(
                '%c[RDP Client] Active renderer',
                'color: #FF9800; font-weight: bold',
                this._activeRenderer
            );
            return;
        }
        this._initCanvasRenderer();
    },

    /**
     * Initialize canvas renderer as fallback
     */
    _initCanvasRenderer() {
        const canvasRenderer = new CanvasRenderer(this.canvas);
        canvasRenderer.init();
        this.renderer = canvasRenderer;
        this._activeRenderer = 'Canvas2D';
        console.info(
            '%c[RDP Client] Active renderer',
            'color: #FF9800; font-weight: bold',
            this._activeRenderer
        );
    },

    /**
     * Switch to canvas renderer (e.g., after WebGL context loss)
     */
    _switchToCanvasRenderer() {
        if (this.renderer) {
            this.renderer.destroy();
        }
        this._initCanvasRenderer();
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

        // Enable WebGL renderer metrics in debug mode
        if (this.renderer && typeof this.renderer.setMetricsEnabled === 'function') {
            this.renderer.setMetricsEnabled(true);
        }
        
        // Log stats every 10 seconds (clear any existing interval first)
        if (this._statsInterval) {
            clearInterval(this._statsInterval);
        }
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
            if (this.renderer && typeof this.renderer.resize === 'function') {
                this.renderer.resize(this.canvas.width, this.canvas.height);
            }
        }
        
        if (this.multiMonitorMode && !this.multiMonitorMessageShown) {
            this.showUserSuccess('Multi-monitor environment detected');
            this.multiMonitorMessageShown = true;
        }

        // Use batching for multiple rectangles (reduces draw calls)
        const useBatching = bitmap.rectangles.length > 1 && 
                           this.renderer && 
                           typeof this.renderer.beginBatch === 'function';

        if (useBatching) {
            this.renderer.beginBatch();
        }

        bitmap.rectangles.forEach((bitmapData) => {
            this.processBitmapData(bitmapData);
        });

        if (useBatching) {
            this.renderer.flush();
        }
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
        // NO_BITMAP_COMPRESSION_HDR flag means RDP6 compression (Planar codec for 32bpp)
        // Without it, 32bpp compressed uses old interleaved 24-bit RLE
        const noHdr = bitmapData.hasNoBitmapCompressionHDR();
        const encoding = isCompressed ? (noHdr ? 'Planar/RDP6' : 'RLE') : 'Raw Bitmap';
        
        let rgba = new Uint8ClampedArray(size * 4);
        const srcData = new Uint8Array(bitmapData.bitmapDataStream);
        
        // Try WASM first (fast path)
        if (WASMCodec.isReady()) {
            const result = WASMCodec.processBitmap(srcData, width, height, bpp, isCompressed, rgba, rowDelta, noHdr);
            
            if (result) {
                if (!this._imageEncodingLogged) {
                    this._imageEncodingLogged = true;
                    this._activeEncoding = `${encoding} (${bpp}bpp)`;
                    console.info(
                        '%c[RDP Session] Active codec',
                        'color: #03A9F4; font-weight: bold',
                        `${this._activeEncoding} via WASM`
                    );
                }
                this.setActiveDecoder('WASM');
                this.renderer.drawRGBA(
                    bitmapData.destLeft,
                    bitmapData.destTop,
                    width,
                    height,
                    rgba
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
                this._activeEncoding = `${encoding} (${bpp}bpp)`;
                console.info(
                    '%c[RDP Session] Active codec',
                    'color: #03A9F4; font-weight: bold',
                    `${this._activeEncoding} via JS-Fallback`
                );
            }
            this.setActiveDecoder('JS-Fallback');
            this.renderer.drawRGBA(
                bitmapData.destLeft,
                bitmapData.destTop,
                width,
                height,
                rgba
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
                // Resize canvas to match cursor dimensions (also clears it)
                this.pointerCacheCanvas.width = newPointerUpdate.width;
                this.pointerCacheCanvas.height = newPointerUpdate.height;
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
                Logger.debug("Resize", `${newWidth}x${newHeight}`);
                
                // Try dynamic resize first, fall back to reconnect
                if (this.serverCapabilities?.displayControlReady) {
                    this.sendResizeRequest(newWidth, newHeight);
                } else {
                    this.showUserInfo('Resizing desktop...');
                    this.reconnectWithNewSize(newWidth, newHeight);
                }
            }
        }, 500);
    },
    
    /**
     * Send dynamic resize request to server
     * @param {number} width
     * @param {number} height
     */
    sendResizeRequest(width, height) {
        if (!this.socket || this.socket.readyState !== WebSocket.OPEN) {
            Logger.warn("Resize", "Cannot resize: not connected");
            return false;
        }
        
        const msg = JSON.stringify({
            type: 'resize',
            width: width,
            height: height
        });
        
        try {
            this.socket.send(msg);
            Logger.debug("Resize", `Sent resize request: ${width}x${height}`);
            
            // Update canvas size locally
            this.canvas.width = width;
            this.canvas.height = height;
            this.originalWidth = width;
            this.originalHeight = height;
            
            // Resize renderer texture
            if (this.renderer && typeof this.renderer.resize === 'function') {
                this.renderer.resize(width, height);
            }
            
            return true;
        } catch (e) {
            Logger.error("Resize", `Failed to send resize: ${e.message}`);
            // Fall back to reconnect
            this.showUserInfo('Resizing desktop...');
            this.reconnectWithNewSize(width, height);
            return false;
        }
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
        if (this.renderer) {
            this.renderer.clear();
        } else {
            this.ctx.clearRect(0, 0, this.canvas.width, this.canvas.height);
        }
        this.canvas.className = '';
    },
    
    // ========================================
    // RemoteFX (RFX) Support
    // ========================================

    // MS-RDPRFX block type constants
    // RFX_TILESET is where tiles and quant tables live
    _RFX_WBT_SYNC:           0xCCC0,
    _RFX_WBT_CODEC_VERSIONS: 0xCCC1,
    _RFX_WBT_CHANNELS:       0xCCC2,
    _RFX_WBT_CONTEXT:        0xCCC3,
    _RFX_WBT_FRAME_BEGIN:    0xCCC4,
    _RFX_WBT_FRAME_END:      0xCCC5,
    _RFX_WBT_REGION:         0xCCC6,
    _RFX_WBT_EXTENSION:      0xCCC7,
    _RFX_WBT_TILESET:        0xCAC2,
    _RFX_CBT_TILE:           0xCAC3,

    /**
     * Handle surface commands from a fastpath update.
     * Dispatches SetSurfaceBits based on codec type.
     * @param {BinaryReader} r
     */
    handleSurfaceCommands(r) {
        if (!this.canvasShown) {
            this.showCanvas();
            this.canvasShown = true;
            this.startPerfStats();
            if (this.renderer && typeof this.renderer.resize === 'function') {
                this.renderer.resize(this.canvas.width, this.canvas.height);
            }
        }

        const commands = parseSurfaceCommands(r);
        for (const { type, command } of commands) {
            if (type === 'surfaceBits') {
                const data = command.bitmapData;
                if (data && data.length >= 2) {
                    const magic = data[0] | (data[1] << 8);
                    // RFX block types: WBT_SYNC=0xCCC0 through WBT_EXTENSION=0xCCC7 and WBT_TILESET=0xCAC2
                    if ((magic >= 0xCCC0 && magic <= 0xCCC7) || magic === 0xCAC2 || magic === 0xCAC3) {
                        this.handleRFXSurface(command);
                    } else {
                        this.handleNSCodecSurface(command);
                    }
                }
            }
            // frameMarker: informational
        }
    },

    /**
     * Handle a NSCodec-encoded surface command.
     * @param {SetSurfaceBitsCommand} cmd
     */
    handleNSCodecSurface(cmd) {
        const data = cmd.bitmapData;
        const width = cmd.width;
        const height = cmd.height;

        if (!width || !height || !data || data.length === 0) return;

        const rgba = new Uint8ClampedArray(width * height * 4);

        // Try NSCodec decoder (data starts with 20-byte NSCodec header)
        if (WASMCodec.isReady() && cmd.bpp >= 24 && data.length >= 20) {
            if (WASMCodec.decodeNSCodec(data, width, height, rgba)) {
                this.renderer.drawRGBA(cmd.destLeft, cmd.destTop, width, height, rgba);
                return;
            }
        }

        // Only use raw BGRA/BGR if the data size EXACTLY matches uncompressed pixels.
        // If size doesn't match, this is compressed data we can't decode — skip it.
        const rawBGRA = width * height * 4;
        const rawBGR = width * height * 3;
        if (data.length === rawBGRA) {
            for (let i = 0; i < width * height; i++) {
                rgba[i * 4]     = data[i * 4 + 2]; // R←B
                rgba[i * 4 + 1] = data[i * 4 + 1]; // G
                rgba[i * 4 + 2] = data[i * 4];     // B←R
                rgba[i * 4 + 3] = 255;
            }
            this.renderer.drawRGBA(cmd.destLeft, cmd.destTop, width, height, rgba);
        } else if (data.length === rawBGR) {
            for (let i = 0; i < width * height; i++) {
                rgba[i * 4]     = data[i * 3 + 2];
                rgba[i * 4 + 1] = data[i * 3 + 1];
                rgba[i * 4 + 2] = data[i * 3];
                rgba[i * 4 + 3] = 255;
            }
            this.renderer.drawRGBA(cmd.destLeft, cmd.destTop, width, height, rgba);
        } else {
            Logger.warn("Surface", `Cannot decode surface: ${data.length} bytes for ${width}x${height} (expected ${rawBGRA} or NSCodec)`);
        }
    },

    /**
     * Handle a single SetSurfaceBits command containing RFX data.
     * Parses the MS-RDPRFX message blocks and decodes tiles.
     * @param {SetSurfaceBitsCommand} cmd
     */
    handleRFXSurface(cmd) {
        if (!WASMCodec.isReady()) {
            Logger.error("RFX", "WASM not loaded");
            return;
        }

        const data = cmd.bitmapData;
        if (!data || data.length < 6) {
            Logger.warn("RFX", "Surface command too short");
            return;
        }

        if (!this._rfxCodecLogged) {
            this._rfxCodecLogged = true;
            console.info(
                '%c[RDP Session] Active codec',
                'color: #03A9F4; font-weight: bold',
                'RemoteFX-Image via WASM'
            );
        }

        const destLeft = cmd.destLeft;
        const destTop = cmd.destTop;

        // Parse RFX message blocks
        let offset = 0;
        let quantTables = [];

        while (offset + 6 <= data.length) {
            const blockType = data[offset] | (data[offset + 1] << 8);
            const blockLen = data[offset + 2] | (data[offset + 3] << 8) |
                             (data[offset + 4] << 16) | (data[offset + 5] << 24);

            if (blockLen < 6 || offset + blockLen > data.length) break;

            if (blockType === this._RFX_WBT_TILESET) {
                this._parseAndRenderTileset(data, offset, blockLen, destLeft, destTop, quantTables);
            }
            // SYNC, CODEC_VERSIONS, CHANNELS, CONTEXT, FRAME_BEGIN, REGION, FRAME_END: skip

            offset += blockLen;
        }
    },

    /**
     * Parse a TS_RFX_TILESET block and decode+render its tiles.
     * @param {Uint8Array} data - Full RFX message
     * @param {number} blockOffset - Offset of tileset block
     * @param {number} blockLen - Length of tileset block
     * @param {number} destLeft - Destination X offset from surface command
     * @param {number} destTop - Destination Y offset from surface command
     * @param {Array} quantTables - Output: parsed quant tables
     */
    _parseAndRenderTileset(data, blockOffset, blockLen, destLeft, destTop, quantTables) {
        // TS_RFX_TILESET header: blockType(2) + blockLen(4) + subtype(2) + idx(2) + flags(2)
        //   + numQuant(1) + tileSize(1) + numTiles(2) + tileDataSize(4)
        const hdrSize = 22;
        if (blockLen < hdrSize) return;

        let off = blockOffset + 6; // skip block header
        off += 2; // subtype
        off += 2; // idx
        off += 2; // flags

        const numQuant = data[off]; off++;
        off++; // tileSize (always 64)
        const numTiles = data[off] | (data[off + 1] << 8); off += 2;
        off += 4; // tileDataSize

        // Parse quantization tables (5 bytes each, packed nibbles)
        quantTables.length = 0;
        for (let i = 0; i < numQuant && off + 5 <= blockOffset + blockLen; i++) {
            quantTables.push(data.slice(off, off + 5));
            off += 5;
        }

        // Parse and decode tiles
        let decoded = 0;
        let failed = 0;
        const blockEnd = blockOffset + blockLen;

        for (let i = 0; i < numTiles && off + 6 <= blockEnd; i++) {
            const tileBlockType = data[off] | (data[off + 1] << 8);
            const tileBlockLen = data[off + 2] | (data[off + 3] << 8) |
                                 (data[off + 4] << 16) | (data[off + 5] << 24);

            if (tileBlockType !== this._RFX_CBT_TILE || tileBlockLen < 13 || off + tileBlockLen > blockEnd) {
                off += Math.max(tileBlockLen, 6);
                failed++;
                continue;
            }

            // Tile header after block header (6 bytes):
            // quantIdxY(1) + quantIdxCb(1) + quantIdxCr(1) + xIdx(2) + yIdx(2) + ...
            const quantIdxY = data[off + 6];
            const quantIdxCb = data[off + 7];
            const quantIdxCr = data[off + 8];

            // Build 15-byte quant buffer for WASM (Y + Cb + Cr, 5 bytes each)
            const quantBuf = new Uint8Array(15);
            if (quantIdxY < quantTables.length) quantBuf.set(quantTables[quantIdxY], 0);
            if (quantIdxCb < quantTables.length) quantBuf.set(quantTables[quantIdxCb], 5);
            if (quantIdxCr < quantTables.length) quantBuf.set(quantTables[quantIdxCr], 10);
            this.rfxDecoder.setQuantRaw(quantBuf);

            // Pass the full tile block (including CBT_TILE header) to WASM decoder
            const tileData = data.subarray(off, off + tileBlockLen);
            const result = WASMCodec.decodeRFXTile(tileData, this.rfxDecoder.tileBuffer);

            if (result) {
                const rgba = new Uint8ClampedArray(this.rfxDecoder.tileBuffer.buffer,
                    this.rfxDecoder.tileBuffer.byteOffset, this.rfxDecoder.tileBuffer.byteLength);
                this.renderer.drawRGBA(
                    destLeft + result.x,
                    destTop + result.y,
                    result.width,
                    result.height,
                    rgba
                );
                decoded++;
            } else {
                failed++;
            }

            off += tileBlockLen;
        }

        if (failed > 0) {
            Logger.warn("RFX", `Decoded ${decoded} tiles, ${failed} failed`);
        } else if (decoded > 0) {
            Logger.debug("RFX", `Decoded ${decoded} tiles`);
        }
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
        return this.rfxDecoder.decodeTileToCanvas(tileData, this.renderer);
    },
    
    /**
     * Process multiple RFX tiles
     * @param {Array<Uint8Array>} tiles - Array of tile data
     */
    processRFXTiles(tiles) {
        let decoded = 0;
        let failed = 0;
        
        for (const tileData of tiles) {
            if (this.rfxDecoder.decodeTileToCanvas(tileData, this.renderer)) {
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

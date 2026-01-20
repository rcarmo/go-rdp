/**
 * Graphics handling for RDP client
 * Handles bitmap rendering, pointer/cursor updates, and canvas management
 * @module graphics
 */

import { Logger } from './logger.js';

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
            console.warn('[Palette] Invalid number of colors:', numberColors);
            return;
        }
        
        console.log('[Palette] Received palette update with', numberColors, 'colors');
        
        const paletteData = r.blob(numberColors * 3);
        
        if (typeof goRLE !== 'undefined' && goRLE.setPalette) {
            goRLE.setPalette(new Uint8Array(paletteData), numberColors);
            console.log('[Palette] Updated', numberColors, 'colors via WASM');
        } else {
            console.warn('[Palette] Go WASM not available for palette update');
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
        
        if (typeof goRLE !== 'undefined' && goRLE.processBitmap) {
            const srcData = new Uint8Array(bitmapData.bitmapDataStream);
            const result = goRLE.processBitmap(srcData, width, height, bpp, isCompressed, rgba, rowDelta);
            
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
            Logger.warn("[Bitmap] Go WASM processBitmap failed, bpp:", bpp, "compressed:", isCompressed);
        } else {
            Logger.error("[Bitmap] Go WASM not loaded - cannot process bitmap");
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
            Logger.debug('[Cursor] Pointer update received, type:', header.updateCode);
            
            if (header.isPTRNull()) {
                Logger.debug('[Cursor] PTRNull - hiding cursor');
                this.canvas.className = 'pointer-cache-null';
                return;
            }

            if (header.isPTRDefault()) {
                Logger.debug('[Cursor] PTRDefault - showing default cursor');
                this.canvas.className = 'pointer-cache-default';
                return;
            }

            if (header.isPTRColor()) {
                return;
            }

            if (header.isPTRNew()) {
                Logger.debug('[Cursor] PTRNew - new cursor image received');
                const newPointerUpdate = parseNewPointerUpdate(r);
                Logger.debug('[Cursor] Cursor details:', {
                    cacheIndex: newPointerUpdate.cacheIndex,
                    hotspot: { x: newPointerUpdate.x, y: newPointerUpdate.y },
                    size: { w: newPointerUpdate.width, h: newPointerUpdate.height }
                });
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
                Logger.debug('[Cursor] Applied cursor class:', className);
                return;
            }

            if (header.isPTRCached()) {
                const cacheIndex = r.uint16(true);
                Logger.debug('[Cursor] PTRCached - using cached cursor index:', cacheIndex);
                const className = 'pointer-cache-' + cacheIndex;
                this.canvas.className = className;
                Logger.debug('[Cursor] Applied cached cursor class:', className);
                return;
            }

            if (header.isPTRPosition()) {
                Logger.debug('[Cursor] PTRPosition - cursor position update (ignored)');
                return;
            }
            
            Logger.debug('[Cursor] Unknown pointer update type');
        } catch (error) {
            console.error('[RDP] Error handling pointer:', error);
            Logger.error('[RDP] Pointer error:', error.message, error.stack);
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
                console.log(`[Resize] Window resized to ${newWidth}x${newHeight}, reconnecting...`);
                this.showUserInfo('Resizing desktop...');
                this.reconnectWithNewSize(newWidth, newHeight);
            }
        }, 500);
    },
    
    /**
     * Show the canvas (hide login form)
     */
    showCanvas() {
        console.log('[RDP] First bitmap received - showing canvas');
        
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
    }
};

export default GraphicsMixin;

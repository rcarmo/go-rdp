/**
 * RDP Client Graphics Module
 * Message handling, bitmap processing, and surface commands
 */

// Surface command type constants
const CMDTYPE_SET_SURFACE_BITS = 0x0001;
const CMDTYPE_FRAME_MARKER = 0x0004;
const CMDTYPE_STREAM_SURFACE_BITS = 0x0006;
const CODEC_ID_NSCODEC = 1;

// Utility function
function buf2hex(buffer) {
    return [...new Uint8Array(buffer)]
        .map(x => x.toString(16).padStart(2, '0'))
        .join('');
}

// Handle incoming WebSocket message
Client.prototype.handleMessage = function (arrayBuffer) {
    if (!this.connected) {
        return;
    }
    
    // Check for capability info message (starts with 0xFF marker)
    const bytes = new Uint8Array(arrayBuffer);
    if (bytes.length > 1 && bytes[0] === 0xFF) {
        try {
            const jsonText = new TextDecoder().decode(bytes.slice(1));
            const caps = JSON.parse(jsonText);
            if (caps.type === 'capabilities') {
                console.log('[RDP Server Capabilities]', caps);
                console.log('  Codecs:', caps.codecs.length > 0 ? caps.codecs.join(', ') : 'None');
                console.log('  Surface Commands:', caps.surfaceCommands);
                console.log('  Color Depth:', caps.colorDepth);
                console.log('  Desktop Size:', caps.desktopSize);
                console.log('  Multifragment Size:', caps.multifragmentSize);
                console.log('  Large Pointer:', caps.largePointer);
                console.log('  Frame Acknowledge:', caps.frameAcknowledge);
                this.serverCapabilities = caps;
                return;
            }
        } catch (e) {
            // Not valid JSON, treat as binary
        }
    }
    
    // Try to parse as JSON (for clipboard and file transfer)
    try {
        const text = new TextDecoder().decode(arrayBuffer);
        const message = JSON.parse(text);
        
        if (message.type === 'clipboard_response') {
            this.handleRemoteClipboard(message.data);
            return;
        }
        
        if (message.type === 'file_transfer_status') {
            this.updateFileStatus(message.message);
            return;
        }
        
        if (message.type === 'error') {
            this.showUserError(message.message);
            this.emitEvent('error', {message: message.message});
            return;
        }
    } catch (e) {
        // Not JSON, handle as binary RDP data
    }
    
    // Handle binary RDP data - may contain multiple updates
    const r = new BinaryReader(arrayBuffer);
    while (!r.eof()) {
        this.processUpdate(r);
    }
};

// Process a single fastpath update
Client.prototype.processUpdate = function (r) {
    const header = parseUpdateHeader(r);
    
    Logger.debug('[RDP] Update received, code:', header.updateCode, 'isPointer:', header.isPointer());

    if (header.isCompressed()) {
        console.warn("compressing is not supported");
        r.skip(header.size);
        return;
    }

    if (header.isBitmap()) {
        this.handleBitmap(r);
        return;
    }

    if (header.isPointer()) {
        this.handlePointer(header, r);
        return;
    }

    if (header.isSynchronize()) {
        return;
    }

    if (header.isSurfCMDs()) {
        this.handleSurfaceCommands(r);
        return;
    }

    if (header.isOrders()) {
        Logger.debug('[RDP] Received orders update, skipping (size:', header.size, 'bytes)');
        r.skip(header.size);
        return;
    }

    if (header.isPalette()) {
        Logger.debug('[RDP] Received palette update, skipping');
        r.skip(header.size);
        return;
    }

    console.warn("unknown update:", header.updateCode);
    if (header.size > 0) {
        r.skip(header.size);
    }
};

// Handle bitmap update
Client.prototype.handleBitmap = function (r) {
    const bitmap = parseBitmapUpdate(r);
    
    if (!this.bitmapUpdateLogged) {
        const firstRect = bitmap.rectangles[0];
        if (firstRect) {
            console.log('[RDP Codec] Standard bitmap update - bpp:', firstRect.bitsPerPixel, 
                        'compressed:', !!(firstRect.flags & 0x0001));
        }
        this.bitmapUpdateLogged = true;
    }
    
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
};

// Process a single bitmap rectangle
Client.prototype.processBitmapData = function(bitmapData) {
    const width = bitmapData.width;
    const height = bitmapData.height;
    const bpp = bitmapData.bitsPerPixel;
    const size = width * height;
    const bytesPerPixel = bpp / 8;
    const rowDelta = width * bytesPerPixel;
    const rawSize = size * bytesPerPixel;
    
    let rgba = new Uint8ClampedArray(size * 4);
    
    // Try Go WASM processBitmap
    if (typeof goRLE !== 'undefined' && goRLE.processBitmap) {
        const srcData = new Uint8Array(bitmapData.bitmapDataStream);
        const result = goRLE.processBitmap(srcData, width, height, bpp, bitmapData.isCompressed(), rgba, rowDelta);
        if (result) {
            this.ctx.putImageData(
                new ImageData(rgba, width, height),
                bitmapData.destLeft,
                bitmapData.destTop
            );
            return;
        }
    }
    
    // JavaScript fallback
    let rawData;
    
    if (!bitmapData.isCompressed()) {
        rawData = new Uint8ClampedArray(bitmapData.bitmapDataStream);
    } else if (bpp === 16 || bpp === 15) {
        rawData = new Uint8ClampedArray(rawSize);
        const srcData = new Uint8Array(bitmapData.bitmapDataStream);
        if (!window.rleDecompress16(srcData, rawData, rowDelta)) {
            return;
        }
    } else {
        if (!this.wasmFallbackWarningShown) {
            console.warn('[Bitmap] WASM not available. 24/32-bit compressed bitmaps require WASM.');
            this.wasmFallbackWarningShown = true;
        }
        return;
    }
    
    // Flip vertically (RDP sends bottom-up)
    window.flipV(rawData, width, height, bytesPerPixel);
    
    // Convert to RGBA
    switch (bpp) {
        case 32:
            window.bgra32toRGBA(rawData, rgba);
            break;
        case 24:
            window.bgr24toRGBA(rawData, rawSize, rgba);
            break;
        case 16:
        case 15:
        default:
            window.rgb565toRGBA(rawData, rawSize, rgba);
            break;
    }
    
    this.ctx.putImageData(
        new ImageData(rgba, width, height),
        bitmapData.destLeft,
        bitmapData.destTop
    );
    
    if (this.bitmapCacheEnabled) {
        this.cacheBitmap(bitmapData, rgba);
    }
};

// Handle surface commands (NSCodec, etc.)
Client.prototype.handleSurfaceCommands = function(r) {
    if (!this.canvasShown) {
        this.showCanvas();
        this.canvasShown = true;
    }

    if (!this.surfaceCommandsLogged) {
        console.log('[RDP Codec] Receiving Surface Commands - NSCodec may be in use');
        this.surfaceCommandsLogged = true;
    }

    const data = r.remaining();
    if (!data || data.length === 0) {
        return;
    }

    let offset = 0;
    while (offset < data.length) {
        if (offset + 2 > data.length) break;

        const cmdType = data[offset] | (data[offset + 1] << 8);
        offset += 2;

        switch (cmdType) {
            case CMDTYPE_SET_SURFACE_BITS:
            case CMDTYPE_STREAM_SURFACE_BITS:
                offset = this.handleSetSurfaceBits(data, offset);
                break;

            case CMDTYPE_FRAME_MARKER:
                offset += 6;
                break;

            default:
                Logger.debug('[Surface] Unknown surface command:', cmdType);
                offset = data.length;
                break;
        }
    }
};

// Handle SetSurfaceBits command
Client.prototype.handleSetSurfaceBits = function(data, offset) {
    if (offset + 20 > data.length) {
        return data.length;
    }

    const destLeft = data[offset] | (data[offset + 1] << 8);
    const destTop = data[offset + 2] | (data[offset + 3] << 8);
    const destRight = data[offset + 4] | (data[offset + 5] << 8);
    const destBottom = data[offset + 6] | (data[offset + 7] << 8);
    const bpp = data[offset + 8];
    const flags = data[offset + 9];
    const codecID = data[offset + 11];
    const width = data[offset + 12] | (data[offset + 13] << 8);
    const height = data[offset + 14] | (data[offset + 15] << 8);
    const bitmapDataLength = data[offset + 16] | (data[offset + 17] << 8) | 
                             (data[offset + 18] << 16) | (data[offset + 19] << 24);

    offset += 20;

    if (offset + bitmapDataLength > data.length) {
        return data.length;
    }

    const bitmapData = data.subarray(offset, offset + bitmapDataLength);
    offset += bitmapDataLength;

    let rgba = null;
    
    if (codecID === CODEC_ID_NSCODEC) {
        if (!this.nscodecLogged) {
            console.log('[RDP Codec] NSCodec in use - codecID:', codecID, 'bpp:', bpp, 'size:', width, 'x', height);
            this.nscodecLogged = true;
        }
        
        if (typeof goRLE !== 'undefined' && goRLE.decodeNSCodec) {
            rgba = new Uint8ClampedArray(width * height * 4);
            const srcData = new Uint8Array(bitmapData);
            if (!goRLE.decodeNSCodec(srcData, width, height, rgba)) {
                rgba = null;
            }
        }
        
        if (!rgba && typeof window.NSCodec !== 'undefined') {
            try {
                rgba = window.NSCodec.decode(new Uint8Array(bitmapData), width, height);
            } catch (e) {
                Logger.debug('[Surface] NSCodec decode error:', e);
                return offset;
            }
        }
        
        if (!rgba) {
            Logger.debug('[Surface] No NSCodec decoder available');
            return offset;
        }
    } else if (codecID === 0) {
        if (!this.rawBitmapLogged) {
            console.log('[RDP Codec] Raw/uncompressed bitmap - codecID:', codecID, 'bpp:', bpp);
            this.rawBitmapLogged = true;
        }
        if (bpp === 32) {
            rgba = window.bgra32toRGBA(bitmapData, width, height);
        } else if (bpp === 24) {
            rgba = window.bgr24toRGBA(bitmapData, width, height);
        } else {
            Logger.debug('[Surface] Unsupported uncompressed bpp:', bpp);
            return offset;
        }
    } else {
        Logger.debug('[Surface] Unsupported codec:', codecID);
        return offset;
    }

    if (rgba) {
        const imageData = new ImageData(new Uint8ClampedArray(rgba), width, height);
        this.ctx.putImageData(imageData, destLeft, destTop);
    }

    return offset;
};

// Bitmap cache management
Client.prototype.initBitmapCache = function() {
    this.bitmapCacheEnabled = true;
    this.bitmapCache = new Map();
    this.bitmapCacheMaxSize = 1000;
    this.bitmapCacheHits = 0;
    this.bitmapCacheMisses = 0;
};

Client.prototype.cacheBitmap = function(bitmapData, rgba) {
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
};

Client.prototype.getCachedBitmap = function(bitmapData) {
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
};

Client.prototype.getBitmapCacheKey = function(bitmapData) {
    const sample = bitmapData.bitmapDataStream.slice(0, Math.min(64, bitmapData.bitmapDataStream.length));
    let hash = 0;
    for (let i = 0; i < sample.length; i++) {
        hash = ((hash << 5) - hash) + sample[i];
        hash = hash & hash;
    }
    return `${bitmapData.width}x${bitmapData.height}x${bitmapData.bitsPerPixel}:${hash}`;
};

Client.prototype.clearBitmapCache = function() {
    if (this.bitmapCache) {
        this.bitmapCache.clear();
    }
    this.bitmapCacheHits = 0;
    this.bitmapCacheMisses = 0;
};

Client.prototype.getBitmapCacheStats = function() {
    return {
        size: this.bitmapCache ? this.bitmapCache.size : 0,
        hits: this.bitmapCacheHits || 0,
        misses: this.bitmapCacheMisses || 0,
        hitRate: this.bitmapCacheHits && this.bitmapCacheMisses 
            ? (this.bitmapCacheHits / (this.bitmapCacheHits + this.bitmapCacheMisses) * 100).toFixed(1) + '%'
            : 'N/A'
    };
};

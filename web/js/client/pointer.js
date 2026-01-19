/**
 * RDP Client Pointer Module
 * Cursor/pointer handling and caching
 */

// Handle pointer/cursor updates
Client.prototype.handlePointer = function (header, r) {
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
            // ptr color is unsupported
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
            this.pointerCacheCanvasCtx.putImageData(newPointerUpdate.getImageData(this.pointerCacheCanvasCtx), 0, 0)

            // Use PNG for better browser compatibility
            const url = this.pointerCacheCanvas.toDataURL('image/png');

            if (this.pointerCache.hasOwnProperty(newPointerUpdate.cacheIndex)) {
                document.getElementsByTagName('head')[0].removeChild(this.pointerCache[newPointerUpdate.cacheIndex]);
                delete this.pointerCache[newPointerUpdate.cacheIndex];
            }

            const style = document.createElement('style');
            const className = 'pointer-cache-' + newPointerUpdate.cacheIndex
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
};

/**
 * Renderer interfaces and Canvas renderer implementation.
 * WebGL renderer is implemented in webgl-renderer.js
 */

export class CanvasRenderer {
    constructor(canvas) {
        this.canvas = canvas;
        this.ctx = null;
    }

    init() {
        this.ctx = this.canvas.getContext('2d');
        return !!this.ctx;
    }

    resize(_width, _height) {
        // Canvas size is managed by the caller
    }

    drawRGBA(x, y, width, height, rgba) {
        if (!this.ctx) {
            return false;
        }
        const imageData = new ImageData(rgba, width, height);
        this.ctx.putImageData(imageData, x, y);
        return true;
    }

    clear() {
        if (!this.ctx) {
            return;
        }
        this.ctx.clearRect(0, 0, this.canvas.width, this.canvas.height);
    }

    destroy() {
        this.ctx = null;
    }

    getContext() {
        return this.ctx;
    }
}

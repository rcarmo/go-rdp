/**
 * WebGL renderer for RGBA blitting.
 * Prefers WebGL2, falls back to WebGL1.
 * Supports batched updates and performance metrics.
 */

import { Logger } from './logger.js';

export class WebGLRenderer {
    constructor(canvas) {
        this.canvas = canvas;
        this.gl = null;
        this.program = null;
        this.texture = null;
        this.positionBuffer = null;
        this.texCoordBuffer = null;
        this.uResolution = null;
        this.uSampler = null;
        this.aPosition = -1;
        this.aTexCoord = -1;
        this.webglVersion = null;
        this.textureWidth = 0;
        this.textureHeight = 0;
        this.contextLost = false;
        this._onContextLost = null;
        this._onContextRestored = null;
        this.onContextLostCallback = null;  // External callback for fallback

        // Batching support
        this._batchMode = false;
        this._batchCount = 0;

        // Performance metrics (debug mode)
        this._metricsEnabled = false;
        this._uploadTimeTotal = 0;
        this._drawTimeTotal = 0;
        this._frameCount = 0;
        this._lastMetricsLog = 0;
    }

    /**
     * Enable performance metrics logging
     * @param {boolean} enabled
     */
    setMetricsEnabled(enabled) {
        this._metricsEnabled = enabled;
        if (enabled) {
            this._resetMetrics();
        }
    }

    _resetMetrics() {
        this._uploadTimeTotal = 0;
        this._drawTimeTotal = 0;
        this._frameCount = 0;
        this._lastMetricsLog = performance.now();
    }

    init() {
        this.gl = this._initContext();
        if (!this.gl) {
            return false;
        }

        // Set up context loss handlers before any GL operations
        this._setupContextLossHandlers();

        if (!this._initProgram()) {
            this.destroy();
            return false;
        }

        this._initBuffers();
        this._initTexture();
        this._configureGL();

        Logger.info('Renderer', `WebGL${this.webglVersion} initialized`);

        return true;
    }

    /**
     * Set up WebGL context loss/restore handlers
     */
    _setupContextLossHandlers() {
        this._onContextLost = (e) => {
            e.preventDefault();
            this.contextLost = true;
            Logger.warn('Renderer', 'WebGL context lost');
            // Notify external handler for potential fallback
            if (this.onContextLostCallback) {
                this.onContextLostCallback();
            }
        };

        this._onContextRestored = () => {
            Logger.info('Renderer', 'WebGL context restored, reinitializing');
            this.contextLost = false;
            // Reinitialize GL state
            if (!this._initProgram()) {
                Logger.error('Renderer', 'Failed to reinitialize after context restore');
                return;
            }
            this._initBuffers();
            this._initTexture();
            this._configureGL();
        };

        this.canvas.addEventListener('webglcontextlost', this._onContextLost);
        this.canvas.addEventListener('webglcontextrestored', this._onContextRestored);
    }

    /**
     * Check if context is valid for rendering
     * @returns {boolean}
     */
    isContextValid() {
        return this.gl && !this.contextLost && !this.gl.isContextLost();
    }

    resize(width, height) {
        if (!this.isContextValid()) {
            return;
        }
        if (width !== this.textureWidth || height !== this.textureHeight) {
            this._resizeTexture(width, height);
            this._updatePositionBuffer(width, height);
        }
        this.gl.viewport(0, 0, this.canvas.width, this.canvas.height);
    }

    drawRGBA(x, y, width, height, rgba) {
        if (!this.isContextValid() || !this.texture) {
            return false;
        }
        const gl = this.gl;

        gl.bindTexture(gl.TEXTURE_2D, this.texture);
        gl.pixelStorei(gl.UNPACK_ALIGNMENT, 1);

        const uploadStart = this._metricsEnabled ? performance.now() : 0;

        try {
            gl.texSubImage2D(
                gl.TEXTURE_2D,
                0,
                x,
                y,
                width,
                height,
                gl.RGBA,
                gl.UNSIGNED_BYTE,
                rgba
            );
        } catch (e) {
            Logger.error('Renderer', `texSubImage2D failed: ${e.message}`);
            return false;
        }

        if (this._metricsEnabled) {
            this._uploadTimeTotal += performance.now() - uploadStart;
        }

        const err = gl.getError();
        if (err !== gl.NO_ERROR) {
            Logger.warn('Renderer', `GL error after texSubImage2D: ${err}`);
            return false;
        }

        // In batch mode, defer the draw call
        if (this._batchMode) {
            this._batchCount++;
            return true;
        }

        this._drawWithMetrics();
        return true;
    }

    /**
     * Begin batching texture uploads. Call flush() to render.
     * Use when processing multiple rectangles in a single frame.
     */
    beginBatch() {
        this._batchMode = true;
        this._batchCount = 0;
    }

    /**
     * End batch mode and render all accumulated texture updates.
     */
    flush() {
        if (!this._batchMode) {
            return;
        }
        this._batchMode = false;

        if (this._batchCount > 0 && this.isContextValid()) {
            this._drawWithMetrics();
        }
        this._batchCount = 0;
    }

    /**
     * Draw with optional metrics tracking
     */
    _drawWithMetrics() {
        const drawStart = this._metricsEnabled ? performance.now() : 0;
        this._draw();
        if (this._metricsEnabled) {
            this._drawTimeTotal += performance.now() - drawStart;
            this._frameCount++;
            this._logMetricsIfNeeded();
        }
    }

    /**
     * Log metrics every 10 seconds if enabled
     */
    _logMetricsIfNeeded() {
        const now = performance.now();
        const elapsed = now - this._lastMetricsLog;
        if (elapsed >= 10000 && this._frameCount > 0) {
            const avgUpload = (this._uploadTimeTotal / this._frameCount).toFixed(2);
            const avgDraw = (this._drawTimeTotal / this._frameCount).toFixed(2);
            const fps = (this._frameCount / (elapsed / 1000)).toFixed(1);
            console.info(
                '%c[WebGL Metrics]',
                'color: #9C27B0; font-weight: bold',
                `FPS: ${fps}`,
                `| Avg upload: ${avgUpload}ms`,
                `| Avg draw: ${avgDraw}ms`,
                `| Frames: ${this._frameCount}`
            );
            this._resetMetrics();
        }
    }

    clear() {
        if (!this.isContextValid()) {
            return;
        }
        const gl = this.gl;
        gl.clearColor(0, 0, 0, 1);
        gl.clear(gl.COLOR_BUFFER_BIT);
    }

    destroy() {
        // Remove event listeners
        if (this._onContextLost) {
            this.canvas.removeEventListener('webglcontextlost', this._onContextLost);
            this._onContextLost = null;
        }
        if (this._onContextRestored) {
            this.canvas.removeEventListener('webglcontextrestored', this._onContextRestored);
            this._onContextRestored = null;
        }

        if (!this.gl) {
            return;
        }
        const gl = this.gl;
        if (this.texture) {
            gl.deleteTexture(this.texture);
            this.texture = null;
        }
        if (this.positionBuffer) {
            gl.deleteBuffer(this.positionBuffer);
            this.positionBuffer = null;
        }
        if (this.texCoordBuffer) {
            gl.deleteBuffer(this.texCoordBuffer);
            this.texCoordBuffer = null;
        }
        if (this.program) {
            gl.deleteProgram(this.program);
            this.program = null;
        }
        this.gl = null;
        this.contextLost = false;
    }

    getContext() {
        return this.gl;
    }

    _initContext() {
        const gl2 = this.canvas.getContext('webgl2', { alpha: false, premultipliedAlpha: false });
        if (gl2) {
            this.webglVersion = 2;
            return gl2;
        }
        const gl1 = this.canvas.getContext('webgl', { alpha: false, premultipliedAlpha: false }) ||
            this.canvas.getContext('experimental-webgl', { alpha: false, premultipliedAlpha: false });
        if (gl1) {
            this.webglVersion = 1;
            return gl1;
        }
        return null;
    }

    _initProgram() {
        const gl = this.gl;
        const vertexSrc = `
            attribute vec2 a_position;
            attribute vec2 a_texCoord;
            varying vec2 v_texCoord;
            uniform vec2 u_resolution;
            void main() {
                vec2 zeroToOne = a_position / u_resolution;
                vec2 clipSpace = zeroToOne * 2.0 - 1.0;
                gl_Position = vec4(clipSpace * vec2(1, -1), 0, 1);
                v_texCoord = a_texCoord;
            }
        `;
        const fragmentSrc = `
            precision mediump float;
            varying vec2 v_texCoord;
            uniform sampler2D u_sampler;
            void main() {
                gl_FragColor = texture2D(u_sampler, v_texCoord);
            }
        `;

        const vert = this._compileShader(gl.VERTEX_SHADER, vertexSrc);
        const frag = this._compileShader(gl.FRAGMENT_SHADER, fragmentSrc);
        if (!vert || !frag) {
            if (vert) gl.deleteShader(vert);
            if (frag) gl.deleteShader(frag);
            return false;
        }

        const program = gl.createProgram();
        gl.attachShader(program, vert);
        gl.attachShader(program, frag);
        gl.linkProgram(program);

        // Shaders can be deleted after linking - they remain attached to the program
        gl.deleteShader(vert);
        gl.deleteShader(frag);

        if (!gl.getProgramParameter(program, gl.LINK_STATUS)) {
            Logger.error('Renderer', `WebGL program link failed: ${gl.getProgramInfoLog(program)}`);
            gl.deleteProgram(program);
            return false;
        }

        this.program = program;
        gl.useProgram(program);
        this.uResolution = gl.getUniformLocation(program, 'u_resolution');
        this.uSampler = gl.getUniformLocation(program, 'u_sampler');
        this.aPosition = gl.getAttribLocation(program, 'a_position');
        this.aTexCoord = gl.getAttribLocation(program, 'a_texCoord');
        return true;
    }

    _compileShader(type, source) {
        const gl = this.gl;
        const shader = gl.createShader(type);
        gl.shaderSource(shader, source);
        gl.compileShader(shader);
        if (!gl.getShaderParameter(shader, gl.COMPILE_STATUS)) {
            Logger.error('Renderer', `WebGL shader compile failed: ${gl.getShaderInfoLog(shader)}`);
            gl.deleteShader(shader);
            return null;
        }
        return shader;
    }

    _initBuffers() {
        const gl = this.gl;
        const width = this.canvas.width;
        const height = this.canvas.height;

        // Position buffer - full screen quad
        const positionBuffer = gl.createBuffer();
        gl.bindBuffer(gl.ARRAY_BUFFER, positionBuffer);
        const positions = new Float32Array([
            0, 0,
            width, 0,
            0, height,
            0, height,
            width, 0,
            width, height
        ]);
        gl.bufferData(gl.ARRAY_BUFFER, positions, gl.DYNAMIC_DRAW);

        // Texture coordinate buffer - static, maps full texture
        const texCoordBuffer = gl.createBuffer();
        gl.bindBuffer(gl.ARRAY_BUFFER, texCoordBuffer);
        const texCoords = new Float32Array([
            0, 0,
            1, 0,
            0, 1,
            0, 1,
            1, 0,
            1, 1
        ]);
        gl.bufferData(gl.ARRAY_BUFFER, texCoords, gl.STATIC_DRAW);

        // Set up vertex attributes
        gl.bindBuffer(gl.ARRAY_BUFFER, positionBuffer);
        gl.enableVertexAttribArray(this.aPosition);
        gl.vertexAttribPointer(this.aPosition, 2, gl.FLOAT, false, 0, 0);

        gl.bindBuffer(gl.ARRAY_BUFFER, texCoordBuffer);
        gl.enableVertexAttribArray(this.aTexCoord);
        gl.vertexAttribPointer(this.aTexCoord, 2, gl.FLOAT, false, 0, 0);

        this.positionBuffer = positionBuffer;
        this.texCoordBuffer = texCoordBuffer;
    }

    /**
     * Update position buffer when canvas/texture size changes
     */
    _updatePositionBuffer(width, height) {
        const gl = this.gl;
        gl.bindBuffer(gl.ARRAY_BUFFER, this.positionBuffer);
        const positions = new Float32Array([
            0, 0,
            width, 0,
            0, height,
            0, height,
            width, 0,
            width, height
        ]);
        gl.bufferData(gl.ARRAY_BUFFER, positions, gl.DYNAMIC_DRAW);
    }

    _initTexture() {
        const gl = this.gl;
        this.texture = gl.createTexture();
        gl.bindTexture(gl.TEXTURE_2D, this.texture);
        gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST);
        gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST);
        gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE);
        gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE);
        this._resizeTexture(this.canvas.width, this.canvas.height);
    }

    _resizeTexture(width, height) {
        const gl = this.gl;
        this.textureWidth = width;
        this.textureHeight = height;
        gl.bindTexture(gl.TEXTURE_2D, this.texture);

        try {
            gl.texImage2D(
                gl.TEXTURE_2D,
                0,
                gl.RGBA,
                width,
                height,
                0,
                gl.RGBA,
                gl.UNSIGNED_BYTE,
                null
            );
        } catch (e) {
            Logger.error('Renderer', `texImage2D failed: ${e.message}`);
            return;
        }

        const err = gl.getError();
        if (err !== gl.NO_ERROR) {
            Logger.error('Renderer', `GL error creating texture ${width}x${height}: ${err}`);
        }

        if (this.uResolution) {
            gl.uniform2f(this.uResolution, width, height);
        }
    }

    _configureGL() {
        const gl = this.gl;
        gl.disable(gl.DEPTH_TEST);
        gl.disable(gl.BLEND);
        gl.viewport(0, 0, this.canvas.width, this.canvas.height);
        gl.clearColor(0, 0, 0, 1);
        gl.clear(gl.COLOR_BUFFER_BIT);
        if (this.uSampler) {
            gl.uniform1i(this.uSampler, 0);
        }
    }

    _draw() {
        const gl = this.gl;
        gl.drawArrays(gl.TRIANGLES, 0, 6);
    }
}

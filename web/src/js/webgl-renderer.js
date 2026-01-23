/**
 * WebGL renderer for RGBA blitting.
 * Prefers WebGL2, falls back to WebGL1.
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
        this.webglVersion = null;
        this.textureWidth = 0;
        this.textureHeight = 0;
    }

    init() {
        this.gl = this._initContext();
        if (!this.gl) {
            return false;
        }

        if (!this._initProgram()) {
            this.destroy();
            return false;
        }

        this._initBuffers();
        this._initTexture();
        this._configureGL();

        Logger.info('Renderer', `WebGL${this.webglVersion} active`);
        console.info(
            '%c[RDP Client] Active renderer',
            'color: #FF9800; font-weight: bold',
            `WebGL${this.webglVersion}`
        );

        return true;
    }

    resize(width, height) {
        if (!this.gl) {
            return;
        }
        if (width !== this.textureWidth || height !== this.textureHeight) {
            this._resizeTexture(width, height);
        }
        this.gl.viewport(0, 0, this.canvas.width, this.canvas.height);
    }

    drawRGBA(x, y, width, height, rgba) {
        if (!this.gl || !this.texture) {
            return false;
        }
        const gl = this.gl;

        gl.bindTexture(gl.TEXTURE_2D, this.texture);
        gl.pixelStorei(gl.UNPACK_ALIGNMENT, 1);

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

        this._draw();
        return true;
    }

    clear() {
        if (!this.gl) {
            return;
        }
        const gl = this.gl;
        gl.clearColor(0, 0, 0, 1);
        gl.clear(gl.COLOR_BUFFER_BIT);
    }

    destroy() {
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
            return false;
        }

        const program = gl.createProgram();
        gl.attachShader(program, vert);
        gl.attachShader(program, frag);
        gl.linkProgram(program);

        if (!gl.getProgramParameter(program, gl.LINK_STATUS)) {
            Logger.error('Renderer', `WebGL program link failed: ${gl.getProgramInfoLog(program)}`);
            return false;
        }

        this.program = program;
        gl.useProgram(program);
        this.uResolution = gl.getUniformLocation(program, 'u_resolution');
        this.uSampler = gl.getUniformLocation(program, 'u_sampler');
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

        const positionBuffer = gl.createBuffer();
        gl.bindBuffer(gl.ARRAY_BUFFER, positionBuffer);
        const positions = new Float32Array([
            0, 0,
            this.canvas.width, 0,
            0, this.canvas.height,
            0, this.canvas.height,
            this.canvas.width, 0,
            this.canvas.width, this.canvas.height
        ]);
        gl.bufferData(gl.ARRAY_BUFFER, positions, gl.STATIC_DRAW);

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

        const aPosition = gl.getAttribLocation(this.program, 'a_position');
        gl.bindBuffer(gl.ARRAY_BUFFER, positionBuffer);
        gl.enableVertexAttribArray(aPosition);
        gl.vertexAttribPointer(aPosition, 2, gl.FLOAT, false, 0, 0);

        const aTexCoord = gl.getAttribLocation(this.program, 'a_texCoord');
        gl.bindBuffer(gl.ARRAY_BUFFER, texCoordBuffer);
        gl.enableVertexAttribArray(aTexCoord);
        gl.vertexAttribPointer(aTexCoord, 2, gl.FLOAT, false, 0, 0);

        this.positionBuffer = positionBuffer;
        this.texCoordBuffer = texCoordBuffer;
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

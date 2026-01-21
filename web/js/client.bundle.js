var RDP = (() => {
  var __defProp = Object.defineProperty;
  var __getOwnPropDesc = Object.getOwnPropertyDescriptor;
  var __getOwnPropNames = Object.getOwnPropertyNames;
  var __hasOwnProp = Object.prototype.hasOwnProperty;
  var __export = (target, all) => {
    for (var name in all)
      __defProp(target, name, { get: all[name], enumerable: true });
  };
  var __copyProps = (to, from, except, desc) => {
    if (from && typeof from === "object" || typeof from === "function") {
      for (let key of __getOwnPropNames(from))
        if (!__hasOwnProp.call(to, key) && key !== except)
          __defProp(to, key, { get: () => from[key], enumerable: !(desc = __getOwnPropDesc(from, key)) || desc.enumerable });
    }
    return to;
  };
  var __toCommonJS = (mod) => __copyProps(__defProp({}, "__esModule", { value: true }), mod);

  // index.js
  var src_exports = {};
  __export(src_exports, {
    Client: () => Client,
    FallbackCodec: () => FallbackCodec,
    Logger: () => Logger2,
    RFXDecoder: () => RFXDecoder,
    WASMCodec: () => WASMCodec,
    default: () => src_default,
    isWASMSupported: () => isWASMSupported
  });

  // logger.js
  var LogLevel = {
    DEBUG: 0,
    INFO: 1,
    WARN: 2,
    ERROR: 3,
    NONE: 4
  };
  var Logger2 = {
    level: LogLevel.WARN,
    // Default to WARN - minimal console output
    /**
     * Set log level from string
     * @param {string} levelStr - 'debug', 'info', 'warn', 'error', 'none'
     */
    setLevel(levelStr) {
      const levels = {
        "debug": LogLevel.DEBUG,
        "info": LogLevel.INFO,
        "warn": LogLevel.WARN,
        "warning": LogLevel.WARN,
        "error": LogLevel.ERROR,
        "none": LogLevel.NONE
      };
      this.level = levels[levelStr.toLowerCase()] ?? LogLevel.WARN;
    },
    /**
     * Log debug message (protocol details, byte dumps)
     * @param {string} category - Log category (e.g., 'Cursor', 'Bitmap')
     * @param {...any} args - Log arguments
     */
    debug(category, ...args) {
      if (this.level <= LogLevel.DEBUG) {
        console.log(`[${category}]`, ...args);
      }
    },
    /**
     * Log info message (connection state, key events)
     * @param {string} category - Log category
     * @param {...any} args - Log arguments
     */
    info(category, ...args) {
      if (this.level <= LogLevel.INFO) {
        console.info(`[${category}]`, ...args);
      }
    },
    /**
     * Log warning message (recoverable issues)
     * @param {string} category - Log category
     * @param {...any} args - Log arguments
     */
    warn(category, ...args) {
      if (this.level <= LogLevel.WARN) {
        console.warn(`[${category}]`, ...args);
      }
    },
    /**
     * Log error message (failures)
     * @param {string} category - Log category
     * @param {...any} args - Log arguments
     */
    error(category, ...args) {
      if (this.level <= LogLevel.ERROR) {
        console.error(`[${category}]`, ...args);
      }
    },
    /**
     * Enable debug logging (convenience method)
     */
    enableDebug() {
      this.level = LogLevel.DEBUG;
    },
    /**
     * Enable info logging
     */
    enableInfo() {
      this.level = LogLevel.INFO;
    },
    /**
     * Disable all logging except errors (default)
     */
    quiet() {
      this.level = LogLevel.ERROR;
    },
    /**
     * Disable all logging
     */
    silent() {
      this.level = LogLevel.NONE;
    }
  };

  // session.js
  function generateSessionId() {
    const array = new Uint8Array(16);
    crypto.getRandomValues(array);
    const hex = Array.from(array, (byte) => byte.toString(16).padStart(2, "0")).join("");
    return "session_" + hex;
  }
  var SessionMixin = {
    /**
     * Initialize session management
     */
    initSession() {
      this.reconnectAttempts = 0;
      this.maxReconnectAttempts = 5;
      this.reconnectDelay = 2e3;
      this.reconnectTimeout = null;
      this.lastConnectionTime = null;
      this.manualDisconnect = false;
      this.sessionId = generateSessionId();
      this.maxSessionTime = 8 * 60 * 60 * 1e3;
      this.maxIdleTime = 30 * 60 * 1e3;
      this.lastActivityTime = null;
      this.sessionTimeout = null;
      this.idleTimeout = null;
      this.warningTimeout = null;
      this.warningShown = false;
      this.loadSession();
    },
    /**
     * Check if auto-reconnect should be attempted
     * @returns {boolean}
     */
    shouldAutoReconnect() {
      return false;
    },
    /**
     * Save session data to cookies
     */
    saveSession() {
      try {
        const expires = new Date(Date.now() + 7 * 24 * 60 * 60 * 1e3).toUTCString();
        document.cookie = `rdp_host=${encodeURIComponent(this.hostEl.value)}; expires=${expires}; path=/; SameSite=Strict`;
        document.cookie = `rdp_user=${encodeURIComponent(this.userEl.value)}; expires=${expires}; path=/; SameSite=Strict`;
      } catch (e) {
        Logger2.warn("Session", `Failed to save: ${e.message}`);
      }
    },
    /**
     * Load session data from cookies
     */
    loadSession() {
      try {
        const cookies = document.cookie.split(";").reduce((acc, cookie) => {
          const [key, value] = cookie.trim().split("=");
          if (key && value)
            acc[key] = decodeURIComponent(value);
          return acc;
        }, {});
        if (cookies.rdp_host)
          this.hostEl.value = cookies.rdp_host;
        if (cookies.rdp_user)
          this.userEl.value = cookies.rdp_user;
      } catch (e) {
        Logger2.debug("Session", `Failed to load: ${e.message}`);
      }
    },
    /**
     * Verify session data integrity
     * @param {Object} session
     * @returns {boolean}
     */
    verifySessionIntegrity(session) {
      const requiredFields = ["host", "user", "timestamp", "sessionId"];
      return requiredFields.every((field) => session.hasOwnProperty(field));
    },
    /**
     * Clear session data
     */
    clearSession() {
      document.cookie = "rdp_host=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;";
      document.cookie = "rdp_user=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;";
      this.manualDisconnect = true;
    },
    /**
     * Schedule a reconnection attempt
     * @param {number} delay - Delay in milliseconds
     */
    scheduleReconnect(delay) {
      if (this.reconnectTimeout) {
        clearTimeout(this.reconnectTimeout);
      }
      if (this.reconnectAttempts >= this.maxReconnectAttempts || this.manualDisconnect) {
        return;
      }
      this.reconnectTimeout = setTimeout(() => {
        if (this.shouldAutoReconnect() && !this.connected) {
          Logger2.debug("Session", `Reconnect attempt ${this.reconnectAttempts + 1}/${this.maxReconnectAttempts}`);
          this.attemptReconnect();
        }
      }, delay);
    },
    /**
     * Attempt to reconnect to the server
     */
    attemptReconnect() {
      if (!this.hostEl.value || !this.userEl.value) {
        return;
      }
      this.reconnectAttempts++;
      if (this.socket && this.socket.readyState !== WebSocket.CLOSED) {
        try {
          this.socket.close();
        } catch (e) {
        }
      }
      const url = new URL(this.websocketURL);
      url.searchParams.set("width", this.canvas.width);
      url.searchParams.set("height", this.canvas.height);
      url.searchParams.set("sessionId", this.sessionId);
      const password = this.passwordEl ? this.passwordEl.value : "";
      this.socket = new WebSocket(url.toString());
      this.socket.onopen = () => {
        const credMsg = JSON.stringify({
          type: "credentials",
          host: this.hostEl.value,
          user: this.userEl.value,
          password
        });
        this.socket.send(credMsg);
        this.initialize();
      };
      this.socket.onmessage = (e) => {
        e.data.arrayBuffer().then((arrayBuffer) => this.handleMessage(arrayBuffer)).catch((err) => Logger2.error("Session", `Failed to read message: ${err.message}`));
      };
      this.socket.onerror = (e) => {
        Logger2.warn("Session", "Reconnection error");
      };
      this.socket.onclose = (e) => {
        if (!this.manualDisconnect && this.reconnectAttempts < this.maxReconnectAttempts) {
          const exponent = Math.max(0, this.reconnectAttempts - 1);
          const exponentialDelay = this.reconnectDelay * Math.pow(2, exponent);
          this.scheduleReconnect(Math.min(exponentialDelay, 3e4));
        }
      };
    },
    /**
     * Start session and idle timeout tracking
     */
    startTimeoutTracking() {
      this.lastConnectionTime = Date.now();
      this.lastActivityTime = Date.now();
      this.sessionTimeout = setTimeout(() => {
        this.handleSessionTimeout();
      }, this.maxSessionTime);
      this.resetIdleTimeout();
    },
    /**
     * Update activity timestamp
     */
    updateActivity() {
      this.lastActivityTime = Date.now();
      this.warningShown = false;
      const warning = document.getElementById("idle-warning");
      if (warning) {
        warning.style.display = "none";
      }
      this.resetIdleTimeout();
    },
    /**
     * Reset idle timeout
     */
    resetIdleTimeout() {
      if (this.idleTimeout) {
        clearTimeout(this.idleTimeout);
      }
      if (this.warningTimeout) {
        clearTimeout(this.warningTimeout);
      }
      this.warningTimeout = setTimeout(() => {
        this.showIdleWarning();
      }, this.maxIdleTime - 5 * 60 * 1e3);
      this.idleTimeout = setTimeout(() => {
        this.handleIdleTimeout();
      }, this.maxIdleTime);
    },
    /**
     * Show idle warning to user
     */
    showIdleWarning() {
      if (this.warningShown)
        return;
      this.warningShown = true;
      let warning = document.getElementById("idle-warning");
      if (!warning) {
        warning = document.createElement("div");
        warning.id = "idle-warning";
        warning.className = "idle-warning";
        warning.innerHTML = "Session will disconnect in 5 minutes due to inactivity. Move mouse or press a key to stay connected.";
        document.body.appendChild(warning);
      }
      warning.style.display = "block";
    },
    /**
     * Handle idle timeout
     */
    handleIdleTimeout() {
      Logger2.debug("Session", "Disconnected due to inactivity");
      this.showUserWarning("Session disconnected due to inactivity");
      this.disconnect();
    },
    /**
     * Handle session timeout
     */
    handleSessionTimeout() {
      Logger2.debug("Session", "Maximum session time reached (8 hours)");
      this.showUserWarning("Session disconnected - maximum session time reached (8 hours)");
      this.disconnect();
    },
    /**
     * Clear all session timeouts
     */
    clearAllTimeouts() {
      if (this.sessionTimeout) {
        clearTimeout(this.sessionTimeout);
        this.sessionTimeout = null;
      }
      if (this.idleTimeout) {
        clearTimeout(this.idleTimeout);
        this.idleTimeout = null;
      }
      if (this.warningTimeout) {
        clearTimeout(this.warningTimeout);
        this.warningTimeout = null;
      }
      if (this.reconnectTimeout) {
        clearTimeout(this.reconnectTimeout);
        this.reconnectTimeout = null;
      }
    }
  };

  // input.js
  function elementOffset(el) {
    let x = 0;
    let y = 0;
    while (el && !isNaN(el.offsetLeft) && !isNaN(el.offsetTop)) {
      x += el.offsetLeft - el.scrollLeft;
      y += el.offsetTop - el.scrollTop;
      el = el.offsetParent;
    }
    return { top: y, left: x };
  }
  function mouseButtonMap(button) {
    switch (button) {
      case 0:
        return 1;
      case 1:
        return 3;
      case 2:
        return 2;
      default:
        return 1;
    }
  }
  var InputMixin = {
    /**
     * Initialize input handling
     */
    initInput() {
      this.isDragging = false;
      this.inputQueue = [];
      this.inputFlushPending = false;
      this.lastMouseMove = null;
      this.mouseThrottleMs = 16;
      this.lastMouseSendTime = 0;
      this.lastActivityUpdate = null;
      this.lastTouchUpdate = null;
      this.handleKeyDown = this.handleKeyDown.bind(this);
      this.handleKeyUp = this.handleKeyUp.bind(this);
      this.handleMouseMove = this.handleMouseMove.bind(this);
      this.handleMouseDown = this.handleMouseDown.bind(this);
      this.handleMouseUp = this.handleMouseUp.bind(this);
      this.handleWheel = this.handleWheel.bind(this);
      this.handleTouchStart = this.handleTouchStart.bind(this);
      this.handleTouchMove = this.handleTouchMove.bind(this);
      this.handleTouchEnd = this.handleTouchEnd.bind(this);
    },
    /**
     * Convert screen coordinates to desktop coordinates
     * @param {number} screenX
     * @param {number} screenY
     * @returns {{x: number, y: number}}
     */
    screenToDesktop(screenX, screenY) {
      const offset = elementOffset(this.canvas);
      const x = Math.floor(screenX - offset.left);
      const y = Math.floor(screenY - offset.top);
      return { x, y };
    },
    /**
     * Queue an input event for sending
     * @param {ArrayBuffer} data
     * @param {boolean} isMouseMove
     */
    queueInput(data, isMouseMove) {
      if (isMouseMove) {
        this.lastMouseMove = data;
      } else {
        this.inputQueue.push(data);
      }
      if (!this.inputFlushPending) {
        this.inputFlushPending = true;
        setTimeout(() => this.flushInputQueue(), 0);
      }
    },
    /**
     * Flush queued input events to the server
     */
    flushInputQueue() {
      this.inputFlushPending = false;
      if (!this.connected || !this.socket || this.socket.readyState !== WebSocket.OPEN) {
        this.inputQueue = [];
        this.lastMouseMove = null;
        return;
      }
      while (this.inputQueue.length > 0) {
        const data = this.inputQueue.shift();
        try {
          this.socket.send(data);
        } catch (e) {
          this.inputQueue = [];
          break;
        }
      }
      const now = Date.now();
      if (this.lastMouseMove && now - this.lastMouseSendTime >= this.mouseThrottleMs) {
        try {
          this.socket.send(this.lastMouseMove);
          this.lastMouseSendTime = now;
        } catch (e) {
        }
        this.lastMouseMove = null;
      } else if (this.lastMouseMove) {
        if (!this.inputFlushPending) {
          this.inputFlushPending = true;
          setTimeout(() => this.flushInputQueue(), this.mouseThrottleMs);
        }
      }
    },
    /**
     * Handle keydown event
     * @param {KeyboardEvent} e
     */
    handleKeyDown(e) {
      if (!this.connected) {
        this.showUserError("Cannot send keystrokes: not connected to server");
        return;
      }
      this.updateActivity();
      const event = new KeyboardEventKeyDown(e.code);
      if (event.keyCode === void 0) {
        this.logError("Key mapping error", { code: e.code, key: e.key });
        this.showUserError("Unsupported key: " + e.key);
        e.preventDefault();
        return false;
      }
      try {
        const data = event.serialize();
        this.queueInput(data, false);
      } catch (error) {
        this.logError("Key send error", { code: e.code, error: error.message });
        this.showUserError("Failed to send keystroke");
      }
      e.preventDefault();
      return false;
    },
    /**
     * Handle keyup event
     * @param {KeyboardEvent} e
     */
    handleKeyUp(e) {
      if (!this.connected) {
        return;
      }
      this.updateActivity();
      const event = new KeyboardEventKeyUp(e.code);
      if (event.keyCode === void 0) {
        Logger2.debug("[Input] Undefined key up:", e.code);
        e.preventDefault();
        return false;
      }
      const data = event.serialize();
      this.queueInput(data, false);
      e.preventDefault();
      return false;
    },
    /**
     * Handle mouse move event
     * @param {MouseEvent} e
     */
    handleMouseMove(e) {
      if (!this.lastActivityUpdate || Date.now() - this.lastActivityUpdate > 1e3) {
        this.updateActivity();
        this.lastActivityUpdate = Date.now();
      }
      try {
        const pos = this.screenToDesktop(e.clientX, e.clientY);
        const event = new MouseMoveEvent(pos.x, pos.y);
        const data = event.serialize();
        this.queueInput(data, true);
      } catch (error) {
        this.logError("Mouse move error", { x: e.clientX, y: e.clientY, error: error.message });
      }
      e.preventDefault();
      return false;
    },
    /**
     * Handle mouse down event
     * @param {MouseEvent} e
     */
    handleMouseDown(e) {
      this.updateActivity();
      this.isDragging = true;
      const pos = this.screenToDesktop(e.clientX, e.clientY);
      const event = new MouseDownEvent(pos.x, pos.y, mouseButtonMap(e.button));
      const data = event.serialize();
      this.queueInput(data, false);
      document.addEventListener("mousemove", this.handleMouseMove);
      document.addEventListener("mouseup", this.handleMouseUp);
      e.preventDefault();
      return false;
    },
    /**
     * Handle mouse up event
     * @param {MouseEvent} e
     */
    handleMouseUp(e) {
      this.isDragging = false;
      const pos = this.screenToDesktop(e.clientX, e.clientY);
      const event = new MouseUpEvent(pos.x, pos.y, mouseButtonMap(e.button));
      const data = event.serialize();
      this.queueInput(data, false);
      document.removeEventListener("mousemove", this.handleMouseMove);
      document.removeEventListener("mouseup", this.handleMouseUp);
      e.preventDefault();
      return false;
    },
    /**
     * Handle wheel event
     * @param {WheelEvent} e
     */
    handleWheel(e) {
      this.updateActivity();
      const pos = this.screenToDesktop(e.clientX, e.clientY);
      const isHorizontal = Math.abs(e.deltaX) > Math.abs(e.deltaY);
      const delta = isHorizontal ? e.deltaX : e.deltaY;
      const step = Math.round(Math.abs(delta) * 15 / 8);
      const event = new MouseWheelEvent(pos.x, pos.y, step, delta > 0, isHorizontal);
      const data = event.serialize();
      this.queueInput(data, false);
      e.preventDefault();
      return false;
    },
    /**
     * Handle touch start event
     * @param {TouchEvent} e
     */
    handleTouchStart(e) {
      if (!this.connected)
        return;
      e.preventDefault();
      this.updateActivity();
      const touch = e.touches[0];
      const mouseEvent = {
        clientX: touch.clientX,
        clientY: touch.clientY,
        button: 0,
        preventDefault: () => {
        }
      };
      return this.handleMouseDown(mouseEvent);
    },
    /**
     * Handle touch move event
     * @param {TouchEvent} e
     */
    handleTouchMove(e) {
      if (!this.connected)
        return;
      e.preventDefault();
      if (!this.lastTouchUpdate || Date.now() - this.lastTouchUpdate > 16) {
        this.updateActivity();
        this.lastTouchUpdate = Date.now();
        const touch = e.touches[0];
        const mouseEvent = {
          clientX: touch.clientX,
          clientY: touch.clientY,
          preventDefault: () => {
          }
        };
        return this.handleMouseMove(mouseEvent);
      }
    },
    /**
     * Handle touch end event
     * @param {TouchEvent} e
     */
    handleTouchEnd(e) {
      if (!this.connected)
        return;
      e.preventDefault();
      this.updateActivity();
      const touch = e.changedTouches[0];
      const mouseEvent = {
        clientX: touch.clientX,
        clientY: touch.clientY,
        button: 0,
        preventDefault: () => {
        }
      };
      return this.handleMouseUp(mouseEvent);
    },
    /**
     * Attach input event listeners to canvas
     */
    attachInputListeners() {
      this.canvas.addEventListener("keydown", this.handleKeyDown, false);
      this.canvas.addEventListener("keyup", this.handleKeyUp, false);
      this.canvas.addEventListener("mousemove", this.handleMouseMove, false);
      this.canvas.addEventListener("mousedown", this.handleMouseDown, false);
      this.canvas.addEventListener("mouseup", this.handleMouseUp, false);
      this.canvas.addEventListener("wheel", this.handleWheel, false);
      this.canvas.addEventListener("contextmenu", (e) => {
        e.preventDefault();
        return false;
      });
      this.canvas.addEventListener("touchstart", this.handleTouchStart, { passive: false });
      this.canvas.addEventListener("touchmove", this.handleTouchMove, { passive: false });
      this.canvas.addEventListener("touchend", this.handleTouchEnd, { passive: false });
    },
    /**
     * Detach input event listeners from canvas
     */
    detachInputListeners() {
      this.canvas.removeEventListener("keydown", this.handleKeyDown);
      this.canvas.removeEventListener("keyup", this.handleKeyUp);
      this.canvas.removeEventListener("mousemove", this.handleMouseMove);
      this.canvas.removeEventListener("mousedown", this.handleMouseDown);
      this.canvas.removeEventListener("mouseup", this.handleMouseUp);
      this.canvas.removeEventListener("wheel", this.handleWheel);
      this.canvas.removeEventListener("touchstart", this.handleTouchStart);
      this.canvas.removeEventListener("touchmove", this.handleTouchMove);
      this.canvas.removeEventListener("touchend", this.handleTouchEnd);
      document.removeEventListener("mousemove", this.handleMouseMove);
      document.removeEventListener("mouseup", this.handleMouseUp);
    }
  };

  // wasm.js
  function isWASMSupported() {
    try {
      if (typeof WebAssembly === "object" && typeof WebAssembly.instantiate === "function") {
        const module = new WebAssembly.Module(
          new Uint8Array([0, 97, 115, 109, 1, 0, 0, 0])
        );
        return module instanceof WebAssembly.Module;
      }
    } catch (e) {
    }
    return false;
  }
  var WASMCodec = {
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
    async init(wasmPath = "js/rle/rle.wasm") {
      if (this.ready) {
        return true;
      }
      if (!this.supported) {
        this.initError = "WebAssembly not supported in this browser";
        Logger2.error("WASM", this.initError);
        return false;
      }
      try {
        if (typeof Go === "undefined") {
          this.initError = "Go class not found. Include wasm_exec.js before initializing.";
          Logger2.error("WASM", this.initError);
          return false;
        }
        this.goInstance = new Go();
        let result;
        if (typeof WebAssembly.instantiateStreaming === "function") {
          try {
            result = await WebAssembly.instantiateStreaming(
              fetch(wasmPath),
              this.goInstance.importObject
            );
          } catch (e) {
            Logger2.warn("WASM", "Streaming failed, using array buffer fallback");
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
        this.goInstance.run(this.wasmInstance);
        if (typeof goRLE === "undefined") {
          this.initError = "goRLE not initialized after running WASM";
          Logger2.error("WASM", this.initError);
          return false;
        }
        this.ready = true;
        this.initError = null;
        Logger2.debug("WASM", "Codec module initialized (RLE + RFX)");
        return true;
      } catch (error) {
        this.initError = error.message;
        Logger2.error("WASM", `Failed to initialize: ${error.message}`);
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
      return this.ready && typeof goRLE !== "undefined";
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
      if (!this.isReady())
        return false;
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
      if (!this.isReady())
        return;
      goRLE.flipVertical(data, width, height, bytesPerPixel);
    },
    /**
     * Convert RGB565 to RGBA
     * @param {Uint8Array} src
     * @param {Uint8Array} dst
     */
    rgb565toRGBA(src, dst) {
      if (!this.isReady())
        return;
      goRLE.rgb565toRGBA(src, dst);
    },
    /**
     * Convert BGR24 to RGBA
     * @param {Uint8Array} src
     * @param {Uint8Array} dst
     */
    bgr24toRGBA(src, dst) {
      if (!this.isReady())
        return;
      goRLE.bgr24toRGBA(src, dst);
    },
    /**
     * Convert BGRA32 to RGBA
     * @param {Uint8Array} src
     * @param {Uint8Array} dst
     */
    bgra32toRGBA(src, dst) {
      if (!this.isReady())
        return;
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
      if (!this.isReady())
        return false;
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
      if (!this.isReady())
        return false;
      return goRLE.decodeNSCodec(src, width, height, dst);
    },
    /**
     * Set palette colors
     * @param {Uint8Array} data - Palette data (RGB triples)
     * @param {number} numColors
     * @returns {boolean}
     */
    setPalette(data, numColors) {
      if (!this.isReady())
        return false;
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
      if (!this.isReady())
        return false;
      return goRLE.setRFXQuant(quantData);
    },
    /**
     * Decode a single RFX tile
     * @param {Uint8Array} tileData - Compressed tile data (CBT_TILE block)
     * @param {Uint8Array} outputBuffer - Output buffer (16384 bytes for 64x64 RGBA)
     * @returns {Object|null} { x, y, width, height } or null on error
     */
    decodeRFXTile(tileData, outputBuffer) {
      if (!this.isReady())
        return null;
      const result = goRLE.decodeRFXTile(tileData, outputBuffer);
      if (result === null || result === void 0) {
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
    RFX_TILE_RGBA_SIZE: 16384
  };
  var RFXDecoder = class {
    constructor() {
      this.tileBuffer = new Uint8Array(WASMCodec.RFX_TILE_RGBA_SIZE);
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
     * Decode a tile and render directly to canvas context
     * @param {Uint8Array} tileData - CBT_TILE block data
     * @param {CanvasRenderingContext2D} ctx - Canvas context
     * @returns {boolean}
     */
    decodeTileToCanvas(tileData, ctx) {
      const result = WASMCodec.decodeRFXTile(tileData, this.tileBuffer);
      if (!result) {
        return false;
      }
      const imageData = new ImageData(
        new Uint8ClampedArray(this.tileBuffer),
        // Creates a copy
        result.width,
        result.height
      );
      ctx.putImageData(imageData, result.x, result.y);
      return true;
    }
  };

  // codec-fallback.js
  var FallbackCodec = {
    palette: new Uint8Array(256 * 4),
    // RGBA palette
    // Pre-computed lookup tables for fast 5/6-bit to 8-bit expansion
    _lut5to8: null,
    _lut6to8: null,
    /**
     * Initialize lookup tables for fast color conversion
     * Call once at startup for best performance
     */
    init() {
      if (this._lut5to8)
        return;
      this._lut5to8 = new Uint8Array(32);
      for (let i = 0; i < 32; i++) {
        this._lut5to8[i] = i << 3 | i >> 2;
      }
      this._lut6to8 = new Uint8Array(64);
      for (let i = 0; i < 64; i++) {
        this._lut6to8[i] = i << 2 | i >> 4;
      }
      Logger2.debug("FallbackCodec", "Initialized color lookup tables");
    },
    /**
     * Check if WASM-free operation is recommended
     * @returns {boolean}
     */
    shouldUse16BitColor() {
      return true;
    },
    /**
     * Get recommended color depth for fallback mode
     * @returns {number}
     */
    getRecommendedColorDepth() {
      return 16;
    },
    /**
     * Set color palette for 8-bit mode
     * @param {Uint8Array} data - RGB palette data (3 bytes per color)
     * @param {number} numColors - Number of colors
     */
    setPalette(data, numColors) {
      const count = Math.min(numColors, 256);
      for (let i = 0; i < count; i++) {
        this.palette[i * 4] = data[i * 3];
        this.palette[i * 4 + 1] = data[i * 3 + 1];
        this.palette[i * 4 + 2] = data[i * 3 + 2];
        this.palette[i * 4 + 3] = 255;
      }
    },
    /**
     * Convert RGB565 to RGBA - OPTIMIZED for performance
     * This is the primary fast path for 16-bit fallback
     * @param {Uint8Array} src - Source RGB565 data (2 bytes per pixel, little-endian)
     * @param {Uint8Array} dst - Destination RGBA buffer (must be 4x pixel count)
     * @returns {boolean} True if conversion succeeded
     */
    rgb565ToRGBA(src, dst) {
      if (!this._lut5to8)
        this.init();
      const pixelCount = src.length >> 1;
      if (src.length === 0)
        return true;
      if (src.length < 2)
        return false;
      if (dst.length < pixelCount * 4) {
        Logger2.warn("FallbackCodec", `rgb565ToRGBA: dst buffer too small (${dst.length} < ${pixelCount * 4})`);
        return false;
      }
      const lut5 = this._lut5to8;
      const lut6 = this._lut6to8;
      const srcView = new DataView(src.buffer, src.byteOffset, src.byteLength);
      for (let i = 0; i < pixelCount; i++) {
        const pixel = srcView.getUint16(i << 1, true);
        const dstIdx = i << 2;
        dst[dstIdx] = lut5[pixel >> 11 & 31];
        dst[dstIdx + 1] = lut6[pixel >> 5 & 63];
        dst[dstIdx + 2] = lut5[pixel & 31];
        dst[dstIdx + 3] = 255;
      }
      return true;
    },
    /**
     * Convert RGB565 to RGBA - Ultra-fast version using 32-bit writes
     * @param {Uint8Array} src - Source RGB565 data
     * @param {Uint8Array} dst - Destination RGBA buffer (must be 4-byte aligned)
     * @returns {boolean} True if conversion succeeded
     */
    rgb565ToRGBA_Fast(src, dst) {
      if (!this._lut5to8)
        this.init();
      const pixelCount = src.length >> 1;
      if (src.length === 0)
        return true;
      if (src.length < 2)
        return false;
      if (dst.length < pixelCount * 4) {
        Logger2.warn("FallbackCodec", `rgb565ToRGBA_Fast: dst buffer too small`);
        return false;
      }
      const lut5 = this._lut5to8;
      const lut6 = this._lut6to8;
      const srcView = new DataView(src.buffer, src.byteOffset, src.byteLength);
      const dstView = new DataView(dst.buffer, dst.byteOffset, dst.byteLength);
      for (let i = 0; i < pixelCount; i++) {
        const pixel = srcView.getUint16(i << 1, true);
        const r = lut5[pixel >> 11 & 31];
        const g = lut6[pixel >> 5 & 63];
        const b = lut5[pixel & 31];
        dstView.setUint32(i << 2, 255 << 24 | b << 16 | g << 8 | r, true);
      }
      return true;
    },
    /**
     * Convert RGB555 to RGBA - OPTIMIZED
     * @param {Uint8Array} src - Source RGB555 data (2 bytes per pixel)
     * @param {Uint8Array} dst - Destination RGBA buffer
     * @returns {boolean} True if conversion succeeded
     */
    rgb555ToRGBA(src, dst) {
      if (!this._lut5to8)
        this.init();
      const pixelCount = src.length >> 1;
      if (src.length === 0)
        return true;
      if (src.length < 2)
        return false;
      if (dst.length < pixelCount * 4) {
        Logger2.warn("FallbackCodec", `rgb555ToRGBA: dst buffer too small`);
        return false;
      }
      const lut5 = this._lut5to8;
      const srcView = new DataView(src.buffer, src.byteOffset, src.byteLength);
      for (let i = 0; i < pixelCount; i++) {
        const pixel = srcView.getUint16(i << 1, true);
        const dstIdx = i << 2;
        dst[dstIdx] = lut5[pixel >> 10 & 31];
        dst[dstIdx + 1] = lut5[pixel >> 5 & 31];
        dst[dstIdx + 2] = lut5[pixel & 31];
        dst[dstIdx + 3] = 255;
      }
      return true;
    },
    /**
     * Convert 8-bit paletted to RGBA
     * @param {Uint8Array} src - Source palette indices
     * @param {Uint8Array} dst - Destination RGBA buffer
     * @returns {boolean} True if conversion succeeded
     */
    palette8ToRGBA(src, dst) {
      if (src.length === 0)
        return true;
      if (dst.length < src.length * 4) {
        Logger2.warn("FallbackCodec", `palette8ToRGBA: dst buffer too small`);
        return false;
      }
      const palette = this.palette;
      for (let i = 0, len = src.length; i < len; i++) {
        const idx = src[i] << 2;
        const dstIdx = i << 2;
        dst[dstIdx] = palette[idx];
        dst[dstIdx + 1] = palette[idx + 1];
        dst[dstIdx + 2] = palette[idx + 2];
        dst[dstIdx + 3] = palette[idx + 3];
      }
      return true;
    },
    /**
     * Convert BGR24 to RGBA
     * @param {Uint8Array} src - Source BGR data (3 bytes per pixel)
     * @param {Uint8Array} dst - Destination RGBA buffer
     * @returns {boolean} True if conversion succeeded
     */
    bgr24ToRGBA(src, dst) {
      const pixelCount = src.length / 3 | 0;
      if (src.length === 0)
        return true;
      if (src.length < 3)
        return false;
      if (dst.length < pixelCount * 4) {
        Logger2.warn("FallbackCodec", `bgr24ToRGBA: dst buffer too small`);
        return false;
      }
      for (let i = 0; i < pixelCount; i++) {
        const srcIdx = i * 3;
        const dstIdx = i << 2;
        dst[dstIdx] = src[srcIdx + 2];
        dst[dstIdx + 1] = src[srcIdx + 1];
        dst[dstIdx + 2] = src[srcIdx];
        dst[dstIdx + 3] = 255;
      }
      return true;
    },
    /**
     * Convert BGRA32 to RGBA - optimized with 32-bit operations
     * @param {Uint8Array} src - Source BGRA data (4 bytes per pixel)
     * @param {Uint8Array} dst - Destination RGBA buffer
     * @returns {boolean} True if conversion succeeded
     */
    bgra32ToRGBA(src, dst) {
      const pixelCount = src.length >> 2;
      if (src.length === 0)
        return true;
      if (src.length < 4)
        return false;
      if (dst.length < pixelCount * 4) {
        Logger2.warn("FallbackCodec", `bgra32ToRGBA: dst buffer too small`);
        return false;
      }
      const srcView = new DataView(src.buffer, src.byteOffset, src.byteLength);
      const dstView = new DataView(dst.buffer, dst.byteOffset, dst.byteLength);
      for (let i = 0; i < pixelCount; i++) {
        const offset = i << 2;
        const bgra = srcView.getUint32(offset, true);
        const b = bgra & 255;
        const g = bgra >> 8 & 255;
        const r = bgra >> 16 & 255;
        const a = bgra >> 24 & 255;
        dstView.setUint32(offset, a << 24 | b << 16 | g << 8 | r, true);
      }
      return true;
    },
    /**
     * Flip image vertically (in-place) - optimized
     * @param {Uint8Array} data - Image data buffer
     * @param {number} width - Image width in pixels
     * @param {number} height - Image height in pixels
     * @param {number} bytesPerPixel - Bytes per pixel (typically 4 for RGBA)
     * @returns {boolean} True if flip succeeded
     */
    flipVertical(data, width, height, bytesPerPixel) {
      if (width <= 0 || height <= 0 || bytesPerPixel <= 0)
        return false;
      const rowSize = width * bytesPerPixel;
      const expectedSize = rowSize * height;
      if (data.length < expectedSize) {
        Logger2.warn("FallbackCodec", `flipVertical: data buffer too small (${data.length} < ${expectedSize})`);
        return false;
      }
      if (height <= 1)
        return true;
      const temp = new Uint8Array(rowSize);
      const halfHeight = height >> 1;
      for (let y = 0; y < halfHeight; y++) {
        const topOffset = y * rowSize;
        const bottomOffset = (height - 1 - y) * rowSize;
        temp.set(data.subarray(topOffset, topOffset + rowSize));
        data.copyWithin(topOffset, bottomOffset, bottomOffset + rowSize);
        data.set(temp, bottomOffset);
      }
      return true;
    },
    /**
     * Process a bitmap with fallback codecs
     * Optimized for 16-bit uncompressed (fastest path)
     * 
     * @param {Uint8Array} src - Source data
     * @param {number} width - Image width
     * @param {number} height - Image height
     * @param {number} bpp - Bits per pixel
     * @param {boolean} isCompressed - Whether data is compressed
     * @param {Uint8Array} dst - Destination RGBA buffer
     * @returns {boolean}
     */
    processBitmap(src, width, height, bpp, isCompressed, dst) {
      try {
        if (!isCompressed && (bpp === 16 || bpp === 15)) {
          if (bpp === 16) {
            this.rgb565ToRGBA(src, dst);
          } else {
            this.rgb555ToRGBA(src, dst);
          }
          this.flipVertical(dst, width, height, 4);
          return true;
        }
        if (!isCompressed) {
          switch (bpp) {
            case 8:
              this.palette8ToRGBA(src, dst);
              break;
            case 24:
              this.bgr24ToRGBA(src, dst);
              break;
            case 32:
              this.bgra32ToRGBA(src, dst);
              break;
            default:
              Logger2.warn("FallbackCodec", `Unsupported uncompressed bpp: ${bpp}`);
              return false;
          }
          this.flipVertical(dst, width, height, 4);
          return true;
        }
        Logger2.debug("FallbackCodec", `Compressed ${bpp}bpp not optimized in JS fallback`);
        return false;
      } catch (e) {
        Logger2.error("FallbackCodec", `Processing failed: ${e.message}`);
        return false;
      }
    }
  };
  FallbackCodec.init();

  // graphics.js
  var GraphicsMixin = {
    /**
     * Initialize graphics subsystem
     */
    initGraphics() {
      this.canvasShown = false;
      this.pointerCache = {};
      this.bitmapCacheEnabled = true;
      this.bitmapCache = /* @__PURE__ */ new Map();
      this.bitmapCacheMaxSize = 1e3;
      this.bitmapCacheHits = 0;
      this.bitmapCacheMisses = 0;
      this.originalWidth = 0;
      this.originalHeight = 0;
      this.resizeTimeout = null;
      this.rfxDecoder = new RFXDecoder();
      this._wasmErrorShown = false;
      this._usingFallback = false;
      this._fallbackWarningShown = false;
      this._capabilitiesLogged = false;
      this.handleResize = this.handleResize.bind(this);
      if (!WASMCodec.isSupported()) {
        Logger2.debug("Graphics", "WebAssembly not available - using JS fallback");
      }
    },
    /**
     * Log client capabilities to the console
     * Called once upon connection
     */
    logCapabilities() {
      if (this._capabilitiesLogged)
        return;
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
      const codecList = [];
      if (caps.codecs.rfx)
        codecList.push("RemoteFX");
      if (caps.codecs.rle)
        codecList.push("RLE");
      if (caps.codecs.nscodec)
        codecList.push("NSCodec");
      if (caps.codecs.fallback)
        codecList.push("JS-Fallback");
      console.info(
        "%c[RDP Client] Capabilities",
        "color: #4CAF50; font-weight: bold",
        "\n  WASM:",
        wasmReady ? "\u2713 loaded" : wasmSupported ? "\u2717 failed" : "\u2717 unsupported",
        "\n  Codecs:",
        codecList.join(", "),
        "\n  Display:",
        `${caps.display.width}\xD7${caps.display.height}`,
        "\n  Color:",
        `${caps.display.colorDepth}bpp`,
        wasmError ? `
  Error: ${wasmError}` : ""
      );
      if (this.emitEvent) {
        this.emitEvent("capabilities", caps);
      }
      return caps;
    },
    /**
     * Get recommended color depth based on codec availability
     * @returns {number} Recommended bits per pixel (16 or 32)
     */
    getRecommendedColorDepth() {
      if (WASMCodec.isReady()) {
        return 32;
      }
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
        Logger2.warn("Palette", `Invalid color count: ${numberColors}`);
        return;
      }
      Logger2.debug("Palette", `Received ${numberColors} colors`);
      const paletteData = r.blob(numberColors * 3);
      const paletteArray = new Uint8Array(paletteData);
      if (WASMCodec.isReady()) {
        WASMCodec.setPalette(paletteArray, numberColors);
        Logger2.debug("Palette", "Updated via WASM");
      }
      FallbackCodec.setPalette(paletteArray, numberColors);
      Logger2.debug("Palette", "Updated in fallback codec");
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
        this.showUserSuccess("Multi-monitor environment detected");
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
      const srcData = new Uint8Array(bitmapData.bitmapDataStream);
      if (WASMCodec.isReady()) {
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
        Logger2.debug("Bitmap", `WASM processBitmap failed, trying fallback`);
      }
      if (!this._usingFallback) {
        this._usingFallback = true;
        const reason = WASMCodec.isReady() ? "WASM decode failed" : WASMCodec.getInitError() || "WASM not loaded";
        Logger2.warn("Bitmap", `Using JavaScript fallback codec (${reason})`);
      }
      const fallbackResult = FallbackCodec.processBitmap(srcData, width, height, bpp, isCompressed, rgba);
      if (fallbackResult) {
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
      if (!this._wasmErrorShown) {
        this._wasmErrorShown = true;
        Logger2.error("Bitmap", `Cannot decode: bpp=${bpp}, compressed=${isCompressed}`);
        if (isCompressed && bpp !== 8) {
          this.showUserError("Some compressed graphics cannot be displayed. Try reducing color depth.");
        }
      }
    },
    /**
     * Initialize bitmap cache
     */
    initBitmapCache() {
      this.bitmapCacheEnabled = true;
      this.bitmapCache = /* @__PURE__ */ new Map();
      this.bitmapCacheMaxSize = 1e3;
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
        hash = (hash << 5) - hash + sample[i];
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
        hitRate: this.bitmapCacheHits && this.bitmapCacheMisses ? (this.bitmapCacheHits / (this.bitmapCacheHits + this.bitmapCacheMisses) * 100).toFixed(1) + "%" : "N/A"
      };
    },
    /**
     * Handle pointer/cursor update
     * @param {Object} header
     * @param {Reader} r
     */
    handlePointer(header, r) {
      try {
        Logger2.debug("Cursor", `Update type: ${header.updateCode}`);
        if (header.isPTRNull()) {
          Logger2.debug("Cursor", "Hidden");
          this.canvas.className = "pointer-cache-null";
          return;
        }
        if (header.isPTRDefault()) {
          Logger2.debug("Cursor", "Default");
          this.canvas.className = "pointer-cache-default";
          return;
        }
        if (header.isPTRColor()) {
          return;
        }
        if (header.isPTRNew()) {
          const newPointerUpdate = parseNewPointerUpdate(r);
          Logger2.debug("Cursor", `New cursor: cache=${newPointerUpdate.cacheIndex}, hotspot=(${newPointerUpdate.x},${newPointerUpdate.y}), size=${newPointerUpdate.width}x${newPointerUpdate.height}`);
          this.pointerCacheCanvasCtx.putImageData(newPointerUpdate.getImageData(this.pointerCacheCanvasCtx), 0, 0);
          const url = this.pointerCacheCanvas.toDataURL("image/png");
          if (this.pointerCache.hasOwnProperty(newPointerUpdate.cacheIndex)) {
            document.getElementsByTagName("head")[0].removeChild(this.pointerCache[newPointerUpdate.cacheIndex]);
            delete this.pointerCache[newPointerUpdate.cacheIndex];
          }
          const style = document.createElement("style");
          const className = "pointer-cache-" + newPointerUpdate.cacheIndex;
          style.innerHTML = "." + className + ' {cursor:url("' + url + '") ' + newPointerUpdate.x + " " + newPointerUpdate.y + ", auto !important}";
          document.getElementsByTagName("head")[0].appendChild(style);
          this.pointerCache[newPointerUpdate.cacheIndex] = style;
          this.canvas.className = className;
          return;
        }
        if (header.isPTRCached()) {
          const cacheIndex = r.uint16(true);
          Logger2.debug("Cursor", `Cached index: ${cacheIndex}`);
          const className = "pointer-cache-" + cacheIndex;
          this.canvas.className = className;
          return;
        }
        if (header.isPTRPosition()) {
          Logger2.debug("Cursor", "Position update (ignored)");
          return;
        }
        Logger2.debug("Cursor", "Unknown pointer type");
      } catch (error) {
        Logger2.error("Cursor", `Error: ${error.message}`);
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
          Logger2.debug("Resize", `${newWidth}x${newHeight}, reconnecting...`);
          this.showUserInfo("Resizing desktop...");
          this.reconnectWithNewSize(newWidth, newHeight);
        }
      }, 500);
    },
    /**
     * Show the canvas (hide login form)
     */
    showCanvas() {
      Logger2.debug("Connection", "First bitmap received - session active");
      const loginForm = document.getElementById("login-form");
      const canvasContainer = document.getElementById("canvas-container");
      if (loginForm) {
        loginForm.style.display = "none";
      }
      if (canvasContainer) {
        canvasContainer.style.display = "block";
      }
      this.canvas.tabIndex = 1e3;
      this.canvas.focus();
    },
    /**
     * Hide the canvas (show login form)
     */
    hideCanvas() {
      const loginForm = document.getElementById("login-form");
      const canvasContainer = document.getElementById("canvas-container");
      if (canvasContainer) {
        canvasContainer.style.display = "none";
      }
      if (loginForm) {
        loginForm.style.display = "block";
      }
    },
    /**
     * Clear the canvas
     */
    clearCanvas() {
      this.ctx.clearRect(0, 0, this.canvas.width, this.canvas.height);
      this.canvas.className = "";
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
        Logger2.error("RFX", "WASM not loaded");
        return;
      }
      Logger2.debug("RFX", `Surface command received, ${data.length} bytes`);
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
        Logger2.warn("RFX", `Decoded ${decoded} tiles, ${failed} failed`);
      } else {
        Logger2.debug("RFX", `Decoded ${decoded} tiles`);
      }
    }
  };

  // clipboard.js
  var ClipboardMixin = {
    /**
     * Initialize clipboard support
     */
    initClipboardSupport() {
      this.clipboardApiSupported = !!(navigator.clipboard && navigator.clipboard.writeText);
      Logger2.debug("Clipboard", `API supported: ${this.clipboardApiSupported}`);
    },
    /**
     * Type text to remote by sending key events
     * This simulates typing the text character by character
     * @param {string} text
     */
    typeTextToRemote(text) {
      if (!this.connected || !text)
        return;
      const maxLength = 4096;
      if (text.length > maxLength) {
        text = text.substring(0, maxLength);
        this.showUserWarning("Text truncated to " + maxLength + " characters");
      }
      Logger2.debug("Clipboard", `Typing ${text.length} chars to remote`);
      let index = 0;
      const typeNext = () => {
        if (index >= text.length || !this.connected)
          return;
        const char = text[index];
        this.sendCharacter(char);
        index++;
        if (index < text.length) {
          setTimeout(typeNext, 10);
        }
      };
      typeNext();
    },
    /**
     * Send a single character as key press
     * @param {string} char
     */
    sendCharacter(char) {
      const code = this.charToKeyCode(char);
      if (!code) {
        Logger2.debug("Clipboard", `No mapping for char code: ${char.charCodeAt(0)}`);
        return;
      }
      const needsShift = this.charNeedsShift(char);
      if (needsShift) {
        const shiftDown = new KeyboardEventKeyDown("ShiftLeft");
        if (shiftDown.keyCode !== void 0) {
          this.queueInput(shiftDown.serialize(), false);
        }
      }
      const keyDown = new KeyboardEventKeyDown(code);
      const keyUp = new KeyboardEventKeyUp(code);
      if (keyDown.keyCode !== void 0) {
        this.queueInput(keyDown.serialize(), false);
        this.queueInput(keyUp.serialize(), false);
      }
      if (needsShift) {
        const shiftUp = new KeyboardEventKeyUp("ShiftLeft");
        if (shiftUp.keyCode !== void 0) {
          this.queueInput(shiftUp.serialize(), false);
        }
      }
    },
    /**
     * Check if character needs shift key
     * @param {string} char
     * @returns {boolean}
     */
    charNeedsShift(char) {
      if (char >= "A" && char <= "Z")
        return true;
      const shiftedChars = '!@#$%^&*()_+{}|:"<>?~';
      return shiftedChars.includes(char);
    },
    /**
     * Map character to JavaScript key code
     * @param {string} char
     * @returns {string|null}
     */
    charToKeyCode(char) {
      const charCode = char.charCodeAt(0);
      if (charCode >= 65 && charCode <= 90 || charCode >= 97 && charCode <= 122) {
        return "Key" + char.toUpperCase();
      }
      if (charCode >= 48 && charCode <= 57) {
        return "Digit" + char;
      }
      const specialMap = {
        " ": "Space",
        "\n": "Enter",
        "\r": "Enter",
        "	": "Tab",
        ".": "Period",
        ",": "Comma",
        ";": "Semicolon",
        ":": "Semicolon",
        // Shift+;
        "'": "Quote",
        '"': "Quote",
        // Shift+'
        "/": "Slash",
        "?": "Slash",
        // Shift+/
        "\\": "Backslash",
        "|": "Backslash",
        // Shift+\
        "[": "BracketLeft",
        "{": "BracketLeft",
        // Shift+[
        "]": "BracketRight",
        "}": "BracketRight",
        // Shift+]
        "-": "Minus",
        "_": "Minus",
        // Shift+-
        "=": "Equal",
        "+": "Equal",
        // Shift+=
        "`": "Backquote",
        "~": "Backquote",
        // Shift+`
        "!": "Digit1",
        // Shift+1
        "@": "Digit2",
        // Shift+2
        "#": "Digit3",
        // Shift+3
        "$": "Digit4",
        // Shift+4
        "%": "Digit5",
        // Shift+5
        "^": "Digit6",
        // Shift+6
        "&": "Digit7",
        // Shift+7
        "*": "Digit8",
        // Shift+8
        "(": "Digit9",
        // Shift+9
        ")": "Digit0",
        // Shift+0
        "<": "Comma",
        // Shift+,
        ">": "Period"
        // Shift+.
      };
      return specialMap[char] || null;
    },
    /**
     * Copy text to local clipboard (for UI use)
     * @param {string} text
     */
    async copyToLocalClipboard(text) {
      if (!this.clipboardApiSupported)
        return false;
      try {
        await navigator.clipboard.writeText(text);
        return true;
      } catch (err) {
        Logger2.warn("Clipboard", `Write failed: ${err.message}`);
        return false;
      }
    }
  };

  // ui.js
  var UIMixin = {
    /**
     * Initialize UI state
     */
    initUI() {
      this.csrfToken = null;
    },
    /**
     * Sanitize user input
     * @param {string} input
     * @returns {string}
     */
    sanitizeInput(input) {
      if (!input)
        return "";
      return input.replace(/[<>'"&]/g, "").trim();
    },
    /**
     * Validate hostname format
     * @param {string} hostname
     * @returns {boolean}
     */
    validateHostname(hostname) {
      if (!hostname)
        return false;
      const pattern = /^([a-zA-Z0-9.-]+|(\d{1,3}\.){3}\d{1,3})(:\d{1,5})?$/;
      return pattern.test(hostname) && hostname.length <= 253;
    },
    /**
     * Generate CSRF token
     * @returns {string}
     */
    generateCSRFToken() {
      return crypto.getRandomValues(new Uint8Array(16)).reduce((hex, byte) => hex + byte.toString(16).padStart(2, "0"), "");
    },
    /**
     * Show error message to user
     * @param {string} message
     */
    showUserError(message) {
      if (window.showToast) {
        window.showToast(message, "error", "Connection Error", 8e3);
      }
      const status = document.getElementById("status");
      if (status) {
        status.style.display = "block";
        status.className = "status-indicator status-disconnected";
        status.textContent = message;
        setTimeout(() => {
          if (status.textContent === message) {
            status.style.display = "none";
          }
        }, 1e4);
      }
    },
    /**
     * Show success message to user
     * @param {string} message
     */
    showUserSuccess(message) {
      if (window.showToast) {
        window.showToast(message, "success", "Success", 5e3);
      }
      const status = document.getElementById("status");
      if (status) {
        status.style.display = "block";
        status.className = "status-indicator status-connected";
        status.textContent = message;
        setTimeout(() => {
          if (status.textContent === message) {
            status.style.display = "none";
          }
        }, 5e3);
      }
    },
    /**
     * Show warning message to user
     * @param {string} message
     */
    showUserWarning(message) {
      if (window.showToast) {
        window.showToast(message, "info", "Warning", 6e3);
      }
    },
    /**
     * Show info message to user
     * @param {string} message
     */
    showUserInfo(message) {
      if (window.showToast) {
        window.showToast(message, "info", "Info", 4e3);
      }
    },
    /**
     * Log error with context
     * @param {string} context
     * @param {Object} details
     */
    logError(context, details) {
      Logger2.error("RDP", `${context}:`, details);
    },
    /**
     * Emit custom event
     * @param {string} name
     * @param {Object} detail
     */
    emitEvent(name, detail = {}) {
      try {
        document.dispatchEvent(new CustomEvent("rdp:" + name, { detail }));
      } catch (error) {
        Logger2.debug("Event", `Dispatch failed: ${error.message}`);
      }
    }
  };

  // audio.js
  var AudioMixin = {
    initAudio() {
      this.audioContext = null;
      this.audioEnabled = false;
      this.audioFormat = null;
      this.audioQueue = [];
      this.audioPlaying = false;
      this.audioGain = null;
      this.audioVolume = 1;
      this.audioBufferSize = 4096;
      this.audioSampleRate = 44100;
      this.audioChannels = 2;
      this.audioBitsPerSample = 16;
    },
    enableAudio() {
      if (this.audioContext) {
        return;
      }
      try {
        this.audioContext = new (window.AudioContext || window.webkitAudioContext)();
        this.audioGain = this.audioContext.createGain();
        this.audioGain.connect(this.audioContext.destination);
        this.audioGain.gain.value = this.audioVolume;
        this.audioEnabled = true;
        Logger.debug("Audio", `Initialized: ${this.audioContext.sampleRate}Hz`);
      } catch (e) {
        Logger.error("Audio", `Failed to initialize: ${e.message}`);
        this.audioEnabled = false;
      }
    },
    disableAudio() {
      if (this.audioContext) {
        this.audioContext.close();
        this.audioContext = null;
      }
      this.audioEnabled = false;
      this.audioQueue = [];
      Logger.debug("Audio", "Disabled");
    },
    setAudioVolume(volume) {
      this.audioVolume = Math.max(0, Math.min(1, volume));
      if (this.audioGain) {
        this.audioGain.gain.value = this.audioVolume;
      }
    },
    handleAudioMessage(data) {
      if (!this.audioEnabled || !this.audioContext) {
        return;
      }
      if (data.length < 4) {
        return;
      }
      const view = new DataView(data.buffer, data.byteOffset, data.byteLength);
      const msgType = data[1];
      const timestamp = view.getUint16(2, true);
      let offset = 4;
      if (msgType === 2 && data.length >= 12) {
        const channels = view.getUint16(4, true);
        const sampleRate = view.getUint32(6, true);
        const bitsPerSample = view.getUint16(10, true);
        if (channels < 1 || channels > 8) {
          Logger.warn("Audio", `Invalid channel count: ${channels}`);
          return;
        }
        if (sampleRate < 8e3 || sampleRate > 192e3) {
          Logger.warn("Audio", `Invalid sample rate: ${sampleRate}`);
          return;
        }
        if (bitsPerSample !== 8 && bitsPerSample !== 16) {
          Logger.warn("Audio", `Unsupported bit depth: ${bitsPerSample}`);
          return;
        }
        this.audioChannels = channels;
        this.audioSampleRate = sampleRate;
        this.audioBitsPerSample = bitsPerSample;
        offset = 12;
        Logger.debug("Audio", `Format: ${this.audioSampleRate}Hz ${this.audioChannels}ch ${this.audioBitsPerSample}bit`);
      }
      const pcmData = data.slice(offset);
      if (pcmData.length === 0) {
        return;
      }
      this.queueAudio(pcmData, timestamp);
    },
    queueAudio(pcmData, timestamp) {
      const samples = this.pcmToFloat32(pcmData);
      if (!samples || samples.length === 0) {
        return;
      }
      const frameCount = Math.floor(samples.length / this.audioChannels);
      if (frameCount === 0) {
        return;
      }
      try {
        const audioBuffer = this.audioContext.createBuffer(
          this.audioChannels,
          frameCount,
          this.audioSampleRate
        );
        for (let channel = 0; channel < this.audioChannels; channel++) {
          const channelData = audioBuffer.getChannelData(channel);
          for (let i = 0; i < frameCount; i++) {
            channelData[i] = samples[i * this.audioChannels + channel];
          }
        }
        this.audioQueue.push({
          buffer: audioBuffer,
          timestamp
        });
        if (!this.audioPlaying) {
          this.playNextAudio();
        }
      } catch (e) {
        Logger.error("Audio", `Buffer creation failed: ${e.message}`);
      }
    },
    pcmToFloat32(pcmData) {
      const view = new DataView(pcmData.buffer, pcmData.byteOffset, pcmData.byteLength);
      const samples = [];
      if (this.audioBitsPerSample === 16) {
        const sampleCount = Math.floor(pcmData.length / 2);
        for (let i = 0; i < sampleCount; i++) {
          const sample = view.getInt16(i * 2, true);
          samples.push(sample / 32768);
        }
      } else if (this.audioBitsPerSample === 8) {
        for (let i = 0; i < pcmData.length; i++) {
          const sample = pcmData[i];
          samples.push((sample - 128) / 128);
        }
      } else {
        Logger.warn("Audio", `Unsupported bit depth: ${this.audioBitsPerSample}`);
        return null;
      }
      return samples;
    },
    playNextAudio() {
      if (this.audioQueue.length === 0) {
        this.audioPlaying = false;
        return;
      }
      this.audioPlaying = true;
      const item = this.audioQueue.shift();
      const source = this.audioContext.createBufferSource();
      source.buffer = item.buffer;
      source.connect(this.audioGain);
      source.onended = () => {
        this.playNextAudio();
      };
      const startTime = this.audioContext.currentTime;
      source.start(startTime);
    },
    // Resume audio context after user interaction (required by browsers)
    resumeAudioContext() {
      if (this.audioContext && this.audioContext.state === "suspended") {
        this.audioContext.resume().then(() => {
          Logger.debug("Audio", "Context resumed");
        });
      }
    }
  };
  var audio_default = AudioMixin;

  // client.js
  function applyMixin(mixin) {
    Object.keys(mixin).forEach((key) => {
      if (typeof mixin[key] === "function") {
        Client.prototype[key] = mixin[key];
      }
    });
  }
  function Client(websocketURL, canvasID, hostID, userID, passwordID) {
    this.websocketURL = websocketURL;
    this.canvas = document.getElementById(canvasID);
    this.hostEl = document.getElementById(hostID);
    this.userEl = document.getElementById(userID);
    this.passwordEl = document.getElementById(passwordID);
    this.ctx = this.canvas.getContext("2d");
    this.pointerCacheCanvas = document.getElementById("pointer-cache");
    this.pointerCacheCanvasCtx = this.pointerCacheCanvas.getContext("2d");
    this.connected = false;
    this.socket = null;
    this.initSession();
    this.initInput();
    this.initGraphics();
    this.initUI();
    this.initAudio();
    this.initialize = this.initialize.bind(this);
    this.handleMessage = this.handleMessage.bind(this);
    this.deinitialize = this.deinitialize.bind(this);
    this.loadSession();
    if (this.shouldAutoReconnect()) {
      this.scheduleReconnect(100);
    }
  }
  applyMixin(SessionMixin);
  applyMixin(InputMixin);
  applyMixin(GraphicsMixin);
  applyMixin(ClipboardMixin);
  applyMixin(UIMixin);
  applyMixin(audio_default);
  Client.prototype.connect = function() {
    if (this.socket && this.socket.readyState === WebSocket.OPEN) {
      Logger2.warn("Connection", "Already established");
      return;
    }
    const host = this.sanitizeInput(this.hostEl.value);
    const user = this.sanitizeInput(this.userEl.value);
    const password = this.passwordEl.value;
    if (!this.validateHostname(host)) {
      Logger2.error("Connection", "Invalid hostname format");
      return;
    }
    this.reconnectAttempts = 0;
    this.manualDisconnect = false;
    this.lastConnectionTime = Date.now();
    this.csrfToken = this.generateCSRFToken();
    const screenWidth = window.innerWidth;
    const screenHeight = window.innerHeight;
    this.originalWidth = screenWidth;
    this.originalHeight = screenHeight;
    this.canvas.width = screenWidth;
    this.canvas.height = screenHeight;
    const colorDepthEl = document.getElementById("colorDepth");
    const colorDepth = colorDepthEl ? colorDepthEl.value : "16";
    const disableNLAEl = document.getElementById("disableNLA");
    const disableNLA = disableNLAEl ? disableNLAEl.checked : false;
    Logger2.debug("Connection", `Connecting to ${host} as ${user} (${screenWidth}x${screenHeight}, ${colorDepth}bpp)`);
    const url = new URL(this.websocketURL);
    url.searchParams.set("width", screenWidth);
    url.searchParams.set("height", screenHeight);
    url.searchParams.set("colorDepth", colorDepth);
    if (disableNLA) {
      url.searchParams.set("disableNLA", "true");
      Logger2.debug("Connection", "NLA disabled");
    }
    url.searchParams.set("audio", "true");
    this.enableAudio();
    Logger2.debug("Audio", "Audio redirection enabled");
    this._pendingCredentials = { host, user, password };
    this.socket = new WebSocket(url.toString());
    const pendingCreds = this._pendingCredentials;
    this.socket.onopen = () => {
      Logger2.debug("Connection", "WebSocket opened, sending credentials");
      if (pendingCreds) {
        const credMsg = JSON.stringify({
          type: "credentials",
          host: pendingCreds.host,
          user: pendingCreds.user,
          password: pendingCreds.password
        });
        this.socket.send(credMsg);
        this._pendingCredentials = null;
      }
      this.initialize();
    };
    this.socket.onmessage = (e) => {
      e.data.arrayBuffer().then((arrayBuffer) => this.handleMessage(arrayBuffer)).catch((err) => Logger2.error("Message", `Failed to read message: ${err.message}`));
    };
    this.socket.onerror = (e) => {
      const errorMsg = e.message || "";
      this.logError("WebSocket connection error", { error: errorMsg, code: e.code });
      if (errorMsg.includes("401") || errorMsg.includes("Unauthorized")) {
        this.showUserError("Authentication failed: Invalid username or password");
      } else if (errorMsg.includes("403") || errorMsg.includes("Forbidden")) {
        this.showUserError("Access denied: You do not have permission to connect");
      } else if (errorMsg.includes("404") || errorMsg.includes("Not Found")) {
        this.showUserError("Server not found: Check the server address");
      } else if (errorMsg.includes("timeout")) {
        this.showUserError("Connection timeout: Server is not responding");
      } else {
        this.showUserError("Connection failed: Unable to connect to server");
      }
      this.emitEvent("error", { message: errorMsg, code: e.code });
    };
    this.socket.onclose = (e) => {
      Logger2.debug("Connection", `WebSocket closed (code=${e.code}, reason=${e.reason || "none"})`);
      this.emitEvent("disconnected", {
        code: e.code,
        reason: e.reason,
        wasClean: e.wasClean,
        manual: this.manualDisconnect
      });
      if (this.manualDisconnect) {
        this.showUserSuccess("Disconnected successfully");
        return;
      }
      if (e.code === 1e3) {
        return;
      } else if (e.code === 1001) {
        this.showUserError("Connection closed: Going away");
      } else if (e.code === 1002) {
        this.showUserError("Connection closed: Protocol error");
      } else if (e.code === 1003) {
        this.showUserError("Connection closed: Unsupported data type");
      } else if (e.code === 1006) {
        this.showUserError("Connection closed abnormally: Check your network connection");
      } else if (e.code === 1015) {
        this.showUserError("TLS handshake failed: Certificate validation error");
      } else {
        this.showUserError(`Connection lost (code: ${e.code})`);
      }
      if (!this.manualDisconnect && this.reconnectAttempts < this.maxReconnectAttempts) {
        const exponent = Math.max(0, this.reconnectAttempts - 1);
        const exponentialDelay = this.reconnectDelay * Math.pow(2, exponent);
        this.scheduleReconnect(Math.min(exponentialDelay, 3e4));
      }
      this.deinitialize();
    };
    this.saveSession();
  };
  Client.prototype.sendAuthentication = function() {
    if (!this.socket || this.socket.readyState !== WebSocket.OPEN) {
      this.showUserError("Connection lost during authentication");
      return;
    }
    try {
      const authData = {
        type: "auth",
        user: this.sanitizeInput(this.userEl.value),
        password: this.passwordEl.value,
        host: this.sanitizeInput(this.hostEl.value),
        sessionId: this.sessionId,
        csrfToken: this.csrfToken,
        timestamp: Date.now()
      };
      this.socket.send(JSON.stringify(authData));
      Logger2.debug("Connection", "Authentication data sent");
    } catch (error) {
      this.showUserError("Failed to send authentication data");
      Logger2.error("Connection", "Authentication send error:", error);
    }
  };
  Client.prototype.initialize = function() {
    if (this.connected) {
      return;
    }
    this.reconnectAttempts = 0;
    this.lastConnectionTime = Date.now();
    window.addEventListener("keydown", this.handleKeyDown);
    window.addEventListener("keyup", this.handleKeyUp);
    this.canvas.addEventListener("mousemove", this.handleMouseMove);
    this.canvas.addEventListener("mousedown", this.handleMouseDown);
    this.canvas.addEventListener("mouseup", this.handleMouseUp);
    this.canvas.addEventListener("contextmenu", this.handleMouseUp);
    this.canvas.addEventListener("click", () => {
      this.canvas.focus();
      this.resumeAudioContext();
    });
    this.canvas.addEventListener("wheel", this.handleWheel);
    this.canvas.addEventListener("touchstart", this.handleTouchStart, { passive: false });
    this.canvas.addEventListener("touchmove", this.handleTouchMove, { passive: false });
    this.canvas.addEventListener("touchend", this.handleTouchEnd, { passive: false });
    window.addEventListener("resize", this.handleResize);
    this.connected = true;
    this.logCapabilities();
  };
  Client.prototype.showCanvas = function() {
    const canvasContainer = document.getElementById("canvas-container");
    const formContainer = document.querySelector(".container");
    if (formContainer) {
      formContainer.style.display = "none";
    }
    if (canvasContainer) {
      canvasContainer.style.cssText = "display: block !important; position: fixed !important; top: 0 !important; left: 0 !important; width: 100vw !important; height: 100vh !important; z-index: 9999 !important; background: #000 !important;";
      void canvasContainer.offsetHeight;
    }
    this.canvas.style.cssText = "display: block !important; visibility: visible !important; opacity: 1 !important; position: absolute !important; top: 0 !important; left: 0 !important; width: " + this.canvas.width + "px !important; height: " + this.canvas.height + "px !important;";
    this.canvas.setAttribute("tabindex", "0");
    this.canvas.style.outline = "none";
    this.canvas.focus();
    void this.canvas.offsetHeight;
    this.initBitmapCache();
    this.initClipboardSupport();
    this.startTimeoutTracking();
    this.emitEvent("connected", {
      host: this.sanitizeInput(this.hostEl.value),
      user: this.sanitizeInput(this.userEl.value)
    });
  };
  Client.prototype.deinitialize = function() {
    this.connected = false;
    this.canvasShown = false;
    window.removeEventListener("keydown", this.handleKeyDown);
    window.removeEventListener("keyup", this.handleKeyUp);
    this.canvas.removeEventListener("mousemove", this.handleMouseMove);
    this.canvas.removeEventListener("mousedown", this.handleMouseDown);
    this.canvas.removeEventListener("mouseup", this.handleMouseUp);
    this.canvas.removeEventListener("contextmenu", this.handleMouseUp);
    this.canvas.removeEventListener("wheel", this.handleWheel);
    this.canvas.removeEventListener("touchstart", this.handleTouchStart);
    this.canvas.removeEventListener("touchmove", this.handleTouchMove);
    this.canvas.removeEventListener("touchend", this.handleTouchEnd);
    window.removeEventListener("resize", this.handleResize);
    this.clearAllTimeouts();
    this.clearBitmapCache();
    this.disableAudio();
    Object.entries(this.pointerCache).forEach(([index, style]) => {
      try {
        if (style && style.parentNode) {
          style.parentNode.removeChild(style);
        }
      } catch (e) {
      }
    });
    this.pointerCache = {};
    this.canvas.classList = [];
    this.ctx.clearRect(0, 0, this.canvas.width, this.canvas.height);
  };
  Client.prototype.handleMessage = function(arrayBuffer) {
    if (!this.connected) {
      return;
    }
    const firstByte = new Uint8Array(arrayBuffer)[0];
    if (firstByte === 254 && this.audioEnabled) {
      this.handleAudioMessage(new Uint8Array(arrayBuffer));
      return;
    }
    if (firstByte === 255) {
      if (arrayBuffer.byteLength > 1024 * 1024) {
        Logger2.warn("Message", "JSON message too large, ignoring");
        return;
      }
      try {
        const jsonData = arrayBuffer.slice(1);
        const text = new TextDecoder().decode(jsonData);
        const message = JSON.parse(text);
        if (message.type === "capabilities") {
          if (message.logLevel) {
            Logger2.setLevel(message.logLevel);
          }
          Logger2.debug("Capabilities", `Server: codecs=${message.codecs?.join(",") || "none"}, colorDepth=${message.colorDepth}, desktop=${message.desktopSize}`);
          this.serverCapabilities = message;
        } else if (message.type === "error") {
          this.showUserError(message.message);
          this.emitEvent("error", { message: message.message });
        }
        return;
      } catch (e) {
        Logger2.warn("Message", `Failed to parse 0xFF message: ${e.message}`);
      }
    }
    if (arrayBuffer.byteLength > 1024 * 1024) {
      this.handleBitmapUpdate(new Uint8Array(arrayBuffer));
      return;
    }
    try {
      const text = new TextDecoder().decode(arrayBuffer);
      const message = JSON.parse(text);
      if (message.type === "clipboard_response") {
        Logger2.debug("Clipboard", "Received remote clipboard data");
        this.handleRemoteClipboard(message.data);
        return;
      }
      if (message.type === "file_transfer_status") {
        this.updateFileStatus(message.message);
        return;
      }
      if (message.type === "error") {
        this.showUserError(message.message);
        this.emitEvent("error", { message: message.message });
        return;
      }
    } catch (e) {
    }
    const r = new BinaryReader(arrayBuffer);
    const header = parseUpdateHeader(r);
    Logger2.debug("Update", `code=${header.updateCode}, pointer=${header.isPointer()}`);
    if (header.isCompressed()) {
      Logger2.warn("Update", "Compression not supported");
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
      Logger2.debug("Update", "Synchronize received");
      return;
    }
    if (header.isPalette()) {
      this.handlePalette(r);
      return;
    }
    if (header.updateCode === 15) {
      return;
    }
    Logger2.debug("Update", `Unknown update code: ${header.updateCode}`);
  };
  Client.prototype.disconnect = function() {
    if (!this.socket) {
      return;
    }
    this.manualDisconnect = true;
    this.clearSession();
    if (this.reconnectTimeout) {
      clearTimeout(this.reconnectTimeout);
      this.reconnectTimeout = null;
    }
    this.deinitialize();
    this.socket.close(1e3);
  };
  Client.prototype.reconnectWithNewSize = function(width, height) {
    this.originalWidth = width;
    this.originalHeight = height;
    if (this.socket) {
      this.manualDisconnect = true;
      this.socket.close(1e3);
    }
    setTimeout(() => {
      this.manualDisconnect = false;
      this.connect();
    }, 500);
  };

  // index.js
  if (typeof window !== "undefined") {
    window.Client = Client;
    window.Logger = Logger2;
    window.WASMCodec = WASMCodec;
    window.RFXDecoder = RFXDecoder;
    window.FallbackCodec = FallbackCodec;
    window.isWASMSupported = isWASMSupported;
  }
  var src_default = Client;
  return __toCommonJS(src_exports);
})();

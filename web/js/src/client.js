/**
 * RDP Web Client - Main module
 * Combines all modules into the Client class
 * @module client
 */

import { Logger } from './logger.js';
import { SessionMixin } from './session.js';
import { InputMixin, elementOffset, mouseButtonMap } from './input.js';
import { GraphicsMixin } from './graphics.js';
import { ClipboardMixin } from './clipboard.js';
import { UIMixin } from './ui.js';
import AudioMixin from './audio.js';

// Re-export Logger for external use
export { Logger };

/**
 * Apply a mixin to the Client prototype
 * @param {Object} mixin
 */
function applyMixin(mixin) {
    Object.keys(mixin).forEach(key => {
        if (typeof mixin[key] === 'function') {
            Client.prototype[key] = mixin[key];
        }
    });
}

/**
 * RDP Web Client
 * @param {string} websocketURL - WebSocket URL for RDP connection
 * @param {string} canvasID - ID of canvas element
 * @param {string} hostID - ID of host input element
 * @param {string} userID - ID of username input element
 * @param {string} passwordID - ID of password input element
 */
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
    
    // Initialize all mixins
    this.initSession();
    this.initInput();
    this.initGraphics();
    this.initUI();
    this.initAudio();
    
    // Bind core methods
    this.initialize = this.initialize.bind(this);
    this.handleMessage = this.handleMessage.bind(this);
    this.deinitialize = this.deinitialize.bind(this);
    
    // Load saved session
    this.loadSession();
    
    // Auto-reconnect if session exists
    if (this.shouldAutoReconnect()) {
        this.scheduleReconnect(100);
    }
}

// Apply all mixins to Client prototype
applyMixin(SessionMixin);
applyMixin(InputMixin);
applyMixin(GraphicsMixin);
applyMixin(ClipboardMixin);
applyMixin(UIMixin);
applyMixin(AudioMixin);

/**
 * Connect to RDP server
 */
Client.prototype.connect = function() {
    if (this.socket && this.socket.readyState === WebSocket.OPEN) {
        Logger.warn("Connection", "Already established");
        return;
    }

    const host = this.sanitizeInput(this.hostEl.value);
    const user = this.sanitizeInput(this.userEl.value);
    const password = this.passwordEl.value;

    if (!this.validateHostname(host)) {
        Logger.error("Connection", "Invalid hostname format");
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

    const colorDepthEl = document.getElementById('colorDepth');
    const colorDepth = colorDepthEl ? colorDepthEl.value : '16';

    const disableNLAEl = document.getElementById('disableNLA');
    const disableNLA = disableNLAEl ? disableNLAEl.checked : false;

    const enableAudioEl = document.getElementById('enableAudio');
    const enableAudio = enableAudioEl ? enableAudioEl.checked : false;

    Logger.debug("Connection", `Connecting to ${host} as ${user} (${screenWidth}x${screenHeight}, ${colorDepth}bpp)`);

    // Build URL with non-sensitive parameters only (no password!)
    const url = new URL(this.websocketURL);
    url.searchParams.set('width', screenWidth);
    url.searchParams.set('height', screenHeight);
    url.searchParams.set('colorDepth', colorDepth);
    if (disableNLA) {
        url.searchParams.set('disableNLA', 'true');
        Logger.debug("Connection", "NLA disabled");
    }
    if (enableAudio) {
        url.searchParams.set('audio', 'true');
        this.enableAudio();
        Logger.debug("Audio", "Audio redirection enabled");
    }

    // Store credentials to send after connection opens
    this._pendingCredentials = { host, user, password };

    this.socket = new WebSocket(url.toString());

    this.socket.onopen = () => {
        Logger.debug("Connection", "WebSocket opened, sending credentials");
        
        // Send credentials securely via WebSocket (not URL)
        if (this._pendingCredentials) {
            const credMsg = JSON.stringify({
                type: 'credentials',
                host: this._pendingCredentials.host,
                user: this._pendingCredentials.user,
                password: this._pendingCredentials.password
            });
            this.socket.send(credMsg);
            // Clear credentials from memory
            this._pendingCredentials = null;
        }
        
        this.initialize();
    };

    this.socket.onmessage = (e) => {
        e.data.arrayBuffer()
            .then((arrayBuffer) => this.handleMessage(arrayBuffer))
            .catch((err) => Logger.error('Message', `Failed to read message: ${err.message}`));
    };

    this.socket.onerror = (e) => {
        const errorMsg = e.message || '';
        this.logError('WebSocket connection error', {error: errorMsg, code: e.code});
        
        if (errorMsg.includes('401') || errorMsg.includes('Unauthorized')) {
            this.showUserError('Authentication failed: Invalid username or password');
        } else if (errorMsg.includes('403') || errorMsg.includes('Forbidden')) {
            this.showUserError('Access denied: You do not have permission to connect');
        } else if (errorMsg.includes('404') || errorMsg.includes('Not Found')) {
            this.showUserError('Server not found: Check the server address');
        } else if (errorMsg.includes('timeout')) {
            this.showUserError('Connection timeout: Server is not responding');
        } else {
            this.showUserError('Connection failed: Unable to connect to server');
        }

        this.emitEvent('error', {message: errorMsg, code: e.code});
    };

    this.socket.onclose = (e) => {
        Logger.debug("Connection", `WebSocket closed (code=${e.code}, reason=${e.reason || 'none'})`);

        this.emitEvent('disconnected', {
            code: e.code,
            reason: e.reason,
            wasClean: e.wasClean,
            manual: this.manualDisconnect
        });
        
        if (this.manualDisconnect) {
            this.showUserSuccess('Disconnected successfully');
            return;
        }
        
        if (e.code === 1000) {
            return;
        } else if (e.code === 1001) {
            this.showUserError('Connection closed: Going away');
        } else if (e.code === 1002) {
            this.showUserError('Connection closed: Protocol error');
        } else if (e.code === 1003) {
            this.showUserError('Connection closed: Unsupported data type');
        } else if (e.code === 1006) {
            this.showUserError('Connection closed abnormally: Check your network connection');
        } else if (e.code === 1015) {
            this.showUserError('TLS handshake failed: Certificate validation error');
        } else {
            this.showUserError(`Connection lost (code: ${e.code})`);
        }
        
        if (!this.manualDisconnect && this.reconnectAttempts < this.maxReconnectAttempts) {
            const exponentialDelay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts);
            this.scheduleReconnect(Math.min(exponentialDelay, 30000));
        }
        this.deinitialize();
    };

    this.saveSession();
};

/**
 * Send authentication data
 */
Client.prototype.sendAuthentication = function() {
    if (!this.socket || this.socket.readyState !== WebSocket.OPEN) {
        this.showUserError('Connection lost during authentication');
        return;
    }

    try {
        const authData = {
            type: 'auth',
            user: this.sanitizeInput(this.userEl.value),
            password: this.passwordEl.value,
            host: this.sanitizeInput(this.hostEl.value),
            sessionId: this.sessionId,
            csrfToken: this.csrfToken,
            timestamp: Date.now()
        };

        this.socket.send(JSON.stringify(authData));
        Logger.debug("Connection", "Authentication data sent");
    } catch (error) {
        this.showUserError('Failed to send authentication data');
        Logger.error("Connection", "Authentication send error:", error);
    }
};

/**
 * Initialize connection
 */
Client.prototype.initialize = function() {
    if (this.connected) {
        return;
    }

    this.reconnectAttempts = 0;
    this.lastConnectionTime = Date.now();

    window.addEventListener('keydown', this.handleKeyDown);
    window.addEventListener('keyup', this.handleKeyUp);
    this.canvas.addEventListener('mousemove', this.handleMouseMove);
    this.canvas.addEventListener('mousedown', this.handleMouseDown);
    this.canvas.addEventListener('mouseup', this.handleMouseUp);
    this.canvas.addEventListener('contextmenu', this.handleMouseUp);
    
    this.canvas.addEventListener('click', () => {
        this.canvas.focus();
        this.resumeAudioContext();
    });
    this.canvas.addEventListener('wheel', this.handleWheel);
    
    this.canvas.addEventListener('touchstart', this.handleTouchStart, { passive: false });
    this.canvas.addEventListener('touchmove', this.handleTouchMove, { passive: false });
    this.canvas.addEventListener('touchend', this.handleTouchEnd, { passive: false });
    
    window.addEventListener('resize', this.handleResize);

    this.connected = true;
    
    // Log capabilities on connection
    this.logCapabilities();
};

/**
 * Show canvas after first bitmap
 */
Client.prototype.showCanvas = function() {
    const canvasContainer = document.getElementById('canvas-container');
    const formContainer = document.querySelector('.container');
    if (formContainer) {
        formContainer.style.display = 'none';
    }
    if (canvasContainer) {
        canvasContainer.style.cssText = 'display: block !important; position: fixed !important; top: 0 !important; left: 0 !important; width: 100vw !important; height: 100vh !important; z-index: 9999 !important; background: #000 !important;';
        void canvasContainer.offsetHeight;
    }
    
    this.canvas.style.cssText = 'display: block !important; visibility: visible !important; opacity: 1 !important; position: absolute !important; top: 0 !important; left: 0 !important; width: ' + this.canvas.width + 'px !important; height: ' + this.canvas.height + 'px !important;';
    
    this.canvas.setAttribute('tabindex', '0');
    this.canvas.style.outline = 'none';
    this.canvas.focus();
    
    void this.canvas.offsetHeight;
    
    this.initBitmapCache();
    this.initClipboardSupport();
    this.startTimeoutTracking();
    
    this.emitEvent('connected', {
        host: this.sanitizeInput(this.hostEl.value),
        user: this.sanitizeInput(this.userEl.value)
    });
};

/**
 * Deinitialize connection
 */
Client.prototype.deinitialize = function() {
    this.connected = false;
    this.canvasShown = false;
    
    window.removeEventListener('keydown', this.handleKeyDown);
    window.removeEventListener('keyup', this.handleKeyUp);
    this.canvas.removeEventListener('mousemove', this.handleMouseMove);
    this.canvas.removeEventListener('mousedown', this.handleMouseDown);
    this.canvas.removeEventListener('mouseup', this.handleMouseUp);
    this.canvas.removeEventListener('contextmenu', this.handleMouseUp);
    this.canvas.removeEventListener('wheel', this.handleWheel);
    
    this.canvas.removeEventListener('touchstart', this.handleTouchStart);
    this.canvas.removeEventListener('touchmove', this.handleTouchMove);
    this.canvas.removeEventListener('touchend', this.handleTouchEnd);
    
    window.removeEventListener('resize', this.handleResize);

    this.clearAllTimeouts();
    this.clearBitmapCache();
    this.disableAudio();

    Object.entries(this.pointerCache).forEach(([index, style]) => {
        document.getElementsByTagName('head')[0].removeChild(style);
    });
    this.pointerCache = {};
    this.canvas.classList = [];

    this.ctx.clearRect(0, 0, this.canvas.width, this.canvas.height);
};

/**
 * Handle incoming message
 * @param {ArrayBuffer} arrayBuffer
 */
Client.prototype.handleMessage = function(arrayBuffer) {
    if (!this.connected) {
        return;
    }
    
    // Check for special message types first
    const firstByte = new Uint8Array(arrayBuffer)[0];
    
    // Audio data (0xFE marker)
    if (firstByte === 0xFE && this.audioEnabled) {
        this.handleAudioMessage(new Uint8Array(arrayBuffer));
        return;
    }
    
    // Capabilities/JSON message (0xFF marker)
    if (firstByte === 0xFF) {
        try {
            // Strip the 0xFF marker and parse JSON
            const jsonData = arrayBuffer.slice(1);
            const text = new TextDecoder().decode(jsonData);
            const message = JSON.parse(text);
            
            if (message.type === 'capabilities') {
                // Sync log level with backend
                if (message.logLevel) {
                    Logger.setLevel(message.logLevel);
                }
                Logger.debug("Capabilities", `Server: codecs=${message.codecs?.join(',') || 'none'}, colorDepth=${message.colorDepth}, desktop=${message.desktopSize}`);
                this.serverCapabilities = message;
            } else if (message.type === 'error') {
                this.showUserError(message.message);
                this.emitEvent('error', {message: message.message});
            }
            return;
        } catch (e) {
            Logger.warn("Message", `Failed to parse 0xFF message: ${e.message}`);
        }
    }
    
    // Try parsing as plain JSON (for clipboard, file transfer, etc.)
    try {
        const text = new TextDecoder().decode(arrayBuffer);
        const message = JSON.parse(text);
        
        if (message.type === 'clipboard_response') {
            Logger.debug("Clipboard", "Received remote clipboard data");
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
    
    const r = new BinaryReader(arrayBuffer);
    const header = parseUpdateHeader(r);
    
    Logger.debug("Update", `code=${header.updateCode}, pointer=${header.isPointer()}`);

    if (header.isCompressed()) {
        Logger.warn("Update", "Compression not supported");
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
        Logger.debug("Update", "Synchronize received");
        return;
    }

    if (header.isPalette()) {
        this.handlePalette(r);
        return;
    }

    if (header.updateCode === 0xF) {
        return;
    }

    Logger.debug("Update", `Unknown update code: ${header.updateCode}`);
};

/**
 * Disconnect from server
 */
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
    this.socket.close(1000);
};

/**
 * Reconnect with new desktop size
 * @param {number} width
 * @param {number} height
 */
Client.prototype.reconnectWithNewSize = function(width, height) {
    this.originalWidth = width;
    this.originalHeight = height;
    
    if (this.socket) {
        this.manualDisconnect = true;
        this.socket.close(1000);
    }
    
    setTimeout(() => {
        this.manualDisconnect = false;
        this.connect();
    }, 500);
};

// Export Client as default and named export
export { Client };
export default Client;

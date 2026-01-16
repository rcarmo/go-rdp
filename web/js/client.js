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
    this.pointerCache = {};
    
    // Session persistence and reconnection
    this.reconnectAttempts = 0;
    this.maxReconnectAttempts = 5;
    this.reconnectDelay = 2000; // 2 seconds
    this.reconnectTimeout = null;
    this.lastConnectionTime = null;
    this.manualDisconnect = false;
    this.sessionId = this.generateSessionId();
    
    // Session timeout and idle detection
    this.maxSessionTime = 8 * 60 * 60 * 1000; // 8 hours
    this.maxIdleTime = 30 * 60 * 1000; // 30 minutes
    this.lastActivityTime = null;
    this.sessionTimeout = null;
    this.idleTimeout = null;
    this.warningTimeout = null;
    this.warningShown = false;

    this.handleKeyDown = this.handleKeyDown.bind(this);
    this.handleKeyUp = this.handleKeyUp.bind(this);
    this.handleMouseMove = this.handleMouseMove.bind(this);
    this.handleMouseDown = this.handleMouseDown.bind(this);
    this.handleMouseUp = this.handleMouseUp.bind(this);
    this.handleWheel = this.handleWheel.bind(this);
    this.handleResize = this.handleResize.bind(this);
    
    // Original desktop size (set when connecting)
    this.originalWidth = 0;
    this.originalHeight = 0;
    this.resizeTimeout = null;
    
    // Input event throttling to prevent overwhelming the connection
    this.inputQueue = [];
    this.inputFlushPending = false;
    this.lastMouseMove = null;
    this.mouseThrottleMs = 16; // ~60fps max for mouse moves
    this.lastMouseSendTime = 0;
    
    // Touch event handlers for mobile support
    this.handleTouchStart = this.handleTouchStart.bind(this);
    this.handleTouchMove = this.handleTouchMove.bind(this);
    this.handleTouchEnd = this.handleTouchEnd.bind(this);
    
    // Clipboard and file transfer
    this.clipboardData = '';
    this.clipboardFormat = 'text';
    this.fileTransferEnabled = false;
    this.pendingFileTransfer = null;
    
    // Multi-monitor support
    this.monitors = [];
    this.currentMonitor = 0;
    this.multiMonitorMode = false;
    this.multiMonitorMessageShown = false;
    this.virtualDesktopWidth = 1024;
    this.virtualDesktopHeight = 768;
    
    // Audio input redirection (microphone)
    this.audioContext = null;
    this.microphoneStream = null;
    this.mediaRecorder = null;
    this.audioChunks = [];
    this.audioRedirectEnabled = false;
    this.microphonePermission = 'prompt'; // 'granted', 'denied', or 'prompt'
    
    this.initialize = this.initialize.bind(this);
    this.handleMessage = this.handleMessage.bind(this);
    this.deinitialize = this.deinitialize.bind(this);
    
    // Load saved session
    this.loadSession();
    
    // Auto-reconnect if session exists and wasn't manually disconnected
    if (this.shouldAutoReconnect()) {
        this.scheduleReconnect(100); // Small delay to ensure page is ready
    }
}

Client.prototype.generateSessionId = function() {
    return 'session_' + Date.now() + '_' + Math.random().toString(36).substr(2, 9);
};

Client.prototype.shouldAutoReconnect = function() {
    // Auto-reconnect disabled - user must manually reconnect
    return false;
};

Client.prototype.saveSession = function() {
    try {
        // Store host and user in cookies (30 day expiry)
        const expires = new Date(Date.now() + 30 * 24 * 60 * 60 * 1000).toUTCString();
        document.cookie = `rdp_host=${encodeURIComponent(this.hostEl.value)}; expires=${expires}; path=/; SameSite=Strict`;
        document.cookie = `rdp_user=${encodeURIComponent(this.userEl.value)}; expires=${expires}; path=/; SameSite=Strict`;
    } catch (e) {
        console.error('Failed to save session:', e);
    }
};

Client.prototype.loadSession = function() {
    try {
        const cookies = document.cookie.split(';').reduce((acc, cookie) => {
            const [key, value] = cookie.trim().split('=');
            if (key && value) acc[key] = decodeURIComponent(value);
            return acc;
        }, {});
        
        if (cookies.rdp_host) this.hostEl.value = cookies.rdp_host;
        if (cookies.rdp_user) this.userEl.value = cookies.rdp_user;
    } catch (e) {
        console.warn('Failed to load saved session:', e);
    }
};

Client.prototype.verifySessionIntegrity = function(session) {
    const requiredFields = ['host', 'user', 'timestamp', 'sessionId'];
    return requiredFields.every(field => session.hasOwnProperty(field));
};

Client.prototype.clearSession = function() {
    // Clear cookies by setting expired date
    document.cookie = 'rdp_host=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;';
    document.cookie = 'rdp_user=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;';
    this.manualDisconnect = true;
};

Client.prototype.scheduleReconnect = function(delay) {
    if (this.reconnectTimeout) {
        clearTimeout(this.reconnectTimeout);
    }
    
    if (this.reconnectAttempts >= this.maxReconnectAttempts || this.manualDisconnect) {
        return;
    }
    
    this.reconnectTimeout = setTimeout(() => {
        if (this.shouldAutoReconnect() && !this.connected) {
            console.log(`Attempting reconnection ${this.reconnectAttempts + 1}/${this.maxReconnectAttempts}`);
            this.attemptReconnect();
        }
    }, delay);
};

Client.prototype.attemptReconnect = function() {
    if (!this.hostEl.value || !this.userEl.value) {
        return;
    }
    
    this.reconnectAttempts++;
    
    const url = new URL(this.websocketURL);
    url.searchParams.set('host', this.hostEl.value);
    url.searchParams.set('user', this.userEl.value);
    url.searchParams.set('password', this.passwordEl.value || '');
    url.searchParams.set('width', this.canvas.width);
    url.searchParams.set('height', this.canvas.height);
    url.searchParams.set('sessionId', this.sessionId);

    this.socket = new WebSocket(url.toString());
    this.socket.onopen = this.initialize;
    this.socket.onmessage = (e) => {
        e.data.arrayBuffer().then((arrayBuffer) => this.handleMessage(arrayBuffer))
    };
    this.socket.onerror = (e) => {
        console.log("Reconnection error:", e);
    };
    this.socket.onclose = (e) => {
        if (!this.manualDisconnect && this.reconnectAttempts < this.maxReconnectAttempts) {
            const exponentialDelay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);
            this.scheduleReconnect(Math.min(exponentialDelay, 30000)); // Max 30 seconds
        }
        this.deinitialize();
    };
};

// Session timeout and idle detection methods
Client.prototype.startTimeoutTracking = function() {
    this.lastActivityTime = Date.now();
    this.clearAllTimeouts();
    
    // Start session timeout
    this.sessionTimeout = setTimeout(() => {
        this.handleSessionTimeout();
    }, this.maxSessionTime);
    
    // Start idle timeout warning
    this.warningTimeout = setTimeout(() => {
        this.showIdleWarning();
    }, this.maxIdleTime - 5 * 60 * 1000); // 5 minutes before idle timeout
    
    // Start idle timeout
    this.idleTimeout = setTimeout(() => {
        this.handleIdleTimeout();
    }, this.maxIdleTime);
};

Client.prototype.updateActivity = function() {
    this.lastActivityTime = Date.now();
    
    // Reset idle timeout
    if (this.idleTimeout) {
        clearTimeout(this.idleTimeout);
    }
    if (this.warningTimeout) {
        clearTimeout(this.warningTimeout);
    }
    this.warningShown = false;
    
    // Start new idle timeout
    this.warningTimeout = setTimeout(() => {
        this.showIdleWarning();
    }, this.maxIdleTime - 5 * 60 * 1000);
    
    this.idleTimeout = setTimeout(() => {
        this.handleIdleTimeout();
    }, this.maxIdleTime);
};

Client.prototype.showIdleWarning = function() {
    if (this.warningShown) return;
    this.warningShown = true;
    
    const message = 'Session will timeout due to inactivity in 5 minutes. Move your mouse or press any key to continue.';
    this.showUserWarning(message);
    
    // Auto-hide warning after 10 seconds
    setTimeout(() => {
        if (this.warningShown) {
            this.warningShown = false;
        }
    }, 10000);
};

Client.prototype.handleIdleTimeout = function() {
    const status = document.getElementById('status');
    if (status) {
        status.style.display = 'block';
        status.className = 'status-indicator status-disconnected';
        status.textContent = 'Session disconnected due to inactivity. Please reconnect to continue.';
    }
    
    this.disconnect();
};

Client.prototype.handleSessionTimeout = function() {
    const status = document.getElementById('status');
    if (status) {
        status.style.display = 'block';
        status.className = 'status-indicator status-disconnected';
        status.textContent = 'Session expired due to time limit. Please reconnect to continue.';
    }
    
    this.disconnect();
    this.clearSession(); // Don't auto-reconnect after session timeout
};

Client.prototype.clearAllTimeouts = function() {
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
    this.warningShown = false;
};

// Security and validation functions
Client.prototype.sanitizeInput = function(input) {
    if (!input) return '';
    return input.replace(/[<>'"&]/g, '').trim();
};

Client.prototype.validateHostname = function(hostname) {
    if (!hostname) return false;
    // Updated regex to properly handle both hostnames and IP addresses with ports
    const pattern = /^([a-zA-Z0-9.-]+|(\d{1,3}\.){3}\d{1,3})(:\d{1,5})?$/;
    return pattern.test(hostname) && hostname.length <= 253;
};

Client.prototype.generateCSRFToken = function() {
    return crypto.getRandomValues(new Uint8Array(16))
        .reduce((hex, byte) => hex + byte.toString(16).padStart(2, '0'), '');
};

Client.prototype.connect = function () {
    // Validation is now handled in the HTML UI
    if (this.socket && this.socket.readyState === WebSocket.OPEN) {
        console.warn("Connection already established");
        return;
    }

    // Sanitize and validate inputs
    const host = this.sanitizeInput(this.hostEl.value);
    const user = this.sanitizeInput(this.userEl.value);
    const password = this.passwordEl.value; // Don't sanitize password yet, handled server-side

    if (!this.validateHostname(host)) {
        console.error('Invalid hostname format');
        return;
    }

    // Reset reconnection state for manual connections
    this.reconnectAttempts = 0;
    this.manualDisconnect = false;
    this.lastConnectionTime = Date.now();

    // Generate CSRF token for this connection
    this.csrfToken = this.generateCSRFToken();

    // Use screen dimensions for fullscreen experience
    const screenWidth = window.innerWidth;
    const screenHeight = window.innerHeight;
    
    // Store original desktop dimensions for resize detection
    this.originalWidth = screenWidth;
    this.originalHeight = screenHeight;
    
    // IMPORTANT: Resize canvas BEFORE connecting to avoid clearing drawn content
    // Setting canvas.width or canvas.height clears the canvas
    this.canvas.width = screenWidth;
    this.canvas.height = screenHeight;

    const url = new URL(this.websocketURL);
    url.searchParams.set('host', host);
    url.searchParams.set('user', user);
    url.searchParams.set('password', password);
    url.searchParams.set('width', screenWidth);
    url.searchParams.set('height', screenHeight);

    this.socket = new WebSocket(url.toString());

    this.socket.onopen = () => {
        this.initialize();
    };

    this.handleMessage = this.handleMessage.bind(this);
    this.socket.onmessage = (e) => {
        e.data.arrayBuffer().then((arrayBuffer) => this.handleMessage(arrayBuffer))
    };

    this.socket.onerror = (e) => {
        const errorMsg = e.message || '';
        this.logError('WebSocket connection error', {error: errorMsg, code: e.code});
        
        // Provide user-friendly error messages based on error type
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
        this.logError('WebSocket connection closed', {code: e.code, reason: e.reason, wasClean: e.wasClean});

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
        
        // Provide specific error messages for close codes
        if (e.code === 1000) {
            // Normal closure
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
            this.scheduleReconnect(Math.min(exponentialDelay, 30000)); // Max 30 seconds
        }
        this.deinitialize();
    };

    // Save session for persistence
    this.saveSession();
};

Client.prototype.sendAuthentication = function() {
    if (!this.socket || this.socket.readyState !== WebSocket.OPEN) {
        this.showUserError('Connection lost during authentication');
        return;
    }

    try {
        // Send authentication data as first message (more secure than URL parameters)
        const authData = {
            type: 'auth',
            user: this.sanitizeInput(this.userEl.value),
            password: this.passwordEl.value, // Send password securely after connection
            host: this.sanitizeInput(this.hostEl.value),
            sessionId: this.sessionId,
            csrfToken: this.csrfToken,
            timestamp: Date.now()
        };

        this.socket.send(JSON.stringify(authData));
    } catch (error) {
        this.showUserError('Failed to send authentication data');
        console.error('Authentication send error:', error);
    }
};

Client.prototype.showUserError = function(message) {
    // Show error using toast notification
    if (window.showToast) {
        window.showToast(message, 'error', 'Connection Error', 8000);
    }
    
    // Also show in status element for backward compatibility
    const status = document.getElementById('status');
    if (status) {
        status.style.display = 'block';
        status.className = 'status-indicator status-disconnected';
        status.textContent = message;
        
        // Auto-hide after 10 seconds
        setTimeout(() => {
            if (status.textContent === message) {
                status.style.display = 'none';
            }
        }, 10000);
    }
};

Client.prototype.showUserSuccess = function(message) {
    // Show success using toast notification
    if (window.showToast) {
        window.showToast(message, 'success', 'Success', 5000);
    }
    
    // Also show in status element for backward compatibility
    const status = document.getElementById('status');
    if (status) {
        status.style.display = 'block';
        status.className = 'status-indicator status-connected';
        status.textContent = message;
        
        // Auto-hide after 5 seconds
        setTimeout(() => {
            if (status.textContent === message) {
                status.style.display = 'none';
            }
        }, 5000);
    }
};

Client.prototype.showUserWarning = function(message) {
    // Show warning using toast notification
    if (window.showToast) {
        window.showToast(message, 'info', 'Warning', 6000);
    }
};

Client.prototype.showUserInfo = function(message) {
    // Show info using toast notification
    if (window.showToast) {
        window.showToast(message, 'info', 'Info', 4000);
    }
};

Client.prototype.logError = function(context, details) {
    // Centralized error logging
    const errorInfo = {
        context: context,
        details: details,
        timestamp: new Date().toISOString(),
        sessionId: this.sessionId,
        userAgent: navigator.userAgent
    };
    
    console.error('RDP Client Error:', errorInfo);
    
    // In production, this could send to a logging service
    // this.sendErrorToService(errorInfo);
};

Client.prototype.emitEvent = function(name, detail = {}) {
    try {
        document.dispatchEvent(new CustomEvent('rdp:' + name, {detail: detail}));
    } catch (error) {
        console.warn('Event dispatch failed', error);
    }
};

Client.prototype.initialize = function () {
    if (this.connected) {
        return;
    }

    // Reset reconnection state on successful connection
    this.reconnectAttempts = 0;
    this.lastConnectionTime = Date.now();

    window.addEventListener('keydown', this.handleKeyDown);
    window.addEventListener('keyup', this.handleKeyUp);
    this.canvas.addEventListener('mousemove', this.handleMouseMove);
    this.canvas.addEventListener('mousedown', this.handleMouseDown);
    this.canvas.addEventListener('mouseup', this.handleMouseUp);
    this.canvas.addEventListener('contextmenu', this.handleMouseUp);
    
    // Ensure canvas gets focus on click for keyboard input
    this.canvas.addEventListener('click', () => this.canvas.focus());
    this.canvas.addEventListener('wheel', this.handleWheel);
    
    // Add touch support for mobile devices
    this.canvas.addEventListener('touchstart', this.handleTouchStart, { passive: false });
    this.canvas.addEventListener('touchmove', this.handleTouchMove, { passive: false });
    this.canvas.addEventListener('touchend', this.handleTouchEnd, { passive: false });
    
    // Handle window resize
    window.addEventListener('resize', this.handleResize);

    this.connected = true;
    
    // DIRECTLY show the canvas - don't rely on events
    const canvasContainer = document.getElementById('canvas-container');
    const formContainer = document.querySelector('.container');
    if (formContainer) {
        formContainer.style.display = 'none';
    }
    if (canvasContainer) {
        // First, force container to be visible
        canvasContainer.style.cssText = 'display: block !important; position: fixed !important; top: 0 !important; left: 0 !important; width: 100vw !important; height: 100vh !important; z-index: 9999 !important; background: #000 !important;';
        
        // Force a reflow by reading offsetHeight
        void canvasContainer.offsetHeight;
    }
    
    // Force canvas to be visible with explicit pixel dimensions
    this.canvas.style.cssText = 'display: block !important; visibility: visible !important; opacity: 1 !important; position: absolute !important; top: 0 !important; left: 0 !important; width: ' + this.canvas.width + 'px !important; height: ' + this.canvas.height + 'px !important;';
    
    // Make canvas focusable and set focus for keyboard input
    this.canvas.setAttribute('tabindex', '0');
    this.canvas.style.outline = 'none'; // Hide focus outline
    this.canvas.focus();
    
    // Force a reflow
    void this.canvas.offsetHeight;
    
    this.emitEvent('connected', {
        host: this.sanitizeInput(this.hostEl.value),
        user: this.sanitizeInput(this.userEl.value)
    });
    
    // Initialize audio input redirection
    this.initAudioRedirection();
    
    // Initialize clipboard and file transfer support
    this.initClipboardSupport();
    this.initFileTransfer();
    
    // Start session timeout and idle detection
    this.startTimeoutTracking();
    
    // Detect and setup multi-monitor support (don't show success message yet)
    this.detectMonitors(false);
    this.addMonitorControls();

    // Initialize WASM module if available (commented out for now as it's optional)
    // this.initializeWASM();
};

Client.prototype.deinitialize = function () {
    window.removeEventListener('keydown', this.handleKeyDown);
    window.removeEventListener('keyup', this.handleKeyUp);
    this.canvas.removeEventListener('mousemove', this.handleMouseMove);
    this.canvas.removeEventListener('mousedown', this.handleMouseDown);
    this.canvas.removeEventListener('mouseup', this.handleMouseUp);
    this.canvas.removeEventListener('contextmenu', this.handleMouseUp);
    this.canvas.removeEventListener('wheel', this.handleWheel);
    
    // Remove touch event listeners
    this.canvas.removeEventListener('touchstart', this.handleTouchStart);
    this.canvas.removeEventListener('touchmove', this.handleTouchMove);
    this.canvas.removeEventListener('touchend', this.handleTouchEnd);
    
    // Remove resize listener
    window.removeEventListener('resize', this.handleResize);

    this.connected = false;
    
    // Stop audio redirection
    this.stopMicrophone();
    
    // Clear all timeout tracking
    this.clearAllTimeouts();

    Object.entries(this.pointerCache).forEach(([index, style]) => {
        document.getElementsByTagName('head')[0].removeChild(style);
    });
    this.pointerCache = {};
    this.canvas.classList = [];

    this.ctx.clearRect(0, 0, this.canvas.width, this.canvas.height);
};

Client.prototype.handleMessage = function (arrayBuffer) {
    if (!this.connected) {
        return;
    }
    
    // Try to parse as JSON first (for clipboard and file transfer)
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
    
    // Handle binary RDP data
    const r = new BinaryReader(arrayBuffer);
    const header = parseUpdateHeader(r);

    if (header.isCompressed()) {
        console.warn("compressing is not supported");

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
        // an artifact of the T.128 protocol ([T128] section 8.6.2) and SHOULD be ignored.
        return;
    }

    console.warn("unknown update:", header.updateCode);
};

function buf2hex(buffer) { // buffer is an ArrayBuffer
    return [...new Uint8Array(buffer)]
        .map(x => x.toString(16).padStart(2, '0'))
        .join('');
}

Client.prototype.handleBitmap = function (r) {
    const bitmap = parseBitmapUpdate(r);
    
    // If this is the first successful bitmap update, show multi-monitor message if applicable
    if (this.multiMonitorMode && !this.multiMonitorMessageShown) {
        this.showUserSuccess('Multi-monitor environment detected');
        this.multiMonitorMessageShown = true;
    }

    bitmap.rectangles.forEach((bitmapData) => {
        const size = bitmapData.width * bitmapData.height;
        const rowDelta = bitmapData.width * 2;
        const resultSize = size * 2;

        if (!bitmapData.isCompressed()) {
            let rgb = new Uint8ClampedArray(bitmapData.bitmapDataStream);
            let rgba = new Uint8ClampedArray(bitmapData.width * bitmapData.height * 4);

            flipV(rgb, bitmapData.width, bitmapData.height);
            rgb2rgba(rgb, resultSize, rgba)

            this.ctx.putImageData(new ImageData(rgba, bitmapData.width, bitmapData.height), bitmapData.destLeft, bitmapData.destTop);

            return;
        }

        // Check if WASM Module is available
        if (typeof Module !== 'undefined' && Module._malloc && Module.ccall) {
            try {
                const inputPtr = Module._malloc(bitmapData.bitmapLength);
                const outputPtr = Module._malloc(resultSize);
                const inputHeap = new Uint8Array(Module.HEAPU8.buffer, inputPtr, bitmapData.bitmapDataStream.length);
                inputHeap.set(new Uint8Array(bitmapData.bitmapDataStream));

                const result = Module.ccall('RleDecompress',
                    'number',
                    ['number', 'number', 'number', 'number'],
                    [
                        inputPtr, bitmapData.bitmapLength,
                        outputPtr,
                        rowDelta,
                    ]
                );

                if (!result) {
                    console.log("bad decompress:", bitmapData);
                    Module._free(inputPtr);
                    Module._free(outputPtr);
                    return;
                }

                let rgb = new Uint8ClampedArray(Module.HEAP8.buffer.slice(outputPtr, outputPtr + resultSize));
                let rgba = new Uint8ClampedArray(bitmapData.width * bitmapData.height * 4);

                flipV(rgb, bitmapData.width, bitmapData.height);
                rgb2rgba(rgb, resultSize, rgba)

                this.ctx.putImageData(new ImageData(rgba, bitmapData.width, bitmapData.height), bitmapData.destLeft, bitmapData.destTop);

                Module._free(inputPtr);
                Module._free(outputPtr);
                return;
            } catch (error) {
                console.error("WASM decompression failed, falling back to JavaScript:", error);
                // Fall through to JavaScript decompression
            }
        }

        // Fallback to JavaScript decompression if WASM is not available
        try {
            let rgb = new Uint8ClampedArray(bitmapData.bitmapDataStream);
            let rgba = new Uint8ClampedArray(bitmapData.width * bitmapData.height * 4);

            flipV(rgb, bitmapData.width, bitmapData.height);
            rgb2rgba(rgb, resultSize, rgba)

            this.ctx.putImageData(new ImageData(rgba, bitmapData.width, bitmapData.height), bitmapData.destLeft, bitmapData.destTop);
        } catch (error) {
            console.error("Failed to decompress bitmap:", error);
        }
    });
};

Client.prototype.handlePointer = function (header, r) {
    if (header.isPTRNull()) {
        this.canvas.classList = ['pointer-cache-null'];

        return;
    }

    if (header.isPTRDefault()) {
        this.canvas.classList = ['pointer-cache-default'];

        return;
    }

    if (header.isPTRColor()) {
        // ptr color is unsupported
        return;
    }

    if (header.isPTRNew()) {
        const newPointerUpdate = parseNewPointerUpdate(r);
        this.pointerCacheCanvasCtx.putImageData(newPointerUpdate.getImageData(this.pointerCacheCanvasCtx), 0, 0)

        const url = this.pointerCacheCanvas.toDataURL('image/webp', 1);

        if (this.pointerCache.hasOwnProperty(newPointerUpdate.cacheIndex)) {
            document.getElementsByTagName('head')[0].removeChild(this.pointerCache[newPointerUpdate.cacheIndex]);

            delete this.pointerCache[newPointerUpdate.cacheIndex];
        }

        const style = document.createElement('style');
        const className = 'pointer-cache-' + newPointerUpdate.cacheIndex
        style.innerHTML = '.' + className + ' {cursor:url("' + url + '") ' + newPointerUpdate.x + ' ' + newPointerUpdate.y + ', auto}';

        document.getElementsByTagName('head')[0].appendChild(style);

        this.pointerCache[newPointerUpdate.cacheIndex] = style;

        this.canvas.classList = [className];

        return;
    }

    if (header.isPTRCached()) {
        const cacheIndex = r.uint16(true);
        const className = 'pointer-cache-' + cacheIndex;

        this.canvas.classList = [className];

        return;
    }

    if (header.isPTRPosition()) {
        // ptr position is unsupported
        return;
    }
};

// Handle window resize - reconnect with new desktop dimensions
Client.prototype.handleResize = function () {
    if (!this.connected) {
        return;
    }
    
    const windowWidth = window.innerWidth;
    const windowHeight = window.innerHeight;
    
    // Only resize if dimensions changed significantly (more than 10 pixels)
    if (Math.abs(windowWidth - this.originalWidth) < 10 && 
        Math.abs(windowHeight - this.originalHeight) < 10) {
        return;
    }
    
    // Debounce resize - wait for user to stop resizing
    if (this.resizeTimeout) {
        clearTimeout(this.resizeTimeout);
    }
    
    this.resizeTimeout = setTimeout(() => {
        // Reconnect with new dimensions
        this.showUserInfo('Resizing session to ' + windowWidth + 'x' + windowHeight + '...');
        
        // Store current credentials
        const host = this.hostEl.value;
        const user = this.userEl.value;
        const password = this.passwordEl.value;
        
        // Disconnect and reconnect
        this.manualDisconnect = true;
        this.disconnect();
        
        // Short delay before reconnecting
        setTimeout(() => {
            this.manualDisconnect = false;
            this.connect();
        }, 500);
    }, 1000); // Wait 1 second after resize stops
};

Client.prototype.handleKeyDown = function (e) {
    if (!this.connected) {
        this.showUserError('Cannot send keystrokes: not connected to server');
        return;
    }

    // Update activity for timeout tracking
    this.updateActivity();

    const event = new KeyboardEventKeyDown(e.code);

    if (event.keyCode === undefined) {
        this.logError('Key mapping error', {code: e.code, key: e.key});
        this.showUserError('Unsupported key: ' + e.key);
        e.preventDefault();
        return false;
    }

    try {
        const data = event.serialize();
        // Queue key events with priority
        this.queueInput(data, false);
    } catch (error) {
        this.logError('Key send error', {code: e.code, error: error.message});
        this.showUserError('Failed to send keystroke');
    }

    e.preventDefault();
    return false;
};

Client.prototype.handleKeyUp = function (e) {
    if (!this.connected) {
        return;
    }

    // Update activity for timeout tracking
    this.updateActivity();

    const event = new KeyboardEventKeyUp(e.code);

    if (event.keyCode === undefined) {
        console.warn("undefined key up:", e)
        e.preventDefault();
        return false;
    }

    const data = event.serialize();
    // Queue key events with priority
    this.queueInput(data, false);

    e.preventDefault();
    return false;
};

function elementOffset(el) {
    let x = 0;
    let y = 0;

    while (el && !isNaN( el.offsetLeft ) && !isNaN( el.offsetTop )) {
        x += el.offsetLeft - el.scrollLeft;
        y += el.offsetTop - el.scrollTop;
        el = el.offsetParent;
    }

    return { top: y, left: x };
}

function mouseButtonMap(button) {
    switch(button) {
        case 0:
            return 1;
        case 2:
            return 2;
        default:
            return 0;
    }
}

// Convert screen coordinates to desktop coordinates
Client.prototype.screenToDesktop = function (screenX, screenY) {
    const offset = elementOffset(this.canvas);
    const x = Math.floor(screenX - offset.left);
    const y = Math.floor(screenY - offset.top);
    return { x: x, y: y };
};

// Queue an input event for sending (prioritizes recent events)
Client.prototype.queueInput = function(data, isMouseMove) {
    if (isMouseMove) {
        // For mouse moves, just store the latest - we'll send it on the next flush
        this.lastMouseMove = data;
    } else {
        // For clicks/keys, queue immediately
        this.inputQueue.push(data);
    }
    
    // Schedule a flush if not already pending
    if (!this.inputFlushPending) {
        this.inputFlushPending = true;
        // Use setTimeout(0) to yield to the browser and allow bitmap processing
        setTimeout(() => this.flushInputQueue(), 0);
    }
};

// Flush queued input events to the server
Client.prototype.flushInputQueue = function() {
    this.inputFlushPending = false;
    
    if (!this.connected || !this.socket || this.socket.readyState !== WebSocket.OPEN) {
        this.inputQueue = [];
        this.lastMouseMove = null;
        return;
    }
    
    // Send all queued click/key events first (they're more important)
    while (this.inputQueue.length > 0) {
        const data = this.inputQueue.shift();
        try {
            this.socket.send(data);
        } catch (e) {
            // Connection error, clear queue
            this.inputQueue = [];
            break;
        }
    }
    
    // Then send the latest mouse position if enough time has passed
    const now = Date.now();
    if (this.lastMouseMove && (now - this.lastMouseSendTime) >= this.mouseThrottleMs) {
        try {
            this.socket.send(this.lastMouseMove);
            this.lastMouseSendTime = now;
        } catch (e) {
            // Ignore mouse send errors
        }
        this.lastMouseMove = null;
    } else if (this.lastMouseMove) {
        // Schedule another flush for the pending mouse move
        if (!this.inputFlushPending) {
            this.inputFlushPending = true;
            setTimeout(() => this.flushInputQueue(), this.mouseThrottleMs);
        }
    }
};

Client.prototype.handleMouseMove = function (e) {
    // Update activity for timeout tracking (throttle to avoid excessive updates)
    if (!this.lastActivityUpdate || Date.now() - this.lastActivityUpdate > 1000) {
        this.updateActivity();
        this.lastActivityUpdate = Date.now();
    }

    try {
        const pos = this.screenToDesktop(e.clientX, e.clientY);
        const event = new MouseMoveEvent(pos.x, pos.y);
        const data = event.serialize();
        
        // Queue mouse move (will be throttled)
        this.queueInput(data, true);
    } catch (error) {
        this.logError('Mouse move error', {x: e.clientX, y: e.clientY, error: error.message});
    }

    e.preventDefault();
    return false;
};

Client.prototype.handleMouseDown = function (e) {
    // Update activity for timeout tracking
    this.updateActivity();

    const pos = this.screenToDesktop(e.clientX, e.clientY);
    const event = new MouseDownEvent(pos.x, pos.y, mouseButtonMap(e.button));
    const data = event.serialize();
    
    // Queue click events with priority
    this.queueInput(data, false);

    e.preventDefault();
    return false;
};

Client.prototype.handleMouseUp = function (e) {
    const pos = this.screenToDesktop(e.clientX, e.clientY);
    const event = new MouseUpEvent(pos.x, pos.y, mouseButtonMap(e.button));
    const data = event.serialize();
    
    // Queue click events with priority
    this.queueInput(data, false);

    e.preventDefault();
    return false;
};

Client.prototype.handleWheel = function (e) {
    // Update activity for timeout tracking
    this.updateActivity();

    const pos = this.screenToDesktop(e.clientX, e.clientY);

    const isHorizontal = Math.abs(e.deltaX) > Math.abs(e.deltaY);
    const delta = isHorizontal?e.deltaX:e.deltaY;
    const step = Math.round(Math.abs(delta) * 15 / 8);

    const event = new MouseWheelEvent(pos.x, pos.y, step, delta > 0, isHorizontal);
    const data = event.serialize();
    
    // Queue wheel events with priority
    this.queueInput(data, false);

    e.preventDefault();
    return false;
};

// Touch event handlers for mobile support
Client.prototype.handleTouchStart = function (e) {
    if (!this.connected) {
        return;
    }
    
    e.preventDefault(); // Prevent scrolling/zooming
    this.updateActivity();
    
    const touch = e.touches[0];
    const mouseEvent = {
        clientX: touch.clientX,
        clientY: touch.clientY,
        button: 0,
        preventDefault: () => {}
    };
    
    return this.handleMouseDown(mouseEvent);
};

Client.prototype.handleTouchMove = function (e) {
    if (!this.connected) {
        return;
    }
    
    e.preventDefault(); // Prevent scrolling/zooming
    
    // Throttle touch move events
    if (!this.lastTouchUpdate || Date.now() - this.lastTouchUpdate > 16) { // ~60fps
        this.updateActivity();
        this.lastTouchUpdate = Date.now();
        
        const touch = e.touches[0];
        const mouseEvent = {
            clientX: touch.clientX,
            clientY: touch.clientY,
            preventDefault: () => {}
        };
        
        return this.handleMouseMove(mouseEvent);
    }
};

Client.prototype.handleTouchEnd = function (e) {
    if (!this.connected) {
        return;
    }
    
    e.preventDefault();
    this.updateActivity();
    
    const touch = e.changedTouches[0];
    const mouseEvent = {
        clientX: touch.clientX,
        clientY: touch.clientY,
        button: 0,
        preventDefault: () => {}
    };
    
    return this.handleMouseUp(mouseEvent);
};

// Clipboard and file transfer methods
Client.prototype.initClipboardSupport = function() {
    // Check if Clipboard API is supported
    if (navigator.clipboard && navigator.clipboard.readText && navigator.clipboard.writeText) {
        this.clipboardApiSupported = true;
        this.setupClipboardSync();
    } else {
        this.clipboardApiSupported = false;
        this.showUserWarning('Clipboard synchronization not supported in this browser');
    }
};

Client.prototype.setupClipboardSync = function() {
    // Monitor local clipboard changes
    document.addEventListener('paste', (e) => {
        this.handleLocalPaste(e);
    });
    
    // Check for clipboard focus/blur events to sync from remote
    document.addEventListener('focus', () => {
        this.requestRemoteClipboard();
    });
};

Client.prototype.handleLocalPaste = function(event) {
    if (!this.connected || !this.clipboardApiSupported) {
        return;
    }
    
    const pasteData = event.clipboardData || window.clipboardData;
    if (!pasteData) return;
    
    const text = pasteData.getData('text/plain');
    if (text) {
        this.sendClipboardToRemote(text);
    }
};

Client.prototype.sendClipboardToRemote = function(text) {
    if (!this.socket || !this.connected) {
        return;
    }
    
    try {
        const clipboardData = {
            type: 'clipboard',
            format: 'text',
            data: this.sanitizeInput(text),
            timestamp: Date.now()
        };
        
        this.socket.send(JSON.stringify(clipboardData));
        
        if (window.showToast) {
            window.showToast('Clipboard content synced to remote session', 'info', 'Clipboard', 3000);
        }
    } catch (error) {
        this.logError('Clipboard sync error', {error: error.message, dataLength: text.length});
        this.showUserError('Failed to sync clipboard to remote session');
    }
};

Client.prototype.requestRemoteClipboard = function() {
    if (!this.socket || !this.connected || !this.clipboardApiSupported) {
        return;
    }
    
    try {
        const request = {
            type: 'clipboard_request',
            timestamp: Date.now()
        };
        
        this.socket.send(JSON.stringify(request));
    } catch (error) {
        this.logError('Clipboard request error', {error: error.message});
    }
};

Client.prototype.handleRemoteClipboard = function(data) {
    if (!this.clipboardApiSupported) {
        return;
    }
    
    navigator.clipboard.writeText(data).then(() => {
        if (window.showToast) {
            window.showToast('Remote clipboard content available locally', 'success', 'Clipboard', 3000);
        }
    }).catch(error => {
        this.logError('Local clipboard write error', {error: error.message});
        this.showUserError('Failed to update local clipboard');
    });
};

// File transfer methods
Client.prototype.detectMonitors = function(showSuccessMessage) {
    // Try to get monitor information using Screen API
    if (window.screen && window.screen.width && window.screen.height) {
        this.monitors = [{
            id: 0,
            width: window.screen.width,
            height: window.screen.height,
            availWidth: window.screen.availWidth || window.screen.width,
            availHeight: window.screen.availHeight || window.screen.height,
            isPrimary: true
        }];
        
        this.virtualDesktopWidth = this.monitors[0].width;
        this.virtualDesktopHeight = this.monitors[0].height;
        
        // Multi-monitor detection disabled
        this.multiMonitorMode = false;
        
        return this.monitors;
    }
    
    // Fallback for older browsers
    this.monitors = [{
        id: 0,
        width: 1024,
        height: 768,
        availWidth: 1024,
        availHeight: 768,
        isPrimary: true
    }];
    
    return this.monitors;
};

Client.prototype.addMonitorControls = function() {
    const form = document.querySelector('#connection-form');
    if (!form) return;
    
    // Create monitor selection controls
    const monitorControls = document.createElement('div');
    monitorControls.className = 'monitor-controls';
    monitorControls.innerHTML = `
        <div class="form-group">
            <label for="monitor-select">Display Monitor</label>
            <select id="monitor-select" class="monitor-select">
                <option value="0">Primary Monitor</option>
            </select>
            <button type="button" id="cycle-monitor-btn" class="btn btn-primary" style="margin-left: 8px;">
                🖥 Cycle Monitor
            </button>
        </div>
        <div class="monitor-info" id="monitor-info">
            <strong>Display Info:</strong> 
            <span id="monitor-details">Detecting...</span>
        </div>
    `;
    
    // Insert before the form submit buttons
    const buttonGroup = form.querySelector('.button-group');
    if (buttonGroup) {
        form.insertBefore(monitorControls, buttonGroup);
    } else {
        // Fallback: append to form
        form.appendChild(monitorControls);
    }
    
    // Populate monitor options
    this.updateMonitorOptions();
    
    // Add event listeners
    document.getElementById('monitor-select').addEventListener('change', (e) => {
        this.switchToMonitor(parseInt(e.target.value));
    });
    
    document.getElementById('cycle-monitor-btn').addEventListener('click', () => {
        this.cycleToNextMonitor();
    });
};

Client.prototype.updateMonitorOptions = function() {
    const select = document.getElementById('monitor-select');
    if (!select) return;
    
    // Clear existing options
    select.innerHTML = '';
    
    // Add monitor options
    this.monitors.forEach((monitor, index) => {
        const option = document.createElement('option');
        option.value = monitor.id;
        option.textContent = `Monitor ${monitor.id + 1} (${monitor.width}x${monitor.height})`;
        if (monitor.id === this.currentMonitor) {
            option.selected = true;
        }
        select.appendChild(option);
    });
    
    this.updateMonitorInfo();
};

Client.prototype.updateMonitorInfo = function() {
    const info = document.getElementById('monitor-info');
    const details = document.getElementById('monitor-details');
    if (!info || !details) return;
    
    const monitor = this.monitors[this.currentMonitor];
    details.textContent = `${monitor.width}x${monitor.height} pixels`;
    
    if (this.multiMonitorMode) {
        details.textContent += ` (Multi-monitor mode)`;
    }
};

Client.prototype.switchToMonitor = function(monitorId) {
    if (monitorId === this.currentMonitor) return;
    
    const monitor = this.monitors.find(m => m.id === monitorId);
    if (!monitor) return;
    
    this.currentMonitor = monitorId;
    
    // Send monitor switch request to server
    if (this.socket && this.connected) {
        try {
            const switchData = {
                type: 'monitor_switch',
                monitorId: monitorId,
                width: monitor.width,
                height: monitor.height,
                timestamp: Date.now()
            };
            
            this.socket.send(JSON.stringify(switchData));
            this.showUserSuccess(`Switched to Monitor ${monitorId + 1}`);
        } catch (error) {
            this.logError('Monitor switch error', {error: error.message, monitorId});
            this.showUserError('Failed to switch monitor');
        }
    }
    
    this.updateMonitorInfo();
};

Client.prototype.cycleToNextMonitor = function() {
    const nextIndex = (this.currentMonitor + 1) % this.monitors.length;
    this.switchToMonitor(this.monitors[nextIndex].id);
};

// Audio input redirection methods
Client.prototype.initAudioRedirection = function() {
    // Check if Web Audio API is supported
    if (navigator.mediaDevices && navigator.mediaDevices.getUserMedia && window.AudioContext) {
        this.audioApiSupported = true;
        this.setupMicrophoneControls();
    } else {
        this.audioApiSupported = false;
        this.showUserWarning('Audio redirection not supported in this browser');
    }
};

Client.prototype.setupMicrophoneControls = function() {
    const container = document.querySelector('.connection-panel');
    if (!container) return;
    
    // Create microphone controls
    const audioControls = document.createElement('div');
    audioControls.className = 'audio-controls';
    audioControls.innerHTML = `
        <div class="form-group">
            <label for="mic-toggle">Microphone Redirection</label>
            <div class="mic-control-group">
                <button type="button" id="mic-toggle-btn" class="btn btn-primary">
                    🎤 Enable Microphone
                </button>
                <span class="mic-status" id="mic-status">Disabled</span>
            </div>
        </div>
        <div class="mic-levels">
            <div class="level-indicator" id="mic-level-indicator"></div>
            <div class="level-bars">
                <div class="level-bar" id="mic-level-1"></div>
                <div class="level-bar" id="mic-level-2"></div>
                <div class="level-bar" id="mic-level-3"></div>
                <div class="level-bar" id="mic-level-4"></div>
                <div class="level-bar" id="mic-level-5"></div>
            </div>
        </div>
    `;
    
    // Insert after connection form
    const form = container.querySelector('#connection-form');
    if (form) {
        form.appendChild(audioControls);
    }
    
    // Add event listeners
    document.getElementById('mic-toggle-btn').addEventListener('click', () => {
        this.toggleMicrophone();
    });
};

Client.prototype.toggleMicrophone = function() {
    const btn = document.getElementById('mic-toggle-btn');
    const status = document.getElementById('mic-status');
    
    if (!btn || !status) return;
    
    if (this.audioRedirectEnabled) {
        // Disable microphone
        this.stopMicrophone();
        this.audioRedirectEnabled = false;
        btn.textContent = '🎤 Enable Microphone';
        btn.className = 'btn btn-primary';
        status.textContent = 'Disabled';
        if (window.showToast) {
            window.showToast('Microphone redirection disabled', 'info', 'Audio', 3000);
        }
    } else {
        // Enable microphone
        this.startMicrophone();
    }
};

Client.prototype.startMicrophone = async function() {
    try {
        // Request microphone permission
        const stream = await navigator.mediaDevices.getUserMedia({
            audio: {
                echoCancellation: true,
                noiseSuppression: true,
                sampleRate: 16000,
                channelCount: 1
            }
        });
        
        this.microphonePermission = 'granted';
        this.microphoneStream = stream;
        
        // Setup audio context
        this.audioContext = new (window.AudioContext || window.webkitAudioContext)();
        this.mediaRecorder = new MediaRecorder(stream, {
            mimeType: this.getSupportedMimeType()
        });
        
        // Setup recording handlers
        this.mediaRecorder.ondataavailable = (event) => {
            if (event.data && event.data.size > 0) {
                this.audioChunks.push(event.data);
                this.sendAudioChunk(event.data);
            }
        };
        
        this.mediaRecorder.onstart = () => {
            this.audioRedirectEnabled = true;
            this.updateMicrophoneUI();
            if (window.showToast) {
                window.showToast('Microphone redirection enabled', 'success', 'Audio', 3000);
            }
        };
        
        this.mediaRecorder.onerror = (event) => {
            this.logError('Microphone error', {error: event.error});
            this.showUserError('Microphone error: ' + event.error.message);
            this.stopMicrophone();
        };
        
        this.mediaRecorder.start(100); // Send chunks every 100ms
        
    } catch (error) {
        if (error.name === 'NotAllowedError') {
            this.microphonePermission = 'denied';
            this.showUserError('Microphone permission denied. Please allow microphone access.');
        } else {
            this.logError('Microphone initialization error', {error: error.message});
            this.showUserError('Failed to initialize microphone: ' + error.message);
        }
    }
};

Client.prototype.stopMicrophone = function() {
    if (this.mediaRecorder && this.mediaRecorder.state !== 'inactive') {
        this.mediaRecorder.stop();
    }
    
    if (this.microphoneStream) {
        this.microphoneStream.getTracks().forEach(track => track.stop());
        this.microphoneStream = null;
    }
    
    if (this.audioContext) {
        this.audioContext.close();
        this.audioContext = null;
    }
    
    this.audioRedirectEnabled = false;
    this.audioChunks = [];
    this.updateMicrophoneUI();
};

Client.prototype.updateMicrophoneUI = function() {
    const btn = document.getElementById('mic-toggle-btn');
    const status = document.getElementById('mic-status');
    
    if (!btn || !status) return;
    
    if (this.audioRedirectEnabled) {
        btn.textContent = '🔴 Disable Microphone';
        btn.className = 'btn btn-danger';
        status.textContent = 'Active';
        this.startMicrophoneLevelIndicator();
    } else {
        btn.textContent = '🎤 Enable Microphone';
        btn.className = 'btn btn-primary';
        status.textContent = 'Disabled';
        this.stopMicrophoneLevelIndicator();
    }
};

Client.prototype.startMicrophoneLevelIndicator = function() {
    // Visual microphone level indicator
    if (!this.audioContext || !this.microphoneStream) return;
    
    const source = this.audioContext.createMediaStreamSource(this.microphoneStream);
    const analyser = this.audioContext.createAnalyser();
    analyser.fftSize = 256;
    
    source.connect(analyser);
    
    const updateLevels = () => {
        if (!this.audioRedirectEnabled) return;
        
        const dataArray = new Uint8Array(analyser.frequencyBinCount);
        analyser.getByteFrequencyData(dataArray);
        
        // Calculate average level
        const average = dataArray.reduce((sum, value) => sum + value, 0) / dataArray.length;
        const normalizedLevel = Math.min(average / 128, 1); // Normalize to 0-1
        
        // Update level bars
        for (let i = 1; i <= 5; i++) {
            const bar = document.getElementById(`mic-level-${i}`);
            if (bar) {
                const shouldShow = i <= (normalizedLevel * 5);
                bar.style.backgroundColor = shouldShow ? '#10b981' : '#e5e7eb';
                bar.style.transform = shouldShow ? 'scaleY(1)' : 'scaleY(0.1)';
            }
        }
        
        if (this.audioRedirectEnabled) {
            requestAnimationFrame(updateLevels);
        }
    };
    
    updateLevels();
};

Client.prototype.stopMicrophoneLevelIndicator = function() {
    for (let i = 1; i <= 5; i++) {
        const bar = document.getElementById(`mic-level-${i}`);
        if (bar) {
            bar.style.backgroundColor = '#e5e7eb';
            bar.style.transform = 'scaleY(0.1)';
        }
    }
};

Client.prototype.getSupportedMimeType = function() {
    const types = [
        'audio/webm',
        'audio/mp4',
        'audio/ogg'
    ];
    
    for (const type of types) {
        if (MediaRecorder.isTypeSupported(type)) {
            return type;
        }
    }
    
    return 'audio/webm'; // Fallback
};

Client.prototype.sendAudioChunk = function(chunk) {
    if (!this.socket || !this.connected) {
        return;
    }
    
    try {
        // Convert chunk to base64 for transmission
        const reader = new FileReader();
        reader.onload = () => {
            const audioData = {
                type: 'audio_chunk',
                data: reader.result,
                timestamp: Date.now(),
                size: chunk.size
            };
            
            this.socket.send(JSON.stringify(audioData));
        };
        
        reader.readAsDataURL(chunk);
    } catch (error) {
        this.logError('Audio chunk send error', {error: error.message});
    }
};

Client.prototype.initFileTransfer = function() {
    // Check if File API is supported
    if (window.File && window.FileReader && window.FileList) {
        this.fileApiSupported = true;
        this.setupFileTransfer();
    } else {
        this.fileApiSupported = false;
        this.showUserWarning('File transfer not supported in this browser');
    }
};

Client.prototype.setupFileTransfer = function() {
    // Add drag and drop support to canvas
    this.canvas.addEventListener('dragover', (e) => {
        e.preventDefault();
        e.stopPropagation();
        this.canvas.style.borderColor = '#2563eb';
        this.canvas.style.boxShadow = '0 0 10px rgba(37, 99, 235, 0.3)';
    });
    
    this.canvas.addEventListener('dragleave', (e) => {
        e.preventDefault();
        e.stopPropagation();
        this.canvas.style.borderColor = '';
        this.canvas.style.boxShadow = '';
    });
    
    this.canvas.addEventListener('drop', (e) => {
        e.preventDefault();
        e.stopPropagation();
        this.canvas.style.borderColor = '';
        this.canvas.style.boxShadow = '';
        
        const files = e.dataTransfer.files;
        if (files.length > 0) {
            this.handleFileUpload(files);
        }
    });
    
    // Add file input button
    this.addFileUploadButton();
};

Client.prototype.addFileUploadButton = function() {
    const container = document.querySelector('.canvas-container');
    if (!container) return;
    
    // Create file upload controls (hidden by default, shown via hot corner menu)
    const controls = document.createElement('div');
    controls.className = 'file-transfer-controls';
    controls.id = 'file-transfer-controls';
    controls.style.display = 'none';
    controls.innerHTML = `
        <div class="file-upload-area">
            <input type="file" id="file-input" multiple style="display: none;">
            <button class="btn btn-primary" id="file-upload-btn">
                📁 Upload Files
            </button>
            <button class="btn" id="file-upload-close" style="padding: 8px 12px;">✕</button>
            <span class="file-status" id="file-status"></span>
        </div>
    `;
    
    container.appendChild(controls);
    
    // Add click handler
    document.getElementById('file-upload-btn').addEventListener('click', () => {
        document.getElementById('file-input').click();
    });
    
    // Add close button handler
    document.getElementById('file-upload-close').addEventListener('click', () => {
        this.hideFileUpload();
    });
    
    // Add change handler
    document.getElementById('file-input').addEventListener('change', (e) => {
        if (e.target.files.length > 0) {
            this.handleFileUpload(e.target.files);
        }
    });
};

// Show file upload controls (called from hot corner menu)
Client.prototype.showFileUpload = function() {
    const controls = document.getElementById('file-transfer-controls');
    if (controls) {
        controls.style.display = 'flex';
    }
};

// Hide file upload controls
Client.prototype.hideFileUpload = function() {
    const controls = document.getElementById('file-transfer-controls');
    if (controls) {
        controls.style.display = 'none';
    }
};

Client.prototype.handleFileUpload = function(files) {
    if (!this.fileApiSupported) {
        this.showUserError('File transfer not supported');
        return;
    }
    
    const fileStatus = document.getElementById('file-status');
    if (fileStatus) {
        fileStatus.textContent = `Preparing ${files.length} file(s)...`;
    }
    
    Array.from(files).forEach((file, index) => {
        this.sendFile(file, index);
    });
};

Client.prototype.sendFile = function(file, index) {
    const reader = new FileReader();
    
    reader.onload = (e) => {
        try {
            const fileData = {
                type: 'file_transfer',
                name: file.name,
                size: file.size,
                type: file.type,
                data: e.target.result.split(',')[1], // Remove data URL prefix
                index: index,
                timestamp: Date.now()
            };
            
            this.socket.send(JSON.stringify(fileData));
            
            this.updateFileStatus(`${file.name} uploaded successfully`);
        } catch (error) {
            this.logError('File transfer error', {error: error.message, file: file.name});
            this.updateFileStatus(`Failed to upload ${file.name}`);
        }
    };
    
    reader.onerror = () => {
        this.logError('File read error', {file: file.name});
        this.updateFileStatus(`Failed to read ${file.name}`);
    };
    
    reader.readAsDataURL(file);
};

Client.prototype.updateFileStatus = function(message) {
    const fileStatus = document.getElementById('file-status');
    if (fileStatus) {
        fileStatus.textContent = message;
        
        // Auto-hide after 5 seconds
        setTimeout(() => {
            if (fileStatus.textContent === message) {
                fileStatus.textContent = '';
            }
        }, 5000);
    }
};

Client.prototype.disconnect = function () {
    if (!this.socket) {
        return;
    }

    // Mark as manual disconnect to prevent auto-reconnection
    this.manualDisconnect = true;
    this.clearSession();
    
    // Clear any pending reconnection timeout
    if (this.reconnectTimeout) {
        clearTimeout(this.reconnectTimeout);
        this.reconnectTimeout = null;
    }

    this.deinitialize();

    this.socket.close(1000); // ok
};

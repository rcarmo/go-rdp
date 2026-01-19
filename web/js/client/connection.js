/**
 * RDP Client Connection Module
 * WebSocket connection, authentication, and lifecycle management
 */

// Establish WebSocket connection to RDP server
Client.prototype.connect = function () {
    console.log('[RDP] connect() called');
    
    if (this.socket && this.socket.readyState === WebSocket.OPEN) {
        console.warn("Connection already established");
        return;
    }

    const host = this.sanitizeInput(this.hostEl.value);
    const user = this.sanitizeInput(this.userEl.value);
    const password = this.passwordEl.value;

    if (!this.validateHostname(host)) {
        console.error('Invalid hostname format');
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

    const url = new URL(this.websocketURL);
    url.searchParams.set('host', host);
    url.searchParams.set('user', user);
    url.searchParams.set('password', password);
    url.searchParams.set('width', screenWidth);
    url.searchParams.set('height', screenHeight);
    url.searchParams.set('colorDepth', colorDepth);

    console.log('[RDP] Connecting to:', url.toString().replace(/password=[^&]+/, 'password=***'));
    
    this.socket = new WebSocket(url.toString());

    this.socket.onopen = () => {
        console.log('[RDP] WebSocket connected');
        this.initialize();
    };

    this.handleMessage = this.handleMessage.bind(this);
    this.socket.onmessage = (e) => {
        e.data.arrayBuffer().then((arrayBuffer) => this.handleMessage(arrayBuffer))
    };

    this.socket.onerror = (e) => {
        console.error('[RDP] WebSocket error:', e);
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
            this.scheduleReconnect(Math.min(exponentialDelay, 30000));
        }
        this.deinitialize();
    };

    this.saveSession();
};

// Send authentication data
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
    } catch (error) {
        this.showUserError('Failed to send authentication data');
        console.error('Authentication send error:', error);
    }
};

// Initialize client after successful connection
Client.prototype.initialize = function () {
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
    
    this.canvas.addEventListener('click', () => this.canvas.focus());
    this.canvas.addEventListener('wheel', this.handleWheel);
    
    this.canvas.addEventListener('touchstart', this.handleTouchStart, { passive: false });
    this.canvas.addEventListener('touchmove', this.handleTouchMove, { passive: false });
    this.canvas.addEventListener('touchend', this.handleTouchEnd, { passive: false });
    
    window.addEventListener('resize', this.handleResize);

    this.connected = true;
};

// Show canvas after first successful bitmap
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
    this.initAudioRedirection();
    this.initClipboardSupport();
    this.initFileTransfer();
    this.startTimeoutTracking();
    this.detectMonitors(false);
    this.addMonitorControls();
    
    this.emitEvent('connected', {
        host: this.sanitizeInput(this.hostEl.value),
        user: this.sanitizeInput(this.userEl.value)
    });
};

// Cleanup when disconnected
Client.prototype.deinitialize = function () {
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

    this.connected = false;
    
    this.stopMicrophone();
    this.clearAllTimeouts();
    this.clearBitmapCache();

    Object.entries(this.pointerCache).forEach(([index, style]) => {
        document.getElementsByTagName('head')[0].removeChild(style);
    });
    this.pointerCache = {};
    this.canvas.classList = [];

    this.ctx.clearRect(0, 0, this.canvas.width, this.canvas.height);
};

// Gracefully disconnect
Client.prototype.disconnect = function () {
    this.manualDisconnect = true;
    
    if (this.reconnectTimeout) {
        clearTimeout(this.reconnectTimeout);
        this.reconnectTimeout = null;
    }
    
    this.emitEvent('disconnecting');
    
    if (this.socket) {
        this.socket.close();
    }
    
    this.deinitialize();
    
    const status = document.getElementById('status');
    if (status) {
        status.style.display = 'block';
        status.className = 'status-indicator status-disconnected';
        status.textContent = 'Disconnected';
    }
};

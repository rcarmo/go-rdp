/**
 * RDP Client Core Module
 * Main Client constructor and core initialization
 */
console.log('[Module] core.js loaded');

function Client(websocketURL, canvasID, hostID, userID, passwordID) {
    console.log('[Client] Constructor called');
    this.websocketURL = websocketURL;
    this.canvas = document.getElementById(canvasID);
    this.hostEl = document.getElementById(hostID);
    this.userEl = document.getElementById(userID);
    this.passwordEl = document.getElementById(passwordID);
    this.ctx = this.canvas.getContext("2d");
    this.pointerCacheCanvas = document.getElementById("pointer-cache");
    this.pointerCacheCanvasCtx = this.pointerCacheCanvas.getContext("2d");
    this.connected = false;
    this.canvasShown = false;
    this.pointerCache = {};
    this.isDragging = false;
    this.dragButton = null;
    
    // Session persistence and reconnection
    this.reconnectAttempts = 0;
    this.maxReconnectAttempts = 5;
    this.reconnectDelay = 2000;
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

    // Bind input handlers
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
    
    // Input event throttling
    this.inputQueue = [];
    this.inputFlushPending = false;
    this.lastMouseMove = null;
    this.mouseThrottleMs = 16; // ~60fps max
    this.lastMouseSendTime = 0;
    
    // Touch event handlers
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
    
    // Audio input redirection
    this.audioContext = null;
    this.microphoneStream = null;
    this.mediaRecorder = null;
    this.audioChunks = [];
    this.audioRedirectEnabled = false;
    this.microphonePermission = 'prompt';
    
    // Bind core handlers
    this.initialize = this.initialize.bind(this);
    this.handleMessage = this.handleMessage.bind(this);
    this.deinitialize = this.deinitialize.bind(this);
    
    // Server capabilities (received after connection)
    this.serverCapabilities = null;
    
    // Bitmap update logging flag
    this.bitmapUpdateLogged = false;
    
    // Load saved session
    this.loadSession();
    
    // Auto-reconnect if session exists
    if (this.shouldAutoReconnect()) {
        this.scheduleReconnect(100);
    }
}

// Generate unique session ID
Client.prototype.generateSessionId = function() {
    return 'session_' + Date.now() + '_' + Math.random().toString(36).substr(2, 9);
};

// Emit custom events
Client.prototype.emitEvent = function(name, detail = {}) {
    detail.sessionId = this.sessionId;
    detail.timestamp = new Date().toISOString();
    
    const event = new CustomEvent('rdp:' + name, { detail });
    document.dispatchEvent(event);
};

// Input validation helpers
Client.prototype.sanitizeInput = function(input) {
    if (typeof input !== 'string') return '';
    return input.replace(/[<>'"&]/g, '');
};

Client.prototype.validateHostname = function(hostname) {
    if (!hostname || typeof hostname !== 'string') return false;
    const hostnameRegex = /^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?)*$/;
    const ipRegex = /^(\d{1,3}\.){3}\d{1,3}$/;
    return hostnameRegex.test(hostname) || ipRegex.test(hostname);
};

Client.prototype.generateCSRFToken = function() {
    const array = new Uint8Array(32);
    crypto.getRandomValues(array);
    return Array.from(array, byte => byte.toString(16).padStart(2, '0')).join('');
};

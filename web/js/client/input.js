/**
 * RDP Client Input Module
 * Keyboard, mouse, and touch event handling
 */

// Utility functions
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
    // Browser: 0=left, 1=middle, 2=right
    // RDP: 1=BUTTON1 (left), 2=BUTTON2 (right), 3=BUTTON3 (middle)
    switch(button) {
        case 0: return 1;  // Left click
        case 1: return 3;  // Middle click
        case 2: return 2;  // Right click
        default: return 1;
    }
}

// Convert screen coordinates to desktop coordinates
Client.prototype.screenToDesktop = function (screenX, screenY) {
    const offset = elementOffset(this.canvas);
    const x = Math.floor(screenX - offset.left);
    const y = Math.floor(screenY - offset.top);
    return { x: x, y: y };
};

// Queue an input event for sending
Client.prototype.queueInput = function(data, isMouseMove) {
    if (isMouseMove) {
        this.lastMouseMove = data;
    } else {
        this.inputQueue.push(data);
    }
    
    if (!this.inputFlushPending) {
        this.inputFlushPending = true;
        setTimeout(() => this.flushInputQueue(), 0);
    }
};

// Flush queued input events to server
Client.prototype.flushInputQueue = function() {
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
    if (this.lastMouseMove && (now - this.lastMouseSendTime) >= this.mouseThrottleMs) {
        try {
            this.socket.send(this.lastMouseMove);
            this.lastMouseSendTime = now;
        } catch (e) {
            // Ignore errors
        }
        this.lastMouseMove = null;
    } else if (this.lastMouseMove) {
        if (!this.inputFlushPending) {
            this.inputFlushPending = true;
            setTimeout(() => this.flushInputQueue(), this.mouseThrottleMs);
        }
    }
};

// Handle window resize
Client.prototype.handleResize = function () {
    if (!this.connected) {
        return;
    }
    
    const windowWidth = window.innerWidth;
    const windowHeight = window.innerHeight;
    
    if (Math.abs(windowWidth - this.originalWidth) < 10 && 
        Math.abs(windowHeight - this.originalHeight) < 10) {
        return;
    }
    
    if (this.resizeTimeout) {
        clearTimeout(this.resizeTimeout);
    }
    
    this.resizeTimeout = setTimeout(() => {
        this.showUserInfo('Resizing session to ' + windowWidth + 'x' + windowHeight + '...');
        
        this.manualDisconnect = true;
        this.disconnect();
        
        setTimeout(() => {
            this.manualDisconnect = false;
            this.connect();
        }, 500);
    }, 1000);
};

// Keyboard handlers
Client.prototype.handleKeyDown = function (e) {
    if (!this.connected) {
        this.showUserError('Cannot send keystrokes: not connected to server');
        return;
    }

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

    this.updateActivity();

    const event = new KeyboardEventKeyUp(e.code);

    if (event.keyCode === undefined) {
        Logger.debug("[Input] Undefined key up:", e.code);
        e.preventDefault();
        return false;
    }

    const data = event.serialize();
    this.queueInput(data, false);

    e.preventDefault();
    return false;
};

// Mouse handlers
Client.prototype.handleMouseMove = function (e) {
    if (!this.lastActivityUpdate || Date.now() - this.lastActivityUpdate > 1000) {
        this.updateActivity();
        this.lastActivityUpdate = Date.now();
    }

    try {
        const pos = this.screenToDesktop(e.clientX, e.clientY);
        const event = new MouseMoveEvent(pos.x, pos.y, this.isDragging ? this.dragButton : null);
        const data = event.serialize();
        this.queueInput(data, !this.isDragging);
    } catch (error) {
        this.logError('Mouse move error', {x: e.clientX, y: e.clientY, error: error.message});
    }

    e.preventDefault();
    return false;
};

Client.prototype.handleMouseDown = function (e) {
    this.updateActivity();
    
    this.isDragging = true;
    this.dragButton = mouseButtonMap(e.button);

    const pos = this.screenToDesktop(e.clientX, e.clientY);
    Logger.debug('[Mouse] Down at', pos.x, pos.y, 'button:', this.dragButton);
    const event = new MouseDownEvent(pos.x, pos.y, this.dragButton);
    const data = event.serialize();
    
    this.queueInput(data, false);
    
    document.addEventListener('mousemove', this.handleMouseMove);
    document.addEventListener('mouseup', this.handleMouseUp);

    e.preventDefault();
    return false;
};

Client.prototype.handleMouseUp = function (e) {
    const button = this.dragButton || mouseButtonMap(e.button);
    this.isDragging = false;
    this.dragButton = null;
    
    const pos = this.screenToDesktop(e.clientX, e.clientY);
    Logger.debug('[Mouse] Up at', pos.x, pos.y, 'button:', button);
    const event = new MouseUpEvent(pos.x, pos.y, button);
    const data = event.serialize();
    
    this.queueInput(data, false);
    
    document.removeEventListener('mousemove', this.handleMouseMove);
    document.removeEventListener('mouseup', this.handleMouseUp);

    e.preventDefault();
    return false;
};

Client.prototype.handleWheel = function (e) {
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
};

// Touch event handlers for mobile support
Client.prototype.handleTouchStart = function (e) {
    if (!this.connected) {
        return;
    }
    
    e.preventDefault();
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
    
    e.preventDefault();
    
    if (!this.lastTouchUpdate || Date.now() - this.lastTouchUpdate > 16) {
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

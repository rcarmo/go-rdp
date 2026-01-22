/**
 * Input handling for RDP client
 * Handles keyboard, mouse, and touch events
 * @module input
 */

import { Logger } from './logger.js';
import { 
    KeyboardEventKeyDown, 
    KeyboardEventKeyUp, 
    MouseMoveEvent, 
    MouseDownEvent, 
    MouseUpEvent, 
    MouseWheelEvent 
} from './protocol.js';

/**
 * Calculate element offset from document
 * @param {HTMLElement} el
 * @returns {{top: number, left: number}}
 */
export function elementOffset(el) {
    let x = 0;
    let y = 0;

    while (el && !isNaN(el.offsetLeft) && !isNaN(el.offsetTop)) {
        x += el.offsetLeft - el.scrollLeft;
        y += el.offsetTop - el.scrollTop;
        el = el.offsetParent;
    }

    return { top: y, left: x };
}

/**
 * Map browser mouse button codes to RDP button codes
 * Browser: 0=left, 1=middle, 2=right
 * RDP: 1=BUTTON1 (left), 2=BUTTON2 (right), 3=BUTTON3 (middle)
 * @param {number} button
 * @returns {number}
 */
export function mouseButtonMap(button) {
    switch (button) {
        case 0: return 1;  // Left click
        case 1: return 3;  // Middle click
        case 2: return 2;  // Right click
        default: return 1; // Default to left click
    }
}

/**
 * Input handling mixin - adds input functionality to Client
 */
export const InputMixin = {
    /**
     * Initialize input handling
     */
    initInput() {
        this.isDragging = false;
        this.inputQueue = [];
        this.inputFlushPending = false;
        this.lastMouseMove = null;
        this.mouseThrottleMs = 16; // ~60fps max for mouse moves
        this.lastMouseSendTime = 0;
        this.lastActivityUpdate = null;
        this.lastTouchUpdate = null;
        
        // Bind event handlers
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
        
        // Send queued click/key events first
        while (this.inputQueue.length > 0) {
            const data = this.inputQueue.shift();
            try {
                this.socket.send(data);
            } catch (e) {
                this.inputQueue = [];
                break;
            }
        }
        
        // Send latest mouse position if enough time has passed
        const now = Date.now();
        if (this.lastMouseMove && (now - this.lastMouseSendTime) >= this.mouseThrottleMs) {
            try {
                this.socket.send(this.lastMouseMove);
                this.lastMouseSendTime = now;
            } catch (e) {
                // Ignore
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
            this.showUserError('Cannot send keystrokes: not connected to server');
            return;
        }

        this.updateActivity();

        const event = new KeyboardEventKeyDown(e.code);

        if (event.keyCode === undefined) {
            this.logError('Key mapping error', { code: e.code, key: e.key });
            this.showUserError('Unsupported key: ' + e.key);
            e.preventDefault();
            return false;
        }

        try {
            const data = event.serialize();
            this.queueInput(data, false);
        } catch (error) {
            this.logError('Key send error', { code: e.code, error: error.message });
            this.showUserError('Failed to send keystroke');
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

        if (event.keyCode === undefined) {
            Logger.debug("[Input] Undefined key up:", e.code);
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
        if (!this.lastActivityUpdate || Date.now() - this.lastActivityUpdate > 1000) {
            this.updateActivity();
            this.lastActivityUpdate = Date.now();
        }

        try {
            const pos = this.screenToDesktop(e.clientX, e.clientY);
            const event = new MouseMoveEvent(pos.x, pos.y);
            const data = event.serialize();
            this.queueInput(data, true);
        } catch (error) {
            this.logError('Mouse move error', { x: e.clientX, y: e.clientY, error: error.message });
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
        
        document.addEventListener('mousemove', this.handleMouseMove);
        document.addEventListener('mouseup', this.handleMouseUp);

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
        
        document.removeEventListener('mousemove', this.handleMouseMove);
        document.removeEventListener('mouseup', this.handleMouseUp);

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
        if (!this.connected) return;
        
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
    },
    
    /**
     * Handle touch move event
     * @param {TouchEvent} e
     */
    handleTouchMove(e) {
        if (!this.connected) return;
        
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
    },
    
    /**
     * Handle touch end event
     * @param {TouchEvent} e
     */
    handleTouchEnd(e) {
        if (!this.connected) return;
        
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
        this.canvas.addEventListener("contextmenu", (e) => { e.preventDefault(); return false; });
        
        // Touch events for mobile
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
        
        document.removeEventListener('mousemove', this.handleMouseMove);
        document.removeEventListener('mouseup', this.handleMouseUp);
    }
};

export default InputMixin;

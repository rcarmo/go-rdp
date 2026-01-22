/**
 * Session management for RDP client
 * Handles persistence, reconnection, and timeout tracking
 * @module session
 */

import { Logger } from './logger.js';

/**
 * Generate a unique session ID using cryptographically secure random values
 * @returns {string}
 */
export function generateSessionId() {
    const array = new Uint8Array(16);
    crypto.getRandomValues(array);
    const hex = Array.from(array, byte => byte.toString(16).padStart(2, '0')).join('');
    return 'session_' + hex;
}

/**
 * Session manager mixin - adds session functionality to Client
 */
export const SessionMixin = {
    /**
     * Initialize session management
     */
    initSession() {
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = 5;
        this.reconnectDelay = 2000;
        this.reconnectTimeout = null;
        this.lastConnectionTime = null;
        this.manualDisconnect = false;
        this.sessionId = generateSessionId();
        
        // Session timeout and idle detection
        this.maxSessionTime = 8 * 60 * 60 * 1000; // 8 hours
        this.maxIdleTime = 30 * 60 * 1000; // 30 minutes
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
        return false; // Disabled - user must manually reconnect
    },
    
    /**
     * Save session data to cookies
     */
    saveSession() {
        try {
            // 7-day expiry for session cookies (not password - just host/user)
            const expires = new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toUTCString();
            document.cookie = `rdp_host=${encodeURIComponent(this.hostEl.value)}; expires=${expires}; path=/; SameSite=Strict`;
            document.cookie = `rdp_user=${encodeURIComponent(this.userEl.value)}; expires=${expires}; path=/; SameSite=Strict`;
        } catch (e) {
            Logger.warn("Session", `Failed to save: ${e.message}`);
        }
    },
    
    /**
     * Load session data from cookies
     */
    loadSession() {
        try {
            const cookies = document.cookie.split(';').reduce((acc, cookie) => {
                const [key, value] = cookie.trim().split('=');
                if (key && value) acc[key] = decodeURIComponent(value);
                return acc;
            }, {});
            
            if (cookies.rdp_host) this.hostEl.value = cookies.rdp_host;
            if (cookies.rdp_user) this.userEl.value = cookies.rdp_user;
        } catch (e) {
            Logger.debug("Session", `Failed to load: ${e.message}`);
        }
    },
    
    /**
     * Verify session data integrity
     * @param {Object} session
     * @returns {boolean}
     */
    verifySessionIntegrity(session) {
        const requiredFields = ['host', 'user', 'timestamp', 'sessionId'];
        return requiredFields.every(field => session.hasOwnProperty(field));
    },
    
    /**
     * Clear session data
     */
    clearSession() {
        document.cookie = 'rdp_host=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;';
        document.cookie = 'rdp_user=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;';
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
                Logger.debug("Session", `Reconnect attempt ${this.reconnectAttempts + 1}/${this.maxReconnectAttempts}`);
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
        
        // Close any existing socket before creating a new one
        if (this.socket && this.socket.readyState !== WebSocket.CLOSED) {
            try {
                this.socket.close();
            } catch (e) {
                // Ignore close errors
            }
        }
        
        // Build URL with non-sensitive parameters only (NO password in URL!)
        const url = new URL(this.websocketURL);
        url.searchParams.set('width', this.canvas.width);
        url.searchParams.set('height', this.canvas.height);
        url.searchParams.set('sessionId', this.sessionId);
        
        // Get password from input (don't persist it)
        const password = this.passwordEl ? this.passwordEl.value : '';

        this.socket = new WebSocket(url.toString());
        this.socket.binaryType = 'arraybuffer';
        this.socket.onopen = () => {
            // Send credentials securely via WebSocket message (not URL)
            const credMsg = JSON.stringify({
                type: 'credentials',
                host: this.hostEl.value,
                user: this.userEl.value,
                password: password
            });
            this.socket.send(credMsg);
            this.initialize();
        };
        this.socket.onmessage = (e) => {
            // With binaryType='arraybuffer', e.data is already an ArrayBuffer
            if (e.data instanceof ArrayBuffer) {
                this.handleMessage(e.data);
            } else if (e.data instanceof Blob) {
                e.data.arrayBuffer()
                    .then((arrayBuffer) => this.handleMessage(arrayBuffer))
                    .catch((err) => Logger.error('Session', `Failed to read message: ${err.message}`));
            }
        };
        this.socket.onerror = (e) => {
            Logger.warn("Session", "Reconnection error");
        };
        this.socket.onclose = (e) => {
            if (!this.manualDisconnect && this.reconnectAttempts < this.maxReconnectAttempts) {
                const exponent = Math.max(0, this.reconnectAttempts - 1);
                const exponentialDelay = this.reconnectDelay * Math.pow(2, exponent);
                this.scheduleReconnect(Math.min(exponentialDelay, 30000));
            }
        };
    },
    
    /**
     * Start session and idle timeout tracking
     */
    startTimeoutTracking() {
        this.lastConnectionTime = Date.now();
        this.lastActivityTime = Date.now();
        
        // Session timeout
        this.sessionTimeout = setTimeout(() => {
            this.handleSessionTimeout();
        }, this.maxSessionTime);
        
        // Initial idle check
        this.resetIdleTimeout();
    },
    
    /**
     * Update activity timestamp
     */
    updateActivity() {
        this.lastActivityTime = Date.now();
        this.warningShown = false;
        
        // Hide any idle warning
        const warning = document.getElementById('idle-warning');
        if (warning) {
            warning.style.display = 'none';
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
        
        // Warning 5 minutes before timeout
        this.warningTimeout = setTimeout(() => {
            this.showIdleWarning();
        }, this.maxIdleTime - 5 * 60 * 1000);
        
        this.idleTimeout = setTimeout(() => {
            this.handleIdleTimeout();
        }, this.maxIdleTime);
    },
    
    /**
     * Show idle warning to user
     */
    showIdleWarning() {
        if (this.warningShown) return;
        this.warningShown = true;
        
        let warning = document.getElementById('idle-warning');
        if (!warning) {
            warning = document.createElement('div');
            warning.id = 'idle-warning';
            warning.className = 'idle-warning';
            warning.innerHTML = 'Session will disconnect in 5 minutes due to inactivity. Move mouse or press a key to stay connected.';
            document.body.appendChild(warning);
        }
        warning.style.display = 'block';
    },
    
    /**
     * Handle idle timeout
     */
    handleIdleTimeout() {
        Logger.debug("Session", "Disconnected due to inactivity");
        this.showUserWarning('Session disconnected due to inactivity');
        this.disconnect();
    },
    
    /**
     * Handle session timeout
     */
    handleSessionTimeout() {
        Logger.debug("Session", "Maximum session time reached (8 hours)");
        this.showUserWarning('Session disconnected - maximum session time reached (8 hours)');
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

export default SessionMixin;

/**
 * RDP Client Session Module
 * Session persistence, reconnection, and timeout handling
 */

// Check if auto-reconnect should be attempted
Client.prototype.shouldAutoReconnect = function() {
    // Auto-reconnect disabled - user must manually reconnect
    return false;
};

// Save session data to cookies
Client.prototype.saveSession = function() {
    try {
        const expires = new Date(Date.now() + 30 * 24 * 60 * 60 * 1000).toUTCString();
        document.cookie = `rdp_host=${encodeURIComponent(this.hostEl.value)}; expires=${expires}; path=/; SameSite=Strict`;
        document.cookie = `rdp_user=${encodeURIComponent(this.userEl.value)}; expires=${expires}; path=/; SameSite=Strict`;
    } catch (e) {
        console.error('Failed to save session:', e);
    }
};

// Load session data from cookies
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

// Verify session data integrity
Client.prototype.verifySessionIntegrity = function(session) {
    const requiredFields = ['host', 'user', 'timestamp', 'sessionId'];
    return requiredFields.every(field => session.hasOwnProperty(field));
};

// Clear saved session data
Client.prototype.clearSession = function() {
    document.cookie = 'rdp_host=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;';
    document.cookie = 'rdp_user=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;';
    this.manualDisconnect = true;
};

// Schedule a reconnection attempt
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

// Attempt to reconnect to the RDP server
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
            this.scheduleReconnect(Math.min(exponentialDelay, 30000));
        }
        this.deinitialize();
    };
};

// Start tracking session and idle timeouts
Client.prototype.startTimeoutTracking = function() {
    this.lastActivityTime = Date.now();
    this.clearAllTimeouts();
    
    this.sessionTimeout = setTimeout(() => {
        this.handleSessionTimeout();
    }, this.maxSessionTime);
    
    this.warningTimeout = setTimeout(() => {
        this.showIdleWarning();
    }, this.maxIdleTime - 5 * 60 * 1000);
    
    this.idleTimeout = setTimeout(() => {
        this.handleIdleTimeout();
    }, this.maxIdleTime);
};

// Update activity timestamp and reset idle timer
Client.prototype.updateActivity = function() {
    this.lastActivityTime = Date.now();
    
    if (this.idleTimeout) {
        clearTimeout(this.idleTimeout);
    }
    if (this.warningTimeout) {
        clearTimeout(this.warningTimeout);
    }
    this.warningShown = false;
    
    this.warningTimeout = setTimeout(() => {
        this.showIdleWarning();
    }, this.maxIdleTime - 5 * 60 * 1000);
    
    this.idleTimeout = setTimeout(() => {
        this.handleIdleTimeout();
    }, this.maxIdleTime);
};

// Show idle warning message
Client.prototype.showIdleWarning = function() {
    if (this.warningShown) return;
    this.warningShown = true;
    
    const message = 'Session will timeout due to inactivity in 5 minutes. Move your mouse or press any key to continue.';
    this.showUserWarning(message);
    
    setTimeout(() => {
        if (this.warningShown) {
            this.warningShown = false;
        }
    }, 10000);
};

// Handle idle timeout
Client.prototype.handleIdleTimeout = function() {
    const status = document.getElementById('status');
    if (status) {
        status.style.display = 'block';
        status.className = 'status-indicator status-disconnected';
        status.textContent = 'Session disconnected due to inactivity. Please reconnect to continue.';
    }
    this.disconnect();
};

// Handle session timeout
Client.prototype.handleSessionTimeout = function() {
    const status = document.getElementById('status');
    if (status) {
        status.style.display = 'block';
        status.className = 'status-indicator status-disconnected';
        status.textContent = 'Session expired due to time limit. Please reconnect to continue.';
    }
    this.disconnect();
    this.clearSession();
};

// Clear all timeout handlers
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

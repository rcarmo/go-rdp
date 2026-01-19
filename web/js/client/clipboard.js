/**
 * RDP Client Clipboard Module
 * Clipboard synchronization between local and remote sessions
 */

// Initialize clipboard support
Client.prototype.initClipboardSupport = function() {
    if (navigator.clipboard && navigator.clipboard.readText && navigator.clipboard.writeText) {
        this.clipboardApiSupported = true;
        this.setupClipboardSync();
    } else {
        this.clipboardApiSupported = false;
        this.showUserWarning('Clipboard synchronization not supported in this browser');
    }
};

// Set up clipboard event listeners
Client.prototype.setupClipboardSync = function() {
    document.addEventListener('paste', (e) => {
        this.handleLocalPaste(e);
    });
    
    document.addEventListener('focus', () => {
        this.requestRemoteClipboard();
    });
};

// Handle paste from local clipboard
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

// Send clipboard content to remote session
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

// Request clipboard content from remote session
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

// Handle clipboard data from remote session
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

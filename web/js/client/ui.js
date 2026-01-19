/**
 * RDP Client UI Module
 * User notifications and status display
 */

// Show error message to user
Client.prototype.showUserError = function(message) {
    if (window.showToast) {
        window.showToast(message, 'error', 'Connection Error', 8000);
    }
    
    const status = document.getElementById('status');
    if (status) {
        status.style.display = 'block';
        status.className = 'status-indicator status-disconnected';
        status.textContent = message;
        
        setTimeout(() => {
            if (status.textContent === message) {
                status.style.display = 'none';
            }
        }, 10000);
    }
};

// Show success message to user
Client.prototype.showUserSuccess = function(message) {
    if (window.showToast) {
        window.showToast(message, 'success', 'Success', 5000);
    }
    
    const status = document.getElementById('status');
    if (status) {
        status.style.display = 'block';
        status.className = 'status-indicator status-connected';
        status.textContent = message;
        
        setTimeout(() => {
            if (status.textContent === message) {
                status.style.display = 'none';
            }
        }, 5000);
    }
};

// Show warning message to user
Client.prototype.showUserWarning = function(message) {
    if (window.showToast) {
        window.showToast(message, 'info', 'Warning', 6000);
    }
};

// Show info message to user
Client.prototype.showUserInfo = function(message) {
    if (window.showToast) {
        window.showToast(message, 'info', 'Info', 4000);
    }
};

// Centralized error logging
Client.prototype.logError = function(context, details) {
    const errorInfo = {
        context: context,
        details: details,
        timestamp: new Date().toISOString(),
        sessionId: this.sessionId,
        userAgent: navigator.userAgent
    };
    
    console.error('RDP Client Error:', errorInfo);
};

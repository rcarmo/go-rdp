/**
 * User interface utilities for RDP client
 * Handles notifications, status updates, and UI helpers
 * @module ui
 */

import { Logger } from './logger.js';

/**
 * UI utilities mixin - adds UI functionality to Client
 */
export const UIMixin = {
    /**
     * Initialize UI state
     */
    initUI() {
        this.csrfToken = null;
    },
    
    /**
     * Sanitize user input
     * @param {string} input
     * @returns {string}
     */
    sanitizeInput(input) {
        if (!input) return '';
        return input.replace(/[<>'"&]/g, '').trim();
    },
    
    /**
     * Validate hostname format
     * @param {string} hostname
     * @returns {boolean}
     */
    validateHostname(hostname) {
        if (!hostname) return false;
        const pattern = /^([a-zA-Z0-9.-]+|(\d{1,3}\.){3}\d{1,3})(:\d{1,5})?$/;
        return pattern.test(hostname) && hostname.length <= 253;
    },
    
    /**
     * Generate CSRF token
     * @returns {string}
     */
    generateCSRFToken() {
        return crypto.getRandomValues(new Uint8Array(16))
            .reduce((hex, byte) => hex + byte.toString(16).padStart(2, '0'), '');
    },
    
    /**
     * Show error message to user
     * @param {string} message
     */
    showUserError(message) {
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
    },
    
    /**
     * Show success message to user
     * @param {string} message
     */
    showUserSuccess(message) {
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
    },
    
    /**
     * Show warning message to user
     * @param {string} message
     */
    showUserWarning(message) {
        if (window.showToast) {
            window.showToast(message, 'info', 'Warning', 6000);
        }
    },
    
    /**
     * Show info message to user
     * @param {string} message
     */
    showUserInfo(message) {
        if (window.showToast) {
            window.showToast(message, 'info', 'Info', 4000);
        }
    },
    
    /**
     * Log error with context
     * @param {string} context
     * @param {Object} details
     */
    logError(context, details) {
        Logger.error("RDP", `${context}:`, details);
    },
    
    /**
     * Emit custom event
     * @param {string} name
     * @param {Object} detail
     */
    emitEvent(name, detail = {}) {
        try {
            document.dispatchEvent(new CustomEvent('rdp:' + name, {detail: detail}));
        } catch (error) {
            Logger.debug("Event", `Dispatch failed: ${error.message}`);
        }
    }
};

export default UIMixin;

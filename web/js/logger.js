// Centralized logging utility
// Enable/disable debug logging via localStorage or global variable
const Logger = {
    _debugEnabled: false,

    init: function() {
        // Check localStorage for debug setting
        const storedDebug = localStorage.getItem('rdp_debug');
        if (storedDebug !== null) {
            this._debugEnabled = storedDebug === 'true';
        }
        
        // Also check global window variable for runtime control
        if (typeof window.RDP_DEBUG !== 'undefined') {
            this._debugEnabled = window.RDP_DEBUG;
        }
        
        if (this._debugEnabled) {
            console.log('[Logger] Debug logging enabled');
        }
    },

    enable: function() {
        this._debugEnabled = true;
        localStorage.setItem('rdp_debug', 'true');
        console.log('[Logger] Debug logging enabled');
    },

    disable: function() {
        this._debugEnabled = false;
        localStorage.setItem('rdp_debug', 'false');
        console.log('[Logger] Debug logging disabled');
    },

    isEnabled: function() {
        return this._debugEnabled;
    },

    debug: function(...args) {
        if (this._debugEnabled) {
            console.log(...args);
        }
    },

    info: function(...args) {
        console.log(...args);
    },

    warn: function(...args) {
        console.warn(...args);
    },

    error: function(...args) {
        console.error(...args);
    }
};

// Initialize on load
Logger.init();

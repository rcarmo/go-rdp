/**
 * Simple Logger utility for RDP client
 * @module logger
 */

export const Logger = {
    enabled: false,
    
    debug: function(...args) {
        if (this.enabled) console.log(...args);
    },
    
    info: function(...args) {
        if (this.enabled) console.info(...args);
    },
    
    warn: function(...args) {
        console.warn(...args);
    },
    
    error: function(...args) {
        console.error(...args);
    },
    
    enable: function() {
        this.enabled = true;
    },
    
    disable: function() {
        this.enabled = false;
    }
};

export default Logger;

/**
 * Simple Logger utility for RDP client
 * Provides leveled logging with configurable verbosity
 * @module logger
 */

/**
 * Log levels
 */
const LogLevel = {
    DEBUG: 0,
    INFO: 1,
    WARN: 2,
    ERROR: 3,
    NONE: 4
};

export const Logger = {
    level: LogLevel.INFO,  // Default to INFO level
    
    /**
     * Set log level from string
     * @param {string} levelStr - 'debug', 'info', 'warn', 'error', 'none'
     */
    setLevel(levelStr) {
        const levels = {
            'debug': LogLevel.DEBUG,
            'info': LogLevel.INFO,
            'warn': LogLevel.WARN,
            'warning': LogLevel.WARN,
            'error': LogLevel.ERROR,
            'none': LogLevel.NONE
        };
        this.level = levels[levelStr.toLowerCase()] ?? LogLevel.INFO;
    },
    
    /**
     * Log debug message (protocol details, byte dumps)
     * @param {string} category - Log category (e.g., 'Cursor', 'Bitmap')
     * @param {...any} args - Log arguments
     */
    debug(category, ...args) {
        if (this.level <= LogLevel.DEBUG) {
            console.log(`[${category}]`, ...args);
        }
    },
    
    /**
     * Log info message (connection state, key events)
     * @param {string} category - Log category
     * @param {...any} args - Log arguments
     */
    info(category, ...args) {
        if (this.level <= LogLevel.INFO) {
            console.info(`[${category}]`, ...args);
        }
    },
    
    /**
     * Log warning message (recoverable issues)
     * @param {string} category - Log category
     * @param {...any} args - Log arguments
     */
    warn(category, ...args) {
        if (this.level <= LogLevel.WARN) {
            console.warn(`[${category}]`, ...args);
        }
    },
    
    /**
     * Log error message (failures)
     * @param {string} category - Log category
     * @param {...any} args - Log arguments
     */
    error(category, ...args) {
        if (this.level <= LogLevel.ERROR) {
            console.error(`[${category}]`, ...args);
        }
    },
    
    /**
     * Enable debug logging (convenience method)
     */
    enableDebug() {
        this.level = LogLevel.DEBUG;
    },
    
    /**
     * Disable all logging except errors
     */
    quiet() {
        this.level = LogLevel.ERROR;
    }
};

// Export LogLevel for external use
export { LogLevel };

export default Logger;

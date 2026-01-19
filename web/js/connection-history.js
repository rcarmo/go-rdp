// Connection History Manager
// Persists recent connections to localStorage
const ConnectionHistory = {
    _storageKey: 'rdp_connection_history',
    _maxHistory: 5,

    // Save a connection to history
    save: function(host, username) {
        if (!host || !username) return;

        const history = this.get();
        
        // Create connection entry (timestamp for sorting)
        const entry = {
            host: host,
            username: username,
            timestamp: Date.now()
        };

        // Remove duplicate if exists (same host + username)
        const filtered = history.filter(item => 
            !(item.host === host && item.username === username)
        );

        // Add to beginning
        filtered.unshift(entry);

        // Keep only max items
        const trimmed = filtered.slice(0, this._maxHistory);

        // Save to localStorage
        try {
            localStorage.setItem(this._storageKey, JSON.stringify(trimmed));
            Logger.debug('[ConnectionHistory] Saved connection:', host, username);
        } catch (e) {
            console.error('[ConnectionHistory] Failed to save:', e);
        }
    },

    // Get all connection history
    get: function() {
        try {
            const data = localStorage.getItem(this._storageKey);
            if (!data) return [];
            
            const parsed = JSON.parse(data);
            return Array.isArray(parsed) ? parsed : [];
        } catch (e) {
            console.error('[ConnectionHistory] Failed to load:', e);
            return [];
        }
    },

    // Clear all history
    clear: function() {
        try {
            localStorage.removeItem(this._storageKey);
            Logger.info('[ConnectionHistory] History cleared');
        } catch (e) {
            console.error('[ConnectionHistory] Failed to clear:', e);
        }
    },

    // Remove a specific entry
    remove: function(host, username) {
        const history = this.get();
        const filtered = history.filter(item => 
            !(item.host === host && item.username === username)
        );
        
        try {
            localStorage.setItem(this._storageKey, JSON.stringify(filtered));
            Logger.debug('[ConnectionHistory] Removed connection:', host, username);
        } catch (e) {
            console.error('[ConnectionHistory] Failed to remove:', e);
        }
    }
};

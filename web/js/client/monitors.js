/**
 * RDP Client Monitor Module
 * Multi-monitor detection and management
 */

// Detect available monitors
Client.prototype.detectMonitors = function(showSuccessMessage) {
    if (window.screen && window.screen.width && window.screen.height) {
        this.monitors = [{
            id: 0,
            width: window.screen.width,
            height: window.screen.height,
            availWidth: window.screen.availWidth || window.screen.width,
            availHeight: window.screen.availHeight || window.screen.height,
            isPrimary: true
        }];
        
        this.virtualDesktopWidth = this.monitors[0].width;
        this.virtualDesktopHeight = this.monitors[0].height;
        this.multiMonitorMode = false;
        
        return this.monitors;
    }
    
    // Fallback for older browsers
    this.monitors = [{
        id: 0,
        width: 1024,
        height: 768,
        availWidth: 1024,
        availHeight: 768,
        isPrimary: true
    }];
    
    return this.monitors;
};

// Add monitor selection controls to UI
Client.prototype.addMonitorControls = function() {
    const form = document.querySelector('#connection-form');
    if (!form) return;
    
    const monitorControls = document.createElement('div');
    monitorControls.className = 'monitor-controls';
    monitorControls.innerHTML = `
        <div class="form-group">
            <label for="monitor-select">Display Monitor</label>
            <select id="monitor-select" class="monitor-select">
                <option value="0">Primary Monitor</option>
            </select>
            <button type="button" id="cycle-monitor-btn" class="btn btn-primary" style="margin-left: 8px;">
                🖥 Cycle Monitor
            </button>
        </div>
        <div class="monitor-info" id="monitor-info">
            <strong>Display Info:</strong> 
            <span id="monitor-details">Detecting...</span>
        </div>
    `;
    
    const buttonGroup = form.querySelector('.button-group');
    if (buttonGroup) {
        form.insertBefore(monitorControls, buttonGroup);
    } else {
        form.appendChild(monitorControls);
    }
    
    this.updateMonitorOptions();
    
    document.getElementById('monitor-select').addEventListener('change', (e) => {
        this.switchToMonitor(parseInt(e.target.value));
    });
    
    document.getElementById('cycle-monitor-btn').addEventListener('click', () => {
        this.cycleToNextMonitor();
    });
};

// Update monitor dropdown options
Client.prototype.updateMonitorOptions = function() {
    const select = document.getElementById('monitor-select');
    if (!select) return;
    
    select.innerHTML = '';
    
    this.monitors.forEach((monitor, index) => {
        const option = document.createElement('option');
        option.value = monitor.id;
        option.textContent = `Monitor ${monitor.id + 1} (${monitor.width}x${monitor.height})`;
        if (monitor.id === this.currentMonitor) {
            option.selected = true;
        }
        select.appendChild(option);
    });
    
    this.updateMonitorInfo();
};

// Update monitor info display
Client.prototype.updateMonitorInfo = function() {
    const info = document.getElementById('monitor-info');
    const details = document.getElementById('monitor-details');
    if (!info || !details) return;
    
    const monitor = this.monitors[this.currentMonitor];
    details.textContent = `${monitor.width}x${monitor.height} pixels`;
    
    if (this.multiMonitorMode) {
        details.textContent += ` (Multi-monitor mode)`;
    }
};

// Switch to a specific monitor
Client.prototype.switchToMonitor = function(monitorId) {
    if (monitorId === this.currentMonitor) return;
    
    const monitor = this.monitors.find(m => m.id === monitorId);
    if (!monitor) return;
    
    this.currentMonitor = monitorId;
    
    if (this.socket && this.connected) {
        try {
            const switchData = {
                type: 'monitor_switch',
                monitorId: monitorId,
                width: monitor.width,
                height: monitor.height,
                timestamp: Date.now()
            };
            
            this.socket.send(JSON.stringify(switchData));
            this.showUserSuccess(`Switched to Monitor ${monitorId + 1}`);
        } catch (error) {
            this.logError('Monitor switch error', {error: error.message, monitorId});
            this.showUserError('Failed to switch monitor');
        }
    }
    
    this.updateMonitorInfo();
};

// Cycle to the next monitor
Client.prototype.cycleToNextMonitor = function() {
    const nextIndex = (this.currentMonitor + 1) % this.monitors.length;
    this.switchToMonitor(this.monitors[nextIndex].id);
};

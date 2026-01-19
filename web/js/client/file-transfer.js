/**
 * RDP Client File Transfer Module
 * File upload and download functionality
 */

// Initialize file transfer support
Client.prototype.initFileTransfer = function() {
    if (window.File && window.FileReader && window.FileList) {
        this.fileApiSupported = true;
        this.setupFileTransfer();
    } else {
        this.fileApiSupported = false;
        this.showUserWarning('File transfer not supported in this browser');
    }
};

// Set up file transfer drag/drop and controls
Client.prototype.setupFileTransfer = function() {
    this.canvas.addEventListener('dragover', (e) => {
        e.preventDefault();
        e.stopPropagation();
        this.canvas.style.borderColor = '#2563eb';
        this.canvas.style.boxShadow = '0 0 10px rgba(37, 99, 235, 0.3)';
    });
    
    this.canvas.addEventListener('dragleave', (e) => {
        e.preventDefault();
        e.stopPropagation();
        this.canvas.style.borderColor = '';
        this.canvas.style.boxShadow = '';
    });
    
    this.canvas.addEventListener('drop', (e) => {
        e.preventDefault();
        e.stopPropagation();
        this.canvas.style.borderColor = '';
        this.canvas.style.boxShadow = '';
        
        const files = e.dataTransfer.files;
        if (files.length > 0) {
            this.handleFileUpload(files);
        }
    });
    
    this.addFileUploadButton();
};

// Add file upload button to UI
Client.prototype.addFileUploadButton = function() {
    const container = document.querySelector('.canvas-container');
    if (!container) return;
    
    const controls = document.createElement('div');
    controls.className = 'file-transfer-controls';
    controls.id = 'file-transfer-controls';
    controls.style.display = 'none';
    controls.innerHTML = `
        <div class="file-upload-area">
            <input type="file" id="file-input" multiple style="display: none;">
            <button class="btn btn-primary" id="file-upload-btn">
                📁 Upload Files
            </button>
            <button class="btn" id="file-upload-close" style="padding: 8px 12px;">✕</button>
            <span class="file-status" id="file-status"></span>
        </div>
    `;
    
    container.appendChild(controls);
    
    document.getElementById('file-upload-btn').addEventListener('click', () => {
        document.getElementById('file-input').click();
    });
    
    document.getElementById('file-upload-close').addEventListener('click', () => {
        this.hideFileUpload();
    });
    
    document.getElementById('file-input').addEventListener('change', (e) => {
        if (e.target.files.length > 0) {
            this.handleFileUpload(e.target.files);
        }
    });
};

// Show file upload controls
Client.prototype.showFileUpload = function() {
    const controls = document.getElementById('file-transfer-controls');
    if (controls) {
        controls.style.display = 'flex';
    }
};

// Hide file upload controls
Client.prototype.hideFileUpload = function() {
    const controls = document.getElementById('file-transfer-controls');
    if (controls) {
        controls.style.display = 'none';
    }
};

// Handle file upload
Client.prototype.handleFileUpload = function(files) {
    if (!this.fileApiSupported) {
        this.showUserError('File transfer not supported');
        return;
    }
    
    const fileStatus = document.getElementById('file-status');
    if (fileStatus) {
        fileStatus.textContent = `Preparing ${files.length} file(s)...`;
    }
    
    Array.from(files).forEach((file, index) => {
        this.sendFile(file, index);
    });
};

// Send a single file to the server
Client.prototype.sendFile = function(file, index) {
    const reader = new FileReader();
    
    reader.onload = (e) => {
        try {
            const fileData = {
                type: 'file_transfer',
                name: file.name,
                size: file.size,
                mimeType: file.type,
                data: e.target.result.split(',')[1],
                index: index,
                timestamp: Date.now()
            };
            
            this.socket.send(JSON.stringify(fileData));
            
            this.updateFileStatus(`${file.name} uploaded successfully`);
        } catch (error) {
            this.logError('File transfer error', {error: error.message, file: file.name});
            this.updateFileStatus(`Failed to upload ${file.name}`);
        }
    };
    
    reader.onerror = () => {
        this.logError('File read error', {file: file.name});
        this.updateFileStatus(`Failed to read ${file.name}`);
    };
    
    reader.readAsDataURL(file);
};

// Update file transfer status display
Client.prototype.updateFileStatus = function(message) {
    const fileStatus = document.getElementById('file-status');
    if (fileStatus) {
        fileStatus.textContent = message;
        
        setTimeout(() => {
            if (fileStatus.textContent === message) {
                fileStatus.textContent = '';
            }
        }, 5000);
    }
};

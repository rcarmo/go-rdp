/**
 * RDP Client Audio Module
 * Microphone redirection and audio capture
 */

// Initialize audio redirection
Client.prototype.initAudioRedirection = function() {
    if (navigator.mediaDevices && navigator.mediaDevices.getUserMedia && window.AudioContext) {
        this.audioApiSupported = true;
        this.setupMicrophoneControls();
    } else {
        this.audioApiSupported = false;
        this.showUserWarning('Audio redirection not supported in this browser');
    }
};

// Set up microphone UI controls
Client.prototype.setupMicrophoneControls = function() {
    const container = document.querySelector('.connection-panel');
    if (!container) return;
    
    const audioControls = document.createElement('div');
    audioControls.className = 'audio-controls';
    audioControls.innerHTML = `
        <div class="form-group">
            <label for="mic-toggle">Microphone Redirection</label>
            <div class="mic-control-group">
                <button type="button" id="mic-toggle-btn" class="btn btn-primary">
                    🎤 Enable Microphone
                </button>
                <span class="mic-status" id="mic-status">Disabled</span>
            </div>
        </div>
        <div class="mic-levels">
            <div class="level-indicator" id="mic-level-indicator"></div>
            <div class="level-bars">
                <div class="level-bar" id="mic-level-1"></div>
                <div class="level-bar" id="mic-level-2"></div>
                <div class="level-bar" id="mic-level-3"></div>
                <div class="level-bar" id="mic-level-4"></div>
                <div class="level-bar" id="mic-level-5"></div>
            </div>
        </div>
    `;
    
    const form = container.querySelector('#connection-form');
    if (form) {
        form.appendChild(audioControls);
    }
    
    document.getElementById('mic-toggle-btn').addEventListener('click', () => {
        this.toggleMicrophone();
    });
};

// Toggle microphone on/off
Client.prototype.toggleMicrophone = function() {
    const btn = document.getElementById('mic-toggle-btn');
    const status = document.getElementById('mic-status');
    
    if (!btn || !status) return;
    
    if (this.audioRedirectEnabled) {
        this.stopMicrophone();
        this.audioRedirectEnabled = false;
        btn.textContent = '🎤 Enable Microphone';
        btn.className = 'btn btn-primary';
        status.textContent = 'Disabled';
        if (window.showToast) {
            window.showToast('Microphone redirection disabled', 'info', 'Audio', 3000);
        }
    } else {
        this.startMicrophone();
    }
};

// Start microphone capture
Client.prototype.startMicrophone = async function() {
    try {
        const stream = await navigator.mediaDevices.getUserMedia({
            audio: {
                echoCancellation: true,
                noiseSuppression: true,
                sampleRate: 16000,
                channelCount: 1
            }
        });
        
        this.microphonePermission = 'granted';
        this.microphoneStream = stream;
        
        this.audioContext = new (window.AudioContext || window.webkitAudioContext)();
        this.mediaRecorder = new MediaRecorder(stream, {
            mimeType: this.getSupportedMimeType()
        });
        
        this.mediaRecorder.ondataavailable = (event) => {
            if (event.data && event.data.size > 0) {
                this.audioChunks.push(event.data);
                this.sendAudioChunk(event.data);
            }
        };
        
        this.mediaRecorder.onstart = () => {
            this.audioRedirectEnabled = true;
            this.updateMicrophoneUI();
            if (window.showToast) {
                window.showToast('Microphone redirection enabled', 'success', 'Audio', 3000);
            }
        };
        
        this.mediaRecorder.onerror = (event) => {
            this.logError('Microphone error', {error: event.error});
            this.showUserError('Microphone error: ' + event.error.message);
            this.stopMicrophone();
        };
        
        this.mediaRecorder.start(100);
        
    } catch (error) {
        if (error.name === 'NotAllowedError') {
            this.microphonePermission = 'denied';
            this.showUserError('Microphone permission denied. Please allow microphone access.');
        } else {
            this.logError('Microphone initialization error', {error: error.message});
            this.showUserError('Failed to initialize microphone: ' + error.message);
        }
    }
};

// Stop microphone capture
Client.prototype.stopMicrophone = function() {
    if (this.mediaRecorder && this.mediaRecorder.state !== 'inactive') {
        this.mediaRecorder.stop();
    }
    
    if (this.microphoneStream) {
        this.microphoneStream.getTracks().forEach(track => track.stop());
        this.microphoneStream = null;
    }
    
    if (this.audioContext) {
        this.audioContext.close();
        this.audioContext = null;
    }
    
    this.audioRedirectEnabled = false;
    this.audioChunks = [];
    this.updateMicrophoneUI();
};

// Update microphone UI state
Client.prototype.updateMicrophoneUI = function() {
    const btn = document.getElementById('mic-toggle-btn');
    const status = document.getElementById('mic-status');
    
    if (!btn || !status) return;
    
    if (this.audioRedirectEnabled) {
        btn.textContent = '🔴 Disable Microphone';
        btn.className = 'btn btn-danger';
        status.textContent = 'Active';
        this.startMicrophoneLevelIndicator();
    } else {
        btn.textContent = '🎤 Enable Microphone';
        btn.className = 'btn btn-primary';
        status.textContent = 'Disabled';
        this.stopMicrophoneLevelIndicator();
    }
};

// Start visual level indicator
Client.prototype.startMicrophoneLevelIndicator = function() {
    if (!this.audioContext || !this.microphoneStream) return;
    
    const source = this.audioContext.createMediaStreamSource(this.microphoneStream);
    const analyser = this.audioContext.createAnalyser();
    analyser.fftSize = 256;
    
    source.connect(analyser);
    
    const updateLevels = () => {
        if (!this.audioRedirectEnabled) return;
        
        const dataArray = new Uint8Array(analyser.frequencyBinCount);
        analyser.getByteFrequencyData(dataArray);
        
        const average = dataArray.reduce((sum, value) => sum + value, 0) / dataArray.length;
        const normalizedLevel = Math.min(average / 128, 1);
        
        for (let i = 1; i <= 5; i++) {
            const bar = document.getElementById(`mic-level-${i}`);
            if (bar) {
                const shouldShow = i <= (normalizedLevel * 5);
                bar.style.backgroundColor = shouldShow ? '#10b981' : '#e5e7eb';
                bar.style.transform = shouldShow ? 'scaleY(1)' : 'scaleY(0.1)';
            }
        }
        
        if (this.audioRedirectEnabled) {
            requestAnimationFrame(updateLevels);
        }
    };
    
    updateLevels();
};

// Stop visual level indicator
Client.prototype.stopMicrophoneLevelIndicator = function() {
    for (let i = 1; i <= 5; i++) {
        const bar = document.getElementById(`mic-level-${i}`);
        if (bar) {
            bar.style.backgroundColor = '#e5e7eb';
            bar.style.transform = 'scaleY(0.1)';
        }
    }
};

// Get supported audio MIME type
Client.prototype.getSupportedMimeType = function() {
    const types = ['audio/webm', 'audio/mp4', 'audio/ogg'];
    
    for (const type of types) {
        if (MediaRecorder.isTypeSupported(type)) {
            return type;
        }
    }
    
    return 'audio/webm';
};

// Send audio chunk to server
Client.prototype.sendAudioChunk = function(chunk) {
    if (!this.socket || !this.connected) {
        return;
    }
    
    try {
        const reader = new FileReader();
        reader.onload = () => {
            const audioData = {
                type: 'audio_chunk',
                data: reader.result,
                timestamp: Date.now(),
                size: chunk.size
            };
            
            this.socket.send(JSON.stringify(audioData));
        };
        
        reader.readAsDataURL(chunk);
    } catch (error) {
        this.logError('Audio chunk send error', {error: error.message});
    }
};

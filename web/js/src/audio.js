// Audio module - handles RDP audio output via Web Audio API

const AudioMixin = {
    initAudio() {
        this.audioContext = null;
        this.audioEnabled = false;
        this.audioFormat = null;
        this.audioQueue = [];
        this.audioPlaying = false;
        this.audioGain = null;
        this.audioVolume = 1.0;
        
        // Audio buffer settings
        this.audioBufferSize = 4096;
        this.audioSampleRate = 44100;
        this.audioChannels = 2;
        this.audioBitsPerSample = 16;
    },

    enableAudio() {
        if (this.audioContext) {
            return; // Already initialized
        }

        try {
            this.audioContext = new (window.AudioContext || window.webkitAudioContext)();
            this.audioGain = this.audioContext.createGain();
            this.audioGain.connect(this.audioContext.destination);
            this.audioGain.gain.value = this.audioVolume;
            this.audioEnabled = true;
            
            Logger.info('Audio', `Initialized: ${this.audioContext.sampleRate}Hz`);
        } catch (e) {
            Logger.error('Audio', `Failed to initialize: ${e.message}`);
            this.audioEnabled = false;
        }
    },

    disableAudio() {
        if (this.audioContext) {
            this.audioContext.close();
            this.audioContext = null;
        }
        this.audioEnabled = false;
        this.audioQueue = [];
        Logger.info('Audio', 'Disabled');
    },

    setAudioVolume(volume) {
        this.audioVolume = Math.max(0, Math.min(1, volume));
        if (this.audioGain) {
            this.audioGain.gain.value = this.audioVolume;
        }
    },

    handleAudioMessage(data) {
        if (!this.audioEnabled || !this.audioContext) {
            return;
        }

        // Parse audio message
        // Format: [0xFE][msgType][timestamp 2 bytes][format info if type=2][PCM data]
        if (data.length < 4) {
            return;
        }

        const view = new DataView(data.buffer, data.byteOffset, data.byteLength);
        const msgType = data[1];
        const timestamp = view.getUint16(2, true);

        let offset = 4;

        // Check for format info (msgType = 2)
        if (msgType === 0x02 && data.length >= 12) {
            const channels = view.getUint16(4, true);
            const sampleRate = view.getUint32(6, true);
            const bitsPerSample = view.getUint16(10, true);
            
            // Validate audio format parameters
            if (channels < 1 || channels > 8) {
                Logger.warn('Audio', `Invalid channel count: ${channels}`);
                return;
            }
            if (sampleRate < 8000 || sampleRate > 192000) {
                Logger.warn('Audio', `Invalid sample rate: ${sampleRate}`);
                return;
            }
            if (bitsPerSample !== 8 && bitsPerSample !== 16) {
                Logger.warn('Audio', `Unsupported bit depth: ${bitsPerSample}`);
                return;
            }
            
            this.audioChannels = channels;
            this.audioSampleRate = sampleRate;
            this.audioBitsPerSample = bitsPerSample;
            offset = 12;
            
            Logger.info('Audio', `Format: ${this.audioSampleRate}Hz ${this.audioChannels}ch ${this.audioBitsPerSample}bit`);
        }

        // Get PCM data
        const pcmData = data.slice(offset);
        if (pcmData.length === 0) {
            return;
        }

        // Queue audio for playback
        this.queueAudio(pcmData, timestamp);
    },

    queueAudio(pcmData, timestamp) {
        // Convert PCM data to Float32 samples
        const samples = this.pcmToFloat32(pcmData);
        if (!samples || samples.length === 0) {
            return;
        }

        // Create audio buffer
        const frameCount = Math.floor(samples.length / this.audioChannels);
        if (frameCount === 0) {
            return;
        }

        try {
            const audioBuffer = this.audioContext.createBuffer(
                this.audioChannels,
                frameCount,
                this.audioSampleRate
            );

            // Deinterleave channels
            for (let channel = 0; channel < this.audioChannels; channel++) {
                const channelData = audioBuffer.getChannelData(channel);
                for (let i = 0; i < frameCount; i++) {
                    channelData[i] = samples[i * this.audioChannels + channel];
                }
            }

            // Queue the buffer
            this.audioQueue.push({
                buffer: audioBuffer,
                timestamp: timestamp
            });

            // Start playback if not already playing
            if (!this.audioPlaying) {
                this.playNextAudio();
            }
        } catch (e) {
            Logger.error('Audio', `Buffer creation failed: ${e.message}`);
        }
    },

    pcmToFloat32(pcmData) {
        const view = new DataView(pcmData.buffer, pcmData.byteOffset, pcmData.byteLength);
        const samples = [];

        if (this.audioBitsPerSample === 16) {
            // 16-bit signed PCM
            const sampleCount = Math.floor(pcmData.length / 2);
            for (let i = 0; i < sampleCount; i++) {
                const sample = view.getInt16(i * 2, true);
                samples.push(sample / 32768.0);
            }
        } else if (this.audioBitsPerSample === 8) {
            // 8-bit unsigned PCM
            for (let i = 0; i < pcmData.length; i++) {
                const sample = pcmData[i];
                samples.push((sample - 128) / 128.0);
            }
        } else {
            Logger.warn('Audio', `Unsupported bit depth: ${this.audioBitsPerSample}`);
            return null;
        }

        return samples;
    },

    playNextAudio() {
        if (this.audioQueue.length === 0) {
            this.audioPlaying = false;
            return;
        }

        this.audioPlaying = true;
        const item = this.audioQueue.shift();

        const source = this.audioContext.createBufferSource();
        source.buffer = item.buffer;
        source.connect(this.audioGain);

        source.onended = () => {
            this.playNextAudio();
        };

        // Play immediately or with minimal delay
        const startTime = this.audioContext.currentTime;
        source.start(startTime);
    },

    // Resume audio context after user interaction (required by browsers)
    resumeAudioContext() {
        if (this.audioContext && this.audioContext.state === 'suspended') {
            this.audioContext.resume().then(() => {
                Logger.info('Audio', 'Context resumed');
            });
        }
    }
};

export default AudioMixin;

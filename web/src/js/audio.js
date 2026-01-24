// Audio module - handles RDP audio output via Web Audio API

import { Logger } from './logger.js';

// Audio format tags (matching WAVE format identifiers from backend)
const WAVE_FORMAT_PCM = 0x0001;
const WAVE_FORMAT_MPEGLAYER3 = 0x0055;

const AudioMixin = {
    initAudio() {
        this.audioContext = null;
        this.audioEnabled = false;
        this.audioFormat = null;
        this.audioQueue = [];
        this.audioPlaying = false;
        this.audioGain = null;
        this.audioVolume = 1.0;
        this._audioEncodingLogged = false;
        
        // Audio buffer settings
        this.audioBufferSize = 4096;
        this.audioSampleRate = 44100;
        this.audioChannels = 2;
        this.audioBitsPerSample = 16;
        this.audioFormatTag = WAVE_FORMAT_PCM; // Default to PCM
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
            
            Logger.debug('Audio', `Initialized: ${this.audioContext.sampleRate}Hz, state=${this.audioContext.state}`);
            
            // Firefox and Chrome suspend AudioContext until user interaction
            if (this.audioContext.state === 'suspended') {
                Logger.debug('Audio', 'Context suspended - will resume on user interaction');
            }
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
        Logger.debug('Audio', 'Disabled');
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
        if (msgType === 0x02 && data.length >= 14) {
            const channels = view.getUint16(4, true);
            const sampleRate = view.getUint32(6, true);
            const bitsPerSample = view.getUint16(10, true);
            const formatTag = view.getUint16(12, true);
            
            // Validate audio format parameters
            if (channels < 1 || channels > 8) {
                Logger.warn('Audio', `Invalid channel count: ${channels}`);
                return;
            }
            if (sampleRate < 8000 || sampleRate > 192000) {
                Logger.warn('Audio', `Invalid sample rate: ${sampleRate}`);
                return;
            }
            // For PCM, validate bit depth; MP3 doesn't use this field the same way
            if (formatTag === WAVE_FORMAT_PCM && bitsPerSample !== 8 && bitsPerSample !== 16) {
                Logger.warn('Audio', `Unsupported PCM bit depth: ${bitsPerSample}`);
                return;
            }
            
            this.audioChannels = channels;
            this.audioSampleRate = sampleRate;
            this.audioBitsPerSample = bitsPerSample;
            this.audioFormatTag = formatTag;
            offset = 14;
            
            const formatName = formatTag === WAVE_FORMAT_MPEGLAYER3 ? 'MP3' : 'PCM';
            const encoding = `${formatName} ${this.audioSampleRate}Hz ${this.audioChannels}ch ${this.audioBitsPerSample}bit`;
            Logger.debug('Audio', `Format: ${encoding}`);
            if (!this._audioEncodingLogged) {
                this._audioEncodingLogged = true;
                console.info(
                    '%c[RDP Session] Audio encoding',
                    'color: #03A9F4; font-weight: bold',
                    encoding
                );
            }
        } else if (msgType === 0x02 && data.length >= 12) {
            // Legacy format without formatTag - assume PCM
            const channels = view.getUint16(4, true);
            const sampleRate = view.getUint32(6, true);
            const bitsPerSample = view.getUint16(10, true);
            
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
            this.audioFormatTag = WAVE_FORMAT_PCM;
            offset = 12;
            
            const encoding = `PCM ${this.audioSampleRate}Hz ${this.audioChannels}ch ${this.audioBitsPerSample}bit`;
            Logger.debug('Audio', `Format: ${encoding}`);
            if (!this._audioEncodingLogged) {
                this._audioEncodingLogged = true;
                console.info(
                    '%c[RDP Session] Audio encoding',
                    'color: #03A9F4; font-weight: bold',
                    encoding
                );
            }
        }

        // Get audio data
        const audioData = data.slice(offset);
        if (audioData.length === 0) {
            return;
        }

        // Queue audio for playback - route based on format
        if (this.audioFormatTag === WAVE_FORMAT_MPEGLAYER3) {
            this.queueMP3Audio(audioData, timestamp);
        } else {
            this.queuePCMAudio(audioData, timestamp);
        }
    },

    queuePCMAudio(pcmData, timestamp) {
        // Convert PCM data to Float32 samples
        const samples = this.pcmToFloat32(pcmData);
        if (!samples || samples.length === 0) {
            Logger.debug('Audio', 'pcmToFloat32 returned no samples');
            return;
        }

        // Create audio buffer
        const frameCount = Math.floor(samples.length / this.audioChannels);
        if (frameCount === 0) {
            Logger.debug('Audio', 'frameCount is 0');
            return;
        }
        
        Logger.debug('Audio', `Queueing ${frameCount} PCM frames, context state: ${this.audioContext?.state}`);

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
            Logger.error('Audio', `PCM buffer creation failed: ${e.message}`);
        }
    },

    queueMP3Audio(mp3Data, timestamp) {
        // Decode MP3 data using Web Audio API's decodeAudioData
        // This is async, so we handle it with a Promise
        Logger.debug('Audio', `Decoding ${mp3Data.length} bytes of MP3 data`);

        // Create an ArrayBuffer copy for decodeAudioData
        const arrayBuffer = mp3Data.buffer.slice(
            mp3Data.byteOffset,
            mp3Data.byteOffset + mp3Data.byteLength
        );

        this.audioContext.decodeAudioData(arrayBuffer)
            .then((audioBuffer) => {
                Logger.debug('Audio', `MP3 decoded: ${audioBuffer.length} frames, ${audioBuffer.numberOfChannels}ch`);
                
                // Queue the decoded buffer
                this.audioQueue.push({
                    buffer: audioBuffer,
                    timestamp: timestamp
                });

                // Start playback if not already playing
                if (!this.audioPlaying) {
                    this.playNextAudio();
                }
            })
            .catch((e) => {
                Logger.warn('Audio', `MP3 decode failed: ${e.message}`);
                // Continue silently - don't break audio pipeline
            });
    },

    // Legacy method name for backwards compatibility
    queueAudio(audioData, timestamp) {
        if (this.audioFormatTag === WAVE_FORMAT_MPEGLAYER3) {
            this.queueMP3Audio(audioData, timestamp);
        } else {
            this.queuePCMAudio(audioData, timestamp);
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
                Logger.debug('Audio', 'Context resumed');
            });
        }
    }
};

export default AudioMixin;

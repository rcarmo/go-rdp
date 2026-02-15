// Audio module - handles RDP audio output via Web Audio API
// Uses buffer accumulation to prevent crackling from tiny per-packet buffers.

import { Logger } from './logger.js';

// Audio format tags (matching WAVE format identifiers from backend)
const WAVE_FORMAT_PCM = 0x0001;
const WAVE_FORMAT_MPEGLAYER3 = 0x0055;

// Minimum PCM bytes to accumulate before scheduling playback (~100ms at 44100Hz stereo 16bit)
const MIN_BUFFER_BYTES = 17640;
// Maximum latency: flush buffer if older than this many ms
const MAX_BUFFER_AGE_MS = 150;

const AudioMixin = {
    initAudio() {
        this.audioContext = null;
        this.audioEnabled = false;
        this.audioGain = null;
        this.audioVolume = 1.0;
        this._audioEncodingLogged = false;

        // Audio format (updated from server messages)
        this.audioSampleRate = 44100;
        this.audioChannels = 2;
        this.audioBitsPerSample = 16;
        this.audioFormatTag = WAVE_FORMAT_PCM;

        // Buffer accumulation: collect small PCM packets into larger chunks
        this._pcmAccum = [];     // array of Uint8Array chunks
        this._pcmAccumBytes = 0; // total accumulated bytes
        this._pcmFlushTimer = null;

        // Gapless playback scheduling
        this.audioNextTime = 0;
        this._scheduledCount = 0; // number of in-flight AudioBufferSourceNodes
    },

    enableAudio() {
        if (this.audioContext) return;
        try {
            // Create context — will be recreated with correct sampleRate once format is known
            this.audioContext = new (window.AudioContext || window.webkitAudioContext)({
                sampleRate: this.audioSampleRate
            });
            this.audioGain = this.audioContext.createGain();
            this.audioGain.connect(this.audioContext.destination);
            this.audioGain.gain.value = this.audioVolume;
            this.audioEnabled = true;
            Logger.debug('Audio', `Initialized: ${this.audioContext.sampleRate}Hz, state=${this.audioContext.state}`);
            if (this.audioContext.state === 'suspended') {
                Logger.debug('Audio', 'Context suspended - will resume on user interaction');
            }
        } catch (e) {
            Logger.error('Audio', `Failed to initialize: ${e.message}`);
            this.audioEnabled = false;
        }
    },

    disableAudio() {
        if (this._pcmFlushTimer) {
            clearTimeout(this._pcmFlushTimer);
            this._pcmFlushTimer = null;
        }
        if (this.audioContext) {
            this.audioContext.close();
            this.audioContext = null;
        }
        this.audioEnabled = false;
        this._pcmAccum = [];
        this._pcmAccumBytes = 0;
        this.audioNextTime = 0;
        this._scheduledCount = 0;
        Logger.debug('Audio', 'Disabled');
    },

    setAudioVolume(volume) {
        this.audioVolume = Math.max(0, Math.min(1, volume));
        if (this.audioGain) {
            this.audioGain.gain.value = this.audioVolume;
        }
    },

    handleAudioMessage(data) {
        if (!this.audioEnabled || !this.audioContext) return;
        if (data.length < 4) return;

        // Ensure context is running; try to resume on every message
        if (this.audioContext.state === 'suspended') {
            this.audioContext.resume();
        }

        const view = new DataView(data.buffer, data.byteOffset, data.byteLength);
        const msgType = data[1];
        let offset = 4;

        // Parse format info (msgType = 2)
        if (msgType === 0x02 && data.length >= 14) {
            const channels = view.getUint16(4, true);
            const sampleRate = view.getUint32(6, true);
            const bitsPerSample = view.getUint16(10, true);
            const formatTag = view.getUint16(12, true);

            if (channels < 1 || channels > 8) return;
            if (sampleRate < 8000 || sampleRate > 192000) return;
            if (formatTag === WAVE_FORMAT_PCM && bitsPerSample !== 8 && bitsPerSample !== 16) return;

            // Flush accumulated data if format changes
            if (this.audioChannels !== channels || this.audioSampleRate !== sampleRate ||
                this.audioBitsPerSample !== bitsPerSample || this.audioFormatTag !== formatTag) {
                this._flushPCMBuffer();

                // Recreate AudioContext at the correct sample rate to avoid resampling
                if (this.audioContext && this.audioContext.sampleRate !== sampleRate) {
                    this.audioContext.close();
                    try {
                        this.audioContext = new (window.AudioContext || window.webkitAudioContext)({
                            sampleRate: sampleRate
                        });
                        this.audioGain = this.audioContext.createGain();
                        this.audioGain.connect(this.audioContext.destination);
                        this.audioGain.gain.value = this.audioVolume;
                        this.audioNextTime = 0;
                        this._scheduledCount = 0;
                        Logger.debug('Audio', `Recreated context at ${sampleRate}Hz`);
                    } catch (e) {
                        Logger.error('Audio', `Failed to recreate context: ${e.message}`);
                    }
                }
            }

            this.audioChannels = channels;
            this.audioSampleRate = sampleRate;
            this.audioBitsPerSample = bitsPerSample;
            this.audioFormatTag = formatTag;
            offset = 14;

            if (!this._audioEncodingLogged) {
                this._audioEncodingLogged = true;
                const formatName = formatTag === WAVE_FORMAT_MPEGLAYER3 ? 'MP3' : 'PCM';
                const encoding = `${formatName} ${sampleRate}Hz ${channels}ch ${bitsPerSample}bit`;
                console.info('%c[RDP Session] Audio encoding', 'color: #03A9F4; font-weight: bold', encoding);
            }
        } else if (msgType === 0x02 && data.length >= 12) {
            const channels = view.getUint16(4, true);
            const sampleRate = view.getUint32(6, true);
            const bitsPerSample = view.getUint16(10, true);
            if (channels < 1 || channels > 8) return;
            if (sampleRate < 8000 || sampleRate > 192000) return;
            if (bitsPerSample !== 8 && bitsPerSample !== 16) return;
            this.audioChannels = channels;
            this.audioSampleRate = sampleRate;
            this.audioBitsPerSample = bitsPerSample;
            this.audioFormatTag = WAVE_FORMAT_PCM;
            offset = 12;
        }

        const audioData = data.slice(offset);
        if (audioData.length === 0) return;

        if (this.audioFormatTag === WAVE_FORMAT_MPEGLAYER3) {
            this._playMP3(audioData);
        } else {
            this._accumulatePCM(audioData);
        }
    },

    /**
     * Accumulate PCM bytes and flush when we have enough for smooth playback.
     */
    _accumulatePCM(pcmBytes) {
        this._pcmAccum.push(pcmBytes);
        this._pcmAccumBytes += pcmBytes.length;

        if (this._pcmAccumBytes >= MIN_BUFFER_BYTES) {
            this._flushPCMBuffer();
        } else if (!this._pcmFlushTimer) {
            // Set a timer to flush partial buffers so audio doesn't stall
            this._pcmFlushTimer = setTimeout(() => {
                this._pcmFlushTimer = null;
                this._flushPCMBuffer();
            }, MAX_BUFFER_AGE_MS);
        }
    },

    /**
     * Convert accumulated PCM bytes into an AudioBuffer and schedule it.
     */
    _flushPCMBuffer() {
        if (this._pcmFlushTimer) {
            clearTimeout(this._pcmFlushTimer);
            this._pcmFlushTimer = null;
        }
        if (this._pcmAccumBytes === 0) return;
        if (!this.audioContext || this.audioContext.state === 'closed') return;

        // Merge accumulated chunks into one Uint8Array
        const merged = new Uint8Array(this._pcmAccumBytes);
        let pos = 0;
        for (const chunk of this._pcmAccum) {
            merged.set(chunk, pos);
            pos += chunk.length;
        }
        this._pcmAccum = [];
        this._pcmAccumBytes = 0;

        // Convert to float32 samples
        const bytesPerSample = this.audioBitsPerSample >> 3;
        const bytesPerFrame = bytesPerSample * this.audioChannels;
        // Truncate to whole frames
        const usableBytes = merged.length - (merged.length % bytesPerFrame);
        if (usableBytes === 0) return;

        const totalSamples = usableBytes / bytesPerSample;
        const frameCount = totalSamples / this.audioChannels;

        const view = new DataView(merged.buffer, merged.byteOffset, usableBytes);
        let audioBuffer;
        try {
            audioBuffer = this.audioContext.createBuffer(this.audioChannels, frameCount, this.audioSampleRate);
        } catch (e) {
            Logger.error('Audio', `createBuffer failed: ${e.message}`);
            return;
        }

        if (this.audioBitsPerSample === 16) {
            for (let ch = 0; ch < this.audioChannels; ch++) {
                const channelData = audioBuffer.getChannelData(ch);
                for (let i = 0; i < frameCount; i++) {
                    channelData[i] = view.getInt16((i * this.audioChannels + ch) * 2, true) / 32768.0;
                }
            }
        } else if (this.audioBitsPerSample === 8) {
            for (let ch = 0; ch < this.audioChannels; ch++) {
                const channelData = audioBuffer.getChannelData(ch);
                for (let i = 0; i < frameCount; i++) {
                    channelData[i] = (merged[i * this.audioChannels + ch] - 128) / 128.0;
                }
            }
        }

        this._scheduleBuffer(audioBuffer);
    },

    /**
     * Schedule an AudioBuffer for gapless playback.
     */
    _scheduleBuffer(audioBuffer) {
        if (!this.audioContext || this.audioContext.state === 'closed') return;

        const source = this.audioContext.createBufferSource();
        source.buffer = audioBuffer;
        source.connect(this.audioGain);

        const now = this.audioContext.currentTime;
        // If we've fallen behind (gap > 50ms), reset to avoid growing latency
        if (this.audioNextTime < now + 0.005) {
            this.audioNextTime = now + 0.02; // 20ms lookahead
        }

        source.start(this.audioNextTime);
        this.audioNextTime += audioBuffer.duration;
        this._scheduledCount++;

        source.onended = () => {
            this._scheduledCount--;
        };
    },

    _playMP3(mp3Data) {
        const arrayBuffer = mp3Data.buffer.slice(mp3Data.byteOffset, mp3Data.byteOffset + mp3Data.byteLength);
        this.audioContext.decodeAudioData(arrayBuffer)
            .then((audioBuffer) => {
                if (!this.audioEnabled || !this.audioContext || this.audioContext.state === 'closed') return;
                this._scheduleBuffer(audioBuffer);
            })
            .catch((e) => {
                Logger.warn('Audio', `MP3 decode failed: ${e.message}`);
            });
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

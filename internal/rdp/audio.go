package rdp

import (
	"bytes"

	"github.com/rcarmo/go-rdp/internal/logging"
	"github.com/rcarmo/go-rdp/internal/protocol/audio"
)

// AudioCallback is called when audio data is available to send to the client
type AudioCallback func(data []byte, format *audio.AudioFormat, timestamp uint16)

// AudioHandler manages RDPSND protocol for audio output
type AudioHandler struct {
	client           *Client
	callback         AudioCallback
	enabled          bool
	serverFormats    []audio.AudioFormat
	selectedFormat   int
	defragmenter     audio.ChannelDefragmenter
	pendingWaveInfo  *audio.WaveInfoPDU
}

// NewAudioHandler creates a new audio handler
func NewAudioHandler(client *Client) *AudioHandler {
	return &AudioHandler{
		client:         client,
		selectedFormat: -1,
	}
}

// SetCallback sets the function to call when audio data is available
func (h *AudioHandler) SetCallback(cb AudioCallback) {
	h.callback = cb
}

// Enable enables audio redirection
func (h *AudioHandler) Enable() {
	h.enabled = true
}

// Disable disables audio redirection
func (h *AudioHandler) Disable() {
	h.enabled = false
}

// IsEnabled returns whether audio is enabled
func (h *AudioHandler) IsEnabled() bool {
	return h.enabled
}

// GetSelectedFormat returns the currently negotiated audio format
func (h *AudioHandler) GetSelectedFormat() *audio.AudioFormat {
	if h.selectedFormat >= 0 && h.selectedFormat < len(h.serverFormats) {
		return &h.serverFormats[h.selectedFormat]
	}
	return nil
}

// HandleChannelData processes RDPSND channel data
func (h *AudioHandler) HandleChannelData(data []byte) error {
	if !h.enabled {
		return nil
	}

	// Parse channel PDU
	chunk, err := audio.ParseChannelData(data)
	if err != nil {
		return err
	}

	// Handle fragmentation
	completeData, complete := h.defragmenter.Process(chunk)
	if !complete {
		return nil // Wait for more fragments
	}

	// Parse RDPSND PDU
	if len(completeData) < 4 {
		return nil
	}

	r := bytes.NewReader(completeData)
	var header audio.PDUHeader
	if err := header.Deserialize(r); err != nil {
		return err
	}

	body := completeData[4:]

	switch header.MsgType {
	case audio.SND_FORMATS:
		return h.handleServerFormats(body)
	case audio.SND_TRAINING:
		return h.handleTraining(body)
	case audio.SND_WAVE:
		return h.handleWave(body)
	case audio.SND_WAVE2:
		return h.handleWave2(body)
	case audio.SND_CLOSE:
		logging.Info("Audio: Server closed audio channel")
		return nil
	default:
		logging.Debug("Audio: Unknown RDPSND message type: 0x%02X", header.MsgType)
	}

	return nil
}

// handleServerFormats processes SNDC_FORMATS from server
func (h *AudioHandler) handleServerFormats(body []byte) error {
	var serverFormats audio.ServerAudioFormats
	if err := serverFormats.Deserialize(body); err != nil {
		return err
	}

	logging.Info("Audio: Server offers %d formats (version %d)", serverFormats.NumFormats, serverFormats.Version)

	h.serverFormats = serverFormats.Formats

	// Find a format we support - require PCM for Web Audio
	selectedIndex := -1
	for i, format := range serverFormats.Formats {
		logging.Debug("Audio:   Format %d: %s", i, format.String())
		// We can handle PCM directly in Web Audio
		if format.FormatTag == audio.WAVE_FORMAT_PCM {
			if selectedIndex == -1 {
				selectedIndex = i
			}
			// Prefer 16-bit stereo 44100 Hz
			if format.BitsPerSample == 16 && format.Channels == 2 && format.SamplesPerSec == 44100 {
				selectedIndex = i
			}
		}
	}

	if selectedIndex == -1 {
		logging.Warn("Audio: No PCM formats offered by server; audio disabled")
		h.selectedFormat = -1
		h.serverFormats = nil
		h.Disable()
		return nil
	}

	// Log negotiated format for debugging
	logging.Info("Audio: Negotiated PCM format: %s", serverFormats.Formats[selectedIndex].String())

	if len(h.serverFormats) == 0 {
		logging.Warn("Audio: No formats available from server")
		return nil
	}

	h.selectedFormat = selectedIndex
	logging.Info("Audio: Selected format %d: %s", selectedIndex, h.serverFormats[selectedIndex].String())

	// Send client response with supported formats
	return h.sendClientFormats(serverFormats.Formats, serverFormats.Version)
}

// sendClientFormats sends SNDC_FORMATS response to server
func (h *AudioHandler) sendClientFormats(formats []audio.AudioFormat, version uint16) error {
	// Echo back formats we support (just PCM for now)
	var supportedFormats []audio.AudioFormat
	for _, format := range formats {
		if format.FormatTag == audio.WAVE_FORMAT_PCM {
			supportedFormats = append(supportedFormats, format)
		}
	}

	// If no PCM formats, disable audio (browser only supports PCM)
	if len(supportedFormats) == 0 {
		logging.Warn("Audio: No PCM formats to send; audio disabled")
		h.Disable()
		return nil
	}

	clientFormats := audio.ClientAudioFormats{
		Flags:              audio.TSSNDCAPS_ALIVE,
		Volume:             0xFFFFFFFF,
		Pitch:              0x00010000,
		DGramPort:          0,
		NumFormats:         uint16(len(supportedFormats)),
		LastBlockConfirmed: 0,
		Version:            version,
		Pad:                0,
		Formats:            supportedFormats,
	}

	body := clientFormats.Serialize()
	pdu := audio.BuildChannelPDU(audio.SND_FORMATS, body)

	// Send on rdpsnd channel
	channelID, ok := h.client.channelIDMap[audio.ChannelRDPSND]
	if !ok {
		logging.Warn("Audio: rdpsnd channel not found")
		return nil
	}

	return h.client.mcsLayer.Send(h.client.userID, channelID, pdu)
}

// handleTraining processes SNDC_TRAINING from server
func (h *AudioHandler) handleTraining(body []byte) error {
	var training audio.TrainingPDU
	if err := training.Deserialize(body); err != nil {
		return err
	}

	logging.Debug("Audio: Training packet timestamp=%d size=%d", training.Timestamp, training.PackSize)

	// Send confirmation
	confirm := audio.TrainingConfirmPDU{
		Timestamp: training.Timestamp,
		PackSize:  training.PackSize,
	}

	pdu := audio.BuildChannelPDU(audio.SND_TRAINING, confirm.Serialize())

	channelID, ok := h.client.channelIDMap[audio.ChannelRDPSND]
	if !ok {
		return nil
	}

	return h.client.mcsLayer.Send(h.client.userID, channelID, pdu)
}

// handleWave processes SNDC_WAVE from server (first part of audio)
func (h *AudioHandler) handleWave(body []byte) error {
	var waveInfo audio.WaveInfoPDU
	if err := waveInfo.Deserialize(body); err != nil {
		return err
	}

	h.pendingWaveInfo = &waveInfo

	// The rest of the data follows in the same PDU after the header
	audioData := body[12:] // Skip WaveInfo header

	// Combine initial data with rest
	fullData := append(waveInfo.InitialData, audioData...)

	if h.callback != nil && len(fullData) > 0 {
		var format *audio.AudioFormat
		if int(waveInfo.FormatNo) < len(h.serverFormats) {
			format = &h.serverFormats[waveInfo.FormatNo]
		}
		if format != nil {
			logging.Debug("Audio: Wave format %s, %d bytes", format.String(), len(fullData))
		}
		h.callback(fullData, format, waveInfo.Timestamp)
	}

	// Send wave confirm
	return h.sendWaveConfirm(waveInfo.Timestamp, waveInfo.BlockNo)
}

// handleWave2 processes SNDC_WAVE2 from server (simplified wave PDU)
func (h *AudioHandler) handleWave2(body []byte) error {
	var wave2 audio.Wave2PDU
	if err := wave2.Deserialize(body); err != nil {
		return err
	}

	if h.callback != nil && len(wave2.Data) > 0 {
		var format *audio.AudioFormat
		if int(wave2.FormatNo) < len(h.serverFormats) {
			format = &h.serverFormats[wave2.FormatNo]
		}
		if format != nil {
			logging.Debug("Audio: Wave2 format %s, %d bytes", format.String(), len(wave2.Data))
		}
		h.callback(wave2.Data, format, wave2.Timestamp)
	}

	// Send wave confirm
	return h.sendWaveConfirm(wave2.Timestamp, wave2.BlockNo)
}

// sendWaveConfirm sends SNDC_WAVECONFIRM to server
func (h *AudioHandler) sendWaveConfirm(timestamp uint16, blockNo uint8) error {
	confirm := audio.WaveConfirmPDU{
		Timestamp:      timestamp,
		ConfirmedBlock: blockNo,
		Padding:        0,
	}

	pdu := audio.BuildChannelPDU(audio.SND_WAVE_CONFIRM, confirm.Serialize())

	channelID, ok := h.client.channelIDMap[audio.ChannelRDPSND]
	if !ok {
		return nil
	}

	return h.client.mcsLayer.Send(h.client.userID, channelID, pdu)
}

// EnableAudio registers the rdpsnd channel for audio redirection
func (c *Client) EnableAudio() {
	if c.channels == nil {
		c.channels = []string{}
	}
	// Check if already added
	for _, ch := range c.channels {
		if ch == audio.ChannelRDPSND {
			return
		}
	}
	c.channels = append(c.channels, audio.ChannelRDPSND)
	// Create audio handler
	c.audioHandler = NewAudioHandler(c)
	c.audioHandler.Enable()
}

// GetAudioHandler returns the audio handler
func (c *Client) GetAudioHandler() *AudioHandler {
	return c.audioHandler
}

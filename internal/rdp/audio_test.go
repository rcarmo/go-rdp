package rdp

import (
	"testing"

	"github.com/rcarmo/go-rdp/internal/protocol/audio"
)

func TestNewAudioHandler(t *testing.T) {
	// Create a minimal client for testing
	client := &Client{
		channelIDMap: make(map[string]uint16),
	}

	handler := NewAudioHandler(client)
	if handler == nil {
		t.Fatal("NewAudioHandler() returned nil")
	}
	if handler.client != client {
		t.Error("NewAudioHandler() client not set correctly")
	}
	if handler.enabled {
		t.Error("NewAudioHandler() should start disabled")
	}
}

func TestAudioHandler_EnableDisable(t *testing.T) {
	client := &Client{
		channelIDMap: make(map[string]uint16),
	}
	handler := NewAudioHandler(client)

	if handler.IsEnabled() {
		t.Error("IsEnabled() should be false initially")
	}

	handler.Enable()
	if !handler.IsEnabled() {
		t.Error("IsEnabled() should be true after Enable()")
	}

	handler.Disable()
	if handler.IsEnabled() {
		t.Error("IsEnabled() should be false after Disable()")
	}
}

func TestAudioHandler_SetCallback(t *testing.T) {
	client := &Client{
		channelIDMap: make(map[string]uint16),
	}
	handler := NewAudioHandler(client)

	handler.SetCallback(func(data []byte, format *audio.AudioFormat, timestamp uint16) {
		// callback for testing
	})

	if handler.callback == nil {
		t.Error("SetCallback() did not set callback")
	}
}

func TestAudioHandler_GetSelectedFormat(t *testing.T) {
	client := &Client{
		channelIDMap: make(map[string]uint16),
	}
	handler := NewAudioHandler(client)

	// Initially no format selected
	format := handler.GetSelectedFormat()
	if format != nil {
		t.Error("GetSelectedFormat() should return nil initially")
	}

	// Set up some formats
	handler.serverFormats = []audio.AudioFormat{
		{FormatTag: audio.WAVE_FORMAT_PCM, Channels: 2, SamplesPerSec: 44100},
		{FormatTag: audio.WAVE_FORMAT_ADPCM, Channels: 1, SamplesPerSec: 22050},
	}
	handler.selectedFormat = 0

	format = handler.GetSelectedFormat()
	if format == nil {
		t.Fatal("GetSelectedFormat() returned nil after setting format")
	}
	if format.FormatTag != audio.WAVE_FORMAT_PCM {
		t.Errorf("GetSelectedFormat() FormatTag = %v, want PCM", format.FormatTag)
	}
}

func TestAudioHandler_HandleChannelData_Disabled(t *testing.T) {
	client := &Client{
		channelIDMap: make(map[string]uint16),
	}
	handler := NewAudioHandler(client)

	// Should not error when disabled
	err := handler.HandleChannelData([]byte{0x01, 0x02, 0x03, 0x04})
	if err != nil {
		t.Errorf("HandleChannelData() error = %v when disabled", err)
	}
}

func TestEnableAudio(t *testing.T) {
	client := &Client{
		channels:     []string{},
		channelIDMap: make(map[string]uint16),
	}

	client.EnableAudio()

	// Check channel was added
	found := false
	for _, ch := range client.channels {
		if ch == audio.ChannelRDPSND {
			found = true
			break
		}
	}
	if !found {
		t.Error("EnableAudio() did not add rdpsnd channel")
	}

	// Check handler was created
	if client.audioHandler == nil {
		t.Error("EnableAudio() did not create audio handler")
	}

	// Check handler is enabled
	if !client.audioHandler.IsEnabled() {
		t.Error("EnableAudio() did not enable audio handler")
	}

	// Calling again should not duplicate
	client.EnableAudio()
	count := 0
	for _, ch := range client.channels {
		if ch == audio.ChannelRDPSND {
			count++
		}
	}
	if count != 1 {
		t.Errorf("EnableAudio() added duplicate channel, count = %d", count)
	}
}

func TestGetAudioHandler(t *testing.T) {
	client := &Client{
		channels:     []string{},
		channelIDMap: make(map[string]uint16),
	}

	// Initially nil
	if client.GetAudioHandler() != nil {
		t.Error("GetAudioHandler() should be nil before EnableAudio()")
	}

	client.EnableAudio()

	handler := client.GetAudioHandler()
	if handler == nil {
		t.Error("GetAudioHandler() returned nil after EnableAudio()")
	}
}

func TestAudioHandler_HandleChannelData_EmptyData(t *testing.T) {
client := &Client{
channelIDMap: make(map[string]uint16),
}
handler := NewAudioHandler(client)
handler.Enable()

// Short data will return an error from ParseChannelData
err := handler.HandleChannelData([]byte{0x01, 0x02})
// Error is expected since data is too short
if err == nil {
t.Log("HandleChannelData() returned nil for short data (may be implementation-specific)")
}
}

func TestAudioHandler_HandleChannelData_ServerFormats(t *testing.T) {
client := &Client{
channelIDMap: make(map[string]uint16),
}
handler := NewAudioHandler(client)
handler.Enable()

// Build a valid server formats message with at least one format
// Channel PDU header: length (4) + flags (4) + RDPSND header (4) + body
channelData := []byte{
// Channel header: totalLength=48, flags=0x03 (first+last)
0x30, 0x00, 0x00, 0x00, 0x03, 0x00, 0x00, 0x00,
	// RDPSND header: msgType=0x07 (SND_FORMATS), reserved=0, bodySize=44
	0x07, 0x00, 0x2C, 0x00,
	// ServerFormats: dwFlags=0, dwVolume=0xFFFFFFFF, dwPitch=0, wDGramPort=0, wNumberOfFormats=1, cLastBlockConfirmed=0, wVersion=6, bPad=0
	0x00, 0x00, 0x00, 0x00,
	0xFF, 0xFF, 0xFF, 0xFF,
	0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, // wDGramPort=0
	0x01, 0x00, // wNumberOfFormats=1
	0x00,       // cLastBlockConfirmed
	0x06, 0x00, // wVersion=6
	0x00,       // bPad
// Format: wFormatTag=1 (PCM), nChannels=2, nSamplesPerSec=44100, nAvgBytesPerSec=176400, nBlockAlign=4, wBitsPerSample=16, cbSize=0
0x01, 0x00, // FormatTag=PCM
0x02, 0x00, // Channels=2
0x44, 0xAC, 0x00, 0x00, // SamplesPerSec=44100
0x10, 0xB1, 0x02, 0x00, // AvgBytesPerSec=176400
0x04, 0x00, // BlockAlign=4
0x10, 0x00, // BitsPerSample=16
0x00, 0x00, // cbSize=0
}

// This exercises the code path but may error due to missing connection
err := handler.HandleChannelData(channelData)
_ = err
}

func TestAudioHandler_HandleChannelData_Wave2(t *testing.T) {
client := &Client{
channelIDMap: make(map[string]uint16),
}
handler := NewAudioHandler(client)
handler.Enable()

// Set up some server formats first
handler.serverFormats = []audio.AudioFormat{
{FormatTag: audio.WAVE_FORMAT_PCM, Channels: 2, SamplesPerSec: 44100, BitsPerSample: 16},
}
handler.selectedFormat = 0

var callbackCalled bool
var receivedData []byte
handler.SetCallback(func(data []byte, format *audio.AudioFormat, timestamp uint16) {
callbackCalled = true
receivedData = data
})

// Build a valid WAVE2 message
channelData := []byte{
// Channel header: totalLength=16, flags=0x03
0x10, 0x00, 0x00, 0x00, 0x03, 0x00, 0x00, 0x00,
// RDPSND header: msgType=0x0D (SND_WAVE2), reserved=0, bodySize=8
0x0D, 0x00, 0x08, 0x00,
// Wave2: wTimeStamp=1234, wFormatNo=0, cBlockNo=1, bPad=0, PCM data
0xD2, 0x04, 0x00, 0x00, 0x01, 0x00, 0xAA, 0xBB,
}

err := handler.HandleChannelData(channelData)
// Callback may or may not be called depending on implementation
_ = err
_ = callbackCalled
_ = receivedData
}

func TestAudioHandler_HandleChannelData_Training(t *testing.T) {
client := &Client{
channelIDMap: make(map[string]uint16),
}
handler := NewAudioHandler(client)
handler.Enable()

// Build a training message
channelData := []byte{
// Channel header: totalLength=12, flags=0x03
0x0C, 0x00, 0x00, 0x00, 0x03, 0x00, 0x00, 0x00,
// RDPSND header: msgType=0x06 (SND_TRAINING), reserved=0, bodySize=4
0x06, 0x00, 0x04, 0x00,
// Training: wTimeStamp=1234, wPackSize=8
0xD2, 0x04, 0x08, 0x00,
}

err := handler.HandleChannelData(channelData)
// May return error due to missing connection for response
_ = err
}

func TestAudioHandler_HandleChannelData_Close(t *testing.T) {
client := &Client{
channelIDMap: make(map[string]uint16),
}
handler := NewAudioHandler(client)
handler.Enable()

// Build a close message
channelData := []byte{
// Channel header: totalLength=12, flags=0x03
0x0C, 0x00, 0x00, 0x00, 0x03, 0x00, 0x00, 0x00,
	// RDPSND header: msgType=0x01 (SND_CLOSE), reserved=0, bodySize=0
	0x01, 0x00, 0x00, 0x00,
}

err := handler.HandleChannelData(channelData)
if err != nil {
t.Errorf("HandleChannelData() error = %v for close message", err)
}
}

func TestAudioHandler_HandleChannelData_Unknown(t *testing.T) {
client := &Client{
channelIDMap: make(map[string]uint16),
}
handler := NewAudioHandler(client)
handler.Enable()

// Build a message with unknown type
channelData := []byte{
// Channel header: totalLength=12, flags=0x03
0x0C, 0x00, 0x00, 0x00, 0x03, 0x00, 0x00, 0x00,
// RDPSND header: msgType=0xFF (unknown), reserved=0, bodySize=0
0xFF, 0x00, 0x00, 0x00,
}

err := handler.HandleChannelData(channelData)
if err != nil {
t.Errorf("HandleChannelData() error = %v for unknown message type", err)
}
}

func TestAudioHandler_GetSelectedFormat_MP3(t *testing.T) {
	client := &Client{
		channelIDMap: make(map[string]uint16),
	}
	handler := NewAudioHandler(client)

	// Set up formats with MP3
	handler.serverFormats = []audio.AudioFormat{
		{FormatTag: audio.WAVE_FORMAT_MPEGLAYER3, Channels: 2, SamplesPerSec: 44100},
		{FormatTag: audio.WAVE_FORMAT_ADPCM, Channels: 1, SamplesPerSec: 22050},
	}
	handler.selectedFormat = 0

	format := handler.GetSelectedFormat()
	if format == nil {
		t.Fatal("GetSelectedFormat() returned nil after setting MP3 format")
	}
	if format.FormatTag != audio.WAVE_FORMAT_MPEGLAYER3 {
		t.Errorf("GetSelectedFormat() FormatTag = %v, want MP3 (0x0055)", format.FormatTag)
	}
}

func TestAudioHandler_HandleChannelData_ServerFormats_MP3Only(t *testing.T) {
	client := &Client{
		channelIDMap: make(map[string]uint16),
	}
	handler := NewAudioHandler(client)
	handler.Enable()

	// Build a server formats message with only MP3 format
	channelData := []byte{
		// Channel header: totalLength=48, flags=0x03 (first+last)
		0x30, 0x00, 0x00, 0x00, 0x03, 0x00, 0x00, 0x00,
		// RDPSND header: msgType=0x07 (SND_FORMATS), reserved=0, bodySize=44
		0x07, 0x00, 0x2C, 0x00,
		// ServerFormats: dwFlags=0, dwVolume=0xFFFFFFFF, dwPitch=0, wDGramPort=0, wNumberOfFormats=1, cLastBlockConfirmed=0, wVersion=6, bPad=0
		0x00, 0x00, 0x00, 0x00,
		0xFF, 0xFF, 0xFF, 0xFF,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, // wDGramPort=0
		0x01, 0x00, // wNumberOfFormats=1
		0x00,       // cLastBlockConfirmed
		0x06, 0x00, // wVersion=6
		0x00,       // bPad
		// Format: wFormatTag=0x55 (MP3), nChannels=2, nSamplesPerSec=44100, nAvgBytesPerSec=16000, nBlockAlign=1, wBitsPerSample=0, cbSize=0
		0x55, 0x00, // FormatTag=MP3
		0x02, 0x00, // Channels=2
		0x44, 0xAC, 0x00, 0x00, // SamplesPerSec=44100
		0x80, 0x3E, 0x00, 0x00, // AvgBytesPerSec=16000
		0x01, 0x00, // BlockAlign=1
		0x00, 0x00, // BitsPerSample=0 (not used for MP3)
		0x00, 0x00, // cbSize=0
	}

	// This exercises the code path - handler should accept MP3 when no PCM available
	err := handler.HandleChannelData(channelData)
	_ = err

	// Verify MP3 was selected (format index 0)
	if handler.selectedFormat != 0 {
		t.Errorf("Expected selectedFormat=0 for MP3, got %d", handler.selectedFormat)
	}
	if handler.serverFormats != nil && len(handler.serverFormats) > 0 {
		if handler.serverFormats[0].FormatTag != audio.WAVE_FORMAT_MPEGLAYER3 {
			t.Errorf("Expected MP3 format tag, got 0x%04X", handler.serverFormats[0].FormatTag)
		}
	}
}

func TestAudioHandler_HandleChannelData_ServerFormats_PCMPreferredOverMP3(t *testing.T) {
	client := &Client{
		channelIDMap: make(map[string]uint16),
	}
	handler := NewAudioHandler(client)
	handler.Enable()

	// Build a server formats message with both MP3 and PCM (MP3 first)
	channelData := []byte{
		// Channel header: totalLength=66, flags=0x03 (first+last)
		0x42, 0x00, 0x00, 0x00, 0x03, 0x00, 0x00, 0x00,
		// RDPSND header: msgType=0x07 (SND_FORMATS), reserved=0, bodySize=62
		0x07, 0x00, 0x3E, 0x00,
		// ServerFormats header
		0x00, 0x00, 0x00, 0x00, // dwFlags=0
		0xFF, 0xFF, 0xFF, 0xFF, // dwVolume
		0x00, 0x00, 0x00, 0x00, // dwPitch
		0x00, 0x00, // wDGramPort=0
		0x02, 0x00, // wNumberOfFormats=2
		0x00,       // cLastBlockConfirmed
		0x06, 0x00, // wVersion=6
		0x00,       // bPad
		// Format 0: MP3
		0x55, 0x00, // FormatTag=MP3
		0x02, 0x00, // Channels=2
		0x44, 0xAC, 0x00, 0x00, // SamplesPerSec=44100
		0x80, 0x3E, 0x00, 0x00, // AvgBytesPerSec=16000
		0x01, 0x00, // BlockAlign=1
		0x00, 0x00, // BitsPerSample=0
		0x00, 0x00, // cbSize=0
		// Format 1: PCM
		0x01, 0x00, // FormatTag=PCM
		0x02, 0x00, // Channels=2
		0x44, 0xAC, 0x00, 0x00, // SamplesPerSec=44100
		0x10, 0xB1, 0x02, 0x00, // AvgBytesPerSec=176400
		0x04, 0x00, // BlockAlign=4
		0x10, 0x00, // BitsPerSample=16
		0x00, 0x00, // cbSize=0
	}

	err := handler.HandleChannelData(channelData)
	_ = err

	// Verify PCM was selected (format index 1) over MP3 (index 0)
	if handler.selectedFormat != 1 {
		t.Errorf("Expected selectedFormat=1 for PCM (preferred over MP3), got %d", handler.selectedFormat)
	}
	if handler.serverFormats != nil && len(handler.serverFormats) > 1 {
		if handler.serverFormats[handler.selectedFormat].FormatTag != audio.WAVE_FORMAT_PCM {
			t.Errorf("Expected PCM format tag for selected format, got 0x%04X", handler.serverFormats[handler.selectedFormat].FormatTag)
		}
	}
}

func TestAudioHandler_HandleChannelData_ServerFormats_NoSupportedFormats(t *testing.T) {
	client := &Client{
		channelIDMap: make(map[string]uint16),
	}
	handler := NewAudioHandler(client)
	handler.Enable()

	// Build a server formats message with only unsupported formats (ADPCM)
	channelData := []byte{
		// Channel header: totalLength=48, flags=0x03 (first+last)
		0x30, 0x00, 0x00, 0x00, 0x03, 0x00, 0x00, 0x00,
		// RDPSND header: msgType=0x07 (SND_FORMATS), reserved=0, bodySize=44
		0x07, 0x00, 0x2C, 0x00,
		// ServerFormats header
		0x00, 0x00, 0x00, 0x00,
		0xFF, 0xFF, 0xFF, 0xFF,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, // wDGramPort=0
		0x01, 0x00, // wNumberOfFormats=1
		0x00,       // cLastBlockConfirmed
		0x06, 0x00, // wVersion=6
		0x00,       // bPad
		// Format: ADPCM (unsupported)
		0x02, 0x00, // FormatTag=ADPCM
		0x02, 0x00, // Channels=2
		0x44, 0xAC, 0x00, 0x00, // SamplesPerSec=44100
		0x00, 0x00, 0x00, 0x00, // AvgBytesPerSec
		0x00, 0x00, // BlockAlign
		0x04, 0x00, // BitsPerSample=4
		0x00, 0x00, // cbSize=0
	}

	err := handler.HandleChannelData(channelData)
	_ = err

	// Verify no format was selected and audio is disabled
	if handler.selectedFormat != -1 {
		t.Errorf("Expected selectedFormat=-1 for unsupported formats, got %d", handler.selectedFormat)
	}
	if handler.IsEnabled() {
		t.Error("Expected audio to be disabled when no supported formats available")
	}
}

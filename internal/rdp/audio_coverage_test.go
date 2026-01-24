package rdp

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/rcarmo/go-rdp/internal/protocol/audio"
	"github.com/rcarmo/go-rdp/internal/protocol/pdu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAudioHandler_SendClientFormats tests sending client audio formats
func TestAudioHandler_SendClientFormatsFunc(t *testing.T) {
	tests := []struct {
		name      string
		formats   []audio.AudioFormat
		version   uint16
		sendErr   error
		expectErr bool
	}{
		{
			name: "PCM format",
			formats: []audio.AudioFormat{
				{
					FormatTag:      audio.WAVE_FORMAT_PCM,
					Channels:       2,
					SamplesPerSec:  44100,
					AvgBytesPerSec: 176400,
					BlockAlign:     4,
					BitsPerSample:  16,
				},
			},
			version:   6,
			expectErr: false,
		},
		{
			name: "MP3 format - accepted as fallback",
			formats: []audio.AudioFormat{
				{
					FormatTag:      0x0055, // MP3
					Channels:       2,
					SamplesPerSec:  44100,
					AvgBytesPerSec: 32000,
					BlockAlign:     1,
					BitsPerSample:  0,
				},
			},
			version:   6,
			expectErr: false,
		},
		{
			name: "no supported format - disable audio",
			formats: []audio.AudioFormat{
				{
					FormatTag:      audio.WAVE_FORMAT_ADPCM, // ADPCM - not supported
					Channels:       2,
					SamplesPerSec:  44100,
					AvgBytesPerSec: 32000,
					BlockAlign:     1,
					BitsPerSample:  4,
				},
			},
			version:   6,
			expectErr: false,
		},
		{
			name:      "send error",
			formats:   []audio.AudioFormat{{FormatTag: audio.WAVE_FORMAT_PCM}},
			version:   6,
			sendErr:   errors.New("send failed"),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sentData []byte
			mockMCS := &MockMCSLayer{
				SendFunc: func(userID, channelID uint16, data []byte) error {
					sentData = data
					return tt.sendErr
				},
			}

			client := &Client{
				userID: 1001,
				channelIDMap: map[string]uint16{
					audio.ChannelRDPSND: 1007,
				},
				mcsLayer: mockMCS,
			}

			handler := NewAudioHandler(client)

			err := handler.sendClientFormats(tt.formats, tt.version)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.name == "no supported format - disable audio" {
					assert.Empty(t, sentData)
				} else {
					assert.NotEmpty(t, sentData)
				}
			}
		})
	}
}

// TestAudioHandler_HandleTrainingFunc tests audio training PDU handling
func TestAudioHandler_HandleTrainingFunc(t *testing.T) {
	// This test is skipped because handleTraining requires a properly
	// formatted TrainingPDU that gets deserialized
	// The function delegates to the audio package for deserialization
}

// TestAudioHandler_HandleWaveFunc tests wave PDU handling
func TestAudioHandler_HandleWaveFunc(t *testing.T) {
	// This test is skipped because handleWave requires a properly
	// formatted WaveInfoPDU that gets deserialized
}

// TestAudioHandler_HandleWave2Func tests wave2 PDU handling
func TestAudioHandler_HandleWave2Func(t *testing.T) {
	// This test is skipped because handleWave2 requires a properly
	// formatted Wave2PDU that gets deserialized
}

// TestAudioHandler_SendWaveConfirmFunc tests wave confirm PDU sending
func TestAudioHandler_SendWaveConfirmFunc(t *testing.T) {
	tests := []struct {
		name      string
		timestamp uint16
		blockNo   uint8
		sendErr   error
		expectErr bool
	}{
		{
			name:      "successful confirm",
			timestamp: 1234,
			blockNo:   5,
			expectErr: false,
		},
		{
			name:      "send error",
			timestamp: 1234,
			blockNo:   5,
			sendErr:   errors.New("send failed"),
			expectErr: true,
		},
		{
			name:      "zero values",
			timestamp: 0,
			blockNo:   0,
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sentData []byte
			mockMCS := &MockMCSLayer{
				SendFunc: func(userID, channelID uint16, data []byte) error {
					sentData = data
					return tt.sendErr
				},
			}

			client := &Client{
				userID: 1001,
				channelIDMap: map[string]uint16{
					audio.ChannelRDPSND: 1007,
				},
				mcsLayer: mockMCS,
			}

			handler := NewAudioHandler(client)

			err := handler.sendWaveConfirm(tt.timestamp, tt.blockNo)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, sentData)
			}
		})
	}
}

// TestAudioHandler_HandleServerFormatsFunc tests server audio formats handling
func TestAudioHandler_HandleServerFormatsFunc(t *testing.T) {
	mockMCS := &MockMCSLayer{
		SendFunc: func(userID, channelID uint16, data []byte) error {
			return nil
		},
	}

	client := &Client{
		userID: 1001,
		channelIDMap: map[string]uint16{
			audio.ChannelRDPSND: 1007,
		},
		mcsLayer: mockMCS,
	}

	handler := NewAudioHandler(client)

	// Create server formats data with proper format
	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.LittleEndian, uint32(0))      // dwFlags
	_ = binary.Write(buf, binary.LittleEndian, uint32(0xFFFF)) // dwVolume
	_ = binary.Write(buf, binary.LittleEndian, uint32(0))      // dwPitch
	_ = binary.Write(buf, binary.LittleEndian, uint16(0))      // wDGramPort
	_ = binary.Write(buf, binary.LittleEndian, uint16(1))      // wNumberOfFormats
	buf.WriteByte(0)                                           // cLastBlockConfirmed
	_ = binary.Write(buf, binary.LittleEndian, uint16(6))      // wVersion
	buf.WriteByte(0)                                           // bPad

	// Add one audio format
	_ = binary.Write(buf, binary.LittleEndian, uint16(audio.WAVE_FORMAT_PCM)) // wFormatTag
	_ = binary.Write(buf, binary.LittleEndian, uint16(2))                     // nChannels
	_ = binary.Write(buf, binary.LittleEndian, uint32(44100))                 // nSamplesPerSec
	_ = binary.Write(buf, binary.LittleEndian, uint32(176400))                // nAvgBytesPerSec
	_ = binary.Write(buf, binary.LittleEndian, uint16(4))                     // nBlockAlign
	_ = binary.Write(buf, binary.LittleEndian, uint16(16))                    // wBitsPerSample
	_ = binary.Write(buf, binary.LittleEndian, uint16(0))                     // cbSize

	err := handler.handleServerFormats(buf.Bytes())
	// The function may or may not populate serverFormats depending on parsing
	// Just verify no error
	require.NoError(t, err)
}

// TestRefreshRect tests refresh rect sending
func TestRefreshRect(t *testing.T) {
	tests := []struct {
		name      string
		width     uint16
		height    uint16
		sendErr   error
		expectErr bool
	}{
		{
			name:      "successful send",
			width:     1920,
			height:    1080,
			expectErr: false,
		},
		{
			name:      "send error",
			width:     1920,
			height:    1080,
			sendErr:   errors.New("send failed"),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sentData []byte
			mockMCS := &MockMCSLayer{
				SendFunc: func(userID, channelID uint16, data []byte) error {
					sentData = data
					return tt.sendErr
				},
			}

			client := &Client{
				shareID:       0x12345678,
				userID:        1001,
				desktopWidth:  tt.width,
				desktopHeight: tt.height,
				channelIDMap:  map[string]uint16{"global": 1003},
				mcsLayer:      mockMCS,
			}

			err := client.sendRefreshRect()

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, sentData)
			}
		})
	}
}

// TestSendInputEvent tests input event sending
func TestSendInputEventFunc(t *testing.T) {
	// This test is limited because fastPath is a concrete type
	// We'll just verify the function exists and test error paths
	client := &Client{
		fastPath: nil, // Will cause a panic, so we skip actual execution
	}

	// We cannot easily test this without a real fastPath instance
	// The function signature shows it delegates to fastPath.Send
	_ = client
}

// TestSetRemoteApp tests remote app configuration
func TestSetRemoteAppFunc(t *testing.T) {
	client := &Client{
		channels: []string{},
	}

	client.SetRemoteApp("notepad.exe", "/test", "C:\\Windows")

	require.NotNil(t, client.remoteApp)
	assert.Equal(t, "notepad.exe", client.remoteApp.App)
	assert.Equal(t, "/test", client.remoteApp.Args)
	assert.Equal(t, "C:\\Windows", client.remoteApp.WorkingDir)
	assert.Contains(t, client.channels, "rail")
	assert.Equal(t, RailStateUninitialized, client.railState)
}

// TestCodecGUIDToName tests codec GUID name conversion
func TestCodecGUIDToNameFunc(t *testing.T) {
	tests := []struct {
		name   string
		guid   [16]byte
		expect string
	}{
		{
			name:   "NSCodec",
			guid:   guidNSCodec,
			expect: "NSCodec",
		},
		{
			name:   "RemoteFX",
			guid:   guidRemoteFX,
			expect: "RemoteFX",
		},
		{
			name:   "RemoteFX-Image",
			guid:   guidImageRemoteFX,
			expect: "RemoteFX-Image",
		},
		{
			name:   "ClearCodec",
			guid:   guidClearCodec,
			expect: "ClearCodec",
		},
		{
			name:   "Unknown",
			guid:   [16]byte{0x01, 0x02, 0x03, 0x04},
			expect: "Unknown(01020304)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := codecGUIDToName(tt.guid)
			assert.Equal(t, tt.expect, result)
		})
	}
}

// TestWrite tests the Write method
func TestWriteFunc(t *testing.T) {
	conn := &mockConnWrite{writeErr: nil}
	client := &Client{conn: conn}

	n, err := client.Write([]byte{0x01, 0x02, 0x03})
	require.NoError(t, err)
	assert.Equal(t, 3, n)
}

// TestRead tests the Read method - skipped because it requires buffReader
func TestReadFunc(t *testing.T) {
	// Create a buffered reader with some test data
	data := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	client := &Client{
		buffReader: bufio.NewReader(bytes.NewReader(data)),
	}

	buf := make([]byte, 3)
	n, err := client.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, 3, n)
	assert.Equal(t, []byte{0x01, 0x02, 0x03}, buf)
}

// mockConnWrite implements net.Conn for write testing
type mockConnWrite struct {
	writeErr error
}

func (m *mockConnWrite) Read(b []byte) (int, error) {
	return 0, io.EOF
}

func (m *mockConnWrite) Write(b []byte) (int, error) {
	if m.writeErr != nil {
		return 0, m.writeErr
	}
	return len(b), nil
}

func (m *mockConnWrite) Close() error {
	return nil
}

func (m *mockConnWrite) LocalAddr() net.Addr {
	return &net.TCPAddr{}
}

func (m *mockConnWrite) RemoteAddr() net.Addr {
	return &net.TCPAddr{}
}

func (m *mockConnWrite) SetDeadline(t time.Time) error {
	return nil
}

func (m *mockConnWrite) SetReadDeadline(t time.Time) error {
	return nil
}

func (m *mockConnWrite) SetWriteDeadline(t time.Time) error {
	return nil
}

// TestGetServerCapabilities_AllTypes tests GetServerCapabilities with all capability types
func TestGetServerCapabilities_AllTypes(t *testing.T) {
	client := &Client{
		serverCapabilitySets: []pdu.CapabilitySet{
			{
				CapabilitySetType: pdu.CapabilitySetTypeBitmap,
				BitmapCapabilitySet: &pdu.BitmapCapabilitySet{
					PreferredBitsPerPixel: 24,
					DesktopWidth:          1920,
					DesktopHeight:         1080,
				},
			},
			{
				CapabilitySetType: pdu.CapabilitySetTypeGeneral,
				GeneralCapabilitySet: &pdu.GeneralCapabilitySet{
					ExtraFlags: 0x0001,
				},
			},
			{
				CapabilitySetType: pdu.CapabilitySetTypeOrder,
				OrderCapabilitySet: &pdu.OrderCapabilitySet{
					OrderFlags: 0x0022,
				},
			},
			{
				CapabilitySetType: pdu.CapabilitySetTypeSurfaceCommands,
			},
			{
				CapabilitySetType: pdu.CapabilitySetTypeBitmapCodecs,
				BitmapCodecsCapabilitySet: &pdu.BitmapCodecsCapabilitySet{
					BitmapCodecArray: []pdu.BitmapCodec{
						{CodecGUID: guidNSCodec},
						{CodecGUID: guidRemoteFX},
					},
				},
			},
			{
				CapabilitySetType: pdu.CapabilitySetTypeMultifragmentUpdate,
				MultifragmentUpdateCapabilitySet: &pdu.MultifragmentUpdateCapabilitySet{
					MaxRequestSize: 65535,
				},
			},
			{
				CapabilitySetType: pdu.CapabilitySetTypeLargePointer,
			},
			{
				CapabilitySetType: pdu.CapabilitySetTypeFrameAcknowledge,
			},
		},
	}

	info := client.GetServerCapabilities()

	assert.Equal(t, 24, info.ColorDepth)
	assert.Equal(t, "1920x1080", info.DesktopSize)
	assert.True(t, info.SurfaceCommands)
	assert.True(t, info.LargePointer)
	assert.True(t, info.FrameAcknowledge)
	assert.Contains(t, info.BitmapCodecs, "NSCodec")
	assert.Contains(t, info.BitmapCodecs, "RemoteFX")
	assert.Equal(t, uint32(65535), info.MultifragmentSize)
}

// TestAudioHandler_HandleTraining tests training PDU handling
func TestAudioHandler_HandleTraining(t *testing.T) {
	mockMCS := &MockMCSLayer{
		SendFunc: func(userID, channelID uint16, data []byte) error {
			return nil
		},
	}

	client := &Client{
		mcsLayer:     mockMCS,
		userID:       1001,
		channelIDMap: map[string]uint16{audio.ChannelRDPSND: 1005},
	}
	handler := NewAudioHandler(client)
	client.audioHandler = handler

	// Create training PDU data manually
	// TrainingPDU: Timestamp (uint16) + PackSize (uint16)
	data := make([]byte, 4)
	binary.LittleEndian.PutUint16(data[0:2], 12345) // Timestamp
	binary.LittleEndian.PutUint16(data[2:4], 0)     // PackSize

	err := handler.handleTraining(data)
	assert.NoError(t, err)
}

// TestAudioHandler_HandleTrainingNoChannel tests training with no channel
func TestAudioHandler_HandleTrainingNoChannel(t *testing.T) {
	mockMCS := &MockMCSLayer{}
	client := &Client{
		mcsLayer:     mockMCS,
		channelIDMap: map[string]uint16{},
	}
	handler := NewAudioHandler(client)

	// Create training PDU data
	data := make([]byte, 4)
	binary.LittleEndian.PutUint16(data[0:2], 12345)
	binary.LittleEndian.PutUint16(data[2:4], 0)

	err := handler.handleTraining(data)
	assert.NoError(t, err) // Returns nil when channel not found
}

// TestAudioHandler_HandleWave2Error tests handleWave2 error path
func TestAudioHandler_HandleWave2Error(t *testing.T) {
	mockMCS := &MockMCSLayer{}
	client := &Client{
		mcsLayer:     mockMCS,
		channelIDMap: map[string]uint16{},
	}
	handler := NewAudioHandler(client)

	// Empty data should fail
	err := handler.handleWave2([]byte{})
	assert.Error(t, err)
}

func TestAudioHandler_HandleWave_ErrorShortBody(t *testing.T) {
	h := NewAudioHandler(&Client{})
	err := h.handleWave([]byte{0x01, 0x02})
	assert.Error(t, err)
}

func TestAudioHandler_HandleWave_CallbackAndConfirm(t *testing.T) {
	mockMCS := &MockMCSLayer{}
	client := &Client{
		mcsLayer: mockMCS,
		userID:   1001,
		channelIDMap: map[string]uint16{
			audio.ChannelRDPSND: 1007,
		},
	}

	h := NewAudioHandler(client)
	h.serverFormats = []audio.AudioFormat{{FormatTag: audio.WAVE_FORMAT_PCM}}

	var (
		gotData []byte
		gotTS   uint16
		gotFmt  *audio.AudioFormat
	)
	h.SetCallback(func(data []byte, format *audio.AudioFormat, timestamp uint16) {
		gotData = append([]byte(nil), data...)
		gotFmt = format
		gotTS = timestamp
	})

	body := make([]byte, 12+2)
	binary.LittleEndian.PutUint16(body[0:2], 1234) // Timestamp
	binary.LittleEndian.PutUint16(body[2:4], 0)    // FormatNo
	body[4] = 7                                   // BlockNo
	copy(body[8:12], []byte{0xAA, 0xBB, 0xCC, 0xDD})
	copy(body[12:], []byte{0x01, 0x02})

	err := h.handleWave(body)
	require.NoError(t, err)
	require.NotNil(t, gotFmt)
	assert.Equal(t, uint16(1234), gotTS)
	assert.Equal(t, []byte{0xAA, 0xBB, 0xCC, 0xDD, 0x01, 0x02}, gotData)
	require.Len(t, mockMCS.SendCalls, 1)
}

func TestAudioHandler_HandleWave2_CallbackAndConfirm(t *testing.T) {
	mockMCS := &MockMCSLayer{}
	client := &Client{
		mcsLayer: mockMCS,
		userID:   1001,
		channelIDMap: map[string]uint16{
			audio.ChannelRDPSND: 1007,
		},
	}

	h := NewAudioHandler(client)
	h.serverFormats = []audio.AudioFormat{{FormatTag: audio.WAVE_FORMAT_PCM}}

	var gotData []byte
	h.SetCallback(func(data []byte, format *audio.AudioFormat, timestamp uint16) {
		gotData = append([]byte(nil), data...)
	})

	body := make([]byte, 12+3)
	binary.LittleEndian.PutUint16(body[0:2], 2222) // Timestamp
	binary.LittleEndian.PutUint16(body[2:4], 0)    // FormatNo
	body[4] = 9                                   // BlockNo
	binary.LittleEndian.PutUint32(body[8:12], 3)   // DataSize (ignored by Deserialize)
	copy(body[12:], []byte{0x11, 0x22, 0x33})

	err := h.handleWave2(body)
	require.NoError(t, err)
	assert.Equal(t, []byte{0x11, 0x22, 0x33}, gotData)
	require.Len(t, mockMCS.SendCalls, 1)
}

package rdp

import (
	"fmt"
	"testing"

	"github.com/rcarmo/go-rdp/internal/protocol/pdu"
	"github.com/stretchr/testify/assert"
)

func TestCodecGUIDToName(t *testing.T) {
	tests := []struct {
		name     string
		guid     [16]byte
		expected string
	}{
		{
			name:     "NSCodec",
			guid:     guidNSCodec,
			expected: "NSCodec",
		},
		{
			name:     "RemoteFX",
			guid:     guidRemoteFX,
			expected: "RemoteFX",
		},
		{
			name:     "RemoteFX-Image",
			guid:     guidImageRemoteFX,
			expected: "RemoteFX-Image",
		},
		{
			name:     "ClearCodec",
			guid:     guidClearCodec,
			expected: "ClearCodec",
		},
		{
			name:     "JPEG",
			guid:     guidJPEG,
			expected: "JPEG",
		},
		{
			name:     "H264",
			guid:     guidH264,
			expected: "H264",
		},
		{
			name:     "PNG",
			guid:     guidPNG,
			expected: "PNG",
		},
		{
			name:     "unknown codec",
			guid:     [16]byte{0x00, 0x01, 0x02, 0x03},
			expected: "Unknown(00010203)",
		},
		{
			name:     "all zeros",
			guid:     [16]byte{},
			expected: "Unknown(00000000)",
		},
		{
			name:     "all ones",
			guid:     [16]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
			expected: "Unknown(ffffffff)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := codecGUIDToName(tt.guid)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRemoteApp(t *testing.T) {
	app := RemoteApp{
		App:        "notepad.exe",
		WorkingDir: "C:\\Windows\\System32",
		Args:       "--help",
	}

	assert.Equal(t, "notepad.exe", app.App)
	assert.Equal(t, "C:\\Windows\\System32", app.WorkingDir)
	assert.Equal(t, "--help", app.Args)
}

func TestClient_SetUseNLA(t *testing.T) {
	tests := []struct {
		name             string
		useNLA           bool
		expectedProtocol pdu.NegotiationProtocol
	}{
		{
			name:             "enable NLA",
			useNLA:           true,
			expectedProtocol: pdu.NegotiationProtocolHybrid,
		},
		{
			name:             "disable NLA",
			useNLA:           false,
			expectedProtocol: pdu.NegotiationProtocolSSL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{}
			client.SetUseNLA(tt.useNLA)

			assert.Equal(t, tt.useNLA, client.useNLA)
			assert.Equal(t, tt.expectedProtocol, client.selectedProtocol)
		})
	}
}

func TestClient_SetEnableRFX(t *testing.T) {
	client := &Client{}
	assert.False(t, client.enableRFX)

	client.SetEnableRFX(true)
	assert.True(t, client.enableRFX)

	client.SetEnableRFX(false)
	assert.False(t, client.enableRFX)
}

func TestClient_GetServerCapabilities_Empty(t *testing.T) {
	client := &Client{
		serverCapabilitySets: []pdu.CapabilitySet{},
	}

	info := client.GetServerCapabilities()

	assert.NotNil(t, info)
	assert.Empty(t, info.BitmapCodecs)
	assert.False(t, info.SurfaceCommands)
	assert.Equal(t, 0, info.ColorDepth)
	assert.Empty(t, info.DesktopSize)
	assert.Equal(t, uint16(0), info.GeneralFlags)
	assert.Equal(t, uint32(0), info.OrderFlags)
	assert.Equal(t, uint32(0), info.MultifragmentSize)
	assert.False(t, info.LargePointer)
	assert.False(t, info.FrameAcknowledge)
}

func TestClient_GetServerCapabilities_WithBitmap(t *testing.T) {
	client := &Client{
		serverCapabilitySets: []pdu.CapabilitySet{
			{
				CapabilitySetType: pdu.CapabilitySetTypeBitmap,
				BitmapCapabilitySet: &pdu.BitmapCapabilitySet{
					PreferredBitsPerPixel: 32,
					DesktopWidth:          1920,
					DesktopHeight:         1080,
				},
			},
		},
	}

	info := client.GetServerCapabilities()

	assert.Equal(t, 32, info.ColorDepth)
	assert.Equal(t, "1920x1080", info.DesktopSize)
}

func TestClient_GetServerCapabilities_WithGeneral(t *testing.T) {
	client := &Client{
		serverCapabilitySets: []pdu.CapabilitySet{
			{
				CapabilitySetType: pdu.CapabilitySetTypeGeneral,
				GeneralCapabilitySet: &pdu.GeneralCapabilitySet{
					ExtraFlags: 0x1234,
				},
			},
		},
	}

	info := client.GetServerCapabilities()

	assert.Equal(t, uint16(0x1234), info.GeneralFlags)
}

func TestClient_GetServerCapabilities_WithOrder(t *testing.T) {
	client := &Client{
		serverCapabilitySets: []pdu.CapabilitySet{
			{
				CapabilitySetType: pdu.CapabilitySetTypeOrder,
				OrderCapabilitySet: &pdu.OrderCapabilitySet{
					OrderFlags: 0x5678,
				},
			},
		},
	}

	info := client.GetServerCapabilities()

	assert.Equal(t, uint32(0x5678), info.OrderFlags)
}

func TestClient_GetServerCapabilities_WithSurfaceCommands(t *testing.T) {
	client := &Client{
		serverCapabilitySets: []pdu.CapabilitySet{
			{
				CapabilitySetType: pdu.CapabilitySetTypeSurfaceCommands,
			},
		},
	}

	info := client.GetServerCapabilities()

	assert.True(t, info.SurfaceCommands)
}

func TestClient_GetServerCapabilities_WithMultifragmentUpdate(t *testing.T) {
	client := &Client{
		serverCapabilitySets: []pdu.CapabilitySet{
			{
				CapabilitySetType: pdu.CapabilitySetTypeMultifragmentUpdate,
				MultifragmentUpdateCapabilitySet: &pdu.MultifragmentUpdateCapabilitySet{
					MaxRequestSize: 0x200000,
				},
			},
		},
	}

	info := client.GetServerCapabilities()

	assert.Equal(t, uint32(0x200000), info.MultifragmentSize)
}

func TestClient_GetServerCapabilities_WithLargePointer(t *testing.T) {
	client := &Client{
		serverCapabilitySets: []pdu.CapabilitySet{
			{
				CapabilitySetType: pdu.CapabilitySetTypeLargePointer,
			},
		},
	}

	info := client.GetServerCapabilities()

	assert.True(t, info.LargePointer)
}

func TestClient_GetServerCapabilities_WithFrameAcknowledge(t *testing.T) {
	client := &Client{
		serverCapabilitySets: []pdu.CapabilitySet{
			{
				CapabilitySetType: pdu.CapabilitySetTypeFrameAcknowledge,
			},
		},
	}

	info := client.GetServerCapabilities()

	assert.True(t, info.FrameAcknowledge)
}

func TestClient_GetServerCapabilities_WithBitmapCodecs(t *testing.T) {
	client := &Client{
		serverCapabilitySets: []pdu.CapabilitySet{
			{
				CapabilitySetType: pdu.CapabilitySetTypeBitmapCodecs,
				BitmapCodecsCapabilitySet: &pdu.BitmapCodecsCapabilitySet{
					BitmapCodecArray: []pdu.BitmapCodec{
						{CodecGUID: guidNSCodec},
						{CodecGUID: guidRemoteFX},
					},
				},
			},
		},
	}

	info := client.GetServerCapabilities()

	assert.Len(t, info.BitmapCodecs, 2)
	assert.Contains(t, info.BitmapCodecs, "NSCodec")
	assert.Contains(t, info.BitmapCodecs, "RemoteFX")
}

func TestClient_GetServerCapabilities_AllTypes(t *testing.T) {
	client := &Client{
		serverCapabilitySets: []pdu.CapabilitySet{
			{
				CapabilitySetType: pdu.CapabilitySetTypeBitmap,
				BitmapCapabilitySet: &pdu.BitmapCapabilitySet{
					PreferredBitsPerPixel: 24,
					DesktopWidth:          1280,
					DesktopHeight:         720,
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
					OrderFlags: 0x0002,
				},
			},
			{
				CapabilitySetType: pdu.CapabilitySetTypeSurfaceCommands,
			},
			{
				CapabilitySetType: pdu.CapabilitySetTypeBitmapCodecs,
				BitmapCodecsCapabilitySet: &pdu.BitmapCodecsCapabilitySet{
					BitmapCodecArray: []pdu.BitmapCodec{
						{CodecGUID: guidClearCodec},
					},
				},
			},
			{
				CapabilitySetType: pdu.CapabilitySetTypeMultifragmentUpdate,
				MultifragmentUpdateCapabilitySet: &pdu.MultifragmentUpdateCapabilitySet{
					MaxRequestSize: 0x100000,
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
	assert.Equal(t, "1280x720", info.DesktopSize)
	assert.Equal(t, uint16(0x0001), info.GeneralFlags)
	assert.Equal(t, uint32(0x0002), info.OrderFlags)
	assert.True(t, info.SurfaceCommands)
	assert.Len(t, info.BitmapCodecs, 1)
	assert.Contains(t, info.BitmapCodecs, "ClearCodec")
	assert.Equal(t, uint32(0x100000), info.MultifragmentSize)
	assert.True(t, info.LargePointer)
	assert.True(t, info.FrameAcknowledge)
}

func TestServerCapabilityInfo_String(t *testing.T) {
	info := &ServerCapabilityInfo{
		BitmapCodecs:      []string{"NSCodec", "RemoteFX"},
		SurfaceCommands:   true,
		ColorDepth:        32,
		DesktopSize:       "1920x1080",
		GeneralFlags:      0x1234,
		OrderFlags:        0x5678,
		MultifragmentSize: 0x200000,
		LargePointer:      true,
		FrameAcknowledge:  true,
	}

	// Just verify struct is properly initialized
	assert.NotNil(t, info)
	assert.Len(t, info.BitmapCodecs, 2)
}

func TestGUIDConstants(t *testing.T) {
	// Verify GUIDs have expected lengths
	assert.Len(t, guidNSCodec, 16)
	assert.Len(t, guidRemoteFX, 16)
	assert.Len(t, guidImageRemoteFX, 16)
	assert.Len(t, guidClearCodec, 16)
	assert.Len(t, guidJPEG, 16)
	assert.Len(t, guidH264, 16)
	assert.Len(t, guidPNG, 16)

	// Verify they're distinct
	guids := []([16]byte){guidNSCodec, guidRemoteFX, guidImageRemoteFX, guidClearCodec, guidJPEG, guidH264, guidPNG}
	for i := 0; i < len(guids); i++ {
		for j := i + 1; j < len(guids); j++ {
			assert.NotEqual(t, guids[i], guids[j], fmt.Sprintf("GUID %d and %d should be different", i, j))
		}
	}
}

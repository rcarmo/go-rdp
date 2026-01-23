package rdpedisp

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCapsPDU_SerializeDeserialize(t *testing.T) {
	tests := []struct {
		name string
		caps CapsPDU
	}{
		{
			name: "basic caps",
			caps: CapsPDU{
				MaxNumMonitors:     1,
				MaxMonitorAreaSize: 1920 * 1080,
			},
		},
		{
			name: "multi-monitor caps",
			caps: CapsPDU{
				MaxNumMonitors:     4,
				MaxMonitorAreaSize: 4096 * 2160 * 4,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := tt.caps.Serialize()
			require.Len(t, data, 16) // 4 (type) + 4 (len) + 4 (monitors) + 4 (area)

			var decoded CapsPDU
			err := decoded.Deserialize(bytes.NewReader(data))
			require.NoError(t, err)

			assert.Equal(t, tt.caps.MaxNumMonitors, decoded.MaxNumMonitors)
			assert.Equal(t, tt.caps.MaxMonitorAreaSize, decoded.MaxMonitorAreaSize)
		})
	}
}

func TestCapsPDU_Deserialize_WrongType(t *testing.T) {
	// Create data with wrong PDU type
	data := []byte{
		0x00, 0x00, 0x00, 0x00, // Wrong type (0 instead of 5)
		0x0C, 0x00, 0x00, 0x00, // Length
		0x01, 0x00, 0x00, 0x00, // Max monitors
		0x00, 0x00, 0x00, 0x00, // Max area
	}

	var caps CapsPDU
	err := caps.Deserialize(bytes.NewReader(data))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected PDU type")
}

func TestMonitorDef_SerializeDeserialize(t *testing.T) {
	tests := []struct {
		name    string
		monitor MonitorDef
	}{
		{
			name: "primary 1080p monitor",
			monitor: MonitorDef{
				Flags:              MonitorFlagPrimary,
				Left:               0,
				Top:                0,
				Width:              1920,
				Height:             1080,
				PhysicalWidth:      527, // mm
				PhysicalHeight:     296, // mm
				Orientation:        OrientationLandscape,
				DesktopScaleFactor: 100,
				DeviceScaleFactor:  100,
			},
		},
		{
			name: "secondary portrait monitor",
			monitor: MonitorDef{
				Flags:              0,
				Left:               1920,
				Top:                0,
				Width:              1080,
				Height:             1920,
				PhysicalWidth:      296,
				PhysicalHeight:     527,
				Orientation:        OrientationPortrait,
				DesktopScaleFactor: 125,
				DeviceScaleFactor:  140,
			},
		},
		{
			name: "negative position",
			monitor: MonitorDef{
				Flags:  MonitorFlagPrimary,
				Left:   -1920,
				Top:    -500,
				Width:  1920,
				Height: 1080,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := tt.monitor.Serialize()
			require.Len(t, data, 40) // 10 fields * 4 bytes each

			var decoded MonitorDef
			err := decoded.Deserialize(bytes.NewReader(data))
			require.NoError(t, err)

			assert.Equal(t, tt.monitor.Flags, decoded.Flags)
			assert.Equal(t, tt.monitor.Left, decoded.Left)
			assert.Equal(t, tt.monitor.Top, decoded.Top)
			assert.Equal(t, tt.monitor.Width, decoded.Width)
			assert.Equal(t, tt.monitor.Height, decoded.Height)
			assert.Equal(t, tt.monitor.PhysicalWidth, decoded.PhysicalWidth)
			assert.Equal(t, tt.monitor.PhysicalHeight, decoded.PhysicalHeight)
			assert.Equal(t, tt.monitor.Orientation, decoded.Orientation)
			assert.Equal(t, tt.monitor.DesktopScaleFactor, decoded.DesktopScaleFactor)
			assert.Equal(t, tt.monitor.DeviceScaleFactor, decoded.DeviceScaleFactor)
		})
	}
}

func TestMonitorLayoutPDU_SerializeDeserialize(t *testing.T) {
	tests := []struct {
		name     string
		monitors []MonitorDef
	}{
		{
			name: "single monitor",
			monitors: []MonitorDef{
				{
					Flags:              MonitorFlagPrimary,
					Width:              1920,
					Height:             1080,
					DesktopScaleFactor: 100,
					DeviceScaleFactor:  100,
				},
			},
		},
		{
			name: "dual monitors",
			monitors: []MonitorDef{
				{
					Flags:  MonitorFlagPrimary,
					Width:  1920,
					Height: 1080,
				},
				{
					Flags:  0,
					Left:   1920,
					Width:  1920,
					Height: 1080,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pdu := MonitorLayoutPDU{Monitors: tt.monitors}
			data := pdu.Serialize()

			// Expected length: 16 (header with MonitorLayoutSize + NumMonitors) + 40 * monitor count
			expectedLen := 16 + 40*len(tt.monitors)
			require.Len(t, data, expectedLen)

			var decoded MonitorLayoutPDU
			err := decoded.Deserialize(bytes.NewReader(data))
			require.NoError(t, err)

			require.Len(t, decoded.Monitors, len(tt.monitors))
			for i, mon := range tt.monitors {
				assert.Equal(t, mon.Flags, decoded.Monitors[i].Flags)
				assert.Equal(t, mon.Width, decoded.Monitors[i].Width)
				assert.Equal(t, mon.Height, decoded.Monitors[i].Height)
			}
		})
	}
}

func TestMonitorLayoutPDU_Deserialize_WrongType(t *testing.T) {
	data := []byte{
		0x00, 0x00, 0x00, 0x00, // Wrong type
		0x10, 0x00, 0x00, 0x00, // Length
		0x28, 0x00, 0x00, 0x00, // MonitorLayoutSize (40)
		0x00, 0x00, 0x00, 0x00, // Monitor count
	}

	var pdu MonitorLayoutPDU
	err := pdu.Deserialize(bytes.NewReader(data))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected PDU type")
}

func TestMonitorLayoutPDU_Deserialize_TooManyMonitors(t *testing.T) {
	data := []byte{
		0x02, 0x00, 0x00, 0x00, // Type
		0x10, 0x00, 0x00, 0x00, // Length
		0x28, 0x00, 0x00, 0x00, // MonitorLayoutSize (40)
		0xFF, 0x00, 0x00, 0x00, // 255 monitors (exceeds limit)
	}

	var pdu MonitorLayoutPDU
	err := pdu.Deserialize(bytes.NewReader(data))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "too many monitors")
}

// TestMonitorLayoutPDU_DimensionConstraints tests that Serialize enforces FreeRDP-compatible constraints
func TestMonitorLayoutPDU_DimensionConstraints(t *testing.T) {
	tests := []struct {
		name           string
		inputWidth     uint32
		inputHeight    uint32
		expectedWidth  uint32
		expectedHeight uint32
	}{
		{
			name:           "odd width becomes even",
			inputWidth:     1921,
			inputHeight:    1080,
			expectedWidth:  1920, // Rounded down to even
			expectedHeight: 1080,
		},
		{
			name:           "width below min becomes 200",
			inputWidth:     100,
			inputHeight:    100,
			expectedWidth:  200,
			expectedHeight: 200,
		},
		{
			name:           "width above max becomes 8192",
			inputWidth:     10000,
			inputHeight:    10000,
			expectedWidth:  8192,
			expectedHeight: 8192,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pdu := MonitorLayoutPDU{
				Monitors: []MonitorDef{{
					Flags:  MonitorFlagPrimary,
					Width:  tt.inputWidth,
					Height: tt.inputHeight,
				}},
			}
			data := pdu.Serialize()

			var decoded MonitorLayoutPDU
			err := decoded.Deserialize(bytes.NewReader(data))
			require.NoError(t, err)

			require.Len(t, decoded.Monitors, 1)
			assert.Equal(t, tt.expectedWidth, decoded.Monitors[0].Width)
			assert.Equal(t, tt.expectedHeight, decoded.Monitors[0].Height)
		})
	}
}

func TestNewSingleMonitorLayout(t *testing.T) {
	tests := []struct {
		width  uint32
		height uint32
	}{
		{1920, 1080},
		{2560, 1440},
		{3840, 2160},
	}

	for _, tt := range tests {
		pdu := NewSingleMonitorLayout(tt.width, tt.height)

		require.Len(t, pdu.Monitors, 1)
		mon := pdu.Monitors[0]

		assert.Equal(t, MonitorFlagPrimary, mon.Flags)
		assert.Equal(t, int32(0), mon.Left)
		assert.Equal(t, int32(0), mon.Top)
		assert.Equal(t, tt.width, mon.Width)
		assert.Equal(t, tt.height, mon.Height)
		assert.Equal(t, OrientationLandscape, mon.Orientation)
		assert.Equal(t, uint32(100), mon.DesktopScaleFactor)
		assert.Equal(t, uint32(100), mon.DeviceScaleFactor)
	}
}

func TestParsePDUType(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		expectType  uint32
		expectError bool
	}{
		{
			name:       "caps PDU type",
			data:       []byte{0x05, 0x00, 0x00, 0x00},
			expectType: PDUTypeCaps,
		},
		{
			name:       "monitor layout PDU type",
			data:       []byte{0x02, 0x00, 0x00, 0x00},
			expectType: PDUTypeMonitorLayout,
		},
		{
			name:        "too short",
			data:        []byte{0x05, 0x00},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pduType, err := ParsePDUType(tt.data)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectType, pduType)
			}
		})
	}
}

func TestChannelName(t *testing.T) {
	assert.Equal(t, "Microsoft::Windows::RDS::DisplayControl", ChannelName)
}

func TestOrientationConstants(t *testing.T) {
	assert.Equal(t, uint32(0), OrientationLandscape)
	assert.Equal(t, uint32(90), OrientationPortrait)
	assert.Equal(t, uint32(180), OrientationLandscapeFlipped)
	assert.Equal(t, uint32(270), OrientationPortraitFlipped)
}

func TestCapsPDU_Deserialize_ReadErrors(t *testing.T) {
	// Test with truncated data
	tests := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"only type", []byte{0x05, 0x00, 0x00, 0x00}},
		{"missing area", []byte{0x05, 0x00, 0x00, 0x00, 0x10, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00}},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var caps CapsPDU
			err := caps.Deserialize(bytes.NewReader(tt.data))
			assert.Error(t, err)
		})
	}
}

func TestMonitorDef_Deserialize_ReadErrors(t *testing.T) {
	// Test with truncated monitor data
	data := []byte{
		0x01, 0x00, 0x00, 0x00, // Flags only
	}
	
	var mon MonitorDef
	err := mon.Deserialize(bytes.NewReader(data))
	assert.Error(t, err)
}

func TestMonitorLayoutPDU_Deserialize_WrongMonitorSize(t *testing.T) {
	// Test with wrong MonitorLayoutSize field
	data := []byte{
		0x02, 0x00, 0x00, 0x00, // Type = MonitorLayout
		0x10, 0x00, 0x00, 0x00, // Length
		0x20, 0x00, 0x00, 0x00, // MonitorLayoutSize = 32 (wrong, should be 40)
		0x00, 0x00, 0x00, 0x00, // NumMonitors = 0
	}
	
	var pdu MonitorLayoutPDU
	err := pdu.Deserialize(bytes.NewReader(data))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected monitor layout size")
}

func TestMonitorLayoutPDU_Deserialize_ReadErrors(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"only type", []byte{0x02, 0x00, 0x00, 0x00}},
		{"missing monitor size", []byte{0x02, 0x00, 0x00, 0x00, 0x10, 0x00, 0x00, 0x00}},
		{"missing count", []byte{0x02, 0x00, 0x00, 0x00, 0x10, 0x00, 0x00, 0x00, 0x28, 0x00, 0x00, 0x00}},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var pdu MonitorLayoutPDU
			err := pdu.Deserialize(bytes.NewReader(tt.data))
			assert.Error(t, err)
		})
	}
}

func TestMonitorLayoutPDU_Serialize_WithTruncation(t *testing.T) {
	// Test that large dimensions are truncated per FreeRDP
	pdu := MonitorLayoutPDU{
		Monitors: []MonitorDef{
			{
				Flags:  MonitorFlagPrimary,
				Width:  9999, // > 8192, should be truncated
				Height: 9999,
			},
		},
	}
	
	data := pdu.Serialize()
	
	var decoded MonitorLayoutPDU
	err := decoded.Deserialize(bytes.NewReader(data))
	require.NoError(t, err)
	
	// Verify truncation
	assert.Equal(t, uint32(8192), decoded.Monitors[0].Width)
	assert.Equal(t, uint32(8192), decoded.Monitors[0].Height)
}

func TestMonitorLayoutPDU_Serialize_WithMinimum(t *testing.T) {
	// Test that small dimensions are expanded per FreeRDP
	pdu := MonitorLayoutPDU{
		Monitors: []MonitorDef{
			{
				Flags:  MonitorFlagPrimary,
				Width:  50, // < 200, should be expanded
				Height: 50,
			},
		},
	}
	
	data := pdu.Serialize()
	
	var decoded MonitorLayoutPDU
	err := decoded.Deserialize(bytes.NewReader(data))
	require.NoError(t, err)
	
	// Verify minimum enforcement
	assert.Equal(t, uint32(200), decoded.Monitors[0].Width)
	assert.Equal(t, uint32(200), decoded.Monitors[0].Height)
}

// ============================================================================
// Microsoft Protocol Test Suite Validation Tests
// Reference: MS-RDPEDISP_ClientTestDesignSpecification.md
// ============================================================================

// TestMonitorLayoutValidation_NoOverlap validates per MS test spec:
// "None of the specified monitors overlap"
func TestMonitorLayoutValidation_NoOverlap(t *testing.T) {
	tests := []struct {
		name     string
		monitors []MonitorDef
		valid    bool
	}{
		{
			name: "non-overlapping side by side",
			monitors: []MonitorDef{
				{Left: 0, Top: 0, Width: 1920, Height: 1080},
				{Left: 1920, Top: 0, Width: 1920, Height: 1080},
			},
			valid: true,
		},
		{
			name: "non-overlapping stacked",
			monitors: []MonitorDef{
				{Left: 0, Top: 0, Width: 1920, Height: 1080},
				{Left: 0, Top: 1080, Width: 1920, Height: 1080},
			},
			valid: true,
		},
		{
			name: "overlapping monitors",
			monitors: []MonitorDef{
				{Left: 0, Top: 0, Width: 1920, Height: 1080},
				{Left: 1000, Top: 0, Width: 1920, Height: 1080}, // Overlaps first by 920 pixels
			},
			valid: false,
		},
		{
			name: "contained monitor",
			monitors: []MonitorDef{
				{Left: 0, Top: 0, Width: 1920, Height: 1080},
				{Left: 100, Top: 100, Width: 800, Height: 600}, // Fully inside first
			},
			valid: false,
		},
		{
			name: "single monitor always valid",
			monitors: []MonitorDef{
				{Left: 0, Top: 0, Width: 1920, Height: 1080},
			},
			valid: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := ValidateNoOverlap(tc.monitors)
			assert.Equal(t, tc.valid, result, "ValidateNoOverlap mismatch for %s", tc.name)
		})
	}
}

// TestMonitorLayoutValidation_Adjacent validates per MS test spec:
// "Each monitor is adjacent to at least one other monitor (even if only at a single point)"
func TestMonitorLayoutValidation_Adjacent(t *testing.T) {
	tests := []struct {
		name     string
		monitors []MonitorDef
		valid    bool
	}{
		{
			name: "adjacent side by side",
			monitors: []MonitorDef{
				{Left: 0, Top: 0, Width: 1920, Height: 1080},
				{Left: 1920, Top: 0, Width: 1920, Height: 1080},
			},
			valid: true,
		},
		{
			name: "adjacent at corner only",
			monitors: []MonitorDef{
				{Left: 0, Top: 0, Width: 1920, Height: 1080},
				{Left: 1920, Top: 1080, Width: 1920, Height: 1080}, // Touches at single point
			},
			valid: true,
		},
		{
			name: "not adjacent - gap between monitors",
			monitors: []MonitorDef{
				{Left: 0, Top: 0, Width: 1920, Height: 1080},
				{Left: 2000, Top: 0, Width: 1920, Height: 1080}, // 80 pixel gap
			},
			valid: false,
		},
		{
			name: "single monitor always valid",
			monitors: []MonitorDef{
				{Left: 0, Top: 0, Width: 1920, Height: 1080},
			},
			valid: true,
		},
		{
			name: "three monitors L-shape",
			monitors: []MonitorDef{
				{Left: 0, Top: 0, Width: 1920, Height: 1080},
				{Left: 1920, Top: 0, Width: 1920, Height: 1080},
				{Left: 0, Top: 1080, Width: 1920, Height: 1080},
			},
			valid: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := ValidateAdjacent(tc.monitors)
			assert.Equal(t, tc.valid, result, "ValidateAdjacent mismatch for %s", tc.name)
		})
	}
}

// TestS1_ResolutionChange_FieldsValid validates per MS test spec S1:
// "All of the fields specified in the DISPLAYCONTROL_MONITOR_LAYOUT_PDU message are valid,
// consistent and within range"
func TestS1_ResolutionChange_FieldsValid(t *testing.T) {
	// Per MS-RDPEDISP 2.2.2.2.1:
	// - Width: 200 to 8192, must be even
	// - Height: 200 to 8192
	// - DesktopScaleFactor: 100 to 500
	// - DeviceScaleFactor: 100 to 500
	// - Orientation: 0, 90, 180, 270

	tests := []struct {
		name    string
		monitor MonitorDef
		valid   bool
	}{
		{
			name: "valid 1080p",
			monitor: MonitorDef{
				Width: 1920, Height: 1080,
				DesktopScaleFactor: 100, DeviceScaleFactor: 100,
				Orientation: OrientationLandscape,
			},
			valid: true,
		},
		{
			name: "valid 4K with scaling",
			monitor: MonitorDef{
				Width: 3840, Height: 2160,
				DesktopScaleFactor: 150, DeviceScaleFactor: 100,
				Orientation: OrientationLandscape,
			},
			valid: true,
		},
		{
			name: "valid portrait",
			monitor: MonitorDef{
				Width: 1080, Height: 1920,
				DesktopScaleFactor: 100, DeviceScaleFactor: 100,
				Orientation: OrientationPortrait,
			},
			valid: true,
		},
		{
			name: "width too small",
			monitor: MonitorDef{
				Width: 100, Height: 1080,
				DesktopScaleFactor: 100, DeviceScaleFactor: 100,
			},
			valid: false,
		},
		{
			name: "width too large",
			monitor: MonitorDef{
				Width: 10000, Height: 1080,
				DesktopScaleFactor: 100, DeviceScaleFactor: 100,
			},
			valid: false,
		},
		{
			name: "odd width invalid",
			monitor: MonitorDef{
				Width: 1921, Height: 1080,
				DesktopScaleFactor: 100, DeviceScaleFactor: 100,
			},
			valid: false,
		},
		{
			name: "scale factor too high",
			monitor: MonitorDef{
				Width: 1920, Height: 1080,
				DesktopScaleFactor: 600, DeviceScaleFactor: 100,
			},
			valid: false,
		},
		{
			name: "invalid orientation",
			monitor: MonitorDef{
				Width: 1920, Height: 1080,
				DesktopScaleFactor: 100, DeviceScaleFactor: 100,
				Orientation: 45, // Invalid
			},
			valid: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := ValidateMonitorDef(&tc.monitor)
			assert.Equal(t, tc.valid, result, "ValidateMonitorDef mismatch for %s", tc.name)
		})
	}
}

// TestS2_OrientationChange_AllOrientations validates per MS test spec S2:
// "Trigger client to change screen orientation from Landscape to Portrait"
func TestS2_OrientationChange_AllOrientations(t *testing.T) {
	orientations := []uint32{
		OrientationLandscape,        // 0
		OrientationPortrait,         // 90
		OrientationLandscapeFlipped, // 180
		OrientationPortraitFlipped,  // 270
	}

	for _, orientation := range orientations {
		monitor := MonitorDef{
			Flags:              MonitorFlagPrimary,
			Width:              1920,
			Height:             1080,
			Orientation:        orientation,
			DesktopScaleFactor: 100,
			DeviceScaleFactor:  100,
		}

		// Verify serialization preserves orientation
		data := monitor.Serialize()
		var decoded MonitorDef
		err := decoded.Deserialize(bytes.NewReader(data))
		require.NoError(t, err)
		assert.Equal(t, orientation, decoded.Orientation, "Orientation %d not preserved", orientation)
	}
}

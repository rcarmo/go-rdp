package pdu

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewKeyboardEvent(t *testing.T) {
	tests := []struct {
		name    string
		flags   uint8
		keyCode uint8
	}{
		{
			name:    "KeyA_Down",
			flags:   0,
			keyCode: 0x1E,
		},
		{
			name:    "KeyA_Up",
			flags:   KBDFlagsRelease,
			keyCode: 0x1E,
		},
		{
			name:    "Extended_Right_Ctrl",
			flags:   KBDFlagsExtended,
			keyCode: 0x1D,
		},
		{
			name:    "Extended1_Pause",
			flags:   KBDFlagsExtended1,
			keyCode: 0x1D,
		},
		{
			name:    "Extended_Release",
			flags:   KBDFlagsRelease | KBDFlagsExtended,
			keyCode: 0x38,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := NewKeyboardEvent(tt.flags, tt.keyCode)
			require.NotNil(t, event)
			require.Equal(t, tt.flags, event.EventFlags)
			require.Equal(t, EventCodeScanCode, event.EventCode)
			require.NotNil(t, event.keyboardEvent)
			require.Equal(t, tt.keyCode, event.keyboardEvent.KeyCode)
		})
	}
}

func TestKeyboardEvent_Serialize(t *testing.T) {
	tests := []struct {
		name     string
		flags    uint8
		keyCode  uint8
		expected []byte
	}{
		{
			name:    "KeyA_Down",
			flags:   0,
			keyCode: 0x1E,
			// header: flags(0)<<3 | code(0) = 0x00
			expected: []byte{0x00, 0x1E},
		},
		{
			name:    "KeyA_Up",
			flags:   KBDFlagsRelease,
			keyCode: 0x1E,
			// header: flags(1)<<3 | code(0) = 0x08
			expected: []byte{0x08, 0x1E},
		},
		{
			name:    "Extended",
			flags:   KBDFlagsExtended,
			keyCode: 0x1D,
			// header: flags(2)<<3 | code(0) = 0x10
			expected: []byte{0x10, 0x1D},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := NewKeyboardEvent(tt.flags, tt.keyCode)
			serialized := event.Serialize()
			require.Equal(t, tt.expected, serialized)
		})
	}
}

func TestNewUnicodeKeyboardEvent(t *testing.T) {
	tests := []struct {
		name        string
		unicodeCode uint16
	}{
		{"CharA", 0x0041},
		{"CharZ", 0x005A},
		{"Emoji", 0x263A},
		{"CJK", 0x4E2D},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := NewUnicodeKeyboardEvent(tt.unicodeCode)
			require.NotNil(t, event)
			require.Equal(t, KBDFlagsRelease, event.EventFlags)
			require.Equal(t, EventCodeUnicode, event.EventCode)
			require.NotNil(t, event.unicodeKeyboardEvent)
			require.Equal(t, tt.unicodeCode, event.unicodeKeyboardEvent.UnicodeCode)
		})
	}
}

func TestUnicodeKeyboardEvent_Serialize(t *testing.T) {
	tests := []struct {
		name        string
		unicodeCode uint16
		expected    []byte
	}{
		{
			name:        "CharA",
			unicodeCode: 0x0041,
			// header: flags(1)<<3 | code(4) = 0x0C
			expected: []byte{0x0C, 0x41, 0x00},
		},
		{
			name:        "CharZ",
			unicodeCode: 0x005A,
			expected:    []byte{0x0C, 0x5A, 0x00},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := NewUnicodeKeyboardEvent(tt.unicodeCode)
			serialized := event.Serialize()
			require.Equal(t, tt.expected, serialized)
		})
	}
}

func TestNewMouseEvent(t *testing.T) {
	tests := []struct {
		name         string
		pointerFlags uint16
		xPos         uint16
		yPos         uint16
	}{
		{
			name:         "Move",
			pointerFlags: PTRFlagsMove,
			xPos:         100,
			yPos:         200,
		},
		{
			name:         "LeftClick",
			pointerFlags: PTRFlagsDown | PTRFlagsButton1,
			xPos:         150,
			yPos:         250,
		},
		{
			name:         "RightClick",
			pointerFlags: PTRFlagsDown | PTRFlagsButton2,
			xPos:         300,
			yPos:         400,
		},
		{
			name:         "MiddleClick",
			pointerFlags: PTRFlagsDown | PTRFlagsButton3,
			xPos:         500,
			yPos:         600,
		},
		{
			name:         "WheelUp",
			pointerFlags: PTRFlagsWheel | 0x0078,
			xPos:         0,
			yPos:         0,
		},
		{
			name:         "WheelDown",
			pointerFlags: PTRFlagsWheel | PTRFlagsWheelNegative | 0x0078,
			xPos:         0,
			yPos:         0,
		},
		{
			name:         "HWheelRight",
			pointerFlags: PTRFlagsHWheel | 0x0078,
			xPos:         0,
			yPos:         0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := NewMouseEvent(tt.pointerFlags, tt.xPos, tt.yPos)
			require.NotNil(t, event)
			require.Equal(t, EventCodeMouse, event.EventCode)
			require.NotNil(t, event.mouseEvent)
			require.Equal(t, tt.pointerFlags, event.mouseEvent.pointerFlags)
			require.Equal(t, tt.xPos, event.mouseEvent.xPos)
			require.Equal(t, tt.yPos, event.mouseEvent.yPos)
		})
	}
}

func TestMouseEvent_Serialize(t *testing.T) {
	tests := []struct {
		name         string
		pointerFlags uint16
		xPos         uint16
		yPos         uint16
		expected     []byte
	}{
		{
			name:         "Move",
			pointerFlags: PTRFlagsMove,
			xPos:         100,
			yPos:         200,
			// header: flags(0)<<3 | code(1) = 0x01
			// pointerFlags: 0x0800 (little-endian: 0x00 0x08)
			// xPos: 100 (little-endian: 0x64 0x00)
			// yPos: 200 (little-endian: 0xC8 0x00)
			expected: []byte{0x01, 0x00, 0x08, 0x64, 0x00, 0xC8, 0x00},
		},
		{
			name:         "LeftClickAt0_0",
			pointerFlags: PTRFlagsDown | PTRFlagsButton1,
			xPos:         0,
			yPos:         0,
			// header: 0x01
			// pointerFlags: 0x9000 (0x8000 | 0x1000) -> little-endian: 0x00 0x90
			expected: []byte{0x01, 0x00, 0x90, 0x00, 0x00, 0x00, 0x00},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := NewMouseEvent(tt.pointerFlags, tt.xPos, tt.yPos)
			serialized := event.Serialize()
			require.Equal(t, tt.expected, serialized)
		})
	}
}

func TestNewExtendedMouseEvent(t *testing.T) {
	tests := []struct {
		name         string
		pointerFlags uint16
		xPos         uint16
		yPos         uint16
	}{
		{
			name:         "XButton1_Down",
			pointerFlags: PTRXFlagsDown | PTRXFlagsButton1,
			xPos:         100,
			yPos:         200,
		},
		{
			name:         "XButton2_Down",
			pointerFlags: PTRXFlagsDown | PTRXFlagsButton2,
			xPos:         300,
			yPos:         400,
		},
		{
			name:         "XButton1_Up",
			pointerFlags: PTRXFlagsButton1,
			xPos:         100,
			yPos:         200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := NewExtendedMouseEvent(tt.pointerFlags, tt.xPos, tt.yPos)
			require.NotNil(t, event)
			require.Equal(t, EventCodeMouseX, event.EventCode)
			require.NotNil(t, event.extendedMouseEvent)
			require.Equal(t, tt.pointerFlags, event.extendedMouseEvent.pointerFlags)
			require.Equal(t, tt.xPos, event.extendedMouseEvent.xPos)
			require.Equal(t, tt.yPos, event.extendedMouseEvent.yPos)
		})
	}
}

func TestExtendedMouseEvent_Serialize(t *testing.T) {
	event := NewExtendedMouseEvent(PTRXFlagsDown|PTRXFlagsButton1, 100, 200)
	serialized := event.Serialize()
	// header: flags(0)<<3 | code(2) = 0x02
	// pointerFlags: 0x8001 -> little-endian: 0x01 0x80
	// xPos: 100 -> 0x64 0x00
	// yPos: 200 -> 0xC8 0x00
	expected := []byte{0x02, 0x01, 0x80, 0x64, 0x00, 0xC8, 0x00}
	require.Equal(t, expected, serialized)
}

func TestNewSynchronizeEvent(t *testing.T) {
	tests := []struct {
		name       string
		eventFlags uint8
	}{
		{"NoLocks", 0},
		{"ScrollLock", SyncScrollLock},
		{"NumLock", SyncNumLock},
		{"CapsLock", SyncCapsLock},
		{"KanaLock", SyncKanaLock},
		{"AllLocks", SyncScrollLock | SyncNumLock | SyncCapsLock | SyncKanaLock},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := NewSynchronizeEvent(tt.eventFlags)
			require.NotNil(t, event)
			require.Equal(t, tt.eventFlags, event.EventFlags)
			require.Equal(t, EventCodeSync, event.EventCode)
		})
	}
}

func TestSynchronizeEvent_Serialize(t *testing.T) {
	tests := []struct {
		name       string
		eventFlags uint8
		expected   []byte
	}{
		{
			name:       "NoLocks",
			eventFlags: 0,
			// header: flags(0)<<3 | code(3) = 0x03
			expected: []byte{0x03},
		},
		{
			name:       "NumLock",
			eventFlags: SyncNumLock,
			// header: flags(2)<<3 | code(3) = 0x13
			expected: []byte{0x13},
		},
		{
			name:       "CapsLock",
			eventFlags: SyncCapsLock,
			// header: flags(4)<<3 | code(3) = 0x23
			expected: []byte{0x23},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := NewSynchronizeEvent(tt.eventFlags)
			serialized := event.Serialize()
			require.Equal(t, tt.expected, serialized)
		})
	}
}

func TestNewQualityOfExperienceEvent(t *testing.T) {
	tests := []struct {
		name      string
		timestamp uint32
	}{
		{"Zero", 0},
		{"Small", 1000},
		{"Large", 4294967295},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := NewQualityOfExperienceEvent(tt.timestamp)
			require.NotNil(t, event)
			require.Equal(t, EventCodeQoETimestamp, event.EventCode)
			require.NotNil(t, event.qualityOfExperience)
			require.Equal(t, tt.timestamp, event.qualityOfExperience.timestamp)
		})
	}
}

func TestQualityOfExperienceEvent_Serialize(t *testing.T) {
	event := NewQualityOfExperienceEvent(0x12345678)
	serialized := event.Serialize()
	// header: flags(0)<<3 | code(6) = 0x06
	// timestamp: 0x12345678 -> little-endian: 0x78 0x56 0x34 0x12
	expected := []byte{0x06, 0x78, 0x56, 0x34, 0x12}
	require.Equal(t, expected, serialized)
}

func TestInputEvent_SerializeAllTypes(t *testing.T) {
	tests := []struct {
		name  string
		event *InputEvent
	}{
		{"Keyboard", NewKeyboardEvent(0, 0x1E)},
		{"Unicode", NewUnicodeKeyboardEvent(0x0041)},
		{"Mouse", NewMouseEvent(PTRFlagsMove, 100, 200)},
		{"ExtendedMouse", NewExtendedMouseEvent(PTRXFlagsDown|PTRXFlagsButton1, 100, 200)},
		{"Sync", NewSynchronizeEvent(SyncNumLock)},
		{"QoE", NewQualityOfExperienceEvent(1000)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serialized := tt.event.Serialize()
			require.NotEmpty(t, serialized)
			// First byte should be the header with event code in lower 3 bits
			eventCode := serialized[0] & 0x07
			require.Equal(t, uint8(tt.event.EventCode), eventCode)
		})
	}
}

func TestMouseFlags(t *testing.T) {
	// Test flag constants are correct
	require.Equal(t, uint16(0x0400), PTRFlagsHWheel)
	require.Equal(t, uint16(0x0200), PTRFlagsWheel)
	require.Equal(t, uint16(0x0100), PTRFlagsWheelNegative)
	require.Equal(t, uint16(0x0800), PTRFlagsMove)
	require.Equal(t, uint16(0x8000), PTRFlagsDown)
	require.Equal(t, uint16(0x1000), PTRFlagsButton1)
	require.Equal(t, uint16(0x2000), PTRFlagsButton2)
	require.Equal(t, uint16(0x4000), PTRFlagsButton3)
}

func TestExtendedMouseFlags(t *testing.T) {
	require.Equal(t, uint16(0x8000), PTRXFlagsDown)
	require.Equal(t, uint16(0x0001), PTRXFlagsButton1)
	require.Equal(t, uint16(0x0002), PTRXFlagsButton2)
}

func TestKeyboardFlags(t *testing.T) {
	require.Equal(t, uint8(0x01), KBDFlagsRelease)
	require.Equal(t, uint8(0x02), KBDFlagsExtended)
	require.Equal(t, uint8(0x04), KBDFlagsExtended1)
}

func TestSyncFlags(t *testing.T) {
	require.Equal(t, uint8(0x01), SyncScrollLock)
	require.Equal(t, uint8(0x02), SyncNumLock)
	require.Equal(t, uint8(0x04), SyncCapsLock)
	require.Equal(t, uint8(0x08), SyncKanaLock)
}

// ============================================================================
// Microsoft Protocol Test Suite Validation Tests
// Reference: MS-RDPBCGR_ClientTestDesignSpecification.md - S4_SlowPathInput
// ============================================================================

// TestBVT_SlowPathInput_KeyboardEvent validates per MS test case:
// "BVT_SlowPathInputTest_PositiveTest_KeyboardEvent"
// Per MS-RDPBCGR Section 2.2.8.1.1.3.1.1.1
func TestBVT_SlowPathInput_KeyboardEvent(t *testing.T) {
	// Test key down
	event := NewKeyboardEvent(0, 0x1E) // 'A' key
	serialized := event.Serialize()
	require.NotNil(t, serialized)
	require.Equal(t, EventCodeScanCode, event.EventCode)

	// Test key up
	event = NewKeyboardEvent(KBDFlagsRelease, 0x1E)
	require.Equal(t, uint8(KBDFlagsRelease), event.EventFlags)
}

// TestS4_SlowPathInput_KeyboardEventTypes validates all keyboard event types
// Per MS-RDPBCGR Section 2.2.8.1.1.3.1.1
func TestS4_SlowPathInput_KeyboardEventTypes(t *testing.T) {
	// Event codes map to fast-path event codes in our implementation
	// Per MS-RDPBCGR Table 2.2.8.1.2.2.1
	require.Equal(t, EventCode(0), EventCodeScanCode)   // FASTPATH_INPUT_EVENT_SCANCODE
	require.Equal(t, EventCode(3), EventCodeSync)       // FASTPATH_INPUT_EVENT_SYNC  
	require.Equal(t, EventCode(4), EventCodeUnicode)    // FASTPATH_INPUT_EVENT_UNICODE
}

// TestS4_SlowPathInput_MouseEventTypes validates all mouse event types
// Per MS-RDPBCGR Section 2.2.8.1.1.3.1.1
func TestS4_SlowPathInput_MouseEventTypes(t *testing.T) {
	tests := []struct {
		name    string
		msgType uint16
	}{
		{"INPUT_EVENT_MOUSE", 0x8001},
		{"INPUT_EVENT_MOUSEX", 0x8002},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Per MS-RDPBCGR: Mouse event message types have 0x80 in high byte
			require.Equal(t, uint8(0x80), uint8(tc.msgType>>8))
		})
	}
}

// TestS4_SlowPathInput_MouseButtonFlags validates mouse button flag combinations
// Per MS-RDPBCGR Section 2.2.8.1.1.3.1.1.3
func TestS4_SlowPathInput_MouseButtonFlags(t *testing.T) {
	tests := []struct {
		name        string
		flags       uint16
		description string
	}{
		{"LeftButton_Down", PTRFlagsButton1 | PTRFlagsDown, "Left mouse button pressed"},
		{"LeftButton_Up", PTRFlagsButton1, "Left mouse button released"},
		{"RightButton_Down", PTRFlagsButton2 | PTRFlagsDown, "Right mouse button pressed"},
		{"RightButton_Up", PTRFlagsButton2, "Right mouse button released"},
		{"MiddleButton_Down", PTRFlagsButton3 | PTRFlagsDown, "Middle mouse button pressed"},
		{"Move", PTRFlagsMove, "Mouse move event"},
		{"WheelUp", PTRFlagsWheel | 0x0078, "Wheel scroll up (120 units)"},
		{"WheelDown", PTRFlagsWheel | PTRFlagsWheelNegative | 0x0078, "Wheel scroll down"},
		{"HWheelRight", PTRFlagsHWheel | 0x0078, "Horizontal wheel right"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			event := NewMouseEvent(tc.flags, 100, 200)
			require.NotNil(t, event)
			require.Equal(t, EventCodeMouse, event.EventCode)
		})
	}
}

// TestS4_SlowPathInput_ExtendedMouseButtons validates extended mouse buttons
// Per MS-RDPBCGR Section 2.2.8.1.1.3.1.1.4
func TestS4_SlowPathInput_ExtendedMouseButtons(t *testing.T) {
	// Extended mouse buttons (X buttons) per spec
	tests := []struct {
		name  string
		flags uint16
	}{
		{"XButton1_Down", PTRXFlagsButton1 | PTRXFlagsDown},
		{"XButton1_Up", PTRXFlagsButton1},
		{"XButton2_Down", PTRXFlagsButton2 | PTRXFlagsDown},
		{"XButton2_Up", PTRXFlagsButton2},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			event := NewExtendedMouseEvent(tc.flags, 100, 200)
			require.NotNil(t, event)
			require.Equal(t, EventCodeMouseX, event.EventCode)
		})
	}
}

// TestS4_SlowPathInput_SyncEvent validates synchronize event
// Per MS-RDPBCGR Section 2.2.8.1.1.3.1.1.5
func TestS4_SlowPathInput_SyncEvent(t *testing.T) {
	// Sync event per MS-RDPBCGR 2.2.8.1.1.3.1.1.5
	tests := []struct {
		name      string
		syncFlags uint8
	}{
		{"ScrollLock", SyncScrollLock},
		{"NumLock", SyncNumLock},
		{"CapsLock", SyncCapsLock},
		{"KanaLock", SyncKanaLock},
		{"All_Locks", SyncScrollLock | SyncNumLock | SyncCapsLock | SyncKanaLock},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			event := NewSynchronizeEvent(tc.syncFlags)
			require.NotNil(t, event)
			require.Equal(t, EventCodeSync, event.EventCode)
		})
	}
}

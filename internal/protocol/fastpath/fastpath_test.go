package fastpath

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Protocol tests
// =============================================================================

func TestNew(t *testing.T) {
	buf := &bytes.Buffer{}
	p := New(buf)

	assert.NotNil(t, p)
	assert.NotNil(t, p.conn)
	assert.NotNil(t, p.updatePDUData)
	assert.Equal(t, 64*1024, len(p.updatePDUData))
}

// =============================================================================
// InputEventPDU tests (send.go)
// =============================================================================

func TestNewInputEventPDU(t *testing.T) {
	eventData := []byte{0x01, 0x02, 0x03}
	pdu := NewInputEventPDU(eventData)

	assert.NotNil(t, pdu)
	assert.Equal(t, uint8(1), pdu.numEvents)
	assert.Equal(t, eventData, pdu.eventData)
}

func TestInputEventPDU_Serialize_TableDriven(t *testing.T) {
	tests := []struct {
		name      string
		pdu       *InputEventPDU
		expected  []byte
	}{
		{
			name: "simple event with short length",
			pdu: &InputEventPDU{
				action:    0,
				numEvents: 1,
				flags:     0,
				eventData: []byte{0x01, 0x02},
			},
			expected: []byte{0x04, 0x04, 0x01, 0x02}, // header=0x04 (numEvents=1<<2), length=4, data
		},
		{
			name: "event with flags",
			pdu: &InputEventPDU{
				action:    0,
				numEvents: 1,
				flags:     0x3, // FASTPATH_INPUT_SECURE_CHECKSUM | FASTPATH_INPUT_ENCRYPTED
				eventData: []byte{0x30, 0x35, 0x6b, 0x5b, 0xb5, 0x34, 0xc8, 0x47, 0x26, 0x18, 0x5e, 0x76, 0x0e, 0xde, 0x28},
			},
			expected: []byte{
				0xc4, 0x11, 0x30, 0x35, 0x6b, 0x5b, 0xb5, 0x34, 0xc8, 0x47, 0x26, 0x18, 0x5e, 0x76, 0x0e, 0xde, 0x28,
			},
		},
		{
			name: "multiple events",
			pdu: &InputEventPDU{
				action:    0,
				numEvents: 3,
				flags:     0,
				eventData: []byte{0xAA, 0xBB, 0xCC},
			},
			expected: []byte{0x0c, 0x05, 0xAA, 0xBB, 0xCC}, // header=0x0c (numEvents=3<<2), length=5, data
		},
		{
			name: "empty event data",
			pdu: &InputEventPDU{
				action:    0,
				numEvents: 0,
				flags:     0,
				eventData: []byte{},
			},
			expected: []byte{0x00, 0x02}, // header=0x00, length=2 (just header+length)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tt.pdu.Serialize()
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestInputEventPDU_SerializeLength(t *testing.T) {
	tests := []struct {
		name        string
		value       int
		expected    []byte
		expectError bool
	}{
		{
			name:     "short length (< 0x7f)",
			value:    10,
			expected: []byte{11}, // value + 1
		},
		{
			name:     "exactly 0x7f",
			value:    0x7f,
			expected: []byte{0x80}, // value + 1 = 0x80, since 0x7f is NOT > 0x7f
		},
		{
			name:     "long length (> 0x7f)",
			value:    0x80,
			expected: []byte{0x80, 0x82}, // 0x82 | 0x8000 = 0x8082 (big endian)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pdu := &InputEventPDU{}
			buf := new(bytes.Buffer)
			err := pdu.SerializeLength(tt.value, buf)
			
			if tt.expectError {
				assert.Error(t, err)
				return
			}
			
			require.NoError(t, err)
			assert.Equal(t, tt.expected, buf.Bytes())
		})
	}
}

func TestProtocol_Send(t *testing.T) {
	tests := []struct {
		name      string
		eventData []byte
		flags     uint8
		numEvents uint8
	}{
		{
			name:      "send simple event",
			eventData: []byte{0x01, 0x02, 0x03},
			flags:     0,
			numEvents: 1,
		},
		{
			name:      "send encrypted event",
			eventData: []byte{0xAA, 0xBB},
			flags:     0x3,
			numEvents: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			p := New(buf)

			pdu := NewInputEventPDU(tt.eventData)
			pdu.flags = tt.flags
			pdu.numEvents = tt.numEvents

			err := p.Send(pdu)
			require.NoError(t, err)
			assert.Greater(t, buf.Len(), 0)
		})
	}
}

// =============================================================================
// UpdatePDU tests (receive.go)
// =============================================================================

func TestUpdatePDU_Deserialize(t *testing.T) {
	tests := []struct {
		name           string
		input          []byte
		expectedAction UpdatePDUAction
		expectedFlags  UpdatePDUFlag
		expectedLen    int
		expectedErr    error
	}{
		{
			name: "fastpath update with 1-byte length",
			// header: 0x00 (action=0, flags=0), length: 0x05, data: 0x01,0x02,0x03,0x04,0x05
			input:          []byte{0x00, 0x05, 0x01, 0x02, 0x03, 0x04, 0x05},
			expectedAction: UpdatePDUActionFastPath,
			expectedFlags:  0,
			expectedLen:    5,
			expectedErr:    nil,
		},
		{
			name: "fastpath update with 2-byte length",
			// header: 0x00, length: 0x80 0x85 (2-byte length = 0x0085 = 133)
			input:          append([]byte{0x00, 0x80, 0x85}, make([]byte, 133)...),
			expectedAction: UpdatePDUActionFastPath,
			expectedFlags:  0,
			expectedLen:    133,
			expectedErr:    nil,
		},
		{
			name: "x224 action returns error",
			// header: 0x03 (action=3 = X224)
			input:          []byte{0x03, 0x05},
			expectedAction: UpdatePDUActionX224,
			expectedErr:    ErrUnexpectedX224,
		},
		{
			name: "encrypted flag returns error",
			// header: 0x80 (flags = 0x2 << 6 = encrypted)
			input:       []byte{0x80, 0x05},
			expectedErr: nil, // will return encryption error
		},
		{
			name: "secure checksum flag returns error",
			// header: 0x40 (flags = 0x1 << 6 = secure checksum)
			input:       []byte{0x40, 0x05},
			expectedErr: nil, // will return checksum error
		},
		{
			name:        "empty input returns EOF",
			input:       []byte{},
			expectedErr: io.EOF,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := bytes.NewBuffer(tt.input)
			pdu := &UpdatePDU{}

			err := pdu.Deserialize(buf)

			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
				return
			}

			if tt.name == "encrypted flag returns error" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "encryption")
				return
			}

			if tt.name == "secure checksum flag returns error" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "checksum")
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedAction, pdu.Action)
			assert.Equal(t, tt.expectedFlags, pdu.Flags)
			assert.Equal(t, tt.expectedLen, len(pdu.Data))
		})
	}
}

func TestUpdatePDU_Deserialize_TooLargePacket(t *testing.T) {
	// Create packet with length > 0x4000
	buf := bytes.NewBuffer([]byte{0x00, 0xC0, 0x01}) // 2-byte length = 0x4001
	pdu := &UpdatePDU{}

	err := pdu.Deserialize(buf)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "too big packet")
}

func TestUpdatePDU_Deserialize_WithPreallocatedData(t *testing.T) {
	input := []byte{0x00, 0x05, 0x01, 0x02, 0x03, 0x04, 0x05}
	buf := bytes.NewBuffer(input)

	pdu := &UpdatePDU{
		Data: make([]byte, 100), // preallocated
	}

	err := pdu.Deserialize(buf)
	require.NoError(t, err)
	assert.Equal(t, 5, len(pdu.Data))
	assert.Equal(t, []byte{0x01, 0x02, 0x03, 0x04, 0x05}, pdu.Data)
}

func TestProtocol_Receive(t *testing.T) {
	// Create valid fastpath update
	data := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	input := append([]byte{0x00, 0x05}, data...)

	buf := bytes.NewBuffer(input)
	p := New(buf)

	pdu, err := p.Receive()
	require.NoError(t, err)
	require.NotNil(t, pdu)
	assert.Equal(t, UpdatePDUActionFastPath, pdu.Action)
	assert.Equal(t, data, pdu.Data)
}

func TestProtocol_Receive_Error(t *testing.T) {
	// Create packet triggering X224 error
	input := []byte{0x03, 0x05}
	buf := bytes.NewBuffer(input)
	p := New(buf)

	pdu, err := p.Receive()
	assert.ErrorIs(t, err, ErrUnexpectedX224)
	assert.Nil(t, pdu)
}

// =============================================================================
// Update tests (receive.go)
// =============================================================================

func TestUpdate_Deserialize(t *testing.T) {
	tests := []struct {
		name                 string
		input                []byte
		expectedUpdateCode   UpdateCode
		expectedFragmentation Fragment
		expectedCompression  Compression
		expectedSize         uint16
		expectedErr          error
	}{
		{
			name: "bitmap update without compression",
			// header: 0x01 (updateCode=1, frag=0, comp=0), size: 0x0005, data: 5 bytes
			input:                append([]byte{0x01, 0x05, 0x00}, make([]byte, 5)...),
			expectedUpdateCode:   UpdateCodeBitmap,
			expectedFragmentation: FragmentSingle,
			expectedCompression:  0,
			expectedSize:         5,
			expectedErr:          nil,
		},
		{
			name: "palette update",
			// header: 0x02 (updateCode=2), size: 0x0003, data: 3 bytes
			input:               append([]byte{0x02, 0x03, 0x00}, make([]byte, 3)...),
			expectedUpdateCode:  UpdateCodePalette,
			expectedSize:        3,
			expectedErr:         nil,
		},
		{
			name: "synchronize update",
			// header: 0x03 (updateCode=3), size: 0x0000
			input:               []byte{0x03, 0x00, 0x00},
			expectedUpdateCode:  UpdateCodeSynchronize,
			expectedSize:        0,
			expectedErr:         nil,
		},
		{
			name: "surface commands update",
			// header: 0x04 (updateCode=4), size: 0x000A, data: 10 bytes
			input:               append([]byte{0x04, 0x0a, 0x00}, make([]byte, 10)...),
			expectedUpdateCode:  UpdateCodeSurfCMDs,
			expectedSize:        10,
			expectedErr:         nil,
		},
		{
			name: "pointer null update",
			input:               []byte{0x05, 0x00, 0x00},
			expectedUpdateCode:  UpdateCodePTRNull,
			expectedSize:        0,
			expectedErr:         nil,
		},
		{
			name: "pointer default update",
			input:               []byte{0x06, 0x00, 0x00},
			expectedUpdateCode:  UpdateCodePTRDefault,
			expectedSize:        0,
			expectedErr:         nil,
		},
		{
			name: "pointer position update",
			// header: 0x08, size: 0x0004, data: 4 bytes (xPos + yPos)
			input:               append([]byte{0x08, 0x04, 0x00}, make([]byte, 4)...),
			expectedUpdateCode:  UpdateCodePTRPosition,
			expectedSize:        4,
			expectedErr:         nil,
		},
		{
			name: "color pointer update",
			input:               append([]byte{0x09, 0x02, 0x00}, make([]byte, 2)...),
			expectedUpdateCode:  UpdateCodeColor,
			expectedSize:        2,
			expectedErr:         nil,
		},
		{
			name: "cached pointer update",
			input:               append([]byte{0x0a, 0x02, 0x00}, make([]byte, 2)...),
			expectedUpdateCode:  UpdateCodeCached,
			expectedSize:        2,
			expectedErr:         nil,
		},
		{
			name: "pointer update",
			input:               append([]byte{0x0b, 0x02, 0x00}, make([]byte, 2)...),
			expectedUpdateCode:  UpdateCodePointer,
			expectedSize:        2,
			expectedErr:         nil,
		},
		{
			name: "large pointer update",
			input:               append([]byte{0x0c, 0x02, 0x00}, make([]byte, 2)...),
			expectedUpdateCode:  UpdateCodeLargePointer,
			expectedSize:        2,
			expectedErr:         nil,
		},
		{
			name: "update with fragmentation first",
			// header: 0x21 (updateCode=1, frag=2<<4=0x20)
			input:                append([]byte{0x21, 0x05, 0x00}, make([]byte, 5)...),
			expectedUpdateCode:   UpdateCodeBitmap,
			expectedFragmentation: FragmentFirst,
			expectedSize:         5,
			expectedErr:          nil,
		},
		{
			name: "update with fragmentation next",
			// header: 0x31 (updateCode=1, frag=3<<4=0x30)
			input:                append([]byte{0x31, 0x05, 0x00}, make([]byte, 5)...),
			expectedUpdateCode:   UpdateCodeBitmap,
			expectedFragmentation: FragmentNext,
			expectedSize:         5,
			expectedErr:          nil,
		},
		{
			name: "update with fragmentation last",
			// header: 0x11 (updateCode=1, frag=1<<4=0x10)
			input:                append([]byte{0x11, 0x05, 0x00}, make([]byte, 5)...),
			expectedUpdateCode:   UpdateCodeBitmap,
			expectedFragmentation: FragmentLast,
			expectedSize:         5,
			expectedErr:          nil,
		},
		{
			name: "update with compression",
			// header: 0x81 (updateCode=1, comp=2<<6=0x80), compressionFlags, size, data
			input:               append([]byte{0x81, 0x01, 0x05, 0x00}, make([]byte, 5)...),
			expectedUpdateCode:  UpdateCodeBitmap,
			expectedCompression: CompressionUsed,
			expectedSize:        5,
			expectedErr:         nil,
		},
		{
			name:        "empty input",
			input:       []byte{},
			expectedErr: io.EOF,
		},
		{
			name:        "incomplete header - missing size",
			input:       []byte{0x01},
			expectedErr: io.EOF,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := bytes.NewBuffer(tt.input)
			update := &Update{}

			err := update.Deserialize(buf)

			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedUpdateCode, update.UpdateCode)
			assert.Equal(t, tt.expectedFragmentation, update.fragmentation)
			assert.Equal(t, tt.expectedCompression, update.compression)
			assert.Equal(t, tt.expectedSize, update.size)
		})
	}
}

func TestUpdateCode_Values(t *testing.T) {
	assert.Equal(t, UpdateCode(0x0), UpdateCodeOrders)
	assert.Equal(t, UpdateCode(0x1), UpdateCodeBitmap)
	assert.Equal(t, UpdateCode(0x2), UpdateCodePalette)
	assert.Equal(t, UpdateCode(0x3), UpdateCodeSynchronize)
	assert.Equal(t, UpdateCode(0x4), UpdateCodeSurfCMDs)
	assert.Equal(t, UpdateCode(0x5), UpdateCodePTRNull)
	assert.Equal(t, UpdateCode(0x6), UpdateCodePTRDefault)
	assert.Equal(t, UpdateCode(0x8), UpdateCodePTRPosition)
	assert.Equal(t, UpdateCode(0x9), UpdateCodeColor)
	assert.Equal(t, UpdateCode(0xa), UpdateCodeCached)
	assert.Equal(t, UpdateCode(0xb), UpdateCodePointer)
	assert.Equal(t, UpdateCode(0xc), UpdateCodeLargePointer)
}

func TestFragment_Values(t *testing.T) {
	assert.Equal(t, Fragment(0x0), FragmentSingle)
	assert.Equal(t, Fragment(0x1), FragmentLast)
	assert.Equal(t, Fragment(0x2), FragmentFirst)
	assert.Equal(t, Fragment(0x3), FragmentNext)
}

func TestUpdatePDUAction_Values(t *testing.T) {
	assert.Equal(t, UpdatePDUAction(0x0), UpdatePDUActionFastPath)
	assert.Equal(t, UpdatePDUAction(0x3), UpdatePDUActionX224)
}

func TestUpdatePDUFlag_Values(t *testing.T) {
	assert.Equal(t, UpdatePDUFlag(0x1), UpdatePDUFlagSecureChecksum)
	assert.Equal(t, UpdatePDUFlag(0x2), UpdatePDUFlagEncrypted)
}

// =============================================================================
// Surface Command tests (surface_commands.go)
// =============================================================================

func TestParseSurfaceCommands(t *testing.T) {
	tests := []struct {
		name        string
		input       []byte
		expectedLen int
		expectedTypes []uint16
	}{
		{
			name:        "empty data",
			input:       []byte{},
			expectedLen: 0,
		},
		{
			name: "frame marker command",
			// cmdType: 0x0004, frameAction: 0x0000 (start), frameId: 0x00000001
			input:        []byte{0x04, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00},
			expectedLen:  1,
			expectedTypes: []uint16{CmdTypeFrameMarker},
		},
		{
			name: "multiple frame markers",
			// First: start frame (0x0004, 0x0000, 0x00000001)
			// Second: end frame (0x0004, 0x0001, 0x00000001)
			input: []byte{
				0x04, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, // frame start
				0x04, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0x00, // frame end
			},
			expectedLen:   2,
			expectedTypes: []uint16{CmdTypeFrameMarker, CmdTypeFrameMarker},
		},
		{
			name: "surface bits command",
			// cmdType: 0x0001 (surface bits)
			// destLeft, destTop, destRight, destBottom: each 2 bytes
			// bpp, flags, reserved, codecID: each 1 byte
			// width, height: each 2 bytes
			// bitmapDataLength: 4 bytes (value=4)
			// bitmapData: 4 bytes
			input: []byte{
				0x01, 0x00, // cmdType
				0x00, 0x00, // destLeft
				0x00, 0x00, // destTop
				0x0a, 0x00, // destRight (10)
				0x0a, 0x00, // destBottom (10)
				0x20,       // bpp (32)
				0x00,       // flags
				0x00,       // reserved
				0x01,       // codecID
				0x0a, 0x00, // width (10)
				0x0a, 0x00, // height (10)
				0x04, 0x00, 0x00, 0x00, // bitmapDataLength
				0xAA, 0xBB, 0xCC, 0xDD, // bitmapData
			},
			expectedLen:   1,
			expectedTypes: []uint16{CmdTypeSurfaceBits},
		},
		{
			name: "stream surface bits command",
			// cmdType: 0x0006 (stream surface bits)
			input: []byte{
				0x06, 0x00, // cmdType
				0x00, 0x00, // destLeft
				0x00, 0x00, // destTop
				0x05, 0x00, // destRight
				0x05, 0x00, // destBottom
				0x18,       // bpp (24)
				0x00,       // flags
				0x00,       // reserved
				0x02,       // codecID
				0x05, 0x00, // width
				0x05, 0x00, // height
				0x02, 0x00, 0x00, 0x00, // bitmapDataLength
				0x11, 0x22, // bitmapData
			},
			expectedLen:   1,
			expectedTypes: []uint16{CmdTypeStreamSurfaceBits},
		},
		{
			name: "unknown command type",
			// cmdType: 0xFF00 (unknown)
			input:        []byte{0x00, 0xFF, 0xAA, 0xBB},
			expectedLen:  1,
			expectedTypes: []uint16{0xFF00},
		},
		{
			name:        "truncated cmdType",
			input:       []byte{0x04},
			expectedLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commands, err := ParseSurfaceCommands(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedLen, len(commands))

			if tt.expectedTypes != nil {
				for i, expectedType := range tt.expectedTypes {
					assert.Equal(t, expectedType, commands[i].CmdType)
				}
			}
		})
	}
}

func TestParseSurfaceCommands_TruncatedData(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
	}{
		{
			name: "truncated frame marker",
			// cmdType present but not enough data for full frame marker
			input: []byte{0x04, 0x00, 0x00, 0x00, 0x01}, // missing 3 bytes
		},
		{
			name: "truncated surface bits header",
			// Only cmdType and partial header
			input: []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0a, 0x00, 0x0a, 0x00},
		},
		{
			name: "truncated surface bits data length",
			// Full header but no data length
			input: []byte{
				0x01, 0x00, // cmdType
				0x00, 0x00, 0x00, 0x00, 0x0a, 0x00, 0x0a, 0x00,
				0x20, 0x00, 0x00, 0x01, 0x0a, 0x00, 0x0a, 0x00,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commands, err := ParseSurfaceCommands(tt.input)
			// Should not error but may return incomplete results
			require.NoError(t, err)
			// The function handles truncated data gracefully
			assert.NotNil(t, commands)
		})
	}
}

func TestParseSetSurfaceBits(t *testing.T) {
	tests := []struct {
		name        string
		input       []byte
		expected    *SetSurfaceBitsCommand
		expectedErr error
	}{
		{
			name: "valid surface bits",
			input: []byte{
				0x10, 0x00, // destLeft (16)
				0x20, 0x00, // destTop (32)
				0x30, 0x00, // destRight (48)
				0x40, 0x00, // destBottom (64)
				0x20,       // bpp (32)
				0x01,       // flags
				0x00,       // reserved
				0x03,       // codecID
				0x14, 0x00, // width (20)
				0x10, 0x00, // height (16)
				0x04, 0x00, 0x00, 0x00, // bitmapDataLength (4)
				0xDE, 0xAD, 0xBE, 0xEF, // bitmapData
			},
			expected: &SetSurfaceBitsCommand{
				DestLeft:   16,
				DestTop:    32,
				DestRight:  48,
				DestBottom: 64,
				BPP:        32,
				Flags:      1,
				Reserved:   0,
				CodecID:    3,
				Width:      20,
				Height:     16,
				BitmapData: []byte{0xDE, 0xAD, 0xBE, 0xEF},
			},
			expectedErr: nil,
		},
		{
			name:        "too short data",
			input:       []byte{0x00, 0x00, 0x00, 0x00, 0x00}, // only 5 bytes
			expectedErr: io.ErrUnexpectedEOF,
		},
		{
			name: "truncated bitmap data",
			input: []byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x20, 0x00, 0x00, 0x01, 0x10, 0x00, 0x10, 0x00,
				0x10, 0x00, 0x00, 0x00, // bitmapDataLength = 16
				0xAA, 0xBB, // only 2 bytes of data
			},
			expectedErr: io.ErrUnexpectedEOF,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := ParseSetSurfaceBits(tt.input)

			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, cmd)
			assert.Equal(t, tt.expected.DestLeft, cmd.DestLeft)
			assert.Equal(t, tt.expected.DestTop, cmd.DestTop)
			assert.Equal(t, tt.expected.DestRight, cmd.DestRight)
			assert.Equal(t, tt.expected.DestBottom, cmd.DestBottom)
			assert.Equal(t, tt.expected.BPP, cmd.BPP)
			assert.Equal(t, tt.expected.Flags, cmd.Flags)
			assert.Equal(t, tt.expected.Reserved, cmd.Reserved)
			assert.Equal(t, tt.expected.CodecID, cmd.CodecID)
			assert.Equal(t, tt.expected.Width, cmd.Width)
			assert.Equal(t, tt.expected.Height, cmd.Height)
			assert.Equal(t, tt.expected.BitmapData, cmd.BitmapData)
		})
	}
}

func TestParseFrameMarker(t *testing.T) {
	tests := []struct {
		name        string
		input       []byte
		expected    *FrameMarkerCommand
		expectedErr error
	}{
		{
			name: "frame start",
			input: []byte{
				0x00, 0x00, // frameAction (start)
				0x01, 0x00, 0x00, 0x00, // frameId (1)
			},
			expected: &FrameMarkerCommand{
				FrameAction: FrameStart,
				FrameID:     1,
			},
		},
		{
			name: "frame end",
			input: []byte{
				0x01, 0x00, // frameAction (end)
				0x42, 0x00, 0x00, 0x00, // frameId (66)
			},
			expected: &FrameMarkerCommand{
				FrameAction: FrameEnd,
				FrameID:     66,
			},
		},
		{
			name:        "too short data",
			input:       []byte{0x00, 0x00, 0x01}, // only 3 bytes
			expectedErr: io.ErrUnexpectedEOF,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := ParseFrameMarker(tt.input)

			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, cmd)
			assert.Equal(t, tt.expected.FrameAction, cmd.FrameAction)
			assert.Equal(t, tt.expected.FrameID, cmd.FrameID)
		})
	}
}

func TestSurfaceCommandConstants(t *testing.T) {
	assert.Equal(t, uint16(0x0001), CmdTypeSurfaceBits)
	assert.Equal(t, uint16(0x0004), CmdTypeFrameMarker)
	assert.Equal(t, uint16(0x0006), CmdTypeStreamSurfaceBits)
	assert.Equal(t, uint16(0x0000), FrameStart)
	assert.Equal(t, uint16(0x0001), FrameEnd)
}

// =============================================================================
// Update Events tests (update_events.go)
// =============================================================================

func TestPaletteEntry_Deserialize(t *testing.T) {
	tests := []struct {
		name        string
		input       []byte
		expectedR   uint8
		expectedG   uint8
		expectedErr error
	}{
		{
			name:      "valid palette entry",
			input:     []byte{0xFF, 0x80, 0x40},
			expectedR: 0xFF,
			expectedG: 0x80, // Fixed: R=0xFF, G=0x80, B=0x40
		},
		{
			name:        "too short",
			input:       []byte{0xFF, 0x80},
			expectedErr: io.EOF,
		},
		{
			name:        "empty",
			input:       []byte{},
			expectedErr: io.EOF,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := bytes.NewBuffer(tt.input)
			entry := &PaletteEntry{}

			err := entry.Deserialize(buf)

			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedR, entry.Red)
			assert.Equal(t, tt.expectedG, entry.Green)
		})
	}
}

func TestPaletteUpdateData_Deserialize(t *testing.T) {
	tests := []struct {
		name             string
		input            []byte
		expectedEntries  int
		expectedErr      error
	}{
		{
			name: "valid palette with 2 entries",
			input: func() []byte {
				buf := new(bytes.Buffer)
				_ = binary.Write(buf, binary.LittleEndian, uint16(0x0002)) // updateType
				_ = binary.Write(buf, binary.LittleEndian, uint16(0x0000)) // padding
				_ = binary.Write(buf, binary.LittleEndian, uint16(2))      // numberColors
				// Entry 1
				buf.Write([]byte{0xFF, 0x00, 0x00}) // R, G, B
				// Entry 2
				buf.Write([]byte{0x00, 0xFF, 0x00}) // R, G, B
				return buf.Bytes()
			}(),
			expectedEntries: 2,
		},
		{
			name: "empty palette",
			input: func() []byte {
				buf := new(bytes.Buffer)
				_ = binary.Write(buf, binary.LittleEndian, uint16(0x0002)) // updateType
				_ = binary.Write(buf, binary.LittleEndian, uint16(0x0000)) // padding
				_ = binary.Write(buf, binary.LittleEndian, uint16(0))      // numberColors
				return buf.Bytes()
			}(),
			expectedEntries: 0,
		},
		{
			name:        "too short header",
			input:       []byte{0x02, 0x00},
			expectedErr: io.EOF,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := bytes.NewBuffer(tt.input)
			data := &paletteUpdateData{}

			err := data.Deserialize(buf)

			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedEntries, len(data.PaletteEntries))
		})
	}
}

func TestCompressedDataHeader_Deserialize(t *testing.T) {
	tests := []struct {
		name        string
		input       []byte
		expectedMain    uint16
		expectedScan    uint16
		expectedUncomp  uint16
		expectedErr error
	}{
		{
			name: "valid compressed header",
			input: func() []byte {
				buf := new(bytes.Buffer)
				_ = binary.Write(buf, binary.LittleEndian, uint16(0x0000)) // cbCompFirstRowSize
				_ = binary.Write(buf, binary.LittleEndian, uint16(0x1000)) // cbCompMainBodySize
				_ = binary.Write(buf, binary.LittleEndian, uint16(0x0040)) // cbScanWidth
				_ = binary.Write(buf, binary.LittleEndian, uint16(0x4000)) // cbUncompressedSize
				return buf.Bytes()
			}(),
			expectedMain:   0x1000,
			expectedScan:   0x0040,
			expectedUncomp: 0x4000,
		},
		{
			name:        "too short",
			input:       []byte{0x00, 0x00, 0x00, 0x00},
			expectedErr: io.EOF,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := bytes.NewBuffer(tt.input)
			header := &CompressedDataHeader{}

			err := header.Deserialize(buf)

			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedMain, header.CbCompMainBodySize)
			assert.Equal(t, tt.expectedScan, header.CbScanWidth)
			assert.Equal(t, tt.expectedUncomp, header.CbUncompressedSize)
		})
	}
}

func TestBitmapData_Deserialize(t *testing.T) {
	tests := []struct {
		name        string
		input       []byte
		expected    *BitmapData
		expectedErr error
	}{
		{
			name: "uncompressed bitmap",
			input: func() []byte {
				buf := new(bytes.Buffer)
				_ = binary.Write(buf, binary.LittleEndian, uint16(0))      // destLeft
				_ = binary.Write(buf, binary.LittleEndian, uint16(0))      // destTop
				_ = binary.Write(buf, binary.LittleEndian, uint16(10))     // destRight
				_ = binary.Write(buf, binary.LittleEndian, uint16(10))     // destBottom
				_ = binary.Write(buf, binary.LittleEndian, uint16(10))     // width
				_ = binary.Write(buf, binary.LittleEndian, uint16(10))     // height
				_ = binary.Write(buf, binary.LittleEndian, uint16(24))     // bitsPerPixel
				_ = binary.Write(buf, binary.LittleEndian, uint16(0))      // flags (no compression)
				_ = binary.Write(buf, binary.LittleEndian, uint16(4))      // bitmapLength
				buf.Write([]byte{0xAA, 0xBB, 0xCC, 0xDD})              // bitmapDataStream
				return buf.Bytes()
			}(),
			expected: &BitmapData{
				DestLeft:     0,
				DestTop:      0,
				DestRight:    10,
				DestBottom:   10,
				Width:        10,
				Height:       10,
				BitsPerPixel: 24,
				Flags:        0,
				BitmapLength: 4,
			},
		},
		{
			name: "compressed with NO_HDR flag",
			input: func() []byte {
				buf := new(bytes.Buffer)
				_ = binary.Write(buf, binary.LittleEndian, uint16(0))                                        // destLeft
				_ = binary.Write(buf, binary.LittleEndian, uint16(0))                                        // destTop
				_ = binary.Write(buf, binary.LittleEndian, uint16(20))                                       // destRight
				_ = binary.Write(buf, binary.LittleEndian, uint16(20))                                       // destBottom
				_ = binary.Write(buf, binary.LittleEndian, uint16(20))                                       // width
				_ = binary.Write(buf, binary.LittleEndian, uint16(20))                                       // height
				_ = binary.Write(buf, binary.LittleEndian, uint16(32))                                       // bitsPerPixel
				_ = binary.Write(buf, binary.LittleEndian, uint16(BitmapDataFlagCompression|BitmapDataFlagNoHDR)) // flags
				_ = binary.Write(buf, binary.LittleEndian, uint16(3))                                        // bitmapLength
				buf.Write([]byte{0x11, 0x22, 0x33})                                                      // bitmapDataStream
				return buf.Bytes()
			}(),
			expected: &BitmapData{
				DestLeft:     0,
				DestTop:      0,
				DestRight:    20,
				DestBottom:   20,
				Width:        20,
				Height:       20,
				BitsPerPixel: 32,
				Flags:        BitmapDataFlagCompression | BitmapDataFlagNoHDR,
				BitmapLength: 3,
			},
		},
		{
			name:        "too short header",
			input:       []byte{0x00, 0x00, 0x00, 0x00},
			expectedErr: io.EOF,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := bytes.NewBuffer(tt.input)
			data := &BitmapData{}

			err := data.Deserialize(buf)

			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected.DestLeft, data.DestLeft)
			assert.Equal(t, tt.expected.DestRight, data.DestRight)
			assert.Equal(t, tt.expected.Width, data.Width)
			assert.Equal(t, tt.expected.Height, data.Height)
			assert.Equal(t, tt.expected.BitsPerPixel, data.BitsPerPixel)
			assert.Equal(t, tt.expected.Flags, data.Flags)
		})
	}
}

func TestBitmapDataFlag_Values(t *testing.T) {
	assert.Equal(t, BitmapDataFlag(0x0001), BitmapDataFlagCompression)
	assert.Equal(t, BitmapDataFlag(0x0400), BitmapDataFlagNoHDR)
}

func TestBitmapUpdateData_Deserialize(t *testing.T) {
	tests := []struct {
		name             string
		input            []byte
		expectedRects    int
		expectedErr      error
	}{
		{
			name: "single rectangle",
			input: func() []byte {
				buf := new(bytes.Buffer)
				_ = binary.Write(buf, binary.LittleEndian, uint16(0x0001)) // updateType
				_ = binary.Write(buf, binary.LittleEndian, uint16(1))      // numberRectangles
				// Bitmap data
				_ = binary.Write(buf, binary.LittleEndian, uint16(0))  // destLeft
				_ = binary.Write(buf, binary.LittleEndian, uint16(0))  // destTop
				_ = binary.Write(buf, binary.LittleEndian, uint16(10)) // destRight
				_ = binary.Write(buf, binary.LittleEndian, uint16(10)) // destBottom
				_ = binary.Write(buf, binary.LittleEndian, uint16(10)) // width
				_ = binary.Write(buf, binary.LittleEndian, uint16(10)) // height
				_ = binary.Write(buf, binary.LittleEndian, uint16(24)) // bitsPerPixel
				_ = binary.Write(buf, binary.LittleEndian, uint16(0))  // flags
				_ = binary.Write(buf, binary.LittleEndian, uint16(2))  // bitmapLength
				buf.Write([]byte{0xAA, 0xBB})                      // data
				return buf.Bytes()
			}(),
			expectedRects: 1,
		},
		{
			name: "zero rectangles",
			input: func() []byte {
				buf := new(bytes.Buffer)
				_ = binary.Write(buf, binary.LittleEndian, uint16(0x0001)) // updateType
				_ = binary.Write(buf, binary.LittleEndian, uint16(0))      // numberRectangles
				return buf.Bytes()
			}(),
			expectedRects: 0,
		},
		{
			name:        "too short header",
			input:       []byte{0x01, 0x00},
			expectedErr: io.EOF,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := bytes.NewBuffer(tt.input)
			data := &bitmapUpdateData{}

			err := data.Deserialize(buf)

			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedRects, len(data.Rectangles))
		})
	}
}

func TestPointerPositionUpdateData_Deserialize(t *testing.T) {
	tests := []struct {
		name        string
		input       []byte
		expectedX   uint16
		expectedY   uint16
		expectedErr error
	}{
		{
			name: "valid position",
			input: func() []byte {
				buf := new(bytes.Buffer)
				_ = binary.Write(buf, binary.LittleEndian, uint16(100)) // xPos
				_ = binary.Write(buf, binary.LittleEndian, uint16(200)) // yPos
				return buf.Bytes()
			}(),
			expectedX: 100,
			expectedY: 200,
		},
		{
			name: "max position",
			input: func() []byte {
				buf := new(bytes.Buffer)
				_ = binary.Write(buf, binary.LittleEndian, uint16(0xFFFF)) // xPos
				_ = binary.Write(buf, binary.LittleEndian, uint16(0xFFFF)) // yPos
				return buf.Bytes()
			}(),
			expectedX: 0xFFFF,
			expectedY: 0xFFFF,
		},
		{
			name:        "too short",
			input:       []byte{0x64, 0x00},
			expectedErr: io.EOF,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := bytes.NewBuffer(tt.input)
			data := &pointerPositionUpdateData{}

			err := data.Deserialize(buf)

			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedX, data.xPos)
			assert.Equal(t, tt.expectedY, data.yPos)
		})
	}
}

func TestColorPointerUpdateData_Deserialize(t *testing.T) {
	tests := []struct {
		name        string
		input       []byte
		expected    *colorPointerUpdateData
		expectedErr error
	}{
		{
			name: "valid color pointer without masks",
			input: func() []byte {
				buf := new(bytes.Buffer)
				_ = binary.Write(buf, binary.LittleEndian, uint16(1))  // cacheIndex
				_ = binary.Write(buf, binary.LittleEndian, uint16(0))  // xPos
				_ = binary.Write(buf, binary.LittleEndian, uint16(0))  // yPos
				_ = binary.Write(buf, binary.LittleEndian, uint16(32)) // width
				_ = binary.Write(buf, binary.LittleEndian, uint16(32)) // height
				_ = binary.Write(buf, binary.LittleEndian, uint16(0))  // lengthAndMask
				_ = binary.Write(buf, binary.LittleEndian, uint16(0))  // lengthXorMask
				buf.WriteByte(0x00)                                // padding
				return buf.Bytes()
			}(),
			expected: &colorPointerUpdateData{
				cacheIndex:    1,
				xPos:          0,
				yPos:          0,
				width:         32,
				height:        32,
				lengthAndMask: 0,
				lengthXorMask: 0,
			},
		},
		{
			name: "valid color pointer with masks",
			input: func() []byte {
				buf := new(bytes.Buffer)
				_ = binary.Write(buf, binary.LittleEndian, uint16(2))   // cacheIndex
				_ = binary.Write(buf, binary.LittleEndian, uint16(10))  // xPos
				_ = binary.Write(buf, binary.LittleEndian, uint16(15))  // yPos
				_ = binary.Write(buf, binary.LittleEndian, uint16(16))  // width
				_ = binary.Write(buf, binary.LittleEndian, uint16(16))  // height
				_ = binary.Write(buf, binary.LittleEndian, uint16(4))   // lengthAndMask
				_ = binary.Write(buf, binary.LittleEndian, uint16(8))   // lengthXorMask
				buf.Write([]byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88}) // xorMaskData
				buf.Write([]byte{0xAA, 0xBB, 0xCC, 0xDD})                          // andMaskData
				buf.WriteByte(0x00)                                                // padding
				return buf.Bytes()
			}(),
			expected: &colorPointerUpdateData{
				cacheIndex:    2,
				xPos:          10,
				yPos:          15,
				width:         16,
				height:        16,
				lengthAndMask: 4,
				lengthXorMask: 8,
			},
		},
		{
			name:        "too short header",
			input:       []byte{0x01, 0x00, 0x00, 0x00},
			expectedErr: io.EOF,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := bytes.NewBuffer(tt.input)
			data := &colorPointerUpdateData{}

			err := data.Deserialize(buf)

			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected.cacheIndex, data.cacheIndex)
			assert.Equal(t, tt.expected.xPos, data.xPos)
			assert.Equal(t, tt.expected.yPos, data.yPos)
			assert.Equal(t, tt.expected.width, data.width)
			assert.Equal(t, tt.expected.height, data.height)
			assert.Equal(t, tt.expected.lengthAndMask, data.lengthAndMask)
			assert.Equal(t, tt.expected.lengthXorMask, data.lengthXorMask)

			if tt.expected.lengthXorMask > 0 {
				assert.Equal(t, int(tt.expected.lengthXorMask), len(data.xorMaskData))
			}
			if tt.expected.lengthAndMask > 0 {
				assert.Equal(t, int(tt.expected.lengthAndMask), len(data.andMaskData))
			}
		})
	}
}

// =============================================================================
// Error handling tests
// =============================================================================

type errorWriter struct {
	err error
}

func (w *errorWriter) Write(p []byte) (n int, err error) {
	return 0, w.err
}

func (w *errorWriter) Read(p []byte) (n int, err error) {
	return 0, w.err
}

func TestProtocol_Send_WriteError(t *testing.T) {
	errWriter := &errorWriter{err: io.ErrClosedPipe}
	p := New(errWriter)

	pdu := NewInputEventPDU([]byte{0x01, 0x02})
	err := p.Send(pdu)

	assert.ErrorIs(t, err, io.ErrClosedPipe)
}

func TestProtocol_Receive_ReadError(t *testing.T) {
	errReader := &errorWriter{err: io.ErrNoProgress}
	p := New(errReader)

	pdu, err := p.Receive()
	assert.ErrorIs(t, err, io.ErrNoProgress)
	assert.Nil(t, pdu)
}

// =============================================================================
// Integration tests
// =============================================================================

func TestRoundTrip_InputEventPDU(t *testing.T) {
	// Test that we can serialize an InputEventPDU and it produces expected bytes
	tests := []struct {
		name      string
		eventData []byte
		flags     uint8
		numEvents uint8
	}{
		{
			name:      "keyboard event",
			eventData: []byte{0x08, 0x1E}, // header + scancode for 'A'
			flags:     0,
			numEvents: 1,
		},
		{
			name:      "mouse event",
			eventData: []byte{0x20, 0x00, 0x08, 0x64, 0x00, 0xC8, 0x00}, // mouse move to (100, 200)
			flags:     0,
			numEvents: 1,
		},
		{
			name:      "multiple events",
			eventData: []byte{0x08, 0x1E, 0x08, 0x1F, 0x08, 0x20}, // multiple key presses
			flags:     0,
			numEvents: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			p := New(buf)

			pdu := &InputEventPDU{
				action:    0,
				numEvents: tt.numEvents,
				flags:     tt.flags,
				eventData: tt.eventData,
			}

			err := p.Send(pdu)
			require.NoError(t, err)

			// Verify the buffer contains the serialized data
			data := buf.Bytes()
			assert.Greater(t, len(data), 0)

			// Verify header byte structure
			header := data[0]
			actualNumEvents := (header >> 2) & 0xf
			actualFlags := (header >> 6) & 0x3
			assert.Equal(t, tt.numEvents, actualNumEvents)
			assert.Equal(t, tt.flags, actualFlags)
		})
	}
}

func TestRoundTrip_UpdatePDU(t *testing.T) {
	// Create a valid UpdatePDU data and deserialize it
	tests := []struct {
		name    string
		data    []byte
		action  UpdatePDUAction
		flags   UpdatePDUFlag
		dataLen int
	}{
		{
			name:    "bitmap update",
			data:    append([]byte{0x00, 0x08}, make([]byte, 8)...), // action=0, len=8, 8 bytes data
			action:  UpdatePDUActionFastPath,
			flags:   0,
			dataLen: 8,
		},
		{
			name:    "synchronize update",
			data:    []byte{0x00, 0x02, 0x00, 0x00}, // action=0, len=2, empty sync
			action:  UpdatePDUActionFastPath,
			flags:   0,
			dataLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := bytes.NewBuffer(tt.data)
			p := New(buf)

			pdu, err := p.Receive()
			require.NoError(t, err)
			require.NotNil(t, pdu)

			assert.Equal(t, tt.action, pdu.Action)
			assert.Equal(t, tt.flags, pdu.Flags)
			assert.Equal(t, tt.dataLen, len(pdu.Data))
		})
	}
}

// =============================================================================
// Compression constant tests
// =============================================================================

func TestCompression_Values(t *testing.T) {
	assert.Equal(t, Compression(0x2), CompressionUsed)
}

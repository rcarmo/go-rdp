package rdp

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChannelFlag_Constants(t *testing.T) {
	tests := []struct {
		name     string
		flag     ChannelFlag
		expected uint32
	}{
		{"ChannelFlagFirst", ChannelFlagFirst, 0x00000001},
		{"ChannelFlagLast", ChannelFlagLast, 0x00000002},
		{"ChannelFlagShowProtocol", ChannelFlagShowProtocol, 0x00000010},
		{"ChannelFlagSuspend", ChannelFlagSuspend, 0x00000020},
		{"ChannelFlagResume", ChannelFlagResume, 0x00000040},
		{"ChannelFlagShadowPersistent", ChannelFlagShadowPersistent, 0x00000080},
		{"ChannelFlagCompressed", ChannelFlagCompressed, 0x00200000},
		{"ChannelFlagAtFront", ChannelFlagAtFront, 0x00400000},
		{"ChannelFlagFlushed", ChannelFlagFlushed, 0x00800000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, uint32(tt.flag))
		})
	}
}

func TestChannelPDUHeader_Serialize(t *testing.T) {
	tests := []struct {
		name     string
		flags    ChannelFlag
		expected []byte
	}{
		{
			name:     "first and last flags",
			flags:    ChannelFlagFirst | ChannelFlagLast,
			expected: []byte{0x08, 0x00, 0x00, 0x00, 0x03, 0x00, 0x00, 0x00},
		},
		{
			name:     "no flags",
			flags:    0,
			expected: []byte{0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		},
		{
			name:     "compressed flag",
			flags:    ChannelFlagCompressed,
			expected: []byte{0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x20, 0x00},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := ChannelPDUHeader{Flags: tt.flags}
			result := header.Serialize()

			require.Len(t, result, 8)
			assert.Equal(t, tt.expected, result)

			// Verify the length field is always 8
			assert.Equal(t, uint32(8), binary.LittleEndian.Uint32(result[0:4]))
			// Verify flags
			assert.Equal(t, uint32(tt.flags), binary.LittleEndian.Uint32(result[4:8]))
		})
	}
}

func TestChannelPDUHeader_Deserialize(t *testing.T) {
	tests := []struct {
		name          string
		data          []byte
		expectedFlags ChannelFlag
		expectError   bool
	}{
		{
			name:          "first and last flags",
			data:          []byte{0x08, 0x00, 0x00, 0x00, 0x03, 0x00, 0x00, 0x00},
			expectedFlags: ChannelFlagFirst | ChannelFlagLast,
			expectError:   false,
		},
		{
			name:          "no flags",
			data:          []byte{0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			expectedFlags: 0,
			expectError:   false,
		},
		{
			name:          "compressed flag",
			data:          []byte{0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x20, 0x00},
			expectedFlags: ChannelFlagCompressed,
			expectError:   false,
		},
		{
			name:          "all flags",
			data:          []byte{0x08, 0x00, 0x00, 0x00, 0xF3, 0x00, 0xE0, 0x00},
			expectedFlags: ChannelFlagFirst | ChannelFlagLast | ChannelFlagShowProtocol | ChannelFlagSuspend | ChannelFlagResume | ChannelFlagShadowPersistent | ChannelFlagCompressed | ChannelFlagAtFront | ChannelFlagFlushed,
			expectError:   false,
		},
		{
			name:        "truncated at flags",
			data:        []byte{0x08, 0x00, 0x00, 0x00},
			expectError: true,
		},
		{
			name:        "truncated at length",
			data:        []byte{0x08, 0x00},
			expectError: true,
		},
		{
			name:        "empty data",
			data:        []byte{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := &ChannelPDUHeader{}
			err := header.Deserialize(bytes.NewReader(tt.data))

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedFlags, header.Flags)
			}
		})
	}
}

func TestChannelPDUHeader_SerializeDeserializeRoundTrip(t *testing.T) {
	flags := []ChannelFlag{
		0,
		ChannelFlagFirst,
		ChannelFlagLast,
		ChannelFlagFirst | ChannelFlagLast,
		ChannelFlagShowProtocol,
		ChannelFlagSuspend,
		ChannelFlagResume,
		ChannelFlagShadowPersistent,
		ChannelFlagCompressed,
		ChannelFlagAtFront,
		ChannelFlagFlushed,
		ChannelFlagFirst | ChannelFlagLast | ChannelFlagCompressed,
	}

	for _, flag := range flags {
		t.Run("flags="+string(rune(flag)), func(t *testing.T) {
			original := ChannelPDUHeader{Flags: flag}
			serialized := original.Serialize()

			deserialized := &ChannelPDUHeader{}
			err := deserialized.Deserialize(bytes.NewReader(serialized))

			require.NoError(t, err)
			assert.Equal(t, original.Flags, deserialized.Flags)
		})
	}
}

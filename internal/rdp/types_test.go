package rdp

import (
	"bufio"
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProtocolCode_IsFastpath(t *testing.T) {
	tests := []struct {
		name     string
		code     ProtocolCode
		expected bool
	}{
		{"0 is fastpath", 0, true},
		{"3 is not fastpath (X224)", 3, false},
		{"4 is fastpath", 4, true},
		{"8 is fastpath", 8, true},
		{"1 is not fastpath", 1, false},
		{"2 is not fastpath", 2, false},
		{"0x80 is fastpath", 0x80, true},
		{"0x81 is not fastpath", 0x81, false},
		{"0xFC is fastpath", 0xFC, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.code.IsFastpath()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestProtocolCode_IsX224(t *testing.T) {
	tests := []struct {
		name     string
		code     ProtocolCode
		expected bool
	}{
		{"3 is X224", 3, true},
		{"0 is not X224", 0, false},
		{"1 is not X224", 1, false},
		{"2 is not X224", 2, false},
		{"4 is not X224", 4, false},
		{"255 is not X224", 255, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.code.IsX224()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestReceiveProtocol(t *testing.T) {
	tests := []struct {
		name          string
		data          []byte
		expectedCode  ProtocolCode
		expectedError bool
	}{
		{
			name:          "fastpath protocol code 0",
			data:          []byte{0x00, 0x01, 0x02},
			expectedCode:  0,
			expectedError: false,
		},
		{
			name:          "X224 protocol code 3",
			data:          []byte{0x03, 0x01, 0x02},
			expectedCode:  3,
			expectedError: false,
		},
		{
			name:          "fastpath protocol code 0x80",
			data:          []byte{0x80, 0x01, 0x02},
			expectedCode:  0x80,
			expectedError: false,
		},
		{
			name:          "protocol code 255",
			data:          []byte{0xFF, 0x01},
			expectedCode:  255,
			expectedError: false,
		},
		{
			name:          "single byte",
			data:          []byte{0x42},
			expectedCode:  0x42,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bufio.NewReader(bytes.NewReader(tt.data))
			code, err := receiveProtocol(reader)

			if tt.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedCode, code)

				// Verify the byte was unread (peek at next byte)
				nextByte, err := reader.ReadByte()
				require.NoError(t, err)
				assert.Equal(t, tt.data[0], nextByte, "UnreadByte should have put the byte back")
			}
		})
	}
}

func TestReceiveProtocol_EmptyReader(t *testing.T) {
	reader := bufio.NewReader(bytes.NewReader([]byte{}))
	_, err := receiveProtocol(reader)
	require.Error(t, err)
	assert.Equal(t, io.EOF, err)
}

func TestReceiveProtocol_MultipleCalls(t *testing.T) {
	data := []byte{0x03, 0x80, 0x00}
	reader := bufio.NewReader(bytes.NewReader(data))

	// First call - should read 0x03 and unread it
	code1, err := receiveProtocol(reader)
	require.NoError(t, err)
	assert.Equal(t, ProtocolCode(0x03), code1)

	// Read and discard the first byte
	_, err = reader.ReadByte()
	require.NoError(t, err)

	// Second call - should read 0x80 and unread it
	code2, err := receiveProtocol(reader)
	require.NoError(t, err)
	assert.Equal(t, ProtocolCode(0x80), code2)
}

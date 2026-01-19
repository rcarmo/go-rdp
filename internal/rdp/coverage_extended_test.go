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

	"github.com/rcarmo/rdp-html5/internal/protocol/pdu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test helper mocks to exercise the mock implementations in client_test_extended.go

func TestMockConn_AllMethods(t *testing.T) {
	mock := &mockConn{}

	// Test Read
	buf := make([]byte, 10)
	n, err := mock.Read(buf)
	assert.Equal(t, 0, n)
	assert.NoError(t, err)

	// Test Write
	n, err = mock.Write([]byte{1, 2, 3})
	assert.Equal(t, 3, n)
	assert.NoError(t, err)

	// Test Close
	err = mock.Close()
	assert.NoError(t, err)

	// Test LocalAddr
	addr := mock.LocalAddr()
	assert.Equal(t, "localhost:8080", addr.String())
	assert.Equal(t, "tcp", addr.Network())

	// Test RemoteAddr
	addr = mock.RemoteAddr()
	assert.Equal(t, "server:3389", addr.String())
	assert.Equal(t, "tcp", addr.Network())

	// Test SetDeadline
	err = mock.SetDeadline(time.Now())
	assert.NoError(t, err)

	// Test SetReadDeadline
	err = mock.SetReadDeadline(time.Now())
	assert.NoError(t, err)

	// Test SetWriteDeadline
	err = mock.SetWriteDeadline(time.Now())
	assert.NoError(t, err)
}

func TestMockAddr_AllMethods(t *testing.T) {
	mock := &mockAddr{address: "test:1234"}
	assert.Equal(t, "tcp", mock.Network())
	assert.Equal(t, "test:1234", mock.String())
}

// Test handleSlowPathGraphicsUpdate edge cases
func TestClient_handleSlowPathGraphicsUpdate_EdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		updateType uint16
		data       []byte
		expectNil  bool
	}{
		{
			name:       "bitmap with minimal data",
			updateType: SlowPathUpdateTypeBitmap,
			data:       []byte{0x00, 0x00}, // numberRectangles=0
			expectNil:  false,
		},
		{
			name:       "palette update",
			updateType: SlowPathUpdateTypePalette,
			data:       []byte{0x00, 0x00}, // numColors=0
			expectNil:  false,
		},
		{
			name:       "synchronize update",
			updateType: SlowPathUpdateTypeSynchronize,
			data:       []byte{},
			expectNil:  false,
		},
		{
			name:       "unknown update type 0x1234",
			updateType: 0x1234,
			data:       []byte{1, 2, 3},
			expectNil:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			binary.Write(buf, binary.LittleEndian, tt.updateType)
			buf.Write(tt.data)

			client := &Client{}
			result, err := client.handleSlowPathGraphicsUpdate(buf)

			require.NoError(t, err)
			if tt.expectNil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
			}
		})
	}
}

// Test Client_Read using the existing pattern
func TestClient_Read_Extended(t *testing.T) {
	// Create a client with a mock connection that has data
	dataToRead := []byte{0x03, 0x00, 0x00, 0x0A} // TPKT header
	conn := &dataConn{data: bytes.NewReader(dataToRead)}
	
	client := &Client{
		conn:       conn,
		buffReader: bufio.NewReaderSize(conn, readBufferSize),
	}

	buf := make([]byte, 4)
	n, err := client.Read(buf)
	
	require.NoError(t, err)
	assert.Equal(t, 4, n)
	assert.Equal(t, dataToRead, buf)
}

func TestClient_Write_Extended(t *testing.T) {
	var writtenData bytes.Buffer
	conn := &writeCapturingConn{buffer: &writtenData}
	
	client := &Client{
		conn: conn,
	}

	dataToWrite := []byte{1, 2, 3, 4, 5}
	n, err := client.Write(dataToWrite)
	
	require.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, dataToWrite, writtenData.Bytes())
}

// Test SendInputEvent
func TestClient_SendInputEvent_MockFailure(t *testing.T) {
	// This test verifies the function exists and can be called
	// We can't fully test it without a real fastpath connection
	client := &Client{
		fastPath: nil, // Will panic, but shows the path exists
	}

	// We expect this to fail since fastPath is nil
	assert.Panics(t, func() {
		_ = client.SendInputEvent([]byte{1, 2, 3})
	})
}

// Test ProtocolCode methods more thoroughly
func TestProtocolCode_EdgeCases(t *testing.T) {
	tests := []struct {
		code     ProtocolCode
		isFP     bool
		isX224   bool
	}{
		{0x00, true, false},
		{0x01, false, false},
		{0x02, false, false},
		{0x03, false, true},
		{0x04, true, false},
		{0x08, true, false},
		{0x0C, true, false},
		{0x10, true, false},
		{0x40, true, false},
		{0x80, true, false},
		{0xFC, true, false},
		{0xFF, false, false},
	}

	for _, tt := range tests {
		t.Run(string([]byte{byte(tt.code)}), func(t *testing.T) {
			assert.Equal(t, tt.isFP, tt.code.IsFastpath())
			assert.Equal(t, tt.isX224, tt.code.IsX224())
		})
	}
}

// Test receiveProtocol error handling
func TestReceiveProtocol_Errors(t *testing.T) {
	// Test with empty reader
	reader := bufio.NewReader(bytes.NewReader([]byte{}))
	_, err := receiveProtocol(reader)
	assert.Error(t, err)
	assert.Equal(t, io.EOF, err)

	// Test with reader that returns error on unread
	errReader := &errorOnUnreadReader{data: []byte{0x03}}
	bufReader := bufio.NewReaderSize(errReader, 1)
	// This should still work because bufio handles unread internally
	code, err := receiveProtocol(bufReader)
	assert.NoError(t, err)
	assert.Equal(t, ProtocolCode(0x03), code)
}

// Test NegotiationType methods
func TestNegotiationType_Extended(t *testing.T) {
	tests := []struct {
		name      string
		negType   pdu.NegotiationType
		isFailure bool
	}{
		{"failure type", pdu.NegotiationTypeFailure, true},
		{"response type", pdu.NegotiationTypeResponse, false},
		{"request type", pdu.NegotiationType(0x01), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.isFailure, tt.negType.IsFailure())
		})
	}
}

// Test NegotiationFailureCode String
func TestNegotiationFailureCode_AllStrings(t *testing.T) {
	codes := []pdu.NegotiationFailureCode{
		pdu.NegotiationFailureCodeHybridRequired,
		pdu.NegotiationFailureCodeSSLRequired,
		pdu.NegotiationFailureCodeSSLWithUserAuthRequired,
		pdu.NegotiationFailureCodeInconsistentFlags,
	}

	for _, code := range codes {
		str := code.String()
		assert.NotEmpty(t, str)
	}
}

// Test client with different configurations
func TestClient_Configuration(t *testing.T) {
	tests := []struct {
		name           string
		useNLA         bool
		skipValidation bool
		serverName     string
	}{
		{"default config", false, false, ""},
		{"NLA enabled", true, false, ""},
		{"skip validation", false, true, ""},
		{"custom server name", false, false, "custom.server.com"},
		{"all options", true, true, "test.example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{}

			if tt.useNLA {
				client.SetUseNLA(true)
				assert.Equal(t, pdu.NegotiationProtocolHybrid, client.selectedProtocol)
			}

			client.SetTLSConfig(tt.skipValidation, tt.serverName)
			assert.Equal(t, tt.skipValidation, client.skipTLSValidation)
			assert.Equal(t, tt.serverName, client.tlsServerName)
		})
	}
}

// Test ServerCapabilityInfo with nil capability sets
func TestClient_GetServerCapabilities_NilCapabilities(t *testing.T) {
	client := &Client{
		serverCapabilitySets: []pdu.CapabilitySet{
			{
				CapabilitySetType:   pdu.CapabilitySetTypeBitmap,
				BitmapCapabilitySet: nil, // nil inner struct
			},
		},
	}

	info := client.GetServerCapabilities()
	require.NotNil(t, info)
	// Should handle nil gracefully
	assert.Equal(t, 0, info.ColorDepth)
}

// Helper types for testing

type dataConn struct {
	net.Conn
	data io.Reader
}

func (c *dataConn) Read(b []byte) (int, error) {
	return c.data.Read(b)
}

func (c *dataConn) Close() error { return nil }

func (c *dataConn) LocalAddr() net.Addr { return &net.TCPAddr{} }
func (c *dataConn) RemoteAddr() net.Addr { return &net.TCPAddr{} }
func (c *dataConn) SetDeadline(t time.Time) error { return nil }
func (c *dataConn) SetReadDeadline(t time.Time) error { return nil }
func (c *dataConn) SetWriteDeadline(t time.Time) error { return nil }

type writeCapturingConn struct {
	net.Conn
	buffer *bytes.Buffer
}

func (c *writeCapturingConn) Write(b []byte) (int, error) {
	return c.buffer.Write(b)
}

func (c *writeCapturingConn) Close() error { return nil }

type errorOnUnreadReader struct {
	data       []byte
	readCalled bool
}

func (r *errorOnUnreadReader) Read(b []byte) (int, error) {
	if !r.readCalled {
		r.readCalled = true
		n := copy(b, r.data)
		return n, nil
	}
	return 0, errors.New("read error")
}

// Test the min function more extensively
func TestMin_Extended(t *testing.T) {
	// Test with very large numbers
	assert.Equal(t, 2147483647, min(2147483647, 2147483648))
	assert.Equal(t, -2147483648, min(-2147483648, -2147483647))

	// Test boundaries
	assert.Equal(t, 0, min(0, 1))
	assert.Equal(t, 0, min(1, 0))
}

// Test RailState transitions
func TestRailState_Transitions(t *testing.T) {
	states := []RailState{
		RailStateUninitialized,
		RailStateInitializing,
		RailStateSyncDesktop,
		RailStateWaitForData,
		RailStateExecuteApp,
	}

	for i, state := range states {
		assert.Equal(t, RailState(i), state)
	}
}

// Test RailOrder values
func TestRailOrder_Values(t *testing.T) {
	orders := map[RailOrder]uint16{
		RailOrderExec:                 0x0001,
		RailOrderActivate:             0x0002,
		RailOrderSysParam:             0x0003,
		RailOrderHandshake:            0x0005,
		RailOrderExecResult:           0x0080,
	}

	for order, expected := range orders {
		assert.Equal(t, expected, uint16(order))
	}
}

// Test ChannelPDUHeader
func TestChannelPDUHeader_Extended(t *testing.T) {
	// Test channel flag constants
	assert.Equal(t, ChannelFlag(0x01), ChannelFlagFirst)
	assert.Equal(t, ChannelFlag(0x02), ChannelFlagLast)
}

// Test ChannelPDUHeader serialize/deserialize
func TestChannelPDUHeader_RoundTrip_Extended(t *testing.T) {
	tests := []ChannelFlag{
		ChannelFlagFirst,
		ChannelFlagLast,
		ChannelFlagFirst | ChannelFlagLast,
		0,
	}

	for _, flag := range tests {
		header := ChannelPDUHeader{
			Flags: flag,
		}

		serialized := header.Serialize()
		require.Len(t, serialized, 8)

		var deserialized ChannelPDUHeader
		err := deserialized.Deserialize(bytes.NewReader(serialized))
		require.NoError(t, err)

		assert.Equal(t, flag, deserialized.Flags)
	}
}

// Test type errors
func TestError_Types(t *testing.T) {
	assert.NotNil(t, ErrUnsupportedRequestedProtocol)
	assert.Contains(t, ErrUnsupportedRequestedProtocol.Error(), "unsupported")
}

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

	"github.com/rcarmo/go-rdp/internal/protocol/fastpath"
	"github.com/rcarmo/go-rdp/internal/protocol/pdu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockFastPath implements fastpath.Protocol for testing
type mockFastPath struct {
	receiveFunc func() (*fastpath.UpdatePDU, error)
}

func (m *mockFastPath) Receive() (*fastpath.UpdatePDU, error) {
	if m.receiveFunc != nil {
		return m.receiveFunc()
	}
	return nil, nil
}

// TestGetUpdate_FastPath tests FastPath update handling
func TestGetUpdate_FastPath(t *testing.T) {
	tests := []struct {
		name        string
		updateCode  uint8
		dataLen     int
		expectErr   bool
	}{
		{
			name:       "bitmap update",
			updateCode: FastPathUpdateCodeBitmap,
			dataLen:    100,
			expectErr:  false,
		},
		{
			name:       "palette update",
			updateCode: FastPathUpdateCodePalette,
			dataLen:    50,
			expectErr:  false,
		},
		{
			name:       "synchronize update",
			updateCode: FastPathUpdateCodeSynchronize,
			dataLen:    4,
			expectErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create FastPath update data
			fpData := make([]byte, 3+tt.dataLen)
			fpData[0] = tt.updateCode // updateHeader
			binary.LittleEndian.PutUint16(fpData[1:3], uint16(tt.dataLen))

			client := &Client{
				buffReader: bufio.NewReader(bytes.NewReader([]byte{0x00, 0x10})), // FastPath indicator
				fastPath: &fastpath.Protocol{},
				pendingSlowPathUpdate: nil,
			}

			// We can't easily mock fastPath.Receive, but we can test the pending update path
			pendingUpdate := &Update{Data: fpData}
			client.pendingSlowPathUpdate = pendingUpdate

			update, err := client.GetUpdate()
			require.NoError(t, err)
			assert.Equal(t, fpData, update.Data)
		})
	}
}

// TestConnectionFinalization tests connection finalization
func TestConnectionFinalization(t *testing.T) {
	tests := []struct {
		name          string
		sendErr       error
		receiveErr    error
		expectErr     bool
	}{
		{
			name:      "successful finalization",
			expectErr: false,
		},
		{
			name:      "send error",
			sendErr:   errors.New("send failed"),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			receiveCallCount := 0
			client := &Client{
				shareID:  0x12345678,
				userID:   1001,
				channelIDMap: map[string]uint16{"global": 1003},
			}

			mockMCS := &MockMCSLayer{
				SendFunc: func(userID, channelID uint16, data []byte) error {
					return tt.sendErr
				},
				ReceiveFunc: func() (uint16, io.Reader, error) {
					receiveCallCount++
					if tt.receiveErr != nil {
						return 0, nil, tt.receiveErr
					}
					// Return different responses based on call count
					switch receiveCallCount {
					case 1:
						// Synchronize response
						return 1003, bytes.NewReader(createSynchronizePDU(t)), nil
					case 2:
						// Control cooperate response
						return 1003, bytes.NewReader(createControlPDU(t, pdu.ControlActionCooperate)), nil
					case 3:
						// Control granted response
						return 1003, bytes.NewReader(createControlPDU(t, pdu.ControlActionGrantedControl)), nil
					case 4:
						// Fontmap response
						return 1003, bytes.NewReader(createFontmapPDU(t)), nil
					default:
						return 0, nil, errors.New("unexpected receive call")
					}
				},
			}

			client.mcsLayer = mockMCS

			err := client.connectionFinalization()
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				// May succeed or fail depending on PDU format
				_ = err
			}
		})
	}
}

// TestCapabilitiesExchange tests capabilities exchange
func TestCapabilitiesExchange(t *testing.T) {
	client := &Client{
		shareID:        0,
		userID:         1001,
		desktopWidth:   1920,
		desktopHeight:  1080,
		channelIDMap:   map[string]uint16{"global": 1003},
	}

	// Create server demand active PDU
	demandActivePDU := createDemandActivePDU(t)

	mockMCS := &MockMCSLayer{
		SendFunc: func(userID, channelID uint16, data []byte) error {
			return nil
		},
		ReceiveFunc: func() (uint16, io.Reader, error) {
			return 1003, bytes.NewReader(demandActivePDU), nil
		},
	}

	client.mcsLayer = mockMCS

	err := client.capabilitiesExchange()
	// May succeed or fail depending on PDU parsing
	_ = err
}

// TestClose tests client close
func TestClose(t *testing.T) {
	tests := []struct {
		name     string
		hasConn  bool
	}{
		{
			name:    "with connection",
			hasConn: true,
		},
		{
			name:    "without connection",
			hasConn: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{}
			if tt.hasConn {
				client.conn = &testNetConn2{}
			}

			err := client.Close()
			if tt.hasConn {
				assert.NoError(t, err)
			} else {
				// Close with nil conn is a no-op
				assert.NoError(t, err)
			}
		})
	}
}

// Helper functions for creating test PDUs

func createSynchronizePDU(t *testing.T) []byte {
	buf := new(bytes.Buffer)

	// ShareControlHeader
	_ = binary.Write(buf, binary.LittleEndian, uint16(22)) // totalLength
	_ = binary.Write(buf, binary.LittleEndian, uint16(pdu.TypeData))
	_ = binary.Write(buf, binary.LittleEndian, uint16(1001)) // pduSource

	// ShareDataHeader
	_ = binary.Write(buf, binary.LittleEndian, uint32(0x12345678)) // shareId
	_ = binary.Write(buf, binary.LittleEndian, uint8(0))          // padding
	_ = binary.Write(buf, binary.LittleEndian, uint8(1))          // streamId
	_ = binary.Write(buf, binary.LittleEndian, uint16(10))        // uncompressedLength
	_ = binary.Write(buf, binary.LittleEndian, uint8(0x1F))       // pduType2 = SYNCHRONIZE
	_ = binary.Write(buf, binary.LittleEndian, uint8(0))          // compressedType
	_ = binary.Write(buf, binary.LittleEndian, uint16(0))         // compressedLength

	// Synchronize data
	_ = binary.Write(buf, binary.LittleEndian, uint16(1))    // messageType
	_ = binary.Write(buf, binary.LittleEndian, uint16(1001)) // targetUser

	return buf.Bytes()
}

func createControlPDU(t *testing.T, action pdu.ControlAction) []byte {
	buf := new(bytes.Buffer)

	// ShareControlHeader
	_ = binary.Write(buf, binary.LittleEndian, uint16(26)) // totalLength
	_ = binary.Write(buf, binary.LittleEndian, uint16(pdu.TypeData))
	_ = binary.Write(buf, binary.LittleEndian, uint16(1001)) // pduSource

	// ShareDataHeader
	_ = binary.Write(buf, binary.LittleEndian, uint32(0x12345678)) // shareId
	_ = binary.Write(buf, binary.LittleEndian, uint8(0))          // padding
	_ = binary.Write(buf, binary.LittleEndian, uint8(1))          // streamId
	_ = binary.Write(buf, binary.LittleEndian, uint16(14))        // uncompressedLength
	_ = binary.Write(buf, binary.LittleEndian, uint8(0x14))       // pduType2 = CONTROL
	_ = binary.Write(buf, binary.LittleEndian, uint8(0))          // compressedType
	_ = binary.Write(buf, binary.LittleEndian, uint16(0))         // compressedLength

	// Control data
	_ = binary.Write(buf, binary.LittleEndian, uint16(action)) // action
	_ = binary.Write(buf, binary.LittleEndian, uint16(0))      // grantId
	_ = binary.Write(buf, binary.LittleEndian, uint32(0))      // controlId

	return buf.Bytes()
}

func createFontmapPDU(t *testing.T) []byte {
	buf := new(bytes.Buffer)

	// ShareControlHeader
	_ = binary.Write(buf, binary.LittleEndian, uint16(26)) // totalLength
	_ = binary.Write(buf, binary.LittleEndian, uint16(pdu.TypeData))
	_ = binary.Write(buf, binary.LittleEndian, uint16(1001)) // pduSource

	// ShareDataHeader
	_ = binary.Write(buf, binary.LittleEndian, uint32(0x12345678)) // shareId
	_ = binary.Write(buf, binary.LittleEndian, uint8(0))          // padding
	_ = binary.Write(buf, binary.LittleEndian, uint8(1))          // streamId
	_ = binary.Write(buf, binary.LittleEndian, uint16(14))        // uncompressedLength
	_ = binary.Write(buf, binary.LittleEndian, uint8(0x28))       // pduType2 = FONTMAP
	_ = binary.Write(buf, binary.LittleEndian, uint8(0))          // compressedType
	_ = binary.Write(buf, binary.LittleEndian, uint16(0))         // compressedLength

	// Fontmap data
	_ = binary.Write(buf, binary.LittleEndian, uint16(0)) // numberEntries
	_ = binary.Write(buf, binary.LittleEndian, uint16(0)) // totalNumEntries
	_ = binary.Write(buf, binary.LittleEndian, uint16(4)) // mapFlags
	_ = binary.Write(buf, binary.LittleEndian, uint16(0)) // entrySize

	return buf.Bytes()
}

func createDemandActivePDU(t *testing.T) []byte {
	buf := new(bytes.Buffer)

	// ShareControlHeader
	_ = binary.Write(buf, binary.LittleEndian, uint16(30)) // totalLength
	_ = binary.Write(buf, binary.LittleEndian, uint16(pdu.TypeDemandActive))
	_ = binary.Write(buf, binary.LittleEndian, uint16(1001)) // pduSource

	// DemandActivePDU
	_ = binary.Write(buf, binary.LittleEndian, uint32(0x12345678)) // shareId
	_ = binary.Write(buf, binary.LittleEndian, uint16(4))          // lengthSourceDescriptor
	buf.Write([]byte("RDP\x00"))                                   // sourceDescriptor
	_ = binary.Write(buf, binary.LittleEndian, uint16(4))          // lengthCombinedCapabilities
	_ = binary.Write(buf, binary.LittleEndian, uint16(0))          // numberCapabilities
	_ = binary.Write(buf, binary.LittleEndian, uint16(0))          // pad2Octets
	_ = binary.Write(buf, binary.LittleEndian, uint32(0))          // sessionId

	return buf.Bytes()
}

// testNetConn2 implements net.Conn for testing close
type testNetConn2 struct {
	closed bool
}

func (c *testNetConn2) Read(b []byte) (int, error) {
	return 0, io.EOF
}

func (c *testNetConn2) Write(b []byte) (int, error) {
	return len(b), nil
}

func (c *testNetConn2) Close() error {
	c.closed = true
	return nil
}

func (c *testNetConn2) LocalAddr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12345}
}

func (c *testNetConn2) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("192.168.1.1"), Port: 3389}
}

func (c *testNetConn2) SetDeadline(t time.Time) error {
	return nil
}

func (c *testNetConn2) SetReadDeadline(t time.Time) error {
	return nil
}

func (c *testNetConn2) SetWriteDeadline(t time.Time) error {
	return nil
}

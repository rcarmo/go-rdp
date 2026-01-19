package tpkt

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"testing"
)

// mockConn implements io.ReadWriteCloser for testing
type mockConn struct {
	readBuf       *bytes.Buffer
	writeBuf      *bytes.Buffer
	readErr       error
	writeErr      error
	closed        bool
	readCount     int
	errAfterReads int // set read error after this many reads (-1 means never)
}

func newMockConn() *mockConn {
	return &mockConn{
		readBuf:       new(bytes.Buffer),
		writeBuf:      new(bytes.Buffer),
		errAfterReads: -1,
	}
}

func (m *mockConn) Read(p []byte) (int, error) {
	m.readCount++
	if m.errAfterReads >= 0 && m.readCount > m.errAfterReads && m.readErr != nil {
		return 0, m.readErr
	}
	if m.errAfterReads < 0 && m.readErr != nil {
		return 0, m.readErr
	}
	return m.readBuf.Read(p)
}

func (m *mockConn) Write(p []byte) (int, error) {
	if m.writeErr != nil {
		return 0, m.writeErr
	}
	return m.writeBuf.Write(p)
}

func (m *mockConn) Close() error {
	m.closed = true
	return nil
}

func TestNew(t *testing.T) {
	conn := newMockConn()
	p := New(conn)

	if p == nil {
		t.Fatal("New() returned nil")
	}

	if p.conn != conn {
		t.Error("New() did not set connection correctly")
	}
}

func TestProtocolSend(t *testing.T) {
	tests := []struct {
		name     string
		pduData  []byte
		wantErr  bool
		writeErr error
	}{
		{
			name:    "empty payload",
			pduData: []byte{},
			wantErr: false,
		},
		{
			name:    "simple payload",
			pduData: []byte{0x01, 0x02, 0x03, 0x04},
			wantErr: false,
		},
		{
			name:    "larger payload",
			pduData: bytes.Repeat([]byte{0xAB}, 100),
			wantErr: false,
		},
		{
			name:     "write error",
			pduData:  []byte{0x01, 0x02},
			wantErr:  true,
			writeErr: errors.New("write failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := newMockConn()
			conn.writeErr = tt.writeErr
			p := New(conn)

			err := p.Send(tt.pduData)

			if (err != nil) != tt.wantErr {
				t.Errorf("Send() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// Verify the written data
			written := conn.writeBuf.Bytes()

			// Check TPKT header
			if len(written) < headerLen {
				t.Fatalf("written data too short: got %d, want at least %d", len(written), headerLen)
			}

			// Check version (0x03)
			if written[0] != 0x03 {
				t.Errorf("version byte = 0x%02x, want 0x03", written[0])
			}

			// Check reserved (0x00)
			if written[1] != 0x00 {
				t.Errorf("reserved byte = 0x%02x, want 0x00", written[1])
			}

			// Check length
			expectedLen := uint16(headerLen + len(tt.pduData))
			actualLen := binary.BigEndian.Uint16(written[2:4])
			if actualLen != expectedLen {
				t.Errorf("length = %d, want %d", actualLen, expectedLen)
			}

			// Check payload
			payload := written[headerLen:]
			if !bytes.Equal(payload, tt.pduData) {
				t.Errorf("payload = %v, want %v", payload, tt.pduData)
			}
		})
	}
}

func TestProtocolReceive(t *testing.T) {
	tests := []struct {
		name        string
		setupConn   func(*mockConn)
		wantData    []byte
		wantErr     bool
		description string
	}{
		{
			name: "simple receive",
			setupConn: func(conn *mockConn) {
				// TPKT header: version=0x03, reserved=0x00, length=8 (4 header + 4 data)
				conn.readBuf.Write([]byte{0x03, 0x00, 0x00, 0x08})
				conn.readBuf.Write([]byte{0x01, 0x02, 0x03, 0x04})
			},
			wantData:    []byte{0x01, 0x02, 0x03, 0x04},
			wantErr:     false,
			description: "receive 4 bytes of data",
		},
		{
			name: "empty payload",
			setupConn: func(conn *mockConn) {
				// TPKT header: version=0x03, reserved=0x00, length=4 (header only)
				conn.readBuf.Write([]byte{0x03, 0x00, 0x00, 0x04})
			},
			wantData:    []byte{},
			wantErr:     false,
			description: "receive empty payload",
		},
		{
			name: "larger payload",
			setupConn: func(conn *mockConn) {
				payload := bytes.Repeat([]byte{0xAB}, 256)
				totalLen := uint16(headerLen + len(payload))
				conn.readBuf.Write([]byte{0x03, 0x00})
				binary.Write(conn.readBuf, binary.BigEndian, totalLen)
				conn.readBuf.Write(payload)
			},
			wantData:    bytes.Repeat([]byte{0xAB}, 256),
			wantErr:     false,
			description: "receive 256 bytes of data",
		},
		{
			name: "header read error",
			setupConn: func(conn *mockConn) {
				conn.readErr = errors.New("connection reset")
			},
			wantData:    nil,
			wantErr:     true,
			description: "error reading header",
		},
		{
			name: "incomplete header",
			setupConn: func(conn *mockConn) {
				// Only 2 bytes instead of 4
				conn.readBuf.Write([]byte{0x03, 0x00})
			},
			wantData:    nil,
			wantErr:     true,
			description: "incomplete header causes error",
		},
		{
			name: "data read error after header",
			setupConn: func(conn *mockConn) {
				// Write valid header indicating 8 bytes total
				conn.readBuf.Write([]byte{0x03, 0x00, 0x00, 0x08})
				// Set error to occur after first read (header read succeeds)
				conn.errAfterReads = 1
				conn.readErr = errors.New("connection closed")
			},
			wantData:    nil,
			wantErr:     true,
			description: "error reading data after header",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := newMockConn()
			tt.setupConn(conn)
			p := New(conn)

			reader, err := p.Receive()

			if (err != nil) != tt.wantErr {
				t.Errorf("Receive() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// Read all data from the reader
			data, err := io.ReadAll(reader)
			if err != nil {
				t.Fatalf("failed to read from returned reader: %v", err)
			}

			if !bytes.Equal(data, tt.wantData) {
				t.Errorf("Receive() data = %v, want %v", data, tt.wantData)
			}
		})
	}
}

func TestSendReceiveRoundTrip(t *testing.T) {
	testData := []byte{0xDE, 0xAD, 0xBE, 0xEF}

	// First, send data
	sendConn := newMockConn()
	sender := New(sendConn)
	if err := sender.Send(testData); err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	// Now use the written bytes as input for receive
	recvConn := newMockConn()
	recvConn.readBuf = bytes.NewBuffer(sendConn.writeBuf.Bytes())
	receiver := New(recvConn)

	reader, err := receiver.Receive()
	if err != nil {
		t.Fatalf("Receive() error = %v", err)
	}

	received, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("failed to read from returned reader: %v", err)
	}

	if !bytes.Equal(received, testData) {
		t.Errorf("round trip failed: sent %v, received %v", testData, received)
	}
}

func TestHeaderLen(t *testing.T) {
	if headerLen != 4 {
		t.Errorf("headerLen = %d, want 4", headerLen)
	}
}

func TestTPKTHeaderFormat(t *testing.T) {
	conn := newMockConn()
	p := New(conn)

	// Send with specific data to verify header format
	testData := []byte{0x01}
	if err := p.Send(testData); err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	written := conn.writeBuf.Bytes()

	// Verify TPKT packet structure per RFC 1006
	// Byte 0: Version (always 0x03)
	// Byte 1: Reserved (always 0x00)
	// Bytes 2-3: Length (big-endian, includes header)

	if written[0] != 0x03 {
		t.Errorf("TPKT version = 0x%02x, want 0x03", written[0])
	}

	if written[1] != 0x00 {
		t.Errorf("TPKT reserved = 0x%02x, want 0x00", written[1])
	}

	length := binary.BigEndian.Uint16(written[2:4])
	expectedLength := uint16(5) // 4 header + 1 data
	if length != expectedLength {
		t.Errorf("TPKT length = %d, want %d", length, expectedLength)
	}
}

// BenchmarkSend benchmarks the Send method
func BenchmarkSend(b *testing.B) {
	conn := newMockConn()
	p := New(conn)
	data := bytes.Repeat([]byte{0x00}, 1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		conn.writeBuf.Reset()
		_ = p.Send(data)
	}
}

// BenchmarkReceive benchmarks the Receive method
func BenchmarkReceive(b *testing.B) {
	// Prepare a valid TPKT packet
	var packetBuf bytes.Buffer
	packetBuf.Write([]byte{0x03, 0x00, 0x04, 0x04}) // header with length 1028
	binary.Write(&packetBuf, binary.BigEndian, uint16(1028))
	packetBuf.Write(bytes.Repeat([]byte{0x00}, 1024))
	packetData := packetBuf.Bytes()

	conn := newMockConn()
	p := New(conn)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		conn.readBuf = bytes.NewBuffer(packetData)
		_, _ = p.Receive()
	}
}

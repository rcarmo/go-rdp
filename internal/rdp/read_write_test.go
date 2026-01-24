package rdp

import (
	"bufio"
	"bytes"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/rcarmo/go-rdp/internal/protocol/pdu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// readWriteTestMockConn is a mock connection for testing Read and Write
type readWriteTestMockConn struct {
	readData    []byte
	readIndex   int
	writeBuffer bytes.Buffer
	readErr     error
	writeErr    error
}

func (m *readWriteTestMockConn) Read(b []byte) (n int, err error) {
	if m.readErr != nil {
		return 0, m.readErr
	}
	if m.readIndex >= len(m.readData) {
		return 0, errors.New("no more data")
	}
	n = copy(b, m.readData[m.readIndex:])
	m.readIndex += n
	return n, nil
}

func (m *readWriteTestMockConn) Write(b []byte) (n int, err error) {
	if m.writeErr != nil {
		return 0, m.writeErr
	}
	return m.writeBuffer.Write(b)
}

func (m *readWriteTestMockConn) Close() error {
	return nil
}

func (m *readWriteTestMockConn) LocalAddr() net.Addr {
	return &mockNetAddr{"localhost:8080"}
}

func (m *readWriteTestMockConn) RemoteAddr() net.Addr {
	return &mockNetAddr{"server:3389"}
}

func (m *readWriteTestMockConn) SetDeadline(t time.Time) error {
	return nil
}

func (m *readWriteTestMockConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (m *readWriteTestMockConn) SetWriteDeadline(t time.Time) error {
	return nil
}

type mockNetAddr struct {
	address string
}

func (m *mockNetAddr) Network() string {
	return "tcp"
}

func (m *mockNetAddr) String() string {
	return m.address
}

func TestClient_Read(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		bufSize     int
		expected    []byte
		expectError bool
	}{
		{
			name:        "read small data",
			data:        []byte{0x01, 0x02, 0x03, 0x04},
			bufSize:     4,
			expected:    []byte{0x01, 0x02, 0x03, 0x04},
			expectError: false,
		},
		{
			name:        "read partial data",
			data:        []byte{0x01, 0x02, 0x03, 0x04, 0x05},
			bufSize:     3,
			expected:    []byte{0x01, 0x02, 0x03},
			expectError: false,
		},
		{
			name:        "read empty data",
			data:        []byte{},
			bufSize:     4,
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockConn := &readWriteTestMockConn{readData: tt.data}
			client := &Client{
				conn:       mockConn,
				buffReader: bufio.NewReader(mockConn),
			}

			buf := make([]byte, tt.bufSize)
			n, err := client.Read(buf)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, len(tt.expected), n)
				assert.Equal(t, tt.expected, buf[:n])
			}
		})
	}
}

func TestClient_Write(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		writeErr    error
		expectError bool
	}{
		{
			name:        "write small data",
			data:        []byte{0x01, 0x02, 0x03, 0x04},
			expectError: false,
		},
		{
			name:        "write large data",
			data:        bytes.Repeat([]byte{0xAA}, 1000),
			expectError: false,
		},
		{
			name:        "write error",
			data:        []byte{0x01},
			writeErr:    errors.New("write error"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockConn := &readWriteTestMockConn{writeErr: tt.writeErr}
			client := &Client{conn: mockConn}

			n, err := client.Write(tt.data)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, len(tt.data), n)
				assert.Equal(t, tt.data, mockConn.writeBuffer.Bytes())
			}
		})
	}
}

func TestClient_initChannels(t *testing.T) {
	tests := []struct {
		name          string
		clientChans   []string
		initialMap    map[string]uint16
		serverData    *pdu.ServerNetworkData
		expectedMap   map[string]uint16
	}{
		{
			name:        "single channel with pre-initialized map",
			clientChans: []string{"cliprdr"},
			initialMap:  make(map[string]uint16),
			serverData: &pdu.ServerNetworkData{
				ChannelIdArray: []uint16{1001},
				MCSChannelId:   1003,
			},
			expectedMap: map[string]uint16{
				"cliprdr": 1001,
				"global":  1003,
			},
		},
		{
			name:        "multiple channels with pre-initialized map",
			clientChans: []string{"cliprdr", "rdpdr", "rdpsnd"},
			initialMap:  make(map[string]uint16),
			serverData: &pdu.ServerNetworkData{
				ChannelIdArray: []uint16{1001, 1002, 1003},
				MCSChannelId:   1004,
			},
			expectedMap: map[string]uint16{
				"cliprdr": 1001,
				"rdpdr":   1002,
				"rdpsnd":  1003,
				"global":  1004,
			},
		},
		{
			name:        "no channels with pre-initialized map",
			clientChans: []string{},
			initialMap:  make(map[string]uint16),
			serverData: &pdu.ServerNetworkData{
				ChannelIdArray: []uint16{},
				MCSChannelId:   1000,
			},
			expectedMap: map[string]uint16{
				"global": 1000,
			},
		},
		{
			name:        "nil channels creates map",
			clientChans: nil,
			initialMap:  nil, // Will be created by initChannels
			serverData: &pdu.ServerNetworkData{
				ChannelIdArray: []uint16{},
				MCSChannelId:   1000,
			},
			expectedMap: map[string]uint16{
				"global": 1000,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				channels:     tt.clientChans,
				channelIDMap: tt.initialMap,
			}

			client.initChannels(tt.serverData)

			assert.Equal(t, tt.expectedMap, client.channelIDMap)
		})
	}
}

func TestClient_initChannels_WithExistingMap(t *testing.T) {
	// When channels is not nil, channelIDMap must be pre-initialized
	client := &Client{
		channels: []string{"rail"},
		channelIDMap: map[string]uint16{
			"existing": 999,
		},
	}

	serverData := &pdu.ServerNetworkData{
		ChannelIdArray: []uint16{2000},
		MCSChannelId:   2001,
	}

	client.initChannels(serverData)

	// Should have the new mapping plus existing
	assert.Equal(t, uint16(2000), client.channelIDMap["rail"])
	assert.Equal(t, uint16(2001), client.channelIDMap["global"])
	assert.Equal(t, uint16(999), client.channelIDMap["existing"])
}

func TestRailPDUClientExecute_Serialize(t *testing.T) {
	tests := []struct {
		name       string
		exeOrFile  string
		workingDir string
		arguments  string
	}{
		{
			name:       "basic execute",
			exeOrFile:  "notepad.exe",
			workingDir: "",
			arguments:  "",
		},
		{
			name:       "with working dir",
			exeOrFile:  "cmd.exe",
			workingDir: "C:\\Windows\\System32",
			arguments:  "",
		},
		{
			name:       "with arguments",
			exeOrFile:  "calc.exe",
			workingDir: "",
			arguments:  "/h",
		},
		{
			name:       "full command",
			exeOrFile:  "powershell.exe",
			workingDir: "C:\\Users\\Test",
			arguments:  "-ExecutionPolicy Bypass -File test.ps1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec := &RailPDUClientExecute{
				Flags:      0,
				ExeOrFile:  tt.exeOrFile,
				WorkingDir: tt.workingDir,
				Arguments:  tt.arguments,
			}

			result := exec.Serialize()

			// Should have at least 8 bytes (2 for flags, 2 for each length field)
			require.GreaterOrEqual(t, len(result), 8)

			// First 2 bytes are flags
			assert.Equal(t, byte(0), result[0])
			assert.Equal(t, byte(0), result[1])
		})
	}
}



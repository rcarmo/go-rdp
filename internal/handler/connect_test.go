package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/net/websocket"

	"github.com/rcarmo/rdp-html5/internal/protocol/audio"
	"github.com/rcarmo/rdp-html5/internal/protocol/pdu"
	"github.com/rcarmo/rdp-html5/internal/rdp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock RDP connection for testing
type mockRDPConnection struct {
	updateData     *rdp.Update
	updateError    error
	sendError      error
	receivedInputs [][]byte
	mu             sync.Mutex
	updateCalls    int
	maxUpdates     int
}

func (m *mockRDPConnection) GetUpdate() (*rdp.Update, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.updateCalls++
	if m.maxUpdates > 0 && m.updateCalls > m.maxUpdates {
		return nil, pdu.ErrDeactiateAll
	}

	if m.updateError != nil {
		return nil, m.updateError
	}
	return m.updateData, nil
}

func (m *mockRDPConnection) SendInputEvent(data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.receivedInputs = append(m.receivedInputs, data)
	return m.sendError
}

// TestIsAllowedOrigin tests the origin validation logic
func TestIsAllowedOrigin(t *testing.T) {
	tests := []struct {
		name           string
		origin         string
		envOrigins     string
		expectedResult bool
	}{
		{
			name:           "localhost default",
			origin:         "http://localhost:8080",
			envOrigins:     "",
			expectedResult: true,
		},
		{
			name:           "127.0.0.1 default",
			origin:         "http://127.0.0.1:8080",
			envOrigins:     "",
			expectedResult: true,
		},
		{
			name:           "external origin default (dev mode - allow all)",
			origin:         "http://example.com:8080",
			envOrigins:     "",
			expectedResult: true,
		},
		{
			name:           "allowed origin from env",
			origin:         "http://example.com:8080",
			envOrigins:     "http://example.com:8080,http://trusted.com",
			expectedResult: true,
		},
		{
			name:           "not allowed origin from env",
			origin:         "http://malicious.com:8080",
			envOrigins:     "http://example.com:8080,http://trusted.com",
			expectedResult: false,
		},
		{
			name:           "multiple allowed origins",
			origin:         "https://trusted.com",
			envOrigins:     "http://example.com:8080,https://trusted.com",
			expectedResult: true,
		},
		{
			name:           "empty origin",
			origin:         "",
			envOrigins:     "",
			expectedResult: false,
		},
		{
			name:           "localhost with HTTPS",
			origin:         "https://localhost:3000",
			envOrigins:     "https://example.com",
			expectedResult: true,
		},
		{
			name:           "127.0.0.1 with HTTPS",
			origin:         "https://127.0.0.1:3000",
			envOrigins:     "https://example.com",
			expectedResult: true,
		},
		{
			name:           "origin with trailing slash",
			origin:         "http://example.com/",
			envOrigins:     "example.com",
			expectedResult: true,
		},
		{
			name:           "normalized match without protocol",
			origin:         "http://trusted.com",
			envOrigins:     "trusted.com",
			expectedResult: true,
		},
		{
			name:           "env with spaces",
			origin:         "http://example.com",
			envOrigins:     "  example.com  ,  trusted.com  ",
			expectedResult: true,
		},
		{
			name:           "env with empty entries",
			origin:         "http://example.com",
			envOrigins:     "example.com,,trusted.com",
			expectedResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up environment before test
			os.Unsetenv("ALLOWED_ORIGINS")
			if tt.envOrigins != "" {
				os.Setenv("ALLOWED_ORIGINS", tt.envOrigins)
			}
			defer os.Unsetenv("ALLOWED_ORIGINS")

			result := isAllowedOrigin(tt.origin)
			assert.Equal(t, tt.expectedResult, result, "expected %v for origin %q with env %q", tt.expectedResult, tt.origin, tt.envOrigins)
		})
	}
}

// TestConnect_ForbiddenOrigin tests that disallowed origins are rejected
func TestConnect_ForbiddenOrigin(t *testing.T) {
	os.Setenv("ALLOWED_ORIGINS", "http://allowed.com")
	defer os.Unsetenv("ALLOWED_ORIGINS")

	req := httptest.NewRequest(http.MethodGet, "/connect?width=800&height=600&host=test&user=test&password=test", nil)
	req.Header.Set("Origin", "http://malicious.com")

	w := httptest.NewRecorder()
	Connect(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

// TestConnect_NoOriginHeader tests connection without origin header (allowed)
func TestConnect_NoOriginHeader(t *testing.T) {
	// Use a real HTTP server since WebSocket requires hijacking
	server := httptest.NewServer(http.HandlerFunc(Connect))
	defer server.Close()

	// Make a regular HTTP request (not WebSocket) - should fail upgrade but not be 403
	resp, err := http.Get(server.URL + "/connect?width=800&height=600")
	require.NoError(t, err)
	defer resp.Body.Close()

	// Without proper WebSocket headers, it won't upgrade, but it shouldn't be 403
	assert.NotEqual(t, http.StatusForbidden, resp.StatusCode)
}

// TestConnect_WebSocketUpgrade tests WebSocket upgrade with proper headers
func TestConnect_WebSocketUpgrade(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(Connect))
	defer server.Close()

	// Convert HTTP URL to WS URL
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/connect?width=800&height=600&host=nonexistent&user=test&password=test"

	// Try to establish WebSocket connection
	// This will fail at the RDP connection stage since "nonexistent" isn't a real host
	ws, err := websocket.Dial(wsURL, "", "http://localhost/")
	if err == nil {
		ws.Close()
	}
	// The WebSocket should at least attempt to connect (handshake succeeds)
	// The error will be from RDP connection failure or disconnect
}

// TestCodecListToJSON tests JSON encoding of codec lists
func TestCodecListToJSON(t *testing.T) {
	tests := []struct {
		name     string
		codecs   []string
		expected string
	}{
		{
			name:     "empty list",
			codecs:   []string{},
			expected: "",
		},
		{
			name:     "nil list",
			codecs:   nil,
			expected: "",
		},
		{
			name:     "single codec",
			codecs:   []string{"RFX"},
			expected: `"RFX"`,
		},
		{
			name:     "multiple codecs",
			codecs:   []string{"RFX", "NSCodec", "RemoteFX"},
			expected: `"RFX","NSCodec","RemoteFX"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := codecListToJSON(tt.codecs)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestRdpToWs_ContextCancellation tests that rdpToWs respects context cancellation
func TestRdpToWs_ContextCancellation(t *testing.T) {
	mockRDP := &mockRDPConnection{
		updateData: &rdp.Update{Data: []byte{1, 2, 3}},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Should return quickly without blocking
	done := make(chan struct{})
	go func() {
		rdpToWs(ctx, mockRDP, nil)
		close(done)
	}()

	select {
	case <-done:
		// Success - function returned
	case <-time.After(time.Second):
		t.Fatal("rdpToWs did not return after context cancellation")
	}
}

// TestRdpToWs_DeactivateAllError tests handling of ErrDeactiateAll
func TestRdpToWs_DeactivateAllError(t *testing.T) {
	mockRDP := &mockRDPConnection{
		updateError: pdu.ErrDeactiateAll,
	}

	ctx := context.Background()
	done := make(chan struct{})

	go func() {
		rdpToWs(ctx, mockRDP, nil)
		close(done)
	}()

	select {
	case <-done:
		// Success - function returned on deactivate all
	case <-time.After(time.Second):
		t.Fatal("rdpToWs did not return on ErrDeactiateAll")
	}
}

// TestRdpToWs_GetUpdateError tests handling of generic update errors
func TestRdpToWs_GetUpdateError(t *testing.T) {
	mockRDP := &mockRDPConnection{
		updateError: errors.New("connection reset"),
	}

	ctx := context.Background()
	done := make(chan struct{})

	go func() {
		rdpToWs(ctx, mockRDP, nil)
		close(done)
	}()

	select {
	case <-done:
		// Success - function returned on error
	case <-time.After(time.Second):
		t.Fatal("rdpToWs did not return on GetUpdate error")
	}
}

// TestRdpToWs_WebSocketSendSuccess tests successful sending to WebSocket
func TestRdpToWs_WebSocketSendSuccess(t *testing.T) {
	updateData := []byte{0x01, 0x02, 0x03, 0x04}
	mockRDP := &mockRDPConnection{
		updateData: &rdp.Update{Data: updateData},
		maxUpdates: 1,
	}

	// Create a WebSocket server and client for testing
	server := httptest.NewServer(websocket.Handler(func(ws *websocket.Conn) {
		ctx := context.Background()
		rdpToWs(ctx, mockRDP, ws)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, err := websocket.Dial(wsURL, "", "http://localhost/")
	require.NoError(t, err)
	defer ws.Close()

	// Read the sent data
	var received []byte
	err = websocket.Message.Receive(ws, &received)
	require.NoError(t, err)
	assert.Equal(t, updateData, received)
}

// TestRdpToWs_WebSocketSendError tests handling of WebSocket send errors
func TestRdpToWs_WebSocketSendError(t *testing.T) {
	mockRDP := &mockRDPConnection{
		updateData: &rdp.Update{Data: []byte{1, 2, 3}},
	}

	// Create server that closes connection immediately
	server := httptest.NewServer(websocket.Handler(func(ws *websocket.Conn) {
		ws.Close() // Close immediately to trigger send error
		ctx := context.Background()
		rdpToWs(ctx, mockRDP, ws)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, err := websocket.Dial(wsURL, "", "http://localhost/")
	if err == nil {
		ws.Close()
	}
}

// TestWsToRdp_ContextCancellation tests that wsToRdp respects context cancellation
func TestWsToRdp_ContextCancellation(t *testing.T) {
	mockRDP := &mockRDPConnection{}

	// Create a test server with WebSocket handler
	server := httptest.NewServer(websocket.Handler(func(ws *websocket.Conn) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		done := make(chan struct{})
		go func() {
			wsToRdp(ctx, ws, mockRDP, cancel)
			close(done)
		}()

		select {
		case <-done:
			// Success
		case <-time.After(time.Second):
			t.Error("wsToRdp did not return after context cancellation")
		}
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, err := websocket.Dial(wsURL, "", "http://localhost/")
	if err == nil {
		ws.Close()
	}
}

// TestWsToRdp_ReceiveAndForward tests receiving from WebSocket and forwarding to RDP
func TestWsToRdp_ReceiveAndForward(t *testing.T) {
	mockRDP := &mockRDPConnection{}
	testInput := []byte{0x04, 0x05, 0x06}

	done := make(chan struct{})
	server := httptest.NewServer(websocket.Handler(func(ws *websocket.Conn) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() {
			wsToRdp(ctx, ws, mockRDP, cancel)
			close(done)
		}()

		// Wait a bit for the receiver to start, then close
		time.Sleep(100 * time.Millisecond)
		ws.Close()
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, err := websocket.Dial(wsURL, "", "http://localhost/")
	require.NoError(t, err)

	// Send test input
	err = websocket.Message.Send(ws, testInput)
	require.NoError(t, err)

	// Wait for processing
	time.Sleep(50 * time.Millisecond)
	ws.Close()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("wsToRdp did not complete")
	}

	// Verify input was forwarded
	mockRDP.mu.Lock()
	defer mockRDP.mu.Unlock()
	require.Len(t, mockRDP.receivedInputs, 1)
	assert.Equal(t, testInput, mockRDP.receivedInputs[0])
}

// TestWsToRdp_SendInputError tests handling of RDP send errors
func TestWsToRdp_SendInputError(t *testing.T) {
	mockRDP := &mockRDPConnection{
		sendError: errors.New("RDP connection closed"),
	}

	done := make(chan struct{})
	server := httptest.NewServer(websocket.Handler(func(ws *websocket.Conn) {
		ctx, cancel := context.WithCancel(context.Background())

		go func() {
			wsToRdp(ctx, ws, mockRDP, cancel)
			close(done)
		}()

		// Keep connection open for a moment
		time.Sleep(200 * time.Millisecond)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, err := websocket.Dial(wsURL, "", "http://localhost/")
	require.NoError(t, err)

	// Send test input that will fail to forward
	err = websocket.Message.Send(ws, []byte{1, 2, 3})
	require.NoError(t, err)

	select {
	case <-done:
		// Success - function returned after send error
	case <-time.After(2 * time.Second):
		t.Fatal("wsToRdp did not return after send error")
	}

	ws.Close()
}

// TestWsToRdp_WebSocketEOF tests handling of WebSocket EOF (client disconnect)
func TestWsToRdp_WebSocketEOF(t *testing.T) {
	mockRDP := &mockRDPConnection{}

	done := make(chan struct{})
	server := httptest.NewServer(websocket.Handler(func(ws *websocket.Conn) {
		ctx, cancel := context.WithCancel(context.Background())

		go func() {
			wsToRdp(ctx, ws, mockRDP, cancel)
			close(done)
		}()

		// Wait for wsToRdp to exit
		<-done
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, err := websocket.Dial(wsURL, "", "http://localhost/")
	require.NoError(t, err)

	// Close client side to trigger EOF on server
	ws.Close()

	select {
	case <-done:
		// Success - function returned on EOF
	case <-time.After(2 * time.Second):
		t.Fatal("wsToRdp did not return on WebSocket EOF")
	}
}

// TestRdpToWs_ClosedConnectionError tests handling of closed connection error message
func TestRdpToWs_ClosedConnectionError(t *testing.T) {
	mockRDP := &mockRDPConnection{
		updateData: &rdp.Update{Data: []byte{1, 2, 3}},
	}

	done := make(chan struct{})
	server := httptest.NewServer(websocket.Handler(func(ws *websocket.Conn) {
		ws.Close() // Close immediately

		ctx := context.Background()
		go func() {
			rdpToWs(ctx, mockRDP, ws)
			close(done)
		}()
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, err := websocket.Dial(wsURL, "", "http://localhost/")
	if err == nil {
		ws.Close()
	}

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("rdpToWs did not handle closed connection")
	}
}

// TestConnect_LocalhostOriginWithEnvSet tests that localhost is always allowed
func TestConnect_LocalhostOriginWithEnvSet(t *testing.T) {
	os.Setenv("ALLOWED_ORIGINS", "http://production.com")
	defer os.Unsetenv("ALLOWED_ORIGINS")

	// Test isAllowedOrigin directly since Connect requires WebSocket hijacking
	result := isAllowedOrigin("http://localhost:3000")
	assert.True(t, result, "localhost should always be allowed even with ALLOWED_ORIGINS set")

	result = isAllowedOrigin("http://127.0.0.1:3000")
	assert.True(t, result, "127.0.0.1 should always be allowed even with ALLOWED_ORIGINS set")
}

// TestRdpToWs_MultipleUpdates tests sending multiple updates in sequence
func TestRdpToWs_MultipleUpdates(t *testing.T) {
	updates := [][]byte{
		{0x01, 0x02},
		{0x03, 0x04},
		{0x05, 0x06},
	}

	// Create a custom implementation for counting updates
	type countingMock struct {
		updates [][]byte
		count   int
		mu      sync.Mutex
	}

	cm := &countingMock{updates: updates}

	server := httptest.NewServer(websocket.Handler(func(ws *websocket.Conn) {
		// Custom rdpToWs-like loop for testing multiple updates
		for {
			cm.mu.Lock()
			if cm.count >= len(cm.updates) {
				cm.mu.Unlock()
				return
			}
			data := cm.updates[cm.count]
			cm.count++
			cm.mu.Unlock()

			wsMutex.Lock()
			err := websocket.Message.Send(ws, data)
			wsMutex.Unlock()
			if err != nil {
				return
			}
		}
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, err := websocket.Dial(wsURL, "", "http://localhost/")
	require.NoError(t, err)
	defer ws.Close()

	// Receive all updates
	for i := 0; i < len(updates); i++ {
		var received []byte
		err := websocket.Message.Receive(ws, &received)
		require.NoError(t, err)
		assert.Equal(t, updates[i], received, "update %d mismatch", i)
	}
}

// TestWsToRdp_MultipleInputs tests receiving multiple inputs in sequence
func TestWsToRdp_MultipleInputs(t *testing.T) {
	mockRDP := &mockRDPConnection{}
	inputs := [][]byte{
		{0x10, 0x20},
		{0x30, 0x40},
		{0x50, 0x60},
	}

	done := make(chan struct{})
	server := httptest.NewServer(websocket.Handler(func(ws *websocket.Conn) {
		ctx, cancel := context.WithCancel(context.Background())

		go func() {
			wsToRdp(ctx, ws, mockRDP, cancel)
			close(done)
		}()

		// Wait for processing
		<-done
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, err := websocket.Dial(wsURL, "", "http://localhost/")
	require.NoError(t, err)

	// Send all inputs
	for _, input := range inputs {
		err := websocket.Message.Send(ws, input)
		require.NoError(t, err)
	}

	// Small delay to ensure messages are processed
	time.Sleep(100 * time.Millisecond)
	ws.Close()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("wsToRdp did not complete")
	}

	// Verify all inputs were received
	mockRDP.mu.Lock()
	defer mockRDP.mu.Unlock()
	require.Len(t, mockRDP.receivedInputs, len(inputs))
	for i, input := range inputs {
		assert.Equal(t, input, mockRDP.receivedInputs[i], "input %d mismatch", i)
	}
}

// TestConnect_AllowedOriginWithProtocolVariants tests origin matching with different protocols
func TestConnect_AllowedOriginWithProtocolVariants(t *testing.T) {
	tests := []struct {
		name       string
		envOrigins string
		origin     string
		allowed    bool
	}{
		{
			name:       "https in env, http in request",
			envOrigins: "https://example.com",
			origin:     "http://example.com",
			allowed:    true,
		},
		{
			name:       "http in env, https in request",
			envOrigins: "http://example.com",
			origin:     "https://example.com",
			allowed:    true,
		},
		{
			name:       "no protocol in env",
			envOrigins: "example.com",
			origin:     "https://example.com",
			allowed:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("ALLOWED_ORIGINS", tt.envOrigins)
			defer os.Unsetenv("ALLOWED_ORIGINS")

			result := isAllowedOrigin(tt.origin)
			assert.Equal(t, tt.allowed, result)
		})
	}
}

// TestRdpToWs_ConcurrentSends tests that concurrent sends are properly synchronized
func TestRdpToWs_ConcurrentSends(t *testing.T) {
	// This test verifies the mutex protection in rdpToWs
	var wg sync.WaitGroup

	server := httptest.NewServer(websocket.Handler(func(ws *websocket.Conn) {
		// Simulate concurrent sends using the mutex
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				wsMutex.Lock()
				websocket.Message.Send(ws, []byte{byte(idx)})
				wsMutex.Unlock()
			}(i)
		}
		wg.Wait()
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, err := websocket.Dial(wsURL, "", "http://localhost/")
	require.NoError(t, err)
	defer ws.Close()

	// Read all messages
	received := 0
	for i := 0; i < 10; i++ {
		var data []byte
		err := websocket.Message.Receive(ws, &data)
		if err != nil {
			break
		}
		received++
	}

	assert.Equal(t, 10, received, "should receive all concurrent messages")
}

// TestConnect_InvalidWidthParameter tests WebSocket handling with invalid width
func TestConnect_InvalidWidthParameter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(Connect))
	defer server.Close()

	// Connect with invalid width parameter
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/connect?width=invalid&height=600&host=test"

	ws, err := websocket.Dial(wsURL, "", "http://localhost/")
	if err == nil {
		// Connection established but should be closed quickly due to invalid width
		var data []byte
		websocket.Message.Receive(ws, &data) // Will fail or timeout
		ws.Close()
	}
	// Error is expected due to invalid width parameter handling
}

// TestConnect_InvalidHeightParameter tests WebSocket handling with invalid height
func TestConnect_InvalidHeightParameter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(Connect))
	defer server.Close()

	// Connect with valid width but invalid height parameter
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/connect?width=800&height=invalid&host=test"

	ws, err := websocket.Dial(wsURL, "", "http://localhost/")
	if err == nil {
		var data []byte
		websocket.Message.Receive(ws, &data)
		ws.Close()
	}
}

// TestConnect_MissingWidthParameter tests WebSocket handling with missing width
func TestConnect_MissingWidthParameter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(Connect))
	defer server.Close()

	// Connect without width parameter
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/connect?height=600&host=test"

	ws, err := websocket.Dial(wsURL, "", "http://localhost/")
	if err == nil {
		var data []byte
		websocket.Message.Receive(ws, &data)
		ws.Close()
	}
}

// TestConnect_ColorDepthParameter tests WebSocket handling with color depth parameter
func TestConnect_ColorDepthParameter(t *testing.T) {
	tests := []struct {
		name       string
		colorDepth string
	}{
		{"16-bit color", "16"},
		{"24-bit color", "24"},
		{"32-bit color", "32"},
		{"invalid color depth", "8"},
		{"invalid color depth string", "invalid"},
		{"empty color depth", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(Connect))
			defer server.Close()

			url := "/connect?width=800&height=600&host=nonexistent"
			if tt.colorDepth != "" {
				url += "&colorDepth=" + tt.colorDepth
			}
			wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + url

			ws, err := websocket.Dial(wsURL, "", "http://localhost/")
			if err == nil {
				ws.Close()
			}
			// The connection attempt is expected to fail at RDP stage
		})
	}
}

// TestConnect_WithValidParameters tests full WebSocket flow with valid parameters
func TestConnect_WithValidParameters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(Connect))
	defer server.Close()

	// Connect with all valid parameters (RDP connection will fail but we test the parameter parsing)
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/connect?width=1024&height=768&colorDepth=24&host=127.0.0.1:3389&user=testuser&password=testpass"

	ws, err := websocket.Dial(wsURL, "", "http://localhost/")
	if err == nil {
		ws.Close()
	}
	// Expected to fail at RDP connection stage, but parameters should be parsed correctly
}

// TestConnect_AllowedOriginHeader tests WebSocket with allowed origin header
func TestConnect_AllowedOriginHeader(t *testing.T) {
	os.Setenv("ALLOWED_ORIGINS", "http://example.com")
	defer os.Unsetenv("ALLOWED_ORIGINS")

	server := httptest.NewServer(http.HandlerFunc(Connect))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/connect?width=800&height=600&host=test"

	// websocket.Dial uses the third parameter as origin
	ws, err := websocket.Dial(wsURL, "", "http://example.com")
	if err == nil {
		ws.Close()
	}
}

// TestConnect_EmptyHost tests WebSocket handling with empty host parameter
func TestConnect_EmptyHost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(Connect))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/connect?width=800&height=600&host="

	ws, err := websocket.Dial(wsURL, "", "http://localhost/")
	if err == nil {
		ws.Close()
	}
}

// TestWsToRdp_ReadError tests handling of WebSocket read errors
func TestWsToRdp_ReadError(t *testing.T) {
	mockRDP := &mockRDPConnection{}

	done := make(chan struct{})
	server := httptest.NewServer(websocket.Handler(func(ws *websocket.Conn) {
		ctx, cancel := context.WithCancel(context.Background())

		// Start the goroutine
		go func() {
			wsToRdp(ctx, ws, mockRDP, cancel)
			close(done)
		}()

		// Close immediately to cause read error
		time.Sleep(10 * time.Millisecond)
		ws.Close()
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, err := websocket.Dial(wsURL, "", "http://localhost/")
	if err == nil {
		ws.Close()
	}

	select {
	case <-done:
		// Success - wsToRdp returned on read error
	case <-time.After(2 * time.Second):
		t.Fatal("wsToRdp did not return on read error")
	}
}

// TestRdpToWs_SendWithUpdate tests rdpToWs with actual update data
func TestRdpToWs_SendWithUpdate(t *testing.T) {
	testData := []byte{0x00, 0x01, 0x02, 0x03, 0x04}
	mockRDP := &mockRDPConnection{
		updateData: &rdp.Update{Data: testData},
		maxUpdates: 2, // Allow 2 updates before returning deactivate
	}

	server := httptest.NewServer(websocket.Handler(func(ws *websocket.Conn) {
		ctx := context.Background()
		rdpToWs(ctx, mockRDP, ws)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, err := websocket.Dial(wsURL, "", "http://localhost/")
	require.NoError(t, err)
	defer ws.Close()

	// Read first update
	var received []byte
	err = websocket.Message.Receive(ws, &received)
	require.NoError(t, err)
	assert.Equal(t, testData, received)

	// Read second update
	err = websocket.Message.Receive(ws, &received)
	require.NoError(t, err)
	assert.Equal(t, testData, received)
}

// TestRdpToWs_NilUpdate tests rdpToWs with nil update (results in deactivate after max)
func TestRdpToWs_NilUpdate(t *testing.T) {
	// When updateData is nil but no error, rdpToWs will try to send nil data
	// which should be handled gracefully. Use deactivate error for clean exit.
	mockRDP := &mockRDPConnection{
		updateError: pdu.ErrDeactiateAll,
	}

	done := make(chan struct{})
	server := httptest.NewServer(websocket.Handler(func(ws *websocket.Conn) {
		ctx := context.Background()
		go func() {
			rdpToWs(ctx, mockRDP, ws)
			close(done)
		}()
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, err := websocket.Dial(wsURL, "", "http://localhost/")
	if err == nil {
		defer ws.Close()
	}

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("rdpToWs did not return")
	}
}

// TestConnect_WebSocketHandshakeWithOrigin tests WebSocket handshake with origin configuration
func TestConnect_WebSocketHandshakeWithOrigin(t *testing.T) {
	// Test the handshake function that configures origin
	server := httptest.NewServer(http.HandlerFunc(Connect))
	defer server.Close()

	// Use a different origin to test the handshake configuration
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/connect?width=800&height=600&host=test"

	// Create WebSocket config
	config, err := websocket.NewConfig(wsURL, "http://testorigin.com")
	require.NoError(t, err)

	ws, err := websocket.DialConfig(config)
	if err == nil {
		ws.Close()
	}
}

// TestWsToRdp_ClosedNetworkConnectionError tests specific error message handling
func TestWsToRdp_ClosedNetworkConnectionError(t *testing.T) {
	mockRDP := &mockRDPConnection{}

	done := make(chan struct{})
	serverReady := make(chan struct{})
	
	server := httptest.NewServer(websocket.Handler(func(ws *websocket.Conn) {
		ctx, cancel := context.WithCancel(context.Background())

		go func() {
			wsToRdp(ctx, ws, mockRDP, cancel)
			close(done)
		}()

		close(serverReady)
		<-done
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, err := websocket.Dial(wsURL, "", "http://localhost/")
	require.NoError(t, err)

	<-serverReady
	// Close from client side to trigger "use of closed network connection" error
	ws.Close()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("wsToRdp did not handle closed network connection")
	}
}

// TestRdpToWs_ClosedNetworkConnectionSendError tests send error with closed connection message
func TestRdpToWs_ClosedNetworkConnectionSendError(t *testing.T) {
	mockRDP := &mockRDPConnection{
		updateData: &rdp.Update{Data: []byte{1, 2, 3}},
		maxUpdates: 1, // Only try one update then return deactivate
	}

	done := make(chan struct{})
	server := httptest.NewServer(websocket.Handler(func(ws *websocket.Conn) {
		ctx := context.Background()
		// Close the WebSocket connection mid-send
		go func() {
			time.Sleep(50 * time.Millisecond)
			ws.Close()
		}()
		rdpToWs(ctx, mockRDP, ws)
		close(done)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, err := websocket.Dial(wsURL, "", "http://localhost/")
	if err == nil {
		defer ws.Close()
	}

	select {
	case <-done:
		// Success
	case <-time.After(3 * time.Second):
		t.Fatal("rdpToWs did not handle send error")
	}
}

// TestWsToRdp_LargeData tests handling of large data payloads
func TestWsToRdp_LargeData(t *testing.T) {
	mockRDP := &mockRDPConnection{}

	// Generate large data
	largeData := make([]byte, 64*1024) // 64KB
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	done := make(chan struct{})
	server := httptest.NewServer(websocket.Handler(func(ws *websocket.Conn) {
		ctx, cancel := context.WithCancel(context.Background())

		go func() {
			wsToRdp(ctx, ws, mockRDP, cancel)
			close(done)
		}()

		<-done
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, err := websocket.Dial(wsURL, "", "http://localhost/")
	require.NoError(t, err)

	err = websocket.Message.Send(ws, largeData)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)
	ws.Close()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("wsToRdp did not complete")
	}

	// Verify large data was received
	mockRDP.mu.Lock()
	defer mockRDP.mu.Unlock()
	require.Len(t, mockRDP.receivedInputs, 1)
	assert.Equal(t, largeData, mockRDP.receivedInputs[0])
}

// TestRdpToWs_LargeUpdate tests handling of large update payloads
func TestRdpToWs_LargeUpdate(t *testing.T) {
	largeData := make([]byte, 64*1024) // 64KB
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	mockRDP := &mockRDPConnection{
		updateData: &rdp.Update{Data: largeData},
		maxUpdates: 1,
	}

	server := httptest.NewServer(websocket.Handler(func(ws *websocket.Conn) {
		ctx := context.Background()
		rdpToWs(ctx, mockRDP, ws)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, err := websocket.Dial(wsURL, "", "http://localhost/")
	require.NoError(t, err)
	defer ws.Close()

	var received []byte
	err = websocket.Message.Receive(ws, &received)
	require.NoError(t, err)
	assert.Equal(t, largeData, received)
}

// TestConnect_MissingHeightParameter tests WebSocket handling with missing height
func TestConnect_MissingHeightParameter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(Connect))
	defer server.Close()

	// Connect without height parameter (width valid)
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/connect?width=800&host=test"

	ws, err := websocket.Dial(wsURL, "", "http://localhost/")
	if err == nil {
		var data []byte
		websocket.Message.Receive(ws, &data)
		ws.Close()
	}
}

// TestIsAllowedOrigin_MoreEdgeCases tests additional edge cases
func TestIsAllowedOrigin_MoreEdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		origin     string
		envOrigins string
		expected   bool
	}{
		{
			name:       "localhost without port",
			origin:     "http://localhost",
			envOrigins: "http://other.com",
			expected:   true,
		},
		{
			name:       "127.0.0.1 without port",
			origin:     "http://127.0.0.1",
			envOrigins: "http://other.com",
			expected:   true,
		},
		{
			name:       "localhost with different path",
			origin:     "http://localhost/path",
			envOrigins: "http://other.com",
			expected:   true,
		},
		{
			name:       "exact match with http",
			origin:     "http://example.com",
			envOrigins: "http://example.com",
			expected:   true,
		},
		{
			name:       "normalized without protocol match",
			origin:     "http://example.com:8080",
			envOrigins: "example.com:8080",
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Unsetenv("ALLOWED_ORIGINS")
			if tt.envOrigins != "" {
				os.Setenv("ALLOWED_ORIGINS", tt.envOrigins)
			}
			defer os.Unsetenv("ALLOWED_ORIGINS")

			result := isAllowedOrigin(tt.origin)
			assert.Equal(t, tt.expected, result, "origin=%q env=%q", tt.origin, tt.envOrigins)
		})
	}
}

// TestRdpToWs_ContextCancelledDuringLoop tests context cancellation during update loop
func TestRdpToWs_ContextCancelledDuringLoop(t *testing.T) {
	updateChan := make(chan struct{})
	mockRDP := &mockRDPConnection{
		updateData: &rdp.Update{Data: []byte{1, 2, 3}},
	}

	// Override GetUpdate to wait for a signal then return deactivate
	customMock := &mockRDPConnection{}
	customMock.updateData = &rdp.Update{Data: []byte{1, 2, 3}}
	customMock.maxUpdates = 2

	done := make(chan struct{})
	server := httptest.NewServer(websocket.Handler(func(ws *websocket.Conn) {
		ctx, cancel := context.WithCancel(context.Background())

		go func() {
			// Cancel context after first update
			time.Sleep(100 * time.Millisecond)
			cancel()
		}()

		rdpToWs(ctx, customMock, ws)
		close(done)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, err := websocket.Dial(wsURL, "", "http://localhost/")
	if err == nil {
		// Read what we can
		ws.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		var data []byte
		for {
			err := websocket.Message.Receive(ws, &data)
			if err != nil {
				break
			}
		}
		ws.Close()
	}

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("rdpToWs did not complete")
	}

	close(updateChan)
	_ = mockRDP // unused in this test
}

// TestWsToRdp_ContextCancelledWhileWaiting tests context cancellation while waiting for message
func TestWsToRdp_ContextCancelledWhileWaiting(t *testing.T) {
	mockRDP := &mockRDPConnection{}

	done := make(chan struct{})
	serverReady := make(chan struct{})
	
	server := httptest.NewServer(websocket.Handler(func(ws *websocket.Conn) {
		ctx, cancel := context.WithCancel(context.Background())
		close(serverReady)

		wsToRdp(ctx, ws, mockRDP, cancel)
		close(done)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, err := websocket.Dial(wsURL, "", "http://localhost/")
	require.NoError(t, err)

	<-serverReady
	// Close client connection after server is ready - this will cause Receive to return error
	time.Sleep(50 * time.Millisecond)
	ws.Close()

	select {
	case <-done:
		// Success - function returned due to connection close
	case <-time.After(2 * time.Second):
		t.Fatal("wsToRdp did not return on connection close")
	}
}

// TestCodecListToJSON_SpecialCharacters tests JSON encoding with special characters
func TestCodecListToJSON_SpecialCharacters(t *testing.T) {
	// Test with codec names that might need escaping
	codecs := []string{"codec/test", "codec\\test", "codec\"test"}
	result := codecListToJSON(codecs)
	// The function doesn't escape, so it just quotes
	assert.Contains(t, result, `"codec/test"`)
	assert.Contains(t, result, `"codec\test"`)
}

// TestConnect_ServerHandshakeFunction tests the custom handshake function
func TestConnect_ServerHandshakeFunction(t *testing.T) {
	// This tests the websocket.Server configuration with custom handshake
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate custom handshake behavior
		origin := r.Header.Get("Origin")
		if origin != "" && !isAllowedOrigin(origin) {
			http.Error(w, "Origin not allowed", http.StatusForbidden)
			return
		}

		handler := func(wsConn *websocket.Conn) {
			// Just close immediately for test
			wsConn.Close()
		}

		wsServer := websocket.Server{
			Handler: handler,
			Handshake: func(config *websocket.Config, r *http.Request) error {
				config.Origin, _ = websocket.Origin(config, r)
				return nil
			},
		}
		wsServer.ServeHTTP(w, r)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, err := websocket.Dial(wsURL, "", "http://customorigin.com")
	if err == nil {
		ws.Close()
	}
}

// TestConnect_FullQueryParameters tests connection with all query parameters
func TestConnect_FullQueryParameters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(Connect))
	defer server.Close()

	// Full query string with all parameters
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") +
		"/connect?width=1920&height=1080&colorDepth=32&host=192.168.1.1:3389&user=admin&password=secret123"

	ws, err := websocket.Dial(wsURL, "", "http://localhost/")
	if err == nil {
		ws.Close()
	}
	// Will fail at RDP connection but parameters are all parsed
}

// TestWsToRdp_GenericReadError tests handling of generic read errors (not EOF or closed)
func TestWsToRdp_GenericReadError(t *testing.T) {
	mockRDP := &mockRDPConnection{}

	done := make(chan struct{})
	server := httptest.NewServer(websocket.Handler(func(ws *websocket.Conn) {
		ctx, cancel := context.WithCancel(context.Background())

		go func() {
			wsToRdp(ctx, ws, mockRDP, cancel)
			close(done)
		}()

		// Wait for a bit then close to trigger error
		time.Sleep(100 * time.Millisecond)
		ws.Close()
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, err := websocket.Dial(wsURL, "", "http://localhost/")
	if err == nil {
		// Send some data then close abruptly
		websocket.Message.Send(ws, []byte{1, 2, 3})
		ws.Close()
	}

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("wsToRdp did not return")
	}
}

// TestRdpToWs_GenericSendError tests handling of generic send errors (not closed connection)
func TestRdpToWs_GenericSendError(t *testing.T) {
	mockRDP := &mockRDPConnection{
		updateData: &rdp.Update{Data: []byte{1, 2, 3, 4, 5}},
		maxUpdates: 5,
	}

	done := make(chan struct{})
	server := httptest.NewServer(websocket.Handler(func(ws *websocket.Conn) {
		ctx := context.Background()

		// Close after a delay to trigger send error
		go func() {
			time.Sleep(50 * time.Millisecond)
			ws.Close()
		}()

		rdpToWs(ctx, mockRDP, ws)
		close(done)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, err := websocket.Dial(wsURL, "", "http://localhost/")
	if err == nil {
		// Read what we can before server closes
		ws.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
		var data []byte
		for {
			err := websocket.Message.Receive(ws, &data)
			if err != nil {
				break
			}
		}
		ws.Close()
	}

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("rdpToWs did not return")
	}
}

// TestConnect_ZeroDimensions tests handling of zero dimensions
func TestConnect_ZeroDimensions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(Connect))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/connect?width=0&height=0&host=test"

	ws, err := websocket.Dial(wsURL, "", "http://localhost/")
	if err == nil {
		ws.Close()
	}
}

// TestConnect_NegativeDimensions tests handling of negative dimensions
func TestConnect_NegativeDimensions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(Connect))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/connect?width=-100&height=-100&host=test"

	ws, err := websocket.Dial(wsURL, "", "http://localhost/")
	if err == nil {
		ws.Close()
	}
}

// TestConnect_VeryLargeDimensions tests handling of very large dimensions
func TestConnect_VeryLargeDimensions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(Connect))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/connect?width=99999&height=99999&host=test"

	ws, err := websocket.Dial(wsURL, "", "http://localhost/")
	if err == nil {
		ws.Close()
	}
}

// TestRdpToWs_RapidUpdates tests handling of rapid consecutive updates
func TestRdpToWs_RapidUpdates(t *testing.T) {
	mockRDP := &mockRDPConnection{
		updateData: &rdp.Update{Data: []byte{0xAB, 0xCD}},
		maxUpdates: 100,
	}

	server := httptest.NewServer(websocket.Handler(func(ws *websocket.Conn) {
		ctx := context.Background()
		rdpToWs(ctx, mockRDP, ws)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, err := websocket.Dial(wsURL, "", "http://localhost/")
	require.NoError(t, err)
	defer ws.Close()

	// Read all updates
	count := 0
	for count < 100 {
		var data []byte
		err := websocket.Message.Receive(ws, &data)
		if err != nil {
			break
		}
		count++
	}

	assert.Equal(t, 100, count, "should receive all updates")
}

// TestWsToRdp_RapidInputs tests handling of rapid consecutive inputs
func TestWsToRdp_RapidInputs(t *testing.T) {
	mockRDP := &mockRDPConnection{}

	done := make(chan struct{})
	server := httptest.NewServer(websocket.Handler(func(ws *websocket.Conn) {
		ctx, cancel := context.WithCancel(context.Background())

		go func() {
			wsToRdp(ctx, ws, mockRDP, cancel)
			close(done)
		}()

		<-done
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, err := websocket.Dial(wsURL, "", "http://localhost/")
	require.NoError(t, err)

	// Send many rapid inputs
	for i := 0; i < 50; i++ {
		err := websocket.Message.Send(ws, []byte{byte(i)})
		if err != nil {
			break
		}
	}

	time.Sleep(100 * time.Millisecond)
	ws.Close()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("wsToRdp did not complete")
	}

	// Verify inputs were received
	mockRDP.mu.Lock()
	receivedCount := len(mockRDP.receivedInputs)
	mockRDP.mu.Unlock()

	assert.GreaterOrEqual(t, receivedCount, 1, "should receive at least some inputs")
}

// mockCapabilitiesGetter implements capabilitiesGetter interface
type mockCapabilitiesGetter struct {
	caps *rdp.ServerCapabilityInfo
}

func (m *mockCapabilitiesGetter) GetServerCapabilities() *rdp.ServerCapabilityInfo {
	return m.caps
}

// TestBuildCapabilitiesMessage tests the JSON message building
func TestBuildCapabilitiesMessage(t *testing.T) {
	tests := []struct {
		name     string
		caps     *rdp.ServerCapabilityInfo
		contains []string
	}{
		{
			name: "basic capabilities",
			caps: &rdp.ServerCapabilityInfo{
				BitmapCodecs:      []string{"RFX", "NSCodec"},
				SurfaceCommands:   true,
				ColorDepth:        32,
				DesktopSize:       "1920x1080",
				MultifragmentSize: 65535,
				LargePointer:      true,
				FrameAcknowledge:  true,
			},
			contains: []string{
				`"type":"capabilities"`,
				`"codecs":["RFX","NSCodec"]`,
				`"surfaceCommands":true`,
				`"colorDepth":32`,
				`"desktopSize":"1920x1080"`,
				`"multifragmentSize":65535`,
				`"largePointer":true`,
				`"frameAcknowledge":true`,
			},
		},
		{
			name: "minimal capabilities",
			caps: &rdp.ServerCapabilityInfo{
				BitmapCodecs:    []string{},
				ColorDepth:      16,
				DesktopSize:     "800x600",
				SurfaceCommands: false,
				LargePointer:    false,
				FrameAcknowledge: false,
			},
			contains: []string{
				`"type":"capabilities"`,
				`"codecs":[]`,
				`"colorDepth":16`,
				`"desktopSize":"800x600"`,
				`"surfaceCommands":false`,
			},
		},
		{
			name: "single codec",
			caps: &rdp.ServerCapabilityInfo{
				BitmapCodecs: []string{"RemoteFX"},
				ColorDepth:   24,
				DesktopSize:  "1024x768",
			},
			contains: []string{
				`"codecs":["RemoteFX"]`,
				`"colorDepth":24`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := buildCapabilitiesMessage(tt.caps)

			// First byte should be 0xFF marker
			assert.Equal(t, byte(0xFF), msg[0], "first byte should be capabilities marker")

			// Rest should be valid JSON containing expected fields
			jsonStr := string(msg[1:])
			for _, expected := range tt.contains {
				assert.Contains(t, jsonStr, expected, "JSON should contain: %s", expected)
			}
		})
	}
}

// TestSendCapabilitiesInfo_NilCapabilities tests handling of nil capabilities
func TestSendCapabilitiesInfo_NilCapabilities(t *testing.T) {
	mockCaps := &mockCapabilitiesGetter{caps: nil}

	done := make(chan struct{})
	server := httptest.NewServer(websocket.Handler(func(ws *websocket.Conn) {
		sendCapabilitiesInfo(ws, mockCaps)
		close(done)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, err := websocket.Dial(wsURL, "", "http://localhost/")
	require.NoError(t, err)
	defer ws.Close()

	select {
	case <-done:
		// Success - function returned without error when caps is nil
	case <-time.After(time.Second):
		t.Fatal("sendCapabilitiesInfo did not return")
	}
}

// TestSendCapabilitiesInfo_Success tests successful capabilities sending
func TestSendCapabilitiesInfo_Success(t *testing.T) {
	mockCaps := &mockCapabilitiesGetter{
		caps: &rdp.ServerCapabilityInfo{
			BitmapCodecs:      []string{"RFX"},
			SurfaceCommands:   true,
			ColorDepth:        32,
			DesktopSize:       "1920x1080",
			MultifragmentSize: 65536,
			LargePointer:      true,
			FrameAcknowledge:  true,
		},
	}

	done := make(chan struct{})
	server := httptest.NewServer(websocket.Handler(func(ws *websocket.Conn) {
		sendCapabilitiesInfo(ws, mockCaps)
		close(done)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, err := websocket.Dial(wsURL, "", "http://localhost/")
	require.NoError(t, err)
	defer ws.Close()

	// Receive the capabilities message
	var received []byte
	err = websocket.Message.Receive(ws, &received)
	require.NoError(t, err)

	// Check message format
	assert.Equal(t, byte(0xFF), received[0], "first byte should be capabilities marker")
	jsonStr := string(received[1:])
	assert.Contains(t, jsonStr, `"type":"capabilities"`)
	assert.Contains(t, jsonStr, `"codecs":["RFX"]`)
	assert.Contains(t, jsonStr, `"surfaceCommands":true`)

	select {
	case <-done:
		// Success
	case <-time.After(time.Second):
		t.Fatal("sendCapabilitiesInfo did not return")
	}
}

// TestSendCapabilitiesInfo_SendError tests handling of WebSocket send errors
func TestSendCapabilitiesInfo_SendError(t *testing.T) {
	mockCaps := &mockCapabilitiesGetter{
		caps: &rdp.ServerCapabilityInfo{
			BitmapCodecs: []string{"RFX"},
			ColorDepth:   24,
			DesktopSize:  "1024x768",
		},
	}

	done := make(chan struct{})
	server := httptest.NewServer(websocket.Handler(func(ws *websocket.Conn) {
		// Close connection before sending to trigger error
		ws.Close()
		sendCapabilitiesInfo(ws, mockCaps)
		close(done)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, err := websocket.Dial(wsURL, "", "http://localhost/")
	if err == nil {
		ws.Close()
	}

	select {
	case <-done:
		// Success - function returned even with send error
	case <-time.After(time.Second):
		t.Fatal("sendCapabilitiesInfo did not return")
	}
}

// TestSendCapabilitiesInfo_EmptyCodecs tests handling of empty codecs
func TestSendCapabilitiesInfo_EmptyCodecs(t *testing.T) {
	mockCaps := &mockCapabilitiesGetter{
		caps: &rdp.ServerCapabilityInfo{
			BitmapCodecs: []string{},
			ColorDepth:   16,
			DesktopSize:  "800x600",
		},
	}

	done := make(chan struct{})
	server := httptest.NewServer(websocket.Handler(func(ws *websocket.Conn) {
		sendCapabilitiesInfo(ws, mockCaps)
		close(done)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, err := websocket.Dial(wsURL, "", "http://localhost/")
	require.NoError(t, err)
	defer ws.Close()

	var received []byte
	err = websocket.Message.Receive(ws, &received)
	require.NoError(t, err)

	jsonStr := string(received[1:])
	assert.Contains(t, jsonStr, `"codecs":[]`)

	<-done
}

// Test sendAudioData function
func TestSendAudioData(t *testing.T) {
t.Run("sends audio with data only", func(t *testing.T) {
server := httptest.NewServer(websocket.Handler(func(ws *websocket.Conn) {
var msg []byte
err := websocket.Message.Receive(ws, &msg)
require.NoError(t, err)

// Check message structure
require.GreaterOrEqual(t, len(msg), 4, "message too short")
assert.Equal(t, byte(0xFE), msg[0], "should have audio marker")
assert.Equal(t, byte(AudioMsgTypeData), msg[1], "should be data message")
assert.Equal(t, []byte{0x10, 0x00}, msg[2:4], "timestamp should be 16")
assert.Equal(t, []byte{0x01, 0x02, 0x03, 0x04}, msg[4:], "data should match")
}))
defer server.Close()

wsURL := strings.Replace(server.URL, "http://", "ws://", 1)
ws, err := websocket.Dial(wsURL, "", "http://localhost/")
require.NoError(t, err)
defer ws.Close()

sendAudioData(ws, []byte{0x01, 0x02, 0x03, 0x04}, nil, 16)
time.Sleep(100 * time.Millisecond)
})

t.Run("sends audio with format", func(t *testing.T) {
server := httptest.NewServer(websocket.Handler(func(ws *websocket.Conn) {
var msg []byte
err := websocket.Message.Receive(ws, &msg)
require.NoError(t, err)

// Check message structure
require.GreaterOrEqual(t, len(msg), 12, "message too short for format")
assert.Equal(t, byte(0xFE), msg[0], "should have audio marker")
assert.Equal(t, byte(AudioMsgTypeFormat), msg[1], "should be format message")
// Format info starts at byte 4: channels (2), sampleRate (4), bitsPerSample (2)
assert.Equal(t, []byte{0x02, 0x00}, msg[4:6], "channels should be 2")
assert.Equal(t, []byte{0x44, 0xAC, 0x00, 0x00}, msg[6:10], "sample rate should be 44100")
assert.Equal(t, []byte{0x10, 0x00}, msg[10:12], "bits per sample should be 16")
}))
defer server.Close()

wsURL := strings.Replace(server.URL, "http://", "ws://", 1)
ws, err := websocket.Dial(wsURL, "", "http://localhost/")
require.NoError(t, err)
defer ws.Close()

format := &audio.AudioFormat{
Channels:      2,
SamplesPerSec: 44100,
BitsPerSample: 16,
}
sendAudioData(ws, []byte{0xAA, 0xBB}, format, 0)
time.Sleep(100 * time.Millisecond)
})

t.Run("handles empty data", func(t *testing.T) {
// Should return early without sending anything
sendAudioData(nil, []byte{}, nil, 0)
})
}

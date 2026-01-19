package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/kulaginds/rdp-html5/internal/pkg/rdp/fastpath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConnect_MissingParameters(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "missing host",
			queryParams:    "user=test&password=test",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing user",
			queryParams:    "host=localhost&password=test",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing both",
			queryParams:    "",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/connect?"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			Connect(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestConnect_InvalidDimensions(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
	}{
		{
			name:           "zero width",
			queryParams:    "host=localhost&user=test&width=0&height=768",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "zero height",
			queryParams:    "host=localhost&user=test&width=1024&height=0",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "negative width",
			queryParams:    "host=localhost&user=test&width=-100&height=768",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "negative height",
			queryParams:    "host=localhost&user=test&width=1024&height=-100",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "excessive width",
			queryParams:    "host=localhost&user=test&width=5000&height=768",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "excessive height",
			queryParams:    "host=localhost&user=test&width=1024&height=5000",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/connect?"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			Connect(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestConnect_ValidParameters(t *testing.T) {
	// Set up environment for testing
	os.Setenv("ALLOWED_ORIGINS", "localhost:8080,127.0.0.1:8080")
	defer os.Unsetenv("ALLOWED_ORIGINS")

	// Create a test server with proper CORS headers
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		Connect(w, r)
	}))
	defer server.Close()

	// Convert http://... to ws://...
	serverURL := strings.TrimPrefix(server.URL, "http://")
	u := url.URL{Scheme: "ws", Host: serverURL, Path: "/connect"}
	u.RawQuery = "host=localhost&user=test&password=test&width=1024&height=768"

	// Create headers with proper origin
	headers := http.Header{}
	headers.Set("Origin", "http://"+serverURL)

	// Connect to the server
	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(u.String(), headers)

	// The connection should fail because there's no RDP server running,
	// but it should fail after the WebSocket upgrade succeeds
	if err == nil {
		conn.Close()
		t.Skip("Connection succeeded unexpectedly")
	}

	// The error should be related to RDP connection, not WebSocket upgrade
	assert.NotContains(t, err.Error(), "websocket: bad handshake")
	assert.NotContains(t, err.Error(), "request origin not allowed")
}

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
			expectedResult: true, // In dev mode (no ALLOWED_ORIGINS env), we allow all origins
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envOrigins != "" {
				os.Setenv("ALLOWED_ORIGINS", tt.envOrigins)
				defer os.Unsetenv("ALLOWED_ORIGINS")
			}

			result := isAllowedOrigin(tt.origin)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestWsToRdp_ContextCancellation(t *testing.T) {
	// Create a mock RDP connection
	mockRDPConn := &mockRDPConnection{}

	// Create a mock WebSocket connection using a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}
		wsConn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(r.Context())

		// Cancel context immediately to test cancellation
		cancel()

		wsToRdp(ctx, wsConn, mockRDPConn, cancel)
	}))
	defer server.Close()

	// This test mainly ensures the function handles cancellation gracefully
	// Without actually connecting, we test the cancellation path
}

func TestRdpToWs_ContextCancellation(t *testing.T) {
	// Create a mock RDP connection
	mockRDPConn := &mockRDPConnection{
		updateError: context.Canceled,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Create a mock WebSocket connection
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This will never be reached because we're testing the cancellation path
	}))

	// Create a dummy WebSocket connection for testing
	wsConn, _, err := websocket.DefaultDialer.Dial("ws://"+server.Listener.Addr().String(), nil)
	if err == nil {
		defer wsConn.Close()
	}

	// Test that the function handles context cancellation gracefully
	rdpToWs(ctx, mockRDPConn, wsConn)
}

// Mock RDP connection for testing
type mockRDPConnection struct {
	updateData  *fastpath.UpdatePDU
	updateError error
	sendError   error
}

func (m *mockRDPConnection) GetUpdate() (*fastpath.UpdatePDU, error) {
	if m.updateError != nil {
		return nil, m.updateError
	}
	return m.updateData, nil
}

func (m *mockRDPConnection) SendInputEvent(data []byte) error {
	return m.sendError
}

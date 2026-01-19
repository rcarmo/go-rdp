package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/kulaginds/rdp-html5/internal/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateServer(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:         "localhost",
			Port:         "8080",
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  120 * time.Second,
		},
		Security: config.SecurityConfig{
			AllowedOrigins:     []string{"localhost:8080"},
			MaxConnections:     10,
			EnableRateLimit:    true,
			RateLimitPerMinute: 60,
			EnableTLS:          false,
			MinTLSVersion:      "1.2",
		},
		Logging: config.LoggingConfig{
			Level:        "info",
			Format:       "text",
			EnableCaller: false,
			File:         "",
		},
	}

	server := createServer(cfg)

	require.NotNil(t, server)
	assert.Equal(t, "localhost:8080", server.Addr)
	assert.Equal(t, 30*time.Second, server.ReadTimeout)
	assert.Equal(t, 30*time.Second, server.WriteTimeout)
	assert.Equal(t, 120*time.Second, server.IdleTimeout)
}

func TestApplySecurityMiddleware(t *testing.T) {
	cfg := &config.Config{
		Security: config.SecurityConfig{
			AllowedOrigins:     []string{"https://example.com"},
			MaxConnections:     10,
			EnableRateLimit:    true,
			RateLimitPerMinute: 60,
			EnableTLS:          false,
		},
	}

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	middleware := applySecurityMiddleware(testHandler, cfg)
	require.NotNil(t, middleware)

	// Create a test request
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://example.com")

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Serve the request through middleware
	middleware.ServeHTTP(rr, req)

	// Check CORS headers
	assert.Equal(t, "https://example.com", rr.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "GET, POST, OPTIONS", rr.Header().Get("Access-Control-Allow-Methods"))
	assert.Equal(t, "Content-Type, Authorization", rr.Header().Get("Access-Control-Allow-Headers"))
	assert.Equal(t, "true", rr.Header().Get("Access-Control-Allow-Credentials"))

	// Check security headers
	assert.Equal(t, "nosniff", rr.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", rr.Header().Get("X-Frame-Options"))
	assert.Equal(t, "1; mode=block", rr.Header().Get("X-XSS-Protection"))
	assert.Equal(t, "max-age=31536000; includeSubDomains", rr.Header().Get("Strict-Transport-Security"))
	assert.Equal(t, "strict-origin-when-cross-origin", rr.Header().Get("Referrer-Policy"))
}

func TestSecurityHeadersMiddleware(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	middleware := securityHeadersMiddleware(testHandler)
	require.NotNil(t, middleware)

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	middleware.ServeHTTP(rr, req)

	// Verify security headers are set
	assert.Equal(t, "nosniff", rr.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", rr.Header().Get("X-Frame-Options"))
	assert.Equal(t, "1; mode=block", rr.Header().Get("X-XSS-Protection"))
	assert.Equal(t, "max-age=31536000; includeSubDomains", rr.Header().Get("Strict-Transport-Security"))
	assert.Equal(t, "strict-origin-when-cross-origin", rr.Header().Get("Referrer-Policy"))
	assert.Contains(t, rr.Header().Get("Content-Security-Policy"), "default-src 'self'")
}

func TestCorsMiddleware(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	tests := []struct {
		name           string
		allowedOrigins []string
		requestOrigin  string
		requestHost    string
		expectAllowed  bool
	}{
		{
			name:           "allowed origin from list",
			allowedOrigins: []string{"https://example.com", "https://app.example.com"},
			requestOrigin:  "https://example.com",
			requestHost:    "example.com:8080",
			expectAllowed:  true,
		},
		{
			name:           "not allowed origin from list",
			allowedOrigins: []string{"https://example.com"},
			requestOrigin:  "https://malicious.com",
			requestHost:    "example.com:8080",
			expectAllowed:  false,
		},
		{
			name:           "same origin when no list configured",
			allowedOrigins: []string{},
			requestOrigin:  "http://localhost:8080",
			requestHost:    "localhost:8080",
			expectAllowed:  true,
		},
		{
			name:           "different origin when no list configured (dev mode - allow all)",
			allowedOrigins: []string{},
			requestOrigin:  "https://malicious.com",
			requestHost:    "localhost:8080",
			expectAllowed:  true, // In dev mode (no ALLOWED_ORIGINS), we allow all origins
		},
		{
			name:           "OPTIONS request",
			allowedOrigins: []string{"https://example.com"},
			requestOrigin:  "https://example.com",
			requestHost:    "example.com:8080",
			expectAllowed:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := corsMiddleware(testHandler, tt.allowedOrigins)

			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set("Origin", tt.requestOrigin)
			req.Host = tt.requestHost

			rr := httptest.NewRecorder()
			middleware.ServeHTTP(rr, req)

			// Check CORS header based on expectation
			if tt.expectAllowed {
				assert.Equal(t, tt.requestOrigin, rr.Header().Get("Access-Control-Allow-Origin"))
			} else {
				assert.Empty(t, rr.Header().Get("Access-Control-Allow-Origin"))
			}

			// Always check methods and headers for allowed origins
			if tt.expectAllowed {
				assert.Equal(t, "GET, POST, OPTIONS", rr.Header().Get("Access-Control-Allow-Methods"))
				assert.Equal(t, "Content-Type, Authorization", rr.Header().Get("Access-Control-Allow-Headers"))
			}
		})
	}
}

func TestSetupLogging(t *testing.T) {
	tests := []struct {
		name   string
		config config.LoggingConfig
	}{
		{
			name: "debug level",
			config: config.LoggingConfig{
				Level:        "debug",
				Format:       "text",
				EnableCaller: false,
				File:         "",
			},
		},
		{
			name: "info level",
			config: config.LoggingConfig{
				Level:        "info",
				Format:       "text",
				EnableCaller: false,
				File:         "",
			},
		},
		{
			name: "warn level",
			config: config.LoggingConfig{
				Level:        "warn",
				Format:       "text",
				EnableCaller: false,
				File:         "",
			},
		},
		{
			name: "error level",
			config: config.LoggingConfig{
				Level:        "error",
				Format:       "text",
				EnableCaller: false,
				File:         "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test just ensures setupLogging doesn't panic
			// In a real test, we'd capture log output and verify flags
			setupLogging(tt.config)
			// If we get here without panic, the test passes
		})
	}
}

func TestStartServer(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:         "localhost",
			Port:         "0", // Let OS choose port
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
			IdleTimeout:  10 * time.Second,
		},
		Security: config.SecurityConfig{
			EnableTLS: false,
		},
	}

	server := createServer(cfg)
	require.NotNil(t, server)

	// Start server in a goroutine
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- startServer(server, cfg)
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Test that server is responding
	resp, err := http.Get(server.Addr)
	if err == nil {
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	}

	// Shutdown server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = server.Shutdown(ctx)
	assert.NoError(t, err)

	// Check for server errors
	select {
	case err := <-serverErr:
		// Server should stop gracefully without error
		assert.NoError(t, err)
	case <-time.After(1 * time.Second):
		t.Fatal("Server shutdown timed out")
	}
}

func TestShowHelp(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run help
	showHelp()

	// Restore stdout
	os.Stdout = oldStdout
	w.Close()

	// Read captured output
	output := make([]byte, 1024)
	n, _ := r.Read(output)

	// Verify help contains expected content
	captured := string(output[:n])
	assert.Contains(t, captured, "RDP HTML5 Client")
	assert.Contains(t, captured, "USAGE:")
	assert.Contains(t, captured, "OPTIONS:")
	assert.Contains(t, captured, "ENVIRONMENT VARIABLES:")
	assert.Contains(t, captured, "EXAMPLES:")
}

func TestShowVersion(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run version
	showVersion()

	// Restore stdout
	os.Stdout = oldStdout
	w.Close()

	// Read captured output
	output := make([]byte, 1024)
	n, _ := r.Read(output)

	// Verify version contains expected content
	captured := string(output[:n])
	assert.Contains(t, captured, "RDP HTML5 Client v2.0.0")
	assert.Contains(t, captured, "Built with Go")
	assert.Contains(t, captured, "Protocol: RDP 10.x")
}

func TestRateLimitMiddleware(t *testing.T) {
	cfg := &config.Config{
		Security: config.SecurityConfig{
			EnableRateLimit:    true,
			RateLimitPerMinute: 60,
		},
	}

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	middleware := rateLimitMiddleware(testHandler, cfg.Security.RateLimitPerMinute)
	require.NotNil(t, middleware)

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	// First request should pass
	middleware.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	// Note: This is a basic test. A comprehensive rate limiting test
	// would require time-based testing or more sophisticated mocking
}

func TestMainFunction(t *testing.T) {
	// Save original os.Args
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	// Test help flag
	os.Args = []string{"server", "-help"}

	// This would normally call os.Exit, so we can't test it directly
	// Instead, we'll test the parsing logic separately
}

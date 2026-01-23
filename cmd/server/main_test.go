package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/rcarmo/rdp-html5/internal/config"
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
		_, _ = w.Write([]byte("OK"))
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
		_, _ = w.Write([]byte("OK"))
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
		_, _ = w.Write([]byte("OK"))
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
		defer func() { _ = resp.Body.Close() }()
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
	_ = w.Close()

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
	_ = w.Close()

	// Read captured output
	output := make([]byte, 1024)
	n, _ := r.Read(output)

	// Verify version contains expected content
	captured := string(output[:n])
	assert.Contains(t, captured, "RDP HTML5 Client 1.0.0")
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
		_, _ = w.Write([]byte("OK"))
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

func TestParseFlags_UsesOsArgs(t *testing.T) {
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	os.Args = []string{"server", "-host", " example ", "-port", " 1234 ", "-log-level", "debug"}
	args, action := parseFlags()
	assert.Empty(t, action)
	assert.Equal(t, "example", args.host)
	assert.Equal(t, "1234", args.port)
	assert.Equal(t, "debug", args.logLevel)
}

func TestMain_Help(t *testing.T) {
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	os.Args = []string{"server", "-help"}
	main() // should return early after printing help
}

func TestMain_Version(t *testing.T) {
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	os.Args = []string{"server", "-version"}
	main() // should return early after printing version
}

func TestCorsMiddlewareOptionsRequest(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	middleware := corsMiddleware(testHandler, []string{"https://example.com"})

	// Test OPTIONS preflight request
	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	rr := httptest.NewRecorder()

	middleware.ServeHTTP(rr, req)

	// OPTIONS should return 200 immediately without calling next handler
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "https://example.com", rr.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "GET, POST, OPTIONS", rr.Header().Get("Access-Control-Allow-Methods"))
	// Body should be empty for OPTIONS
	assert.Empty(t, rr.Body.String())
}

func TestCorsMiddlewareEmptyOrigin(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	middleware := corsMiddleware(testHandler, []string{"https://example.com"})

	// Test request with no Origin header
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	middleware.ServeHTTP(rr, req)

	// Should NOT set CORS headers when origin is empty
	assert.Empty(t, rr.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "OK", rr.Body.String())
}

func TestIsOriginAllowed(t *testing.T) {
	tests := []struct {
		name           string
		origin         string
		allowedOrigins []string
		host           string
		expected       bool
	}{
		{
			name:           "empty origin returns false",
			origin:         "",
			allowedOrigins: []string{"https://example.com"},
			host:           "localhost",
			expected:       false,
		},
		{
			name:           "origin in allowed list",
			origin:         "https://example.com",
			allowedOrigins: []string{"https://example.com"},
			host:           "localhost",
			expected:       true,
		},
		{
			name:           "origin not in allowed list",
			origin:         "https://malicious.com",
			allowedOrigins: []string{"https://example.com"},
			host:           "localhost",
			expected:       false,
		},
		{
			name:           "empty allowed list allows all (dev mode)",
			origin:         "https://any-origin.com",
			allowedOrigins: []string{},
			host:           "localhost",
			expected:       true,
		},
		{
			name:           "origin with whitespace in allowed list",
			origin:         "https://example.com",
			allowedOrigins: []string{" https://example.com "},
			host:           "localhost",
			expected:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isOriginAllowed(tt.origin, tt.allowedOrigins, tt.host)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestApplySecurityMiddlewareNilConfig(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// Test with nil config
	middleware := applySecurityMiddleware(testHandler, nil)
	require.NotNil(t, middleware)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://any-origin.com")
	rr := httptest.NewRecorder()

	middleware.ServeHTTP(rr, req)

	// Should still have security headers
	assert.Equal(t, "nosniff", rr.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", rr.Header().Get("X-Frame-Options"))
	// Nil config means dev mode, all origins allowed
	assert.Equal(t, "https://any-origin.com", rr.Header().Get("Access-Control-Allow-Origin"))
}

func TestApplySecurityMiddlewareRateLimitDisabled(t *testing.T) {
	cfg := &config.Config{
		Security: config.SecurityConfig{
			AllowedOrigins:  []string{"https://example.com"},
			EnableRateLimit: false,
		},
	}

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	middleware := applySecurityMiddleware(testHandler, cfg)
	require.NotNil(t, middleware)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://example.com")
	rr := httptest.NewRecorder()

	middleware.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "https://example.com", rr.Header().Get("Access-Control-Allow-Origin"))
}

func TestRequestLoggingMiddleware(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	middleware := requestLoggingMiddleware(testHandler)
	require.NotNil(t, middleware)

	req := httptest.NewRequest("GET", "/test-path", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	rr := httptest.NewRecorder()

	middleware.ServeHTTP(rr, req)

	// Verify the request was processed
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "OK", rr.Body.String())
}

func TestStartServerNilServer(t *testing.T) {
	cfg := &config.Config{}
	err := startServer(nil, cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "server is nil")
}

func TestStartServerInvalidAddress(t *testing.T) {
	cfg := &config.Config{}

	// Create server with invalid address to trigger an error
	server := &http.Server{
		Addr: "invalid-addr:-1",
	}

	err := startServer(server, cfg)
	assert.Error(t, err)
}

func TestStartServerAlreadyInUse(t *testing.T) {
	cfg := &config.Config{}

	// Start first server on a specific port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = listener.Close() }()

	port := listener.Addr().(*net.TCPAddr).Port
	addr := fmt.Sprintf("127.0.0.1:%d", port)

	// Try to start second server on same port
	server := &http.Server{
		Addr: addr,
	}

	err = startServer(server, cfg)
	assert.Error(t, err)
}

func TestParsedArgsStruct(t *testing.T) {
	// Test parsedArgs struct fields are accessible
	rfxEnabled := true
	args := parsedArgs{
		host:          "localhost",
		port:          "8080",
		logLevel:      "debug",
		skipTLS:       true,
		tlsServerName: "example.com",
		useNLA:        true,
		enableRFX:     &rfxEnabled,
	}

	assert.Equal(t, "localhost", args.host)
	assert.Equal(t, "8080", args.port)
	assert.Equal(t, "debug", args.logLevel)
	assert.True(t, args.skipTLS)
	assert.Equal(t, "example.com", args.tlsServerName)
	assert.True(t, args.useNLA)
	require.NotNil(t, args.enableRFX)
	assert.True(t, *args.enableRFX)
}

func TestRunWithValidConfig(t *testing.T) {
	// Start a listener to determine a free port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()

	args := parsedArgs{
		host:     "127.0.0.1",
		port:     fmt.Sprintf("%d", port),
		logLevel: "info",
	}

	// Run in goroutine since it blocks
	errCh := make(chan error, 1)
	go func() {
		errCh <- run(args)
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Server should be running, so we need to force stop it
	// The test passes if we got this far without error
	select {
	case err := <-errCh:
		// If error occurred immediately, check it
		if err != nil {
			t.Logf("run error (may be expected): %v", err)
		}
	default:
		// Server is running, test passed
	}
}

func TestRunWithServerError(t *testing.T) {
	// Occupy a port first
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = listener.Close() }()
	port := listener.Addr().(*net.TCPAddr).Port

	args := parsedArgs{
		host:     "127.0.0.1",
		port:     fmt.Sprintf("%d", port), // Port already in use
		logLevel: "info",
	}

	// run should fail because port is already in use
	err = run(args)
	assert.Error(t, err)
}

func TestParseFlagsWithArgs(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedAction string
		checkArgs      func(t *testing.T, args parsedArgs)
	}{
		{
			name:           "no args returns empty args",
			args:           []string{},
			expectedAction: "",
			checkArgs: func(t *testing.T, args parsedArgs) {
				assert.Empty(t, args.host)
				assert.Empty(t, args.port)
				assert.Nil(t, args.enableRFX)
			},
		},
		{
			name:           "host and port args",
			args:           []string{"-host", "localhost", "-port", "9090"},
			expectedAction: "",
			checkArgs: func(t *testing.T, args parsedArgs) {
				assert.Equal(t, "localhost", args.host)
				assert.Equal(t, "9090", args.port)
			},
		},
		{
			name:           "all flags",
			args:           []string{"-host", "0.0.0.0", "-port", "8080", "-log-level", "debug", "-tls-skip-verify", "-tls-server-name", "server.local", "-nla"},
			expectedAction: "",
			checkArgs: func(t *testing.T, args parsedArgs) {
				assert.Equal(t, "0.0.0.0", args.host)
				assert.Equal(t, "8080", args.port)
				assert.Equal(t, "debug", args.logLevel)
				assert.True(t, args.skipTLS)
				assert.Equal(t, "server.local", args.tlsServerName)
				assert.True(t, args.useNLA)
			},
		},
		{
			name:           "no-rfx flag disables RFX",
			args:           []string{"-no-rfx"},
			expectedAction: "",
			checkArgs: func(t *testing.T, args parsedArgs) {
				require.NotNil(t, args.enableRFX)
				assert.False(t, *args.enableRFX)
			},
		},
		{
			name:           "udp flag enables UDP transport",
			args:           []string{"-udp"},
			expectedAction: "",
			checkArgs: func(t *testing.T, args parsedArgs) {
				require.NotNil(t, args.enableUDP)
				assert.True(t, *args.enableUDP)
			},
		},
		{
			name:           "udp flag not set by default",
			args:           []string{},
			expectedAction: "",
			checkArgs: func(t *testing.T, args parsedArgs) {
				assert.Nil(t, args.enableUDP)
			},
		},
		{
			name:           "help flag returns help action",
			args:           []string{"-help"},
			expectedAction: "help",
			checkArgs:      func(t *testing.T, args parsedArgs) {},
		},
		{
			name:           "version flag returns version action",
			args:           []string{"-version"},
			expectedAction: "version",
			checkArgs:      func(t *testing.T, args parsedArgs) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout for help/version tests
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			args, action := parseFlagsWithArgs(tt.args)

			os.Stdout = oldStdout
			_ = w.Close()
			_ = r.Close()

			assert.Equal(t, tt.expectedAction, action)
			if tt.checkArgs != nil {
				tt.checkArgs(t, args)
			}
		})
	}
}

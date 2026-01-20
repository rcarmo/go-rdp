package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		want    *Config
		wantErr bool
	}{
		{
			name:    "default configuration",
			envVars: map[string]string{},
			want: &Config{
				Server: ServerConfig{
					Host:         "0.0.0.0",
					Port:         "8080",
					ReadTimeout:  30 * time.Second,
					WriteTimeout: 30 * time.Second,
					IdleTimeout:  120 * time.Second,
				},
				RDP: RDPConfig{
					DefaultWidth:  1024,
					DefaultHeight: 768,
					MaxWidth:      3840,
					MaxHeight:     2160,
					BufferSize:    65536,
					Timeout:       10 * time.Second,
				},
				Security: SecurityConfig{
					AllowedOrigins:     []string{},
					MaxConnections:     100,
					RateLimitPerMinute: 60,
					EnableTLS:          false,
					TLSCertFile:        "",
					TLSKeyFile:         "",
					MinTLSVersion:      "1.2",
				},
				Logging: LoggingConfig{
					Level:        "info",
					Format:       "text",
					EnableCaller: false,
					File:         "",
				},
			},
			wantErr: false,
		},
		{
			name: "custom environment variables",
			envVars: map[string]string{
				"SERVER_HOST":        "127.0.0.1",
				"SERVER_PORT":        "9090",
				"LOG_LEVEL":          "debug",
				"MAX_CONNECTIONS":    "50",
				"RDP_DEFAULT_WIDTH":  "1920",
				"RDP_DEFAULT_HEIGHT": "1080",
			},
			want: &Config{
				Server: ServerConfig{
					Host:         "127.0.0.1",
					Port:         "9090",
					ReadTimeout:  30 * time.Second,
					WriteTimeout: 30 * time.Second,
					IdleTimeout:  120 * time.Second,
				},
				RDP: RDPConfig{
					DefaultWidth:  1920,
					DefaultHeight: 1080,
					MaxWidth:      3840,
					MaxHeight:     2160,
					BufferSize:    65536,
					Timeout:       10 * time.Second,
				},
				Security: SecurityConfig{
					AllowedOrigins:     []string{},
					MaxConnections:     50,
					EnableRateLimit:    true,
					RateLimitPerMinute: 60,
					EnableTLS:          false, // Don't enable TLS without cert files
					TLSCertFile:        "",
					TLSKeyFile:         "",
					MinTLSVersion:      "1.2",
				},
				Logging: LoggingConfig{
					Level:        "debug",
					Format:       "text",
					EnableCaller: false,
					File:         "",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			for k := range tt.envVars {
				os.Unsetenv(k)
			}

			// Set test environment variables
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			// Load configuration
			cfg, err := Load()

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want.Server.Host, cfg.Server.Host)
			assert.Equal(t, tt.want.Server.Port, cfg.Server.Port)
			assert.Equal(t, tt.want.RDP.DefaultWidth, cfg.RDP.DefaultWidth)
			assert.Equal(t, tt.want.RDP.DefaultHeight, cfg.RDP.DefaultHeight)
			assert.Equal(t, tt.want.Security.MaxConnections, cfg.Security.MaxConnections)
			assert.Equal(t, tt.want.Logging.Level, cfg.Logging.Level)
			assert.Equal(t, tt.want.Security.EnableTLS, cfg.Security.EnableTLS)

			// Clean up environment
			for k := range tt.envVars {
				os.Unsetenv(k)
			}
		})
	}
}

func TestLoadWithOverrides(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		opts    LoadOptions
		want    *Config
	}{
		{
			name:    "command-line overrides",
			envVars: map[string]string{},
			opts: LoadOptions{
				Host:              "192.168.1.100",
				Port:              "443",
				LogLevel:          "warn",
				SkipTLSValidation: true,
			},
			want: &Config{
				Server: ServerConfig{
					Host:         "192.168.1.100",
					Port:         "443",
					ReadTimeout:  30 * time.Second,
					WriteTimeout: 30 * time.Second,
					IdleTimeout:  120 * time.Second,
				},
				Logging: LoggingConfig{
					Level:        "warn",
					Format:       "text",
					EnableCaller: false,
					File:         "",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			for k := range tt.envVars {
				os.Unsetenv(k)
			}

			cfg, err := LoadWithOverrides(tt.opts)

			require.NoError(t, err)
			assert.Equal(t, tt.want.Server.Host, cfg.Server.Host)
			assert.Equal(t, tt.want.Server.Port, cfg.Server.Port)
			assert.Equal(t, tt.want.Logging.Level, cfg.Logging.Level)

			// Clean up environment
			for k := range tt.envVars {
				os.Unsetenv(k)
			}
		})
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid configuration",
			cfg: &Config{
				Server:   ServerConfig{Host: "0.0.0.0", Port: "8080"},
				RDP:      RDPConfig{DefaultWidth: 1024, DefaultHeight: 768, MaxWidth: 3840, MaxHeight: 2160, BufferSize: 65536},
				Security: SecurityConfig{MaxConnections: 100, RateLimitPerMinute: 60, EnableRateLimit: true},
				Logging:  LoggingConfig{Level: "info", Format: "text"},
			},
			wantErr: false,
		},
		{
			name: "missing server port",
			cfg: &Config{
				Server:   ServerConfig{Host: "0.0.0.0", Port: ""},
				Security: SecurityConfig{MaxConnections: 10, RateLimitPerMinute: 10, EnableRateLimit: true},
				Logging:  LoggingConfig{Level: "info", Format: "text"},
			},
			wantErr: true,
			errMsg:  "server port cannot be empty",
		},
		{
			name: "invalid port range",
			cfg: &Config{
				Server:   ServerConfig{Host: "0.0.0.0", Port: "99999"},
				Security: SecurityConfig{MaxConnections: 10, RateLimitPerMinute: 10, EnableRateLimit: true},
				Logging:  LoggingConfig{Level: "info", Format: "text"},
			},
			wantErr: true,
			errMsg:  "invalid server port",
		},
		{
			name: "invalid RDP dimensions",
			cfg: &Config{
				Server:   ServerConfig{Host: "0.0.0.0", Port: "8080"},
				RDP:      RDPConfig{DefaultWidth: -1, DefaultHeight: 768, MaxWidth: 3840, MaxHeight: 2160, BufferSize: 65536},
				Security: SecurityConfig{MaxConnections: 10, RateLimitPerMinute: 10, EnableRateLimit: true},
				Logging:  LoggingConfig{Level: "info", Format: "text"},
			},
			wantErr: true,
			errMsg:  "default dimensions must be positive",
		},
		{
			name: "max dimensions less than defaults",
			cfg: &Config{
				Server:   ServerConfig{Host: "0.0.0.0", Port: "8080"},
				RDP:      RDPConfig{DefaultWidth: 2000, DefaultHeight: 1200, MaxWidth: 1000, MaxHeight: 800, BufferSize: 65536},
				Security: SecurityConfig{MaxConnections: 10, RateLimitPerMinute: 10, EnableRateLimit: true},
				Logging:  LoggingConfig{Level: "info", Format: "text"},
			},
			wantErr: true,
			errMsg:  "max dimensions must be >= default dimensions",
		},
		{
			name: "invalid buffer size",
			cfg: &Config{
				Server:   ServerConfig{Host: "0.0.0.0", Port: "8080"},
				RDP:      RDPConfig{DefaultWidth: 1024, DefaultHeight: 768, MaxWidth: 3840, MaxHeight: 2160, BufferSize: 0},
				Security: SecurityConfig{MaxConnections: 10, RateLimitPerMinute: 10, EnableRateLimit: true},
				Logging:  LoggingConfig{Level: "info", Format: "text"},
			},
			wantErr: true,
			errMsg:  "buffer size must be positive",
		},
		{
			name: "invalid log level",
			cfg: &Config{
				Server:   ServerConfig{Host: "0.0.0.0", Port: "8080"},
				RDP:      RDPConfig{DefaultWidth: 1024, DefaultHeight: 768, MaxWidth: 3840, MaxHeight: 2160, BufferSize: 65536},
				Security: SecurityConfig{MaxConnections: 10, RateLimitPerMinute: 10, EnableRateLimit: true},
				Logging:  LoggingConfig{Level: "invalid", Format: "text"},
			},
			wantErr: true,
			errMsg:  "invalid log level",
		},
		{
			name: "invalid log format",
			cfg: &Config{
				Server:   ServerConfig{Host: "0.0.0.0", Port: "8080"},
				RDP:      RDPConfig{DefaultWidth: 1024, DefaultHeight: 768, MaxWidth: 3840, MaxHeight: 2160, BufferSize: 65536},
				Security: SecurityConfig{MaxConnections: 10, RateLimitPerMinute: 10, EnableRateLimit: true},
				Logging:  LoggingConfig{Level: "info", Format: "xml"},
			},
			wantErr: true,
			errMsg:  "invalid log format",
		},
		{
			name: "TLS enabled without certs",
			cfg: &Config{
				Server: ServerConfig{Host: "0.0.0.0", Port: "8080"},
				Security: SecurityConfig{
					EnableTLS:          true,
					MaxConnections:     10,
					RateLimitPerMinute: 10,
					EnableRateLimit:    true,
					TLSCertFile:        "",
					TLSKeyFile:         "",
				},
				RDP:     RDPConfig{DefaultWidth: 1024, DefaultHeight: 768, MaxWidth: 3840, MaxHeight: 2160, BufferSize: 65536},
				Logging: LoggingConfig{Level: "info", Format: "text"},
			},
			wantErr: true,
			errMsg:  "TLS certificate and key files must be specified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestGetEnvWithDefault(t *testing.T) {
	key := "TEST_CONFIG_VAR"
	defaultValue := "default"
	testValue := "test_value"

	// Test when env var is not set
	os.Unsetenv(key)
	result := getEnvWithDefault(key, defaultValue)
	assert.Equal(t, defaultValue, result)

	// Test when env var is set
	os.Setenv(key, testValue)
	result = getEnvWithDefault(key, defaultValue)
	assert.Equal(t, testValue, result)

	// Clean up
	os.Unsetenv(key)
}

func TestGetIntWithDefault(t *testing.T) {
	key := "TEST_INT_VAR"
	defaultValue := 42
	testValue := "100"

	// Test when env var is not set
	os.Unsetenv(key)
	result := getIntWithDefault(key, defaultValue)
	assert.Equal(t, defaultValue, result)

	// Test when env var is set with valid integer
	os.Setenv(key, testValue)
	result = getIntWithDefault(key, defaultValue)
	assert.Equal(t, 100, result)

	// Test when env var is set with invalid integer
	os.Setenv(key, "invalid")
	result = getIntWithDefault(key, defaultValue)
	assert.Equal(t, defaultValue, result)

	// Clean up
	os.Unsetenv(key)
}

func TestGetBoolWithDefault(t *testing.T) {
	key := "TEST_BOOL_VAR"
	defaultValue := false

	// Test when env var is not set
	os.Unsetenv(key)
	result := getBoolWithDefault(key, defaultValue)
	assert.Equal(t, defaultValue, result)

	// Test when env var is set with true
	os.Setenv(key, "true")
	result = getBoolWithDefault(key, defaultValue)
	assert.Equal(t, true, result)

	// Test when env var is set with false
	os.Setenv(key, "false")
	result = getBoolWithDefault(key, defaultValue)
	assert.Equal(t, false, result)

	// Test when env var is set with invalid boolean
	os.Setenv(key, "invalid")
	result = getBoolWithDefault(key, defaultValue)
	assert.Equal(t, defaultValue, result)

	// Clean up
	os.Unsetenv(key)
}

func TestGetDurationWithDefault(t *testing.T) {
	key := "TEST_DURATION_VAR"
	defaultValue := 30 * time.Second
	testValue := "60s"

	// Test when env var is not set
	os.Unsetenv(key)
	result := getDurationWithDefault(key, defaultValue)
	assert.Equal(t, defaultValue, result)

	// Test when env var is set with valid duration
	os.Setenv(key, testValue)
	result = getDurationWithDefault(key, defaultValue)
	assert.Equal(t, 60*time.Second, result)

	// Test when env var is set with invalid duration
	os.Setenv(key, "invalid")
	result = getDurationWithDefault(key, defaultValue)
	assert.Equal(t, defaultValue, result)

	// Clean up
	os.Unsetenv(key)
}

func TestGetStringSliceWithDefault(t *testing.T) {
	key := "TEST_SLICE_VAR"
	defaultValue := []string{"default1", "default2"}
	testValue := "value1,value2,value3"

	// Test when env var is not set
	os.Unsetenv(key)
	result := getStringSliceWithDefault(key, defaultValue)
	assert.Equal(t, defaultValue, result)

	// Test when env var is set with comma-separated values
	os.Setenv(key, testValue)
	result = getStringSliceWithDefault(key, defaultValue)
	assert.Equal(t, []string{"value1", "value2", "value3"}, result)

	// Test when env var is empty
	os.Setenv(key, "")
	result = getStringSliceWithDefault(key, defaultValue)
	assert.Equal(t, defaultValue, result)

	// Clean up
	os.Unsetenv(key)
}

func TestGetOverrideOrEnv(t *testing.T) {
	key := "TEST_OVERRIDE_VAR"
	override := "override_value"
	envValue := "env_value"
	defaultValue := "default_value"

	// Test when override is provided
	os.Setenv(key, envValue)
	result := getOverrideOrEnv(override, key, defaultValue)
	assert.Equal(t, override, result)

	// Test when override is empty but env var is set
	os.Setenv(key, envValue)
	result = getOverrideOrEnv("", key, defaultValue)
	assert.Equal(t, envValue, result)

	// Test when both override and env are empty
	os.Unsetenv(key)
	result = getOverrideOrEnv("", key, defaultValue)
	assert.Equal(t, defaultValue, result)

	// Clean up
	os.Unsetenv(key)
}

func TestSplitString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		sep      string
		expected []string
	}{
		{
			name:     "normal comma separation",
			input:    "a,b,c",
			sep:      ",",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "with whitespace",
			input:    "a, b , c",
			sep:      ",",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "empty input",
			input:    "",
			sep:      ",",
			expected: []string{},
		},
		{
			name:     "empty elements",
			input:    "a,,c",
			sep:      ",",
			expected: []string{"a", "c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitString(tt.input, tt.sep)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetGlobalConfig(t *testing.T) {
	// Test that GetGlobalConfig returns nil before any config is stored
	// This tests the thread-safe global config getter
	cfg := GetGlobalConfig()
	// Initially may be nil or have a default value
	_ = cfg
}

package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// globalConfig stores the configuration loaded with command-line overrides
// This allows other packages to access the same configuration that was loaded by the server
var (
	globalConfig *Config
	configMutex  sync.Mutex
)

// Config holds the application configuration
type Config struct {
	Server   ServerConfig   `json:"server"`
	RDP      RDPConfig      `json:"rdp"`
	Security SecurityConfig `json:"security"`
	Logging  LoggingConfig  `json:"logging"`
}

// LoadOptions holds command-line override options
type LoadOptions struct {
	Host              string
	Port              string
	LogLevel          string
	ConfigFile        string
	SkipTLSValidation bool
	TLSServerName     string
	UseNLA            bool
}

// ServerConfig holds server-specific configuration
type ServerConfig struct {
	Host         string        `json:"host" env:"SERVER_HOST" default:"0.0.0.0"`
	Port         string        `json:"port" env:"SERVER_PORT" default:"8080"`
	ReadTimeout  time.Duration `json:"readTimeout" env:"SERVER_READ_TIMEOUT" default:"30s"`
	WriteTimeout time.Duration `json:"writeTimeout" env:"SERVER_WRITE_TIMEOUT" default:"30s"`
	IdleTimeout  time.Duration `json:"idleTimeout" env:"SERVER_IDLE_TIMEOUT" default:"120s"`
}

// RDPConfig holds RDP-specific configuration
type RDPConfig struct {
	DefaultWidth  int           `json:"defaultWidth" env:"RDP_DEFAULT_WIDTH" default:"1024"`
	DefaultHeight int           `json:"defaultHeight" env:"RDP_DEFAULT_HEIGHT" default:"768"`
	MaxWidth      int           `json:"maxWidth" env:"RDP_MAX_WIDTH" default:"3840"`
	MaxHeight     int           `json:"maxHeight" env:"RDP_MAX_HEIGHT" default:"2160"`
	BufferSize    int           `json:"bufferSize" env:"RDP_BUFFER_SIZE" default:"65536"`
	Timeout       time.Duration `json:"timeout" env:"RDP_TIMEOUT" default:"10s"`
}

// SecurityConfig holds security-related configuration
type SecurityConfig struct {
	AllowedOrigins     []string `json:"allowedOrigins" env:"ALLOWED_ORIGINS" default:""`
	MaxConnections     int      `json:"maxConnections" env:"MAX_CONNECTIONS" default:"100"`
	EnableRateLimit    bool     `json:"enableRateLimit" env:"ENABLE_RATE_LIMIT" default:"true"`
	RateLimitPerMinute int      `json:"rateLimitPerMinute" env:"RATE_LIMIT_PER_MINUTE" default:"60"`
	EnableTLS          bool     `json:"enableTLS" env:"ENABLE_TLS" default:"false"`
	TLSCertFile        string   `json:"tlsCertFile" env:"TLS_CERT_FILE" default:""`
	TLSKeyFile         string   `json:"tlsKeyFile" env:"TLS_KEY_FILE" default:""`
	MinTLSVersion      string   `json:"minTLSVersion" env:"MIN_TLS_VERSION" default:"1.2"`
	SkipTLSValidation  bool     `json:"skipTLSValidation" env:"SKIP_TLS_VALIDATION" default:"false"`
	TLSServerName      string   `json:"tlsServerName" env:"TLS_SERVER_NAME" default:""`
	UseNLA             bool     `json:"useNLA" env:"USE_NLA" default:"true"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level        string `json:"level" env:"LOG_LEVEL" default:"info"`
	Format       string `json:"format" env:"LOG_FORMAT" default:"text"`
	EnableCaller bool   `json:"enableCaller" env:"LOG_ENABLE_CALLER" default:"false"`
	File         string `json:"file" env:"LOG_FILE" default:""`
}

// Load loads configuration from environment variables with defaults
func Load() (*Config, error) {
	return LoadWithOverrides(LoadOptions{})
}

// LoadWithOverrides loads configuration with command-line overrides
func LoadWithOverrides(opts LoadOptions) (*Config, error) {
	config := &Config{}

	// Server config
	config.Server.Host = getOverrideOrEnv(opts.Host, "SERVER_HOST", "0.0.0.0")
	config.Server.Port = getOverrideOrEnv(opts.Port, "SERVER_PORT", "8080")
	config.Server.ReadTimeout = getDurationWithDefault("SERVER_READ_TIMEOUT", 30*time.Second)
	config.Server.WriteTimeout = getDurationWithDefault("SERVER_WRITE_TIMEOUT", 30*time.Second)
	config.Server.IdleTimeout = getDurationWithDefault("SERVER_IDLE_TIMEOUT", 120*time.Second)

	// RDP config
	config.RDP.DefaultWidth = getIntWithDefault("RDP_DEFAULT_WIDTH", 1024)
	config.RDP.DefaultHeight = getIntWithDefault("RDP_DEFAULT_HEIGHT", 768)
	config.RDP.MaxWidth = getIntWithDefault("RDP_MAX_WIDTH", 3840)
	config.RDP.MaxHeight = getIntWithDefault("RDP_MAX_HEIGHT", 2160)
	config.RDP.BufferSize = getIntWithDefault("RDP_BUFFER_SIZE", 65536)
	config.RDP.Timeout = getDurationWithDefault("RDP_TIMEOUT", 10*time.Second)

	// Security config
	config.Security.AllowedOrigins = getStringSliceWithDefault("ALLOWED_ORIGINS", []string{})
	config.Security.MaxConnections = getIntWithDefault("MAX_CONNECTIONS", 100)
	config.Security.EnableRateLimit = getBoolWithDefault("ENABLE_RATE_LIMIT", true)
	config.Security.RateLimitPerMinute = getIntWithDefault("RATE_LIMIT_PER_MINUTE", 60)
	config.Security.EnableTLS = getBoolWithDefault("ENABLE_TLS", false)
	config.Security.TLSCertFile = getEnvWithDefault("TLS_CERT_FILE", "")
	config.Security.TLSKeyFile = getEnvWithDefault("TLS_KEY_FILE", "")
	config.Security.MinTLSVersion = getEnvWithDefault("MIN_TLS_VERSION", "1.2")
	config.Security.SkipTLSValidation = getBoolWithDefault("SKIP_TLS_VALIDATION", false) || opts.SkipTLSValidation
	config.Security.TLSServerName = getOverrideOrEnv(opts.TLSServerName, "TLS_SERVER_NAME", "")
	// NLA enabled by default for security; set USE_NLA=false to disable
	config.Security.UseNLA = getBoolWithDefault("USE_NLA", true)
	if opts.UseNLA {
		config.Security.UseNLA = true
	}

	// Logging config
	config.Logging.Level = getOverrideOrEnv(opts.LogLevel, "LOG_LEVEL", "info")
	config.Logging.Format = getEnvWithDefault("LOG_FORMAT", "text")
	config.Logging.EnableCaller = getBoolWithDefault("LOG_ENABLE_CALLER", false)
	config.Logging.File = getEnvWithDefault("LOG_FILE", "")

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Store the configuration globally so other packages can access it
	configMutex.Lock()
	globalConfig = config
	configMutex.Unlock()

	return config, nil
}

// GetGlobalConfig returns the globally stored configuration
// This should be used by packages that need access to the configuration
// loaded by the server with command-line overrides
func GetGlobalConfig() *Config {
	configMutex.Lock()
	defer configMutex.Unlock()
	return globalConfig
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate server config
	if c.Server.Port == "" {
		return fmt.Errorf("server port cannot be empty")
	}

	if port, err := strconv.Atoi(c.Server.Port); err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("invalid server port: %s", c.Server.Port)
	}

	// Validate RDP config
	if c.RDP.DefaultWidth <= 0 || c.RDP.DefaultHeight <= 0 {
		return fmt.Errorf("default dimensions must be positive")
	}

	if c.RDP.MaxWidth < c.RDP.DefaultWidth || c.RDP.MaxHeight < c.RDP.DefaultHeight {
		return fmt.Errorf("max dimensions must be >= default dimensions")
	}

	if c.RDP.BufferSize <= 0 {
		return fmt.Errorf("buffer size must be positive")
	}

	// Validate security config
	if c.Security.EnableTLS {
		if c.Security.TLSCertFile == "" || c.Security.TLSKeyFile == "" {
			return fmt.Errorf("TLS certificate and key files must be specified when TLS is enabled")
		}

		if _, err := os.Stat(c.Security.TLSCertFile); os.IsNotExist(err) {
			return fmt.Errorf("TLS certificate file does not exist: %s", c.Security.TLSCertFile)
		}

		if _, err := os.Stat(c.Security.TLSKeyFile); os.IsNotExist(err) {
			return fmt.Errorf("TLS key file does not exist: %s", c.Security.TLSKeyFile)
		}
	}

	if c.Security.MaxConnections <= 0 {
		return fmt.Errorf("max connections must be positive")
	}

	if c.Security.RateLimitPerMinute <= 0 {
		return fmt.Errorf("rate limit per minute must be positive")
	}

	// Validate logging config
	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}

	if !validLogLevels[c.Logging.Level] {
		return fmt.Errorf("invalid log level: %s", c.Logging.Level)
	}

	validLogFormats := map[string]bool{
		"text": true,
		"json": true,
	}

	if !validLogFormats[c.Logging.Format] {
		return fmt.Errorf("invalid log format: %s", c.Logging.Format)
	}

	return nil
}

// Helper functions for environment variable parsing
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntWithDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getBoolWithDefault(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getDurationWithDefault(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getStringSliceWithDefault(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return splitString(value, ",")
	}
	return defaultValue
}

// getOverrideOrEnv returns command-line override value, env value, or default
func getOverrideOrEnv(override, envKey, defaultValue string) string {
	if override != "" {
		return override
	}
	return getEnvWithDefault(envKey, defaultValue)
}

func splitString(s, sep string) []string {
	if s == "" {
		return []string{}
	}

	var result []string
	for _, part := range strings.Split(s, sep) {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

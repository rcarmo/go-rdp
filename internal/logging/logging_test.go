package logging

import (
	"bytes"
	"log"
	"strings"
	"testing"
)

func TestSetLevel(t *testing.T) {
	tests := []struct {
		name  string
		level Level
	}{
		{"Debug", LevelDebug},
		{"Info", LevelInfo},
		{"Warn", LevelWarn},
		{"Error", LevelError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetLevel(tt.level)
			if Default().GetLevel() != tt.level {
				t.Errorf("SetLevel(%v) = %v, want %v", tt.level, Default().GetLevel(), tt.level)
			}
		})
	}
}

func TestSetLevelFromString(t *testing.T) {
	tests := []struct {
		input    string
		expected Level
	}{
		{"debug", LevelDebug},
		{"DEBUG", LevelDebug},
		{"info", LevelInfo},
		{"INFO", LevelInfo},
		{"warn", LevelWarn},
		{"warning", LevelWarn},
		{"error", LevelError},
		{"ERROR", LevelError},
		{"invalid", LevelInfo}, // defaults to info
		{"", LevelInfo},        // defaults to info
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			SetLevelFromString(tt.input)
			if Default().GetLevel() != tt.expected {
				t.Errorf("SetLevelFromString(%q) = %v, want %v", tt.input, Default().GetLevel(), tt.expected)
			}
		})
	}
}

func TestLoggingOutput(t *testing.T) {
	// Create a custom logger for testing
	var buf bytes.Buffer
	testLogger := &Logger{
		level:  LevelDebug,
		logger: log.New(&buf, "", 0),
	}

	// Test Debug level
	testLogger.SetLevel(LevelDebug)
	buf.Reset()
	testLogger.Debug("test debug %d", 1)
	if !strings.Contains(buf.String(), "[DEBUG]") || !strings.Contains(buf.String(), "test debug 1") {
		t.Errorf("Debug() output = %q, want to contain [DEBUG] and 'test debug 1'", buf.String())
	}

	// Test that Debug is suppressed at Info level
	testLogger.SetLevel(LevelInfo)
	buf.Reset()
	testLogger.Debug("should not appear")
	if buf.Len() != 0 {
		t.Errorf("Debug() at Info level should produce no output, got %q", buf.String())
	}

	// Test Info at Info level
	buf.Reset()
	testLogger.Info("test info")
	if !strings.Contains(buf.String(), "[INFO]") {
		t.Errorf("Info() output = %q, want to contain [INFO]", buf.String())
	}

	// Test Warn
	buf.Reset()
	testLogger.Warn("test warn")
	if !strings.Contains(buf.String(), "[WARN]") {
		t.Errorf("Warn() output = %q, want to contain [WARN]", buf.String())
	}

	// Test Error
	buf.Reset()
	testLogger.Error("test error")
	if !strings.Contains(buf.String(), "[ERROR]") {
		t.Errorf("Error() output = %q, want to contain [ERROR]", buf.String())
	}
}

func TestGetLevel(t *testing.T) {
	SetLevel(LevelWarn)
	if Default().GetLevel() != LevelWarn {
		t.Errorf("GetLevel() = %v, want %v", Default().GetLevel(), LevelWarn)
	}
}

func TestGetLevelString(t *testing.T) {
	tests := []struct {
		level    Level
		expected string
	}{
		{LevelDebug, "DEBUG"},
		{LevelInfo, "INFO"},
		{LevelWarn, "WARN"},
		{LevelError, "ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			SetLevel(tt.level)
			result := GetLevelString()
			if result != tt.expected {
				t.Errorf("GetLevelString() = %q, want %q", result, tt.expected)
			}
		})
	}
}

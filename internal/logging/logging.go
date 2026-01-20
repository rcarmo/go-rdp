// Package logging provides a simple leveled logger for the RDP client.
package logging

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
)

// Level represents log severity levels
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

var levelNames = map[Level]string{
	LevelDebug: "DEBUG",
	LevelInfo:  "INFO",
	LevelWarn:  "WARN",
	LevelError: "ERROR",
}

// Logger provides leveled logging
type Logger struct {
	level  Level
	mu     sync.RWMutex
	logger *log.Logger
}

var (
	defaultLogger *Logger
	once          sync.Once
)

// Default returns the default logger instance
func Default() *Logger {
	once.Do(func() {
		defaultLogger = &Logger{
			level:  LevelInfo,
			logger: log.New(os.Stderr, "", log.LstdFlags|log.LUTC),
		}
	})
	return defaultLogger
}

// SetLevel sets the minimum log level
func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// SetLevelFromString sets the log level from a string
func (l *Logger) SetLevelFromString(levelStr string) {
	switch strings.ToLower(levelStr) {
	case "debug":
		l.SetLevel(LevelDebug)
	case "info":
		l.SetLevel(LevelInfo)
	case "warn", "warning":
		l.SetLevel(LevelWarn)
	case "error":
		l.SetLevel(LevelError)
	default:
		l.SetLevel(LevelInfo)
	}
}

// GetLevel returns the current log level
func (l *Logger) GetLevel() Level {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.level
}

// GetLevelString returns the current log level as a string
func (l *Logger) GetLevelString() string {
	return levelNames[l.GetLevel()]
}

// GetLevelString returns the default logger's level as a string
func GetLevelString() string {
	return Default().GetLevelString()
}

func (l *Logger) log(level Level, format string, args ...interface{}) {
	l.mu.RLock()
	currentLevel := l.level
	l.mu.RUnlock()

	if level < currentLevel {
		return
	}

	prefix := levelNames[level]
	msg := fmt.Sprintf(format, args...)
	l.logger.Printf("[%s] %s", prefix, msg)
}

// Debug logs a debug message
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(LevelDebug, format, args...)
}

// Info logs an info message
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(LevelInfo, format, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(LevelWarn, format, args...)
}

// Error logs an error message
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(LevelError, format, args...)
}

// Package-level convenience functions

// SetLevel sets the default logger's level
func SetLevel(level Level) {
	Default().SetLevel(level)
}

// SetLevelFromString sets the default logger's level from a string
func SetLevelFromString(levelStr string) {
	Default().SetLevelFromString(levelStr)
}

// Debug logs a debug message to the default logger
func Debug(format string, args ...interface{}) {
	Default().Debug(format, args...)
}

// Info logs an info message to the default logger
func Info(format string, args ...interface{}) {
	Default().Info(format, args...)
}

// Warn logs a warning message to the default logger
func Warn(format string, args ...interface{}) {
	Default().Warn(format, args...)
}

// Error(format string, args ...interface{}) logs an error message to the default logger
func Error(format string, args ...interface{}) {
	Default().Error(format, args...)
}

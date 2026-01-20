# Logging Package

> Internal leveled logging for the RDP HTML5 client

## Overview

This package provides a simple, thread-safe leveled logging system for the RDP client. It filters log messages based on severity level, allowing operators to control verbosity without code changes.

## Log Levels

| Level | Value | Description |
|-------|-------|-------------|
| `DEBUG` | 0 | Detailed protocol information, byte dumps, timing |
| `INFO` | 1 | Connection state changes, important milestones |
| `WARN` | 2 | Recoverable issues, deprecation notices |
| `ERROR` | 3 | Failures that affect operation |

## Architecture

```
┌─────────────────────────────────────────────────────┐
│                 Package API                         │
├─────────────────────────────────────────────────────┤
│  SetLevel() SetLevelFromString()                    │
│  Debug() Info() Warn() Error()                      │
└──────────────────────┬──────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────┐
│              Default Logger                          │
│  ┌──────────────────────────────────────────────┐   │
│  │ level: Level  (current threshold)             │   │
│  │ mu: sync.RWMutex (thread safety)              │   │
│  │ logger: *log.Logger (output backend)          │   │
│  └──────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────┘
```

## Key Types

### Level

```go
type Level int

const (
    LevelDebug Level = iota  // 0
    LevelInfo                 // 1
    LevelWarn                 // 2
    LevelError                // 3
)
```

### Logger

```go
type Logger struct {
    level  Level           // Minimum level to output
    mu     sync.RWMutex    // Thread-safe access
    logger *log.Logger     // Standard library logger
}
```

## Usage

### Setting Log Level

```go
import "github.com/rcarmo/rdp-html5/internal/logging"

// From code
logging.SetLevel(logging.LevelDebug)

// From configuration string
logging.SetLevelFromString("info")  // Accepts: debug, info, warn, warning, error
```

### Logging Messages

```go
logging.Debug("Protocol detail: %x", bytes)
logging.Info("Connection established to %s", host)
logging.Warn("Retry attempt %d", count)
logging.Error("Connection failed: %v", err)
```

## Configuration

The logging level is typically set during server startup:

```go
// In cmd/server/main.go
func setupLogging(cfg config.LoggingConfig) {
    level := cfg.Level
    if level == "" {
        level = "info"  // Default
    }
    logging.SetLevelFromString(level)
}
```

### Environment Variable

Set via `LOG_LEVEL` environment variable:
```bash
LOG_LEVEL=debug ./rdp-html5
```

### Command Line

```bash
./rdp-html5 -log-level debug
```

## Output Format

Log messages are formatted as:
```
2024/01/15 10:30:45 [INFO] Connection established to 192.168.1.100:3389
2024/01/15 10:30:45 [DEBUG] NLA: Sent negotiate message (127 bytes)
2024/01/15 10:30:46 [WARN] Audio: No compatible format found
2024/01/15 10:30:47 [ERROR] RDP connect: authentication failed
```

## Thread Safety

The logger is fully thread-safe:
- Level reads/writes use `sync.RWMutex`
- Multiple goroutines can log simultaneously
- Level changes take effect immediately

## Best Practices

### Level Selection Guidelines

| Scenario | Recommended Level |
|----------|-------------------|
| Production | `INFO` or `WARN` |
| Development | `DEBUG` |
| Troubleshooting | `DEBUG` |
| Performance testing | `WARN` or `ERROR` |

### Message Guidelines

- **DEBUG**: Protocol bytes, timing, internal state
- **INFO**: User-visible state changes, key milestones
- **WARN**: Unexpected but handled conditions
- **ERROR**: Failures that need attention

## Integration Points

```
┌─────────────────┐
│  cmd/server     │──▶ SetLevelFromString() on startup
└─────────────────┘
         │
┌────────▼────────┐
│ internal/rdp    │──▶ Debug() for protocol details
│                 │──▶ Info() for authentication success
│                 │──▶ Warn() for fallback scenarios
└─────────────────┘
         │
┌────────▼────────┐
│ internal/handler│──▶ Info() for capability negotiation
│                 │──▶ Error() for connection failures
└─────────────────┘
```

## Dependencies

- `log` - Standard library logger backend
- `sync` - Thread-safe level access

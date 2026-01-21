# internal/config

Central configuration management for the RDP HTML5 server.

## Overview

This package provides a structured, validated approach to managing application settings from environment variables and command-line arguments. It implements a three-layer configuration model:

1. **Defaults** - Sensible default values for all settings
2. **Environment Variables** - Override defaults via `${VAR_NAME}`
3. **CLI Arguments** - Override environment variables via flags

## Files

| File | Purpose |
|------|---------|
| `config.go` | Configuration structs, loading, and validation |
| `config_test.go` | Comprehensive unit tests |

## Configuration Structure

```go
type Config struct {
    Server   ServerConfig   // HTTP server settings
    RDP      RDPConfig      // Remote Desktop Protocol settings
    Security SecurityConfig // Security controls
    Logging  LoggingConfig  // Logging configuration
}
```

## Environment Variables

### Server Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_HOST` | `0.0.0.0` | Listen address |
| `SERVER_PORT` | `8080` | Listen port |
| `SERVER_READ_TIMEOUT` | `30s` | HTTP read timeout |
| `SERVER_WRITE_TIMEOUT` | `30s` | HTTP write timeout |
| `SERVER_IDLE_TIMEOUT` | `120s` | Keep-alive idle timeout |

### RDP Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `RDP_DEFAULT_WIDTH` | `1024` | Default desktop width |
| `RDP_DEFAULT_HEIGHT` | `768` | Default desktop height |
| `RDP_MAX_WIDTH` | `3840` | Maximum allowed width |
| `RDP_MAX_HEIGHT` | `2160` | Maximum allowed height |
| `RDP_BUFFER_SIZE` | `65536` | Network buffer size |
| `RDP_TIMEOUT` | `10s` | Connection timeout |

### Security Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `ALLOWED_ORIGINS` | (empty) | Comma-separated CORS origins |
| `MAX_CONNECTIONS` | `100` | Maximum concurrent connections |
| `ENABLE_RATE_LIMIT` | `true` | Enable request rate limiting |
| `RATE_LIMIT_PER_MINUTE` | `60` | Requests per minute per client |
| `ENABLE_TLS` | `false` | Enable HTTPS |
| `TLS_CERT_FILE` | (empty) | Path to TLS certificate |
| `TLS_KEY_FILE` | (empty) | Path to TLS private key |
| `MIN_TLS_VERSION` | `1.2` | Minimum TLS version |
| `TLS_SKIP_VERIFY` | `false` | Skip RDP server TLS validation |
| `TLS_SERVER_NAME` | (empty) | Override RDP server TLS name |
| `USE_NLA` | `true` | Enable Network Level Auth |

### Logging Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `LOG_LEVEL` | `info` | Log level: debug, info, warn, error |
| `LOG_FORMAT` | `text` | Output format: text, json |
| `LOG_ENABLE_CALLER` | `false` | Include caller information |
| `LOG_FILE` | (empty) | Log file path (empty = stdout) |

## Usage

### Loading Configuration

```go
// Load with defaults + environment variables
cfg, err := config.Load()

// Load with CLI overrides
opts := config.LoadOptions{
    Host:     "127.0.0.1",
    Port:     8443,
    LogLevel: "debug",
}
cfg, err := config.LoadWithOverrides(opts)
```

### Global Access

```go
// Store configuration globally (thread-safe)
config.SetGlobalConfig(cfg)

// Access from any package
cfg := config.GetGlobalConfig()
fmt.Println(cfg.Server.Port)
```

### Validation

The `Validate()` method checks:
- Port numbers are in valid range (1-65535)
- TLS files exist when TLS is enabled
- Desktop dimensions are within limits
- Log levels are valid values

```go
cfg, err := config.Load()
if err != nil {
    log.Fatal("Configuration validation failed:", err)
}
```

## CLI Override Priority

```
CLI Flags > Environment Variables > Defaults
```

When using `LoadWithOverrides()`, CLI options take precedence:

```go
// Environment: SERVER_PORT=8080
// CLI: -port 9000
// Result: Port = 9000
```

## Architecture

```
┌─────────────────────────────────────────┐
│             LoadWithOverrides()         │
├─────────────────────────────────────────┤
│                                         │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  │
│  │ Default │──│ Env Var │──│   CLI   │  │
│  │ Values  │  │ Values  │  │ Values  │  │
│  └─────────┘  └─────────┘  └─────────┘  │
│       │            │            │       │
│       ▼            ▼            ▼       │
│  ┌─────────────────────────────────────┐│
│  │         Merged Configuration        ││
│  └─────────────────────────────────────┘│
│                    │                    │
│                    ▼                    │
│  ┌─────────────────────────────────────┐│
│  │           Validate()                ││
│  └─────────────────────────────────────┘│
│                    │                    │
│                    ▼                    │
│  ┌─────────────────────────────────────┐│
│  │        Global Config Store          ││
│  │        (mutex-protected)            ││
│  └─────────────────────────────────────┘│
└─────────────────────────────────────────┘
```

## Related Packages

- `cmd/server` - Uses config for server startup
- `internal/handler` - Uses security settings for CORS
- `internal/rdp` - Uses RDP settings for connections

# Configuration

The application is configured via environment variables.

For the authoritative list and defaults, see `internal/pkg/config/config.go`.

## Server configuration

```bash
export SERVER_HOST=0.0.0.0
export SERVER_PORT=8080
export SERVER_READ_TIMEOUT=30s
export SERVER_WRITE_TIMEOUT=30s
export SERVER_IDLE_TIMEOUT=120s
```

## Security configuration

```bash
# CORS Configuration
# If ALLOWED_ORIGINS is not set, all origins are allowed (development mode)
# For production, explicitly set allowed origins (localhost/127.0.0.1 always allowed)
export ALLOWED_ORIGINS="https://example.com,https://app.example.com"

export MAX_CONNECTIONS=100
export ENABLE_RATE_LIMIT=true
export RATE_LIMIT_PER_MINUTE=60

export ENABLE_TLS=false
export MIN_TLS_VERSION=1.2
```

## RDP configuration

```bash
export RDP_DEFAULT_WIDTH=1024
export RDP_DEFAULT_HEIGHT=768
export RDP_MAX_WIDTH=3840
export RDP_MAX_HEIGHT=2160
export RDP_BUFFER_SIZE=65536
export RDP_TIMEOUT=10s
```

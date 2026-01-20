# Configuration

The application is configured via environment variables.

For the authoritative list and defaults, see `internal/config/config.go`.

## Server Configuration

```bash
export SERVER_HOST=0.0.0.0
export SERVER_PORT=8080
export SERVER_READ_TIMEOUT=30s
export SERVER_WRITE_TIMEOUT=30s
export SERVER_IDLE_TIMEOUT=120s
```

## Logging Configuration

```bash
# Log level: debug, info, warn, error (default: info)
export LOG_LEVEL=info

# Log format: text or json (default: text)
export LOG_FORMAT=text

# Enable caller info in logs (default: false)
export LOG_ENABLE_CALLER=false

# Log to file instead of stdout (optional)
export LOG_FILE=/var/log/rdp-html5.log
```

The log level is automatically synchronized to the browser client when a connection is established.

## Security Configuration

```bash
# CORS Configuration
# If ALLOWED_ORIGINS is not set, all origins are allowed (development mode)
# For production, explicitly set allowed origins (localhost/127.0.0.1 always allowed)
export ALLOWED_ORIGINS="https://example.com,https://app.example.com"

export MAX_CONNECTIONS=100
export ENABLE_RATE_LIMIT=true
export RATE_LIMIT_PER_MINUTE=60

# TLS for the web interface (HTTPS)
export ENABLE_TLS=false
export TLS_CERT_FILE=/path/to/cert.pem
export TLS_KEY_FILE=/path/to/key.pem
export MIN_TLS_VERSION=1.2
```

## RDP Connection Configuration

```bash
export RDP_DEFAULT_WIDTH=1024
export RDP_DEFAULT_HEIGHT=768
export RDP_MAX_WIDTH=3840
export RDP_MAX_HEIGHT=2160
export RDP_BUFFER_SIZE=65536
export RDP_TIMEOUT=10s

# Skip TLS certificate validation when connecting to RDP servers
# Set to true for self-signed certificates (NOT recommended for production)
export SKIP_TLS_VALIDATION=false

# Override TLS server name for certificate validation
export TLS_SERVER_NAME=

# Enable Network Level Authentication (default: true)
export USE_NLA=true
```

## Docker Configuration

When running in Docker, pass environment variables with `-e`:

```bash
docker run -d \
  -p 8080:8080 \
  -e LOG_LEVEL=info \
  -e SKIP_TLS_VALIDATION=true \
  ghcr.io/rcarmo/rdp-html5:latest
```

Or use a `.env` file with docker-compose:

```yaml
services:
  rdp-html5:
    image: ghcr.io/rcarmo/rdp-html5:latest
    ports:
      - "8080:8080"
    environment:
      - LOG_LEVEL=info
      - SKIP_TLS_VALIDATION=false
```

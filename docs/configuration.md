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
# For production, explicitly set allowed origins
export ALLOWED_ORIGINS="https://example.com,https://app.example.com"

export MAX_CONNECTIONS=100

# Rate limiting (NOTE: Currently a placeholder - not enforced)
# These settings are parsed but have no effect in the current implementation
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
export TLS_SKIP_VERIFY=false

# Override TLS server name for certificate validation
export TLS_SERVER_NAME=

# Allow connecting without enforcing SNI (lab/testing only)
export TLS_ALLOW_ANY_SERVER_NAME=false

# Enable Network Level Authentication (default: true)
export USE_NLA=true

# Enable RemoteFX codec support (default: true)
# Set to false to disable RFX and use simpler codecs for testing
export RDP_ENABLE_RFX=true

# Enable UDP transport (experimental, default: false)
# When enabled, the client will attempt to use UDP for data transfer
export RDP_ENABLE_UDP=false

# Prefer PCM audio for best quality (default: false)
# When false (default), prefer compressed audio (AAC/MP3) to minimize bandwidth (~128-192 kbps)
# When true, prefer PCM for lowest latency and best quality (requires ~1.4 Mbps)
export RDP_PREFER_PCM_AUDIO=false
```

## Command-Line Flags

The server also accepts command-line flags that override environment variables:

```bash
./rdp-html5 [options]

Options:
  -host                      Server listen host (default: 0.0.0.0)
  -port                      Server listen port (default: 8080)
  -log-level                 Log level: debug, info, warn, error
  -tls-skip-verify           Skip TLS certificate validation
  -tls-server-name           Override TLS server name (SNI)
  -tls-allow-any-server-name Allow connecting without enforcing SNI (lab/testing only)
  -nla                       Enable Network Level Authentication
  -no-rfx                    Disable RemoteFX codec support
  -udp                       Enable UDP transport (experimental)
  -prefer-pcm-audio          Prefer PCM audio (best quality, high bandwidth)
  -version                   Show version information
  -help                      Show help message
```

### Flag Reference

#### Server Options

- **`-host`** - Server listen address
  - Default: `0.0.0.0` (all interfaces)
  - Override: `SERVER_HOST` environment variable
  - Example: `-host 127.0.0.1` (localhost only)

- **`-port`** - Server listen port
  - Default: `8080`
  - Override: `SERVER_PORT` environment variable
  - Example: `-port 3000`

#### Logging

- **`-log-level`** - Controls logging verbosity
  - Values: `debug`, `info`, `warn`, `error`
  - Default: `info`
  - Override: `LOG_LEVEL` environment variable
  - Note: Automatically synchronized to browser client on connection

#### TLS/Security

- **`-tls-skip-verify`** - Skip TLS certificate validation when connecting to RDP servers
  - Use for self-signed certificates or testing
  - **WARNING**: Not recommended for production (vulnerable to MITM attacks)
  - Override: `TLS_SKIP_VERIFY=true` environment variable

- **`-tls-server-name`** - Override TLS server name for certificate validation
  - Use when RDP server's certificate doesn't match its hostname/IP
  - Provides explicit Server Name Indication (SNI) value
  - Override: `TLS_SERVER_NAME` environment variable
  - Example: `-tls-server-name rdp.example.com`

- **`-tls-allow-any-server-name`** - Allow connecting without enforcing SNI
  - Disables server name validation entirely
  - **LAB/TESTING ONLY** - do not use in production
  - Override: `TLS_ALLOW_ANY_SERVER_NAME=true` environment variable

#### RDP Protocol

- **`-nla`** - Enable Network Level Authentication (NLA/CredSSP)
  - Required by most modern Windows servers
  - Override: `USE_NLA=true` environment variable
  - Note: Default behavior depends on server requirements

- **`-no-rfx`** - Disable RemoteFX codec support
  - Use for testing simpler codecs (RLE, NSCodec)
  - RemoteFX provides better compression but higher CPU usage
  - Override: `RDP_ENABLE_RFX=false` environment variable

- **`-udp`** - Enable UDP transport (experimental)
  - Uses UDP for lossy graphics/audio data
  - Reduces latency over high-latency links
  - **EXPERIMENTAL**: May not work with all servers/networks
  - Override: `RDP_ENABLE_UDP=true` environment variable

#### Audio

- **`-prefer-pcm-audio`** - Prefer PCM audio over compressed formats
  - **Default (flag not set)**: Prioritize AAC → MP3 → PCM (~128-192 kbps bandwidth)
  - **Flag set**: Prioritize PCM → AAC → MP3 (lowest latency, ~1.4 Mbps bandwidth)
  - Use for high-bandwidth LANs where audio quality is critical
  - Override: `RDP_PREFER_PCM_AUDIO=true` environment variable

Example:
```bash
# Run with RFX disabled for testing
./rdp-html5 -port 8080 -no-rfx -log-level debug

# Run with PCM audio for best quality in high-bandwidth LAN
./rdp-html5 -prefer-pcm-audio

# Run with UDP transport (experimental)
./rdp-html5 -udp -log-level debug

# Run with self-signed cert and debug logging
./rdp-html5 -tls-skip-verify -log-level debug

# Run with custom TLS server name
./rdp-html5 -tls-server-name rdp.example.com
```

## Docker Configuration

When running in Docker, pass environment variables with `-e`:

```bash
docker run -d \
  -p 8080:8080 \
  -e LOG_LEVEL=info \
  -e TLS_SKIP_VERIFY=true \
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
      - TLS_SKIP_VERIFY=false
```

# Debugging

## Server-side Logging

The Go backend uses a leveled logging system with four levels: DEBUG, INFO, WARN, ERROR.

### Setting the Log Level

Set via environment variable:

```bash
# Options: debug, info, warn, error (default: info)
export LOG_LEVEL=debug
./bin/rdp-html5
```

Or with Docker:

```bash
docker run -e LOG_LEVEL=debug -p 8080:8080 ghcr.io/rcarmo/rdp-html5:latest
```

### What Each Level Shows

| Level | Description |
|-------|-------------|
| `debug` | Protocol details, byte dumps, timing, internal state |
| `info` | Connection events, capability negotiation, audio format selection |
| `warn` | Recoverable issues, fallbacks, missing optional features |
| `error` | Failures that affect operation |

### Log Output Examples

```
[INFO] Server starting on 0.0.0.0:8080
[INFO] Client connected from 192.168.1.100
[INFO] RDP connection established to 10.0.0.5:3389
[DEBUG] NLA: Received NTLM challenge, flags=0xe2888235
[WARN] Audio: Unsupported format, falling back to PCM
[ERROR] Connection lost: read tcp: connection reset by peer
```

## Client-side Logging

The browser client has a matching leveled logger that synchronizes with the server's log level.

### Automatic Level Sync

When a connection is established, the server sends its log level to the browser. The browser automatically adjusts to match.

### Manual Override

Override the log level in the browser console:

```javascript
// Set specific level
Logger.setLevel('debug');  // debug, info, warn, error

// Convenience methods
Logger.enableDebug();  // Set to debug
Logger.quiet();        // Set to error only

// Check current level
Logger.getLevel();     // Returns current level string
```

### What Gets Logged (Client)

| Level | Logged Information |
|-------|-------------------|
| `debug` | Bitmap updates, cursor cache, WebSocket frames, timing |
| `info` | Connection state, capabilities, channel events |
| `warn` | Performance issues, unsupported features |
| `error` | Connection failures, rendering errors |

### Browser Console Examples

```
[Connection] WebSocket opened, sending credentials
[Capabilities] Server: codecs=nscodec,remotefx, colorDepth=32, desktop=1920x1080
[Config] Log level synced: info
[Clipboard] Initialized clipboard handler
[Audio] Audio context created, sample rate: 48000
```

## Common Issues

### Connection Fails Immediately

1. Check server logs for TLS errors
2. Try with `SKIP_TLS_VALIDATION=true` if using self-signed certs
3. Verify the RDP server is accessible from the backend

### NLA Authentication Fails

1. Try disabling NLA: check "Disable NLA" in the connection dialog
2. Check server logs for NTLM negotiation details at debug level
3. See [KNOWN_ISSUES.md](../KNOWN_ISSUES.md) for NLA limitations

### No Display Updates

1. Enable debug logging on both server and client
2. Check for bitmap decode errors in server logs
3. Verify WebSocket messages are being received in browser Network tab

### Performance Issues

1. Check network latency between browser → backend → RDP server
2. Try reducing color depth (16-bit instead of 32-bit)
3. Check for excessive debug logging (set to `info` or `warn`)

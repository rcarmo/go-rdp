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

The browser client has a matching leveled logger. By default, it only logs warnings and errors to keep the console clean.

### Capabilities Logging

Upon connection, the client logs its capabilities to the console:

```
[RDP Client] Capabilities
  WASM: ✓ loaded
  Codecs: RemoteFX, RLE, NSCodec
  Display: 1920×1080
  Color: 32bpp
```

If WASM is unavailable:
```
[RDP Client] Capabilities
  WASM: ✗ unsupported
  Codecs: JS-Fallback
  Display: 1920×1080
  Color: 16bpp
```

### Manual Override

Override the log level in the browser console:

```javascript
// Set specific level
Logger.setLevel('debug');  // debug, info, warn, error

// Convenience methods
Logger.enableDebug();  // Set to debug
Logger.enableInfo();   // Set to info
Logger.quiet();        // Set to error only
Logger.silent();       // Disable all logging

// Check current level
Logger.getLevel();     // Returns current level string
```

### What Gets Logged (Client)

| Level | Logged Information |
|-------|-------------------|
| `debug` | Bitmap updates, cursor cache, WebSocket frames, timing |
| `info` | Connection state, capabilities, channel events |
| `warn` | Performance issues, unsupported features (default) |
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
2. Try with `TLS_SKIP_VERIFY=true` if using self-signed certs, or `TLS_ALLOW_ANY_SERVER_NAME=true` in lab scenarios
3. Verify the RDP server is accessible from the backend

### NLA Authentication Fails

1. Try disabling NLA: check "Disable NLA" in the connection dialog
2. Check server logs for NTLM negotiation details at debug level
3. NLA has known limitations with some server configurations

### No Display Updates

1. Enable debug logging on both server and client
2. Check for bitmap decode errors in server logs
3. Verify WebSocket messages are being received in browser Network tab

### Performance Issues

1. Check network latency between browser → backend → RDP server
2. Try reducing color depth (16-bit instead of 32-bit)
3. Check for excessive debug logging (set to `info` or `warn`)

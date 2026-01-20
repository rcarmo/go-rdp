# internal/handler

WebSocket handler bridging browser clients to RDP servers.

## Overview

This package implements the HTTP/WebSocket endpoint that:
1. Accepts WebSocket connections from browser clients
2. Creates RDP client connections to remote servers
3. Bidirectionally forwards data between WebSocket and RDP

It acts as the central coordination point between the web frontend and the RDP protocol stack.

## Files

| File | Purpose |
|------|---------|
| `connect.go` | Main HTTP/WebSocket handler implementation |
| `connect_test.go` | Unit tests with mock RDP connections |

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                          Browser Client                              │
│  ┌────────────────┐                          ┌────────────────────┐  │
│  │ Keyboard/Mouse │                          │ Canvas Display     │  │
│  │ Events         │                          │ Updates            │  │
│  └───────┬────────┘                          └─────────▲──────────┘  │
└──────────┼─────────────────────────────────────────────┼─────────────┘
           │                                             │
           ▼                                             │
┌──────────────────────────────────────────────────────────────────────┐
│                         WebSocket Connection                          │
└──────────────────────────────────────────────────────────────────────┘
           │                                             ▲
           ▼                                             │
┌──────────────────────────────────────────────────────────────────────┐
│                       handler.Connect()                               │
│  ┌─────────────────┐                    ┌──────────────────────────┐ │
│  │   wsToRdp()     │                    │      rdpToWs()           │ │
│  │   goroutine     │                    │      blocking loop       │ │
│  │                 │                    │                          │ │
│  │ ws.ReadMessage()│                    │ rdp.GetUpdate()          │ │
│  │       │         │                    │       │                  │ │
│  │       ▼         │                    │       ▼                  │ │
│  │ rdp.SendInput() │                    │ ws.WriteMessage()        │ │
│  └─────────────────┘                    └──────────────────────────┘ │
└──────────────────────────────────────────────────────────────────────┘
           │                                             ▲
           ▼                                             │
┌──────────────────────────────────────────────────────────────────────┐
│                         RDP Client (internal/rdp)                     │
└──────────────────────────────────────────────────────────────────────┘
           │                                             ▲
           ▼                                             │
┌──────────────────────────────────────────────────────────────────────┐
│                         Remote Desktop Server                         │
└──────────────────────────────────────────────────────────────────────┘
```

## HTTP Endpoint

### `GET /connect`

Upgrades HTTP to WebSocket and establishes RDP connection.

**Query Parameters:**

| Parameter | Required | Description |
|-----------|----------|-------------|
| `host` | Yes | RDP server hostname or IP |
| `user` | Yes | Username for authentication |
| `password` | Yes | Password for authentication |
| `width` | No | Desktop width (default: 1024) |
| `height` | No | Desktop height (default: 768) |
| `colorDepth` | No | Color depth (default: 32) |
| `audio` | No | Enable audio redirection (default: false) |
| `disableNLA` | No | Disable NLA authentication (default: false) |

**Example:**
```
ws://localhost:8080/connect?host=192.168.1.100&user=admin&password=secret&width=1920&height=1080
```

## Message Protocol

### Server → Client Messages

#### Capabilities Message (0xFF prefix)
Sent immediately after connection to inform client of server capabilities.

```
[0xFF] [JSON payload]
```

```json
{
  "type": "capabilities",
  "codecs": ["nscodec", "rle"],
  "surfaceCommands": true,
  "colorDepth": 32,
  "desktopSize": "1920x1080",
  "multifragmentSize": 16384,
  "largePointer": true,
  "frameAcknowledge": true
}
```

#### Audio Messages (0xFE prefix)

**PCM Audio Data (0xFE 0x01):**
```
[0xFE] [0x01] [timestamp:2 bytes] [PCM data...]
```

**Audio Format Info (0xFE 0x02):**
```
[0xFE] [0x02] [timestamp:2 bytes] [channels:2] [sampleRate:4] [bitsPerSample:2] [data...]
```

#### Screen Updates (raw binary)
FastPath bitmap updates forwarded directly from RDP server.

### Client → Server Messages

Raw binary input events forwarded directly to RDP server via FastPath.

## Connection Flow

```
1. HTTP Request → /connect with query parameters
2. CORS Validation → Check origin against allowlist
3. WebSocket Upgrade → Upgrade HTTP to WebSocket
4. RDP Connection → Create client, configure TLS/NLA
5. Send Capabilities → Inform browser of server features
6. Start goroutines:
   - wsToRdp: Forward input events
   - rdpToWs: Forward screen updates
7. Wait for disconnect from either side
8. Cleanup: Close RDP connection, WebSocket
```

## CORS Handling

Origins are validated against the `ALLOWED_ORIGINS` configuration:

- **Empty list (dev mode)**: All origins allowed
- **Configured list**: Only matching origins allowed
- **Special cases**: `localhost`, `127.0.0.1` always allowed in dev

## Thread Safety

- **WebSocket writes** are protected by a mutex to prevent interleaving
- **RDP client** operations are thread-safe
- **Goroutine coordination** via done channels for clean shutdown

## Error Handling

| Error | Handling |
|-------|----------|
| CORS rejection | HTTP 403 Forbidden |
| WebSocket upgrade failure | HTTP 400 Bad Request |
| RDP connection failure | WebSocket close with error |
| RDP deactivation | Clean WebSocket close |
| Network timeout | Connection cleanup |

## Related Packages

- `internal/rdp` - RDP client implementation
- `internal/config` - Security and connection settings
- `web/` - Frontend that connects to this handler

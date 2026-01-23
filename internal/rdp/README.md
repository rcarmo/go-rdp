# internal/rdp

Core RDP (Remote Desktop Protocol) client implementation.

## Specification References

- [MS-RDPBCGR](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-rdpbcgr/) - Remote Desktop Protocol: Basic Connectivity and Graphics Remoting
- [MS-RDPBCGR Section 1.3](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-rdpbcgr/023f1e69-cfe8-4ee6-9ee0-7e759fb4e4ee) - Connection Sequence Overview
- [MS-RDPBCGR Section 3](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-rdpbcgr/89d1f7b5-ad39-480d-8a1f-f66a92aee5fb) - Protocol Details

## Overview

This package implements the MS-RDPBCGR (Remote Desktop Protocol: Basic Connectivity and Graphics Remoting) specification. It handles:

- TCP connection establishment with TLS/NLA security
- Protocol negotiation and capability exchange
- Channel multiplexing for various data streams
- Input event transmission (keyboard, mouse)
- Screen update reception via FastPath and slow-path protocols
- Virtual channel support (audio, RemoteApp)

## Files

| File | Purpose |
|------|---------|
| `client.go` | Main Client struct and configuration |
| `types.go` | Type definitions and constants |
| `errors.go` | Error types and error handling |
| **Connection** ||
| `connect.go` | Connection initiation, TLS, protocol negotiation |
| `capabilities_exchange.go` | Capability set exchange |
| `connection_finalization.go` | Final handshake steps |
| **Security** ||
| `tls.go` | TLS connection upgrade |
| `nla.go` | Network Level Authentication (CredSSP) |
| **I/O** ||
| `read.go` | Network read operations |
| `write.go` | Network write operations |
| `get_update.go` | Receive screen updates |
| `send_input_event.go` | Send keyboard/mouse input |
| **Channels** ||
| `virtual_channels.go` | Virtual channel management |
| `audio.go` | Audio redirection channel |
| `rail.go` | RemoteApp integration |
| **Operations** ||
| `close.go` | Connection cleanup |
| `refresh_rect.go` | Request screen refresh |
| `frame_ack.go` | Frame acknowledgment |
| `mcs_interface.go` | MCS layer interface definition |

## Architecture

### Protocol Stack

```
┌─────────────────────────────────────────────────────────────────────┐
│                         RDP Client                                   │
│                       (this package)                                 │
│                                                                      │
│  ┌───────────────┐  ┌───────────────┐  ┌───────────────────────────┐│
│  │ GetUpdate()   │  │ SendInput()   │  │ Virtual Channels          ││
│  │ (receive)     │  │ (transmit)    │  │ (audio, rail, etc.)       ││
│  └───────┬───────┘  └───────┬───────┘  └───────────┬───────────────┘│
│          │                  │                      │                │
│          ▼                  ▼                      ▼                │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │                     FastPath Protocol                           ││
│  │                 (optimized data path)                           ││
│  └─────────────────────────────┬───────────────────────────────────┘│
│                                │                                    │
│  ┌─────────────────────────────▼───────────────────────────────────┐│
│  │                        MCS Layer                                ││
│  │                  (channel multiplexing)                         ││
│  │                  (protocol/mcs)                                 ││
│  └─────────────────────────────┬───────────────────────────────────┘│
│                                │                                    │
│  ┌─────────────────────────────▼───────────────────────────────────┐│
│  │                       X.224 Layer                               ││
│  │                   (connection-oriented)                         ││
│  │                   (protocol/x224)                               ││
│  └─────────────────────────────┬───────────────────────────────────┘│
│                                │                                    │
│  ┌─────────────────────────────▼───────────────────────────────────┐│
│  │                       TPKT Layer                                ││
│  │                      (framing)                                  ││
│  │                      (protocol/tpkt)                            ││
│  └─────────────────────────────┬───────────────────────────────────┘│
│                                │                                    │
│  ┌─────────────────────────────▼───────────────────────────────────┐│
│  │                   TLS (optional)                                ││
│  │                   (crypto/tls)                                  ││
│  └─────────────────────────────┬───────────────────────────────────┘│
└────────────────────────────────┼────────────────────────────────────┘
                                 │
                                 ▼
                          TCP Connection
```

### Connection Flow

```
Connect()
│
├── 1. connectionInitiation()
│       ├── Send ClientConnectionRequest
│       ├── Receive ServerConnectionConfirm
│       ├── Parse negotiation flags
│       └── Upgrade to TLS/NLA if required
│
├── 2. basicSettingsExchange()
│       ├── Create GCC Conference Create Request
│       ├── Send Client Core Data (dimensions, color depth)
│       ├── Send Client Security Data
│       ├── Send Client Network Data (channel list)
│       └── Receive Server Core/Security/Network Data
│
├── 3. channelConnection()
│       ├── ErectDomain()
│       ├── AttachUser()
│       └── JoinChannels() (global, user, virtual channels)
│
├── 4. secureSettingsExchange()
│       └── Send ClientInfo (credentials, flags, working dir)
│
├── 5. licensing()
│       └── Handle license negotiation PDUs
│
├── 6. capabilitiesExchange()
│       ├── Receive ServerDemandActive (server capabilities)
│       └── Send ClientConfirmActive (client capabilities)
│
└── 7. connectionFinalization()
        ├── Send ClientSynchronize
        ├── Send ClientControlCooperate
        ├── Send ClientControlRequestControl
        ├── Send ClientFontList
        └── Wait for server acknowledgments
```

## Key Structs

### Client

```go
type Client struct {
    // Network
    conn       net.Conn
    buffReader *bufio.Reader
    
    // Protocol layers
    tpktLayer  *tpkt.Protocol
    x224Layer  *x224.Protocol
    mcsLayer   MCSLayer
    fastPath   *fastpath.Protocol
    
    // Credentials
    domain, username, password string
    
    // Session configuration
    desktopWidth, desktopHeight uint16
    colorDepth                  uint16
    
    // Capabilities
    serverCapabilitySets *ServerCapabilityInfo
    
    // Channels
    channels     []string
    channelIDMap map[string]uint16
    
    // Security
    selectedProtocol     SelectedProtocol
    skipTLSValidation    bool
    tlsServerName        string
    useNLA               bool
    
    // Audio
    audioHandler *AudioHandler
}
```

### ServerCapabilityInfo

```go
type ServerCapabilityInfo struct {
    BitmapCodecs        []pdu.BitmapCodec
    SurfaceCommands     uint32
    ColorDepth          uint16
    DesktopWidth        uint16
    DesktopHeight       uint16
    MultifragmentSize   uint32
    LargePointerSupport bool
    FrameAcknowledge    bool
}
```

## Usage

### Basic Connection

```go
client := rdp.NewClient()
client.SetCredentials("DOMAIN", "username", "password")
client.SetDesktopSize(1920, 1080)
client.SetColorDepth(32)

err := client.Connect("192.168.1.100:3389")
if err != nil {
    log.Fatal(err)
}
defer client.Close()
```

### With TLS and NLA

```go
client := rdp.NewClient()
client.SetCredentials("", "admin", "password")
client.SetTLSConfig(true, "server.example.com") // skip validation, server name
client.SetNLA(true)

err := client.Connect("server.example.com:3389")
```

### Receiving Updates

```go
for {
    update, err := client.GetUpdate()
    if err != nil {
        if err == rdp.ErrDeactivateAll {
            break // Session ended
        }
        log.Error(err)
        continue
    }
    
    // Forward update.Data to display
    handleScreenUpdate(update.Data)
}
```

### Sending Input

```go
// Raw FastPath input event bytes
err := client.SendInputEvent(inputData)
```

## Protocol Features

### FastPath vs Slow-Path

| Feature | FastPath | Slow-Path |
|---------|----------|-----------|
| Header size | 2-3 bytes | ~20 bytes |
| Encryption | Optional | Per-spec |
| Use case | Screen updates | Control messages |
| Detection | Header byte bits 0-1 | `0x03` marker |

### Virtual Channels

| Channel | Purpose |
|---------|---------|
| Global | Primary control channel |
| User | Per-user data channel |
| rdpsnd | Audio output |
| rdpdr | Device redirection |
| rail | RemoteApp |
| cliprdr | Clipboard |

### Capability Sets

The client advertises and negotiates:
- Bitmap capabilities (color depth, compression)
- Input capabilities (keyboard, mouse, touch)
- Sound capabilities (audio formats)
- Surface commands (codec support)
- Pointer capabilities (cursor handling)
- Order capabilities (drawing primitives)

## Testing

```bash
# Run all tests
go test ./internal/rdp/...

# With coverage
go test -cover ./internal/rdp/...

# Verbose
go test -v ./internal/rdp/...
```

## Related Packages

- `internal/protocol/*` - Protocol layer implementations
- `internal/auth` - NLA/CredSSP authentication
- `internal/codec` - Bitmap decompression
- `internal/handler` - WebSocket bridge

## References

- **MS-RDPBCGR** - RDP Basic Connectivity and Graphics Remoting
- **MS-RDPEGDI** - Graphics Device Interface Extensions
- **ITU T.125** - MCS Protocol
- **ITU T.124** - GCC Protocol

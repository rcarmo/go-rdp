# RDP HTML5 Client Architecture

> Comprehensive Technical Documentation  
> Last Updated: January 20, 2026

## Table of Contents

1. [Overview](#overview)
2. [System Architecture](#system-architecture)
3. [Data Flow](#data-flow)
4. [Backend Components](#backend-components)
5. [Protocol Stack](#protocol-stack)
6. [Frontend Components](#frontend-components)
7. [Authentication System](#authentication-system)
8. [Audio Subsystem](#audio-subsystem)
9. [Configuration Management](#configuration-management)
10. [Security Considerations](#security-considerations)
11. [Performance Optimizations](#performance-optimizations)
12. [Deployment](#deployment)

---

## Overview

This project implements a browser-based Remote Desktop Protocol (RDP) client using:

- **Go** backend server acting as an RDP-to-WebSocket bridge
- **WebAssembly** module for high-performance bitmap decompression
- **HTML5 Canvas** for rendering remote desktop display
- **Web Audio API** for audio redirection

### Key Capabilities

| Feature | Status | Description |
|---------|--------|-------------|
| Display | ✅ | FastPath and slow-path bitmap updates |
| Input | ✅ | Mouse and keyboard via FastPath |
| Authentication | ✅ | NLA (CredSSP/NTLMv2), TLS, standard RDP |
| Color Depths | ✅ | 8, 15, 16, 24, 32-bit |
| Codecs | ✅ | RLE, NSCodec, Planar |
| Audio | ✅ | RDPSND channel with PCM output |
| Clipboard | ✅ | Text copy/paste |

---

## System Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Browser                                         │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                        HTML5 Client                                  │    │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────────────────┐ │    │
│  │  │  Canvas  │  │  Input   │  │  Audio   │  │    WebSocket API     │ │    │
│  │  │ Renderer │  │ Handler  │  │ Playback │  │                      │ │    │
│  │  └────┬─────┘  └────┬─────┘  └────┬─────┘  └──────────┬───────────┘ │    │
│  │       │              │              │                   │            │    │
│  │  ┌────▼──────────────▼──────────────▼───────────────────▼──────────┐ │    │
│  │  │                    WASM Module (TinyGo)                         │ │    │
│  │  │  RLE Decompress │ NSCodec │ Color Convert │ Bitmap Flip         │ │    │
│  │  └─────────────────────────────────────────────────────────────────┘ │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      │ WebSocket (Binary)
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Go Backend Server                                  │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                     WebSocket Handler                                │    │
│  │                  (internal/handler/connect.go)                       │    │
│  └──────────────────────────────┬──────────────────────────────────────┘    │
│                                  │                                           │
│  ┌──────────────────────────────▼──────────────────────────────────────┐    │
│  │                        RDP Client Core                               │    │
│  │                      (internal/rdp/client.go)                        │    │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌────────────┐  │    │
│  │  │   Connect   │  │  GetUpdate  │  │ SendInput   │  │   Audio    │  │    │
│  │  │   Manager   │  │   Reader    │  │   Writer    │  │  Handler   │  │    │
│  │  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘  └─────┬──────┘  │    │
│  └─────────┼────────────────┼────────────────┼───────────────┼─────────┘    │
│            │                │                │               │              │
│  ┌─────────▼────────────────▼────────────────▼───────────────▼─────────┐    │
│  │                      Protocol Stack                                  │    │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐             │    │
│  │  │ FastPath │  │   PDU    │  │   MCS    │  │   GCC    │             │    │
│  │  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘             │    │
│  │       └──────────────┴────────────┴─────────────┘                    │    │
│  │                              │                                       │    │
│  │  ┌───────────────────────────▼───────────────────────────────────┐  │    │
│  │  │                     X.224 / TPKT                               │  │    │
│  │  └───────────────────────────────────────────────────────────────┘  │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      │ TCP :3389
                                      ▼
                        ┌─────────────────────────┐
                        │      RDP Server         │
                        │  (Windows / XRDP)       │
                        └─────────────────────────┘
```

---

## Data Flow

### Connection Establishment

```
Browser                    Go Server                   RDP Server
   │                           │                            │
   │──GET /connect?params────▶│                            │
   │                           │──TCP Connect────────────▶│
   │                           │◀─────────────────────────│
   │                           │                            │
   │                           │──X.224 Connection Req───▶│
   │                           │◀─X.224 Connection Conf───│
   │                           │                            │
   │                           │══TLS Handshake══════════▶│
   │                           │◀════════════════════════│
   │                           │                            │
   │                           │──NLA (if enabled)───────▶│
   │                           │◀─────────────────────────│
   │                           │                            │
   │                           │──MCS Connect-Initial────▶│
   │                           │◀─MCS Connect-Response────│
   │                           │                            │
   │                           │──Channel Join────────────▶│
   │                           │◀─────────────────────────│
   │                           │                            │
   │                           │──Client Info─────────────▶│
   │                           │◀─License Response────────│
   │                           │                            │
   │                           │──Capability Confirm──────▶│
   │                           │◀─────────────────────────│
   │                           │                            │
   │◀─WebSocket Upgrade───────│                            │
   │                           │                            │
   │◀─Capabilities JSON───────│                            │
   │                           │                            │
```

### Runtime Data Flow

```
┌────────────────────────────────────────────────────────────────────────┐
│                     Incoming (RDP → Browser)                            │
├────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│   RDP Server                                                            │
│       │                                                                 │
│       ▼                                                                 │
│   ┌───────────────────────────────────────────────────────────────┐    │
│   │ FastPath/X.224 Update                                          │    │
│   │ • Bitmap data (compressed)                                     │    │
│   │ • Pointer updates                                              │    │
│   │ • Palette changes                                              │    │
│   └───────────────────────────────────────────────────────────────┘    │
│       │                                                                 │
│       ▼                                                                 │
│   GetUpdate() [internal/rdp/get_update.go]                             │
│       │                                                                 │
│       ▼                                                                 │
│   rdpToWs() [internal/handler/connect.go]                              │
│       │                                                                 │
│       ▼                                                                 │
│   WebSocket.Send() → Binary message                                    │
│       │                                                                 │
│       ▼                                                                 │
│   handleMessage() [web/js/src/client.js]                               │
│       │                                                                 │
│       ├──▶ parseBitmapUpdate()                                         │
│       │         │                                                       │
│       │         ▼                                                       │
│       │    goRLE.processBitmap() [WASM]                                │
│       │         │                                                       │
│       │         ▼                                                       │
│       │    ctx.putImageData() [Canvas]                                 │
│       │                                                                 │
│       └──▶ parsePointerUpdate() → Update cursor                        │
│                                                                         │
└────────────────────────────────────────────────────────────────────────┘

┌────────────────────────────────────────────────────────────────────────┐
│                     Outgoing (Browser → RDP)                            │
├────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│   User Input (mouse move, click, keypress)                             │
│       │                                                                 │
│       ▼                                                                 │
│   InputMixin [web/js/src/mixins/input.js]                              │
│       │                                                                 │
│       ▼                                                                 │
│   ┌───────────────────────────────────────────────────────────────┐    │
│   │ FastPath Input PDU                                             │    │
│   │ • Mouse: flags (2) + X (2) + Y (2)                             │    │
│   │ • Keyboard: flags (1) + keyCode (2)                            │    │
│   │ • Sync: flags (2)                                              │    │
│   └───────────────────────────────────────────────────────────────┘    │
│       │                                                                 │
│       ▼                                                                 │
│   WebSocket.Send() → Binary message                                    │
│       │                                                                 │
│       ▼                                                                 │
│   wsToRdp() [internal/handler/connect.go]                              │
│       │                                                                 │
│       ▼                                                                 │
│   SendInputEvent() [internal/rdp/send_input_event.go]                  │
│       │                                                                 │
│       ▼                                                                 │
│   FastPath.Send() → RDP Server                                         │
│                                                                         │
└────────────────────────────────────────────────────────────────────────┘
```

---

## Backend Components

### Directory Structure

```
cmd/
└── server/
    └── main.go              # Entry point, HTTP server setup

internal/
├── auth/                    # NTLMv2/CredSSP authentication
│   ├── credssp.go          # TSRequest encoding/decoding
│   ├── ntlm.go             # NTLM message generation
│   └── md4.go              # MD4 hash implementation
│
├── codec/                   # Bitmap compression codecs
│   ├── decoder.go          # Main codec dispatcher
│   ├── rle8.go             # 8-bit RLE decompression
│   ├── rle16.go            # 16-bit RLE decompression
│   ├── rle24.go            # 24-bit RLE decompression
│   ├── rle32.go            # 32-bit RLE decompression
│   ├── nscodec.go          # NSCodec (AYCoCg) decompression
│   ├── planar.go           # Planar codec
│   └── bitmap.go           # Color conversion, flip
│
├── config/                  # Configuration management
│   └── config.go           # Load from env/flags
│
├── handler/                 # WebSocket handler
│   └── connect.go          # WS→RDP bridge logic
│
├── logging/                 # Leveled logging
│   └── logging.go          # Debug/Info/Warn/Error
│
├── protocol/                # RDP protocol stack
│   ├── audio/              # RDPSND channel
│   ├── encoding/           # BER/PER encoding
│   ├── fastpath/           # FastPath I/O
│   ├── gcc/                # T.124 GCC
│   ├── mcs/                # T.125 MCS
│   ├── pdu/                # PDU definitions
│   ├── tpkt/               # TPKT framing
│   └── x224/               # X.224 connection
│
└── rdp/                     # Core RDP client
    ├── client.go           # Client struct, initialization
    ├── connect.go          # Connection sequence
    ├── nla.go              # NLA authentication
    ├── audio.go            # Audio handler
    ├── get_update.go       # Receive updates
    ├── send_input_event.go # Send input
    └── capabilities.go     # Capability negotiation
```

### RDP Client Core

The `Client` struct in `internal/rdp/client.go` manages the RDP session:

```go
type Client struct {
    // Network
    conn       net.Conn
    buffReader *bufio.Reader
    
    // Protocol layers
    tpktLayer    *tpkt.Protocol
    x224Layer    *x224.Protocol
    mcsLayer     MCSLayer
    fastPath     *fastpath.FastPath
    
    // Session state
    userID            uint16
    channelIDMap      map[string]uint16
    selectedProtocol  pdu.NegotiationProtocol
    
    // Configuration
    desktopWidth  int
    desktopHeight int
    colorDepth    int
    useNLA        bool
    
    // Handlers
    audioHandler  *AudioHandler
}
```

### Connection Sequence

The connection follows MS-RDPBCGR specification:

```go
func (c *Client) Connect() error {
    // Phase 1: Connection Initiation
    c.connectionInitiation()  // X.224 + protocol negotiation
    
    // Phase 2: Basic Settings Exchange  
    c.basicSettingsExchange() // MCS Connect + GCC
    
    // Phase 3: Channel Connection
    c.channelConnection()     // Erect domain, attach user, join channels
    
    // Phase 4: Secure Settings Exchange
    c.secureSettingsExchange() // Client Info PDU
    
    // Phase 5: Licensing
    c.licensing()             // License validation
    
    // Phase 6: Capabilities Exchange
    c.capabilitiesExchange()  // Demand/Confirm Active PDU
    
    // Phase 7: Connection Finalization
    c.connectionFinalization() // Synchronize, control, font list
    
    return nil
}
```

---

## Protocol Stack

### Layer Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    Application Layer                             │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │ PDU (Protocol Data Units)                                    │ │
│  │ • Share Control Header                                       │ │
│  │ • Share Data Header                                          │ │
│  │ • Capability Sets                                            │ │
│  │ • Input/Output PDUs                                          │ │
│  └─────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────▼───────────────────────────────────┐
│                    Presentation Layer                            │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │ FastPath (Optimized channel)                                 │ │
│  │ • 1-byte header + data                                       │ │
│  │ • Bitmap updates (0x01)                                      │ │
│  │ • Pointer updates (0x04)                                     │ │
│  │ • Input events                                               │ │
│  └─────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────▼───────────────────────────────────┐
│                    Session Layer                                 │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │ MCS (Multipoint Communication Service) - T.125               │ │
│  │ • Connect Initial/Response                                   │ │
│  │ • Erect Domain Request                                       │ │
│  │ • Attach User Request/Confirm                                │ │
│  │ • Channel Join Request/Confirm                               │ │
│  │ • Send Data Request/Indication                               │ │
│  └─────────────────────────────────────────────────────────────┘ │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │ GCC (Generic Conference Control) - T.124                     │ │
│  │ • Conference Create Request/Response                         │ │
│  │ • Client/Server Core Data                                    │ │
│  │ • Client/Server Security Data                                │ │
│  │ • Client/Server Network Data                                 │ │
│  └─────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────▼───────────────────────────────────┐
│                    Transport Layer                               │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │ X.224 (ISO 8073)                                             │ │
│  │ • Connection Request (0xE0)                                  │ │
│  │ • Connection Confirm (0xD0)                                  │ │
│  │ • Data (0xF0)                                                │ │
│  │ • Negotiation Request/Response                               │ │
│  └─────────────────────────────────────────────────────────────┘ │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │ TPKT (RFC 1006)                                              │ │
│  │ • Version (1 byte) = 0x03                                    │ │
│  │ • Reserved (1 byte) = 0x00                                   │ │
│  │ • Length (2 bytes, big-endian)                               │ │
│  └─────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────▼───────────────────────────────────┐
│                    Network Layer                                 │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │ TLS 1.0-1.2 (when SSL/Hybrid negotiated)                     │ │
│  └─────────────────────────────────────────────────────────────┘ │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │ TCP (port 3389)                                              │ │
│  └─────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

### FastPath Format

FastPath provides optimized update delivery:

```
FastPath Update PDU:
┌─────────────────────────────────────────────────────────┐
│ fpUpdateHeader (1 byte)                                  │
│   bits 0-3: updateCode                                   │
│     0x00 = orders                                        │
│     0x01 = bitmap                                        │
│     0x02 = palette                                       │
│     0x03 = synchronize                                   │
│     0x04 = surface commands                              │
│     0x05 = pointer (hidden)                              │
│     0x06 = pointer (default)                             │
│     0x09 = pointer (new)                                 │
│   bits 4-5: fragmentation                                │
│   bits 6-7: compression                                  │
├─────────────────────────────────────────────────────────┤
│ size (1-2 bytes, variable)                               │
├─────────────────────────────────────────────────────────┤
│ updateData (variable)                                    │
└─────────────────────────────────────────────────────────┘
```

---

## Frontend Components

### Directory Structure

```
web/
├── index.html              # Main HTML page
├── js/
│   └── src/
│       ├── client.js       # Main client class
│       └── mixins/
│           ├── session.js  # Connection management
│           ├── input.js    # Mouse/keyboard handling
│           ├── graphics.js # Canvas rendering
│           ├── clipboard.js# Copy/paste
│           ├── ui.js       # UI state
│           └── audio.js    # Audio playback
└── wasm/
    └── main.go             # TinyGo WASM module
```

### Client Architecture

The JavaScript client uses a mixin-based architecture:

```javascript
class RDPClient {
    constructor(canvas) {
        this.canvas = canvas;
        this.ctx = canvas.getContext('2d');
        this.ws = null;
        
        // Apply mixins
        Object.assign(this, SessionMixin);
        Object.assign(this, InputMixin);
        Object.assign(this, GraphicsMixin);
        Object.assign(this, ClipboardMixin);
        Object.assign(this, UIMixin);
        Object.assign(this, AudioMixin);
    }
    
    connect(host, user, password, options) {
        const params = new URLSearchParams({
            host, user, password,
            width: options.width,
            height: options.height,
            colorDepth: options.colorDepth,
            audio: options.audio
        });
        
        this.ws = new WebSocket(`ws://${location.host}/connect?${params}`);
        this.ws.binaryType = 'arraybuffer';
        this.ws.onmessage = (e) => this.handleMessage(e.data);
    }
}
```

### WASM Module

The TinyGo WASM module exposes codec functions to JavaScript:

```go
// web/wasm/main.go
func main() {
    js.Global().Set("goRLE", map[string]interface{}{
        "decompressRLE16": js.FuncOf(decompressRLE16),
        "flipVertical":    js.FuncOf(flipVertical),
        "rgb565toRGBA":    js.FuncOf(rgb565toRGBA),
        "bgr24toRGBA":     js.FuncOf(bgr24toRGBA),
        "bgra32toRGBA":    js.FuncOf(bgra32toRGBA),
        "processBitmap":   js.FuncOf(processBitmap),
        "decodeNSCodec":   js.FuncOf(decodeNSCodec),
        "setPalette":      js.FuncOf(setPalette),
        "setRFXQuant":     js.FuncOf(setRFXQuant),
        "decodeRFXTile":   js.FuncOf(decodeRFXTile),
    })
    
    select {} // Keep alive
}
```

### JavaScript Fallback Codecs

When WASM is unavailable (older browsers, disabled WebAssembly), the client falls back to pure JavaScript implementations:

```javascript
// web/js/src/codec-fallback.js
const FallbackCodec = {
    rgb565ToRGBA(src, dst) { /* ... */ },
    rgb555ToRGBA(src, dst) { /* ... */ },
    bgr24ToRGBA(src, dst)  { /* ... */ },
    bgra32ToRGBA(src, dst) { /* ... */ },
    palette8ToRGBA(src, dst) { /* ... */ },
    flipVertical(data, width, height, bytesPerPixel) { /* ... */ },
    processBitmap(src, width, height, bpp, isCompressed, dst) { /* ... */ }
};
```

**Fallback Limitations:**
- Compressed formats (RLE, NSCodec, RemoteFX) not supported
- Recommend 16-bit color depth for best performance
- ~2-5x slower than WASM for color conversion

### Capabilities Detection

Upon connection, the client logs its capabilities:

```
[RDP Client] Capabilities
  WASM: ✓ loaded
  Codecs: RemoteFX, RLE, NSCodec
  Display: 1920×1080
  Color: 32bpp
```

### Bitmap Processing Pipeline

```
┌─────────────────────────────────────────────────────────────────┐
│                  Bitmap Processing Pipeline                      │
└─────────────────────────────────────────────────────────────────┘

Compressed Bitmap from Server
         │
         ▼
┌─────────────────────┐
│ Detect Compression  │
│ • RLE (most common) │
│ • NSCodec           │
│ • Planar            │
│ • Uncompressed      │
└─────────┬───────────┘
          │
          ▼
┌─────────────────────┐     ┌─────────────────────┐
│  goRLE.processBitmap│────▶│ 1. Decompress       │
│  (WASM)             │     │ 2. Flip vertical    │
│                     │     │ 3. Convert to RGBA  │
└─────────┬───────────┘     └─────────────────────┘
          │
          ▼
┌─────────────────────┐
│ ImageData object    │
│ (width × height × 4)│
└─────────┬───────────┘
          │
          ▼
┌─────────────────────┐
│ ctx.putImageData()  │
│ at (destLeft,       │
│     destTop)        │
└─────────────────────┘
```

---

## Authentication System

### Authentication Methods

| Method | Protocol | Security Level |
|--------|----------|----------------|
| Standard RDP | None | Low (legacy) |
| TLS | SSL/TLS | Medium |
| NLA (CredSSP) | TLS + NTLMv2 | High |

### NLA Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                    NLA Authentication Flow                       │
└─────────────────────────────────────────────────────────────────┘

Client                                              Server
   │                                                    │
   │──────────── TLS Handshake ────────────────────────▶│
   │◀───────────────────────────────────────────────────│
   │                                                    │
   │  TSRequest (version=6, negoTokens=[NTLM Negotiate])│
   │─────────────────────────────────────────────────▶│
   │                                                    │
   │  TSRequest (version=6, negoTokens=[NTLM Challenge])│
   │◀─────────────────────────────────────────────────│
   │                                                    │
   │  TSRequest (negoTokens=[NTLM Auth], pubKeyAuth)   │
   │─────────────────────────────────────────────────▶│
   │                                                    │
   │  TSRequest (pubKeyAuth verified)                   │
   │◀─────────────────────────────────────────────────│
   │                                                    │
   │  TSRequest (authInfo=[encrypted credentials])      │
   │─────────────────────────────────────────────────▶│
   │                                                    │
   │  ─────────── Continue RDP Connection ────────────▶│
   │                                                    │
```

### NTLM Message Structure

```go
// internal/auth/ntlm.go

// Negotiate Message (Type 1)
type NegotiateMessage struct {
    Signature     [8]byte  // "NTLMSSP\0"
    MessageType   uint32   // 0x00000001
    NegotiateFlags uint32  // NTLM capabilities
    DomainName    Field    // Optional
    Workstation   Field    // Optional
    Version       [8]byte  // OS version
}

// Challenge Message (Type 2) - from server
type ChallengeMessage struct {
    Signature      [8]byte
    MessageType    uint32   // 0x00000002
    TargetName     Field
    NegotiateFlags uint32
    ServerChallenge [8]byte // 8-byte nonce
    Reserved       [8]byte
    TargetInfo     Field    // AV_PAIR list
}

// Authenticate Message (Type 3)
type AuthenticateMessage struct {
    Signature         [8]byte
    MessageType       uint32  // 0x00000003
    LmChallengeResponse Field
    NtChallengeResponse Field
    DomainName        Field
    UserName          Field
    Workstation       Field
    EncryptedRandomSession Field
    NegotiateFlags    uint32
    MIC               [16]byte
}
```

---

## Audio Subsystem

### Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     Audio Subsystem                              │
└─────────────────────────────────────────────────────────────────┘

RDP Server (RDPSND Virtual Channel)
         │
         │ SNDC_FORMATS (server audio formats)
         ▼
┌─────────────────────────────────────────────────────────────────┐
│ AudioHandler (internal/rdp/audio.go)                            │
│                                                                  │
│  handleServerFormats()                                           │
│    • Parse available formats                                     │
│    • Select PCM format (preferred: 16-bit, 44.1kHz, stereo)     │
│    • Send SNDC_FORMATS response                                  │
│                                                                  │
│  handleTraining()                                                │
│    • Respond with SNDC_TRAINING_CONFIRM                          │
│                                                                  │
│  handleWave() / handleWave2()                                    │
│    • Extract audio data                                          │
│    • Call callback with PCM data                                 │
│    • Send SNDC_WAVE_CONFIRM                                      │
└─────────────────────────────────────────────────────────────────┘
         │
         │ AudioCallback(data, format, timestamp)
         ▼
┌─────────────────────────────────────────────────────────────────┐
│ WebSocket Handler (internal/handler/connect.go)                  │
│                                                                  │
│  sendAudioData()                                                 │
│    • Build audio message: 0xFE marker + format + data            │
│    • Send over WebSocket                                         │
└─────────────────────────────────────────────────────────────────┘
         │
         │ WebSocket binary message
         ▼
┌─────────────────────────────────────────────────────────────────┐
│ AudioMixin (web/js/src/mixins/audio.js)                          │
│                                                                  │
│  initAudio()                                                     │
│    • Create AudioContext                                         │
│    • Set sample rate from format                                 │
│                                                                  │
│  handleAudioMessage()                                            │
│    • Parse format info                                           │
│    • Decode PCM to Float32                                       │
│    • Schedule playback via AudioBufferSourceNode                 │
└─────────────────────────────────────────────────────────────────┘
```

### Supported Audio Formats

| Format Tag | Name | Support |
|------------|------|---------|
| 0x0001 | WAVE_FORMAT_PCM | ✅ Full |
| 0x0011 | WAVE_FORMAT_ADPCM | ❌ |
| 0x0055 | WAVE_FORMAT_MPEGLAYER3 | ❌ |
| 0xFFFE | WAVE_FORMAT_EXTENSIBLE | ❌ |

---

## Configuration Management

### Configuration Structure

```go
// internal/config/config.go

type Config struct {
    Server   ServerConfig
    RDP      RDPConfig
    Security SecurityConfig
    Logging  LoggingConfig
}

type ServerConfig struct {
    Host         string        // Listen address
    Port         string        // Listen port
    ReadTimeout  time.Duration
    WriteTimeout time.Duration
    IdleTimeout  time.Duration
}

type SecurityConfig struct {
    AllowedOrigins    []string
    MaxConnections    int
    EnableTLS         bool
    SkipTLSValidation bool
    TLSServerName     string
    UseNLA            bool
    EnableRateLimit   bool
    RateLimitPerMinute int
}

type LoggingConfig struct {
    Level  string  // debug, info, warn, error
    Format string
    File   string
}
```

### Configuration Sources (Priority Order)

1. **Command-line flags** (highest priority)
2. **Environment variables**
3. **Default values** (lowest priority)

| Setting | Flag | Environment Variable | Default |
|---------|------|---------------------|---------|
| Host | `-host` | `SERVER_HOST` | `0.0.0.0` |
| Port | `-port` | `SERVER_PORT` | `8080` |
| Log Level | `-log-level` | `LOG_LEVEL` | `info` |
| Skip TLS | `-skip-tls-verify` | `SKIP_TLS_VALIDATION` | `false` |
| Allow any SNI | `-tls-allow-any-server-name` | `TLS_ALLOW_ANY_SERVER_NAME` | `false` (Docker default: `true`) |
| Use NLA | `-nla` | `USE_NLA` | `true` |

---

## Security Considerations

### Transport Security

```
┌─────────────────────────────────────────────────────────────────┐
│                    Security Layers                               │
└─────────────────────────────────────────────────────────────────┘

Browser ←───────────────────→ Go Server ←─────────────────→ RDP Server
         WSS (optional)                   TLS 1.0-1.2

┌─────────────────────────────────────────────────────────────────┐
│ Browser → Server                                                 │
│ • WebSocket over HTTPS (recommended for production)              │
│ • Origin validation (ALLOWED_ORIGINS)                            │
│ • Rate limiting (configurable)                                   │
│ • Security headers (CSP, X-Frame-Options, etc.)                  │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│ Server → RDP                                                     │
│ • TLS encryption (SSL or Hybrid protocol)                        │
│ • NLA authentication (CredSSP + NTLMv2)                          │
│ • Certificate validation (configurable)                          │
└─────────────────────────────────────────────────────────────────┘
```

### Security Headers

The server applies these security headers:

```go
w.Header().Set("X-Content-Type-Options", "nosniff")
w.Header().Set("X-Frame-Options", "DENY")
w.Header().Set("X-XSS-Protection", "1; mode=block")
w.Header().Set("Strict-Transport-Security", "max-age=31536000")
w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
w.Header().Set("Content-Security-Policy", 
    "default-src 'self'; script-src 'self' 'unsafe-inline' 'wasm-unsafe-eval'; ...")
```

### Credential Handling

As of v1.0.0, credentials are transmitted securely via WebSocket message:

1. Browser opens WebSocket with non-sensitive parameters only (width, height, colorDepth)
2. After WebSocket opens, browser sends credentials as JSON message:
   ```json
   {"type": "credentials", "host": "...", "user": "...", "password": "..."}
   ```
3. Server receives credentials via WebSocket (not visible in URL/logs)
4. Credentials are never stored on server (stateless bridge)
5. Encrypted via NLA/TLS before reaching RDP server

This approach prevents credentials from appearing in:
- Browser history
- Server access logs
- Proxy logs
- Referrer headers

---

## Performance Optimizations

### WASM Acceleration

| Operation | JavaScript | WASM | Speedup |
|-----------|------------|------|---------|
| RLE16 decompress | ~50ms | ~2ms | 25x |
| Vertical flip | ~20ms | ~1ms | 20x |
| Color conversion | ~30ms | ~1ms | 30x |

### Buffer Management

```go
// internal/rdp/client.go
const (
    readBufferSize  = 64 * 1024  // 64KB read buffer
    writeBufferSize = 16 * 1024  // 16KB write buffer
)

// Buffered reader reduces syscall overhead
c.buffReader = bufio.NewReaderSize(c.conn, readBufferSize)
```

### FastPath Optimization

FastPath reduces per-update overhead:

| Mode | Header Size | Use Case |
|------|-------------|----------|
| Slow Path (X.224) | ~20 bytes | Connection setup |
| FastPath | 2-3 bytes | Runtime updates |

---

## Deployment

### Docker Container

```dockerfile
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o rdp-html5 ./cmd/server

FROM alpine:latest
COPY --from=builder /app/rdp-html5 /app/
COPY --from=builder /app/web /app/web
ENV SKIP_TLS_VALIDATION=true
EXPOSE 8080
CMD ["/app/rdp-html5"]
```

### Environment Variables

```bash
# Server configuration
SERVER_HOST=0.0.0.0
SERVER_PORT=8080

# Security
SKIP_TLS_VALIDATION=true   # For self-signed RDP certs
USE_NLA=true               # Enable NLA authentication
ALLOWED_ORIGINS=https://example.com

# Logging
LOG_LEVEL=info             # debug, info, warn, error
```

### Health Check

```bash
curl http://localhost:8080/  # Should return HTML
```

---

## Appendix: Message Format Reference

### WebSocket Message Types

| Prefix | Type | Direction | Description |
|--------|------|-----------|-------------|
| `0x00-0x0F` | FastPath Update | Server→Client | Bitmap/pointer updates |
| `0xFE` | Audio Data | Server→Client | PCM audio samples |
| `0xFF` | JSON Metadata | Server→Client | Capabilities, errors |
| (none) | Input Event | Client→Server | Mouse/keyboard |

### Capability Message

```json
{
  "type": "capabilities",
  "codecs": ["nscodec", "remotefx"],
  "surfaceCommands": true,
  "colorDepth": 32,
  "desktopSize": "1920x1080",
  "multifragmentSize": 65536,
  "largePointer": true,
  "frameAcknowledge": true
}
```

---

## References

- [MS-RDPBCGR](https://docs.microsoft.com/en-us/openspecs/windows_protocols/ms-rdpbcgr) - RDP Basic Connectivity
- [MS-RDPEGDI](https://docs.microsoft.com/en-us/openspecs/windows_protocols/ms-rdpegdi) - Graphics Device Interface
- [MS-RDPEFS](https://docs.microsoft.com/en-us/openspecs/windows_protocols/ms-rdpefs) - File System Virtual Channel
- [MS-RDPEA](https://docs.microsoft.com/en-us/openspecs/windows_protocols/ms-rdpea) - Audio Output Virtual Channel
- [MS-CSSP](https://docs.microsoft.com/en-us/openspecs/windows_protocols/ms-cssp) - Credential Security Support Provider
- [RFC 1006](https://tools.ietf.org/html/rfc1006) - TPKT
- [ITU-T T.124](https://www.itu.int/rec/T-REC-T.124) - GCC
- [ITU-T T.125](https://www.itu.int/rec/T-REC-T.125) - MCS

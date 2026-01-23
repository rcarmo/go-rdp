# internal/protocol/fastpath

RDP FastPath protocol for optimized screen updates and input events.

## Specification References

- [MS-RDPBCGR Section 2.2.9](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-rdpbcgr/68b5ee54-d0d5-4d65-8d81-e1c4025f7597) - Server Fast-Path Update PDU
- [MS-RDPBCGR Section 2.2.8](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-rdpbcgr/d364d31f-6b6a-4105-a9d0-5e48047a9a68) - Client Fast-Path Input Event PDU
- [MS-RDPBCGR Section 5.3.8](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-rdpbcgr/0cb7d420-ea0d-4e4b-8a87-b8fb79de99ce) - Fast-Path PDU Security

## Overview

FastPath is an optimized data path in RDP that bypasses the full protocol stack for performance-critical operations:
- **Screen updates** - Bitmap data from server to client
- **Input events** - Keyboard and mouse from client to server

FastPath reduces header overhead from ~20 bytes (slow-path) to 2-3 bytes.

## Files

| File | Purpose |
|------|---------|
| `protocol.go` | Main Protocol struct and configuration |
| `send.go` | Sending FastPath PDUs |
| `receive.go` | Receiving and parsing FastPath PDUs |
| `update_events.go` | Screen update event types |
| `surface_commands.go` | Surface command parsing |
| `fastpath_test.go`, `send_test.go` | Unit tests |

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                        RDP Client                                    │
│                    (internal/rdp)                                    │
└───────────────────────────────┬─────────────────────────────────────┘
                                │
        ┌───────────────────────┴───────────────────────┐
        │                                               │
        ▼                                               ▼
┌───────────────────────┐                 ┌───────────────────────────┐
│   FastPath Send       │                 │   FastPath Receive        │
│                       │                 │                           │
│  ┌─────────────────┐  │                 │  ┌─────────────────────┐  │
│  │ InputEventPDU   │  │                 │  │ Update PDU          │  │
│  │ - Keyboard      │  │                 │  │ - Bitmap            │  │
│  │ - Mouse         │  │                 │  │ - Palette           │  │
│  │ - Sync          │  │                 │  │ - Pointer           │  │
│  └────────┬────────┘  │                 │  │ - Surface Commands  │  │
│           │           │                 │  └──────────┬──────────┘  │
│           ▼           │                 │             │             │
│  ┌─────────────────┐  │                 │             ▼             │
│  │ TPKT Frame      │  │                 │  ┌─────────────────────┐  │
│  └─────────────────┘  │                 │  │ BitmapData          │  │
└───────────┬───────────┘                 │  │ - Compressed        │  │
            │                             │  │ - Uncompressed      │  │
            ▼                             │  └─────────────────────┘  │
     ┌──────────────┐                     └───────────────────────────┘
     │    Server    │
     └──────────────┘
```

## FastPath Header

### Output (Server → Client)

```
Byte 0: Action + Flags
├── Bits 0-1: Action (0 = FastPath)
├── Bits 2-3: Reserved
├── Bits 4-5: Flags (encryption, compressed)
└── Bits 6-7: Number of events (0-3)

Byte 1: Length (low byte)
Byte 2: Length (high byte, if needed)

[Encryption signature: 8 bytes if encrypted]
[PDU data...]
```

### Input (Client → Server)

```
Byte 0: Action + Event Count
├── Bits 0-1: Action (0 = FastPath)
├── Bits 2-5: Number of events
└── Bits 6-7: Flags

Byte 1-2: Length
[Events...]
```

## Key Structs

### Protocol

```go
type Protocol struct {
    rw            io.ReadWriter
    updatePDUData []byte  // Reusable buffer
}
```

### Update (Screen Data)

```go
type Update struct {
    UpdateType     uint8
    Fragmentation  uint8
    Compression    uint8
    CompressionLen uint16
    Data           []byte
}
```

### BitmapData

```go
type BitmapData struct {
    DestLeft       uint16
    DestTop        uint16
    DestRight      uint16
    DestBottom     uint16
    Width          uint16
    Height         uint16
    BitsPerPixel   uint16
    Flags          uint16  // Compressed, NoSkip, etc.
    BitmapDataLength uint16
    BitmapData     []byte
}
```

### InputEventPDU

```go
type InputEventPDU struct {
    NumEvents uint8
    Data      []byte  // Serialized input events
}
```

## Update Types

| Type | Value | Description |
|------|-------|-------------|
| `FASTPATH_UPDATETYPE_ORDERS` | 0x0 | Drawing orders |
| `FASTPATH_UPDATETYPE_BITMAP` | 0x1 | Bitmap data |
| `FASTPATH_UPDATETYPE_PALETTE` | 0x2 | Color palette |
| `FASTPATH_UPDATETYPE_SYNCHRONIZE` | 0x3 | Sync marker |
| `FASTPATH_UPDATETYPE_SURFCMDS` | 0x4 | Surface commands |
| `FASTPATH_UPDATETYPE_PTR_NULL` | 0x5 | Hide pointer |
| `FASTPATH_UPDATETYPE_PTR_DEFAULT` | 0x6 | Default pointer |
| `FASTPATH_UPDATETYPE_PTR_POSITION` | 0x8 | Pointer position |
| `FASTPATH_UPDATETYPE_COLOR` | 0x9 | Color pointer |
| `FASTPATH_UPDATETYPE_CACHED` | 0xA | Cached pointer |
| `FASTPATH_UPDATETYPE_POINTER` | 0xB | New pointer |
| `FASTPATH_UPDATETYPE_LARGE_POINTER` | 0xC | Large pointer |

## Surface Commands

Surface commands enable advanced codec-based rendering:

### SetSurfaceBits

```go
type SetSurfaceBitsCommand struct {
    DestLeft     uint16
    DestTop      uint16
    DestRight    uint16
    DestBottom   uint16
    BitsPerPixel uint8
    CodecID      uint8   // NSCodec, RemoteFX, etc.
    Width        uint16
    Height       uint16
    BitmapData   []byte
}
```

### FrameMarker

```go
type FrameMarkerCommand struct {
    FrameAction uint16  // Begin/End
    FrameID     uint32
}
```

## Usage

### Receiving Updates

```go
fp := fastpath.NewProtocol(conn)

for {
    updates, err := fp.ReceiveUpdatePDU()
    if err != nil {
        break
    }
    
    for _, update := range updates {
        switch update.UpdateType {
        case FASTPATH_UPDATETYPE_BITMAP:
            handleBitmap(update.Data)
        case FASTPATH_UPDATETYPE_SURFCMDS:
            handleSurfaceCommands(update.Data)
        }
    }
}
```

### Sending Input

```go
inputPDU := fastpath.NewInputEventPDU(inputData)
err := fp.SendInputEventPDU(inputPDU)
```

## Bitmap Compression

Bitmap updates may be compressed using:

| Method | Description |
|--------|-------------|
| RLE | Interleaved Run-Length Encoding |
| Planar | RDP6 Planar codec |
| NSCodec | Network Screen Codec |

The `Flags` field in `BitmapData` indicates compression:

```go
const (
    BITMAP_COMPRESSION = 0x0001
    NO_BITMAP_SKIP     = 0x0002
)
```

## Detection

FastPath packets are identified by the first byte:

```go
// FastPath: bits 0-1 are 00 (action = 0)
// Slow-path: byte is 0x03 (X.224 data)
if data[0]&0x03 == 0 {
    // FastPath
} else if data[0] == 0x03 {
    // Slow-path (X.224)
}
```

## Performance Benefits

| Aspect | FastPath | Slow-Path |
|--------|----------|-----------|
| Header size | 2-3 bytes | ~20 bytes |
| Encryption | Optional | Per-spec |
| Channel routing | None | MCS overhead |
| Best for | Screen data | Control messages |

## References

- **MS-RDPBCGR** Section 2.2.9 - Fast-Path Output Update PDU
- **MS-RDPBCGR** Section 2.2.8 - Fast-Path Input Event PDU
- **MS-RDPEGDI** - Surface Commands Extension

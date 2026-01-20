# internal/protocol/pdu

RDP Protocol Data Unit definitions.

## Overview

This package contains all RDP PDU (Protocol Data Unit) definitions used throughout the connection lifecycle. It implements structures and serialization for:

- Connection initiation and settings exchange
- Capability negotiation
- Input events
- Licensing
- Data transfer
- Error handling

This is the largest protocol package with 38+ files defining 60+ PDU types.

## Files

### Connection Phase

| File | Purpose |
|------|---------|
| `connection_initiation.go` | X.224 connection negotiation |
| `basic_settings_exchange.go` | Client/server core data |
| `secure_settings_exchange.go` | Client info PDU |
| `connection_finalization.go` | Synchronize, control, font list |
| `licensing.go` | License negotiation PDUs |

### Capabilities

| File | Capability Type |
|------|-----------------|
| `cap_bitmap.go` | Bitmap rendering |
| `cap_general.go` | General capabilities |
| `cap_input.go` | Input (keyboard, mouse) |
| `cap_order.go` | Drawing order support |
| `cap_pointer.go` | Pointer (cursor) handling |
| `cap_sound.go` | Audio capabilities |
| `cap_surface.go` | Surface commands, codecs |
| `cap_brush.go` | Brush patterns |
| `cap_cache.go` | Bitmap caching |
| `cap_colorcache.go` | Color table caching |
| `cap_control.go` | Control capabilities |
| `cap_share.go` | Session sharing |
| `cap_window.go` | Window activation |
| `cap_draw_gdiplus.go` | GDI+ drawing |
| `cap_draw_ninegrid.go` | Nine-grid scaling |
| `cap_offscreen_cache.go` | Off-screen caching |
| `cap_font.go` | Font support |
| `cap_glyph_cache.go` | Glyph caching |
| `cap_virtual_channel.go` | Virtual channel caps |
| `caps.go` | Capability set container |

### Data Transfer

| File | Purpose |
|------|---------|
| `data.go` | Share data PDU wrapper |
| `input_events.go` | Keyboard/mouse events |
| `error_info.go` | Error info PDU |
| `frame_ack.go` | Frame acknowledgment |

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                        RDP Session                                   │
└───────────────────────────────┬─────────────────────────────────────┘
                                │
┌───────────────────────────────▼─────────────────────────────────────┐
│                          PDU Layer                                   │
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │                    Share Control Header                         ││
│  │         (PDU type, version, source, length)                     ││
│  └─────────────────────────────────────────────────────────────────┘│
│                                │                                     │
│      ┌─────────────┬──────────┴──────────┬─────────────┐           │
│      │             │                     │             │           │
│      ▼             ▼                     ▼             ▼           │
│  ┌────────┐  ┌──────────┐  ┌──────────────────┐  ┌──────────┐     │
│  │Demand  │  │ Confirm  │  │     Data PDU     │  │ Deactivate│     │
│  │Active  │  │ Active   │  │                  │  │   All    │     │
│  │        │  │          │  │ ┌──────────────┐ │  │          │     │
│  │Caps    │  │ Caps     │  │ │Share Data Hdr│ │  │          │     │
│  │Exchange│  │ Exchange │  │ └──────┬───────┘ │  │          │     │
│  └────────┘  └──────────┘  │        │         │  └──────────┘     │
│                            │  ┌─────▼─────┐   │                   │
│                            │  │ PDU Type2 │   │                   │
│                            │  │ (payload) │   │                   │
│                            │  └───────────┘   │                   │
│                            └──────────────────┘                   │
└─────────────────────────────────────────────────────────────────────┘
```

## Key Structures

### Share Control Header

```go
type ShareControlHeader struct {
    TotalLength uint16
    PDUType     uint16
    PDUSource   uint16
}
```

### PDU Types

| Type | Value | Description |
|------|-------|-------------|
| PDUTYPE_DEMANDACTIVEPDU | 0x0011 | Server demands capabilities |
| PDUTYPE_CONFIRMACTIVEPDU | 0x0013 | Client confirms capabilities |
| PDUTYPE_DEACTIVATEALLPDU | 0x0016 | Session deactivation |
| PDUTYPE_DATAPDU | 0x0017 | Data transfer |

### Share Data Header

```go
type ShareDataHeader struct {
    ShareID            uint32
    Pad1               uint8
    StreamID           uint8
    UncompressedLength uint16
    PDUType2           uint8
    CompressedType     uint8
    CompressedLength   uint16
}
```

### PDU Type2 (Data PDU subtypes)

| Type | Value | Description |
|------|-------|-------------|
| PDUTYPE2_UPDATE | 0x02 | Graphics update |
| PDUTYPE2_CONTROL | 0x14 | Control actions |
| PDUTYPE2_POINTER | 0x1B | Pointer update |
| PDUTYPE2_INPUT | 0x1C | Input events |
| PDUTYPE2_SYNCHRONIZE | 0x1F | Synchronization |
| PDUTYPE2_FONTLIST | 0x27 | Font list |
| PDUTYPE2_FONTMAP | 0x28 | Font mapping |
| PDUTYPE2_SET_ERROR_INFO | 0x2F | Error info |
| PDUTYPE2_SAVE_SESSION_INFO | 0x26 | Session info |
| PDUTYPE2_FRAME_ACK | 0x38 | Frame acknowledgment |

## Capability Exchange

### Server Demand Active

Server sends supported capabilities:

```go
type ServerDemandActive struct {
    ShareID            uint32
    LengthSourceDesc   uint16
    LengthCombinedCaps uint16
    SourceDescriptor   []byte
    NumberCapabilities uint16
    Pad2Octets         uint16
    CapabilitySets     []CapabilitySet
}
```

### Client Confirm Active

Client responds with negotiated capabilities:

```go
type ClientConfirmActive struct {
    ShareID            uint32
    OriginatorID       uint16
    LengthSourceDesc   uint16
    LengthCombinedCaps uint16
    SourceDescriptor   []byte
    NumberCapabilities uint16
    Pad2Octets         uint16
    CapabilitySets     []CapabilitySet
}
```

### Capability Set

```go
type CapabilitySet struct {
    CapabilitySetType   uint16
    LengthCapability    uint16
    // Type-specific capability data
    BitmapCapabilitySet        *BitmapCapabilitySet
    GeneralCapabilitySet       *GeneralCapabilitySet
    OrderCapabilitySet         *OrderCapabilitySet
    // ... 20+ capability types
}
```

## Input Events

### Input PDU

```go
type InputPDU struct {
    NumEvents  uint16
    Pad2Octets uint16
    Events     []InputEvent
}
```

### Input Event Types

| Type | Value | Description |
|------|-------|-------------|
| INPUT_EVENT_SYNC | 0x0000 | Synchronize toggle keys |
| INPUT_EVENT_SCANCODE | 0x0004 | Keyboard scancode |
| INPUT_EVENT_UNICODE | 0x0005 | Unicode character |
| INPUT_EVENT_MOUSE | 0x8001 | Mouse event |
| INPUT_EVENT_MOUSEX | 0x8002 | Extended mouse |

### Keyboard Event

```go
type KeyboardEvent struct {
    EventTime    uint32
    MessageType  uint16  // INPUT_EVENT_SCANCODE
    KeyboardFlags uint16
    KeyCode      uint16
    Pad2Octets   uint16
}
```

### Mouse Event

```go
type MouseEvent struct {
    EventTime    uint32
    MessageType  uint16  // INPUT_EVENT_MOUSE
    PointerFlags uint16
    XPos         uint16
    YPos         uint16
}
```

## Client Info PDU

Sent during secure settings exchange:

```go
type ClientInfo struct {
    CodePage             uint32
    Flags                uint32
    Domain               string
    UserName             string
    Password             string
    AlternateShell       string
    WorkingDir           string
    ClientAddressFamily  uint16
    ClientAddress        string
    ClientDir            string
    // ... timezone, session info
}
```

## Usage

### Parsing Server Demand Active

```go
var demandActive pdu.ServerDemandActive
err := demandActive.Deserialize(reader)

for _, cap := range demandActive.CapabilitySets {
    switch cap.CapabilitySetType {
    case CapabilitySetTypeBitmap:
        handleBitmapCaps(cap.BitmapCapabilitySet)
    case CapabilitySetTypeSound:
        handleSoundCaps(cap.SoundCapabilitySet)
    }
}
```

### Building Client Confirm Active

```go
caps := []pdu.CapabilitySet{
    pdu.NewGeneralCapabilitySet(),
    pdu.NewBitmapCapabilitySet(32, 1920, 1080),
    pdu.NewOrderCapabilitySet(),
    pdu.NewInputCapabilitySet(),
    pdu.NewSoundCapabilitySet(),
    // ...
}

confirmActive := pdu.ClientConfirmActive{
    ShareID:        demandActive.ShareID,
    CapabilitySets: caps,
}
data := confirmActive.Serialize()
```

### Creating Input Events

```go
// Keyboard event
keyEvent := pdu.NewKeyboardEvent(scancode, pressed)
inputPDU := pdu.NewInputPDU([]InputEvent{keyEvent})

// Mouse event
mouseEvent := pdu.NewMouseEvent(x, y, buttons)
inputPDU := pdu.NewInputPDU([]InputEvent{mouseEvent})
```

## References

- **MS-RDPBCGR** Section 2.2.1 - Connection Sequence
- **MS-RDPBCGR** Section 2.2.7 - Capability Sets
- **MS-RDPBCGR** Section 2.2.8 - Input PDUs

# internal/protocol/x224

ISO 8073 X.224 connection-oriented transport layer.

## Overview

This package implements the X.224 (ISO 8073 Class 0) protocol layer for RDP connection negotiation. X.224 provides:

- Connection request/confirm handshake
- Protocol negotiation (RDP, TLS, NLA)
- Data PDU framing

X.224 sits between TPKT framing and the MCS layer.

## Files

| File | Purpose |
|------|---------|
| `protocol.go` | Main Protocol struct and interface |
| `connect.go` | Connection request/confirm PDUs |
| `send.go` | Sending X.224 PDUs |
| `receive.go` | Receiving X.224 PDUs |
| `errors.go` | Error definitions |
| `connect_test.go` | Unit tests |

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                         MCS Layer                                    │
│                      (protocol/mcs)                                  │
└───────────────────────────────┬─────────────────────────────────────┘
                                │
┌───────────────────────────────▼─────────────────────────────────────┐
│                        X.224 Layer                                   │
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │                   Connection Phase                              ││
│  │                                                                  ││
│  │  ┌────────────────────┐      ┌────────────────────────────────┐ ││
│  │  │ Connection Request │ ───► │ Connection Confirm             │ ││
│  │  │ - Requested protos │      │ - Selected protocol            │ ││
│  │  │ - Cookie (optional)│      │ - Server flags                 │ ││
│  │  └────────────────────┘      └────────────────────────────────┘ ││
│  └─────────────────────────────────────────────────────────────────┘│
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │                      Data Phase                                 ││
│  │                                                                  ││
│  │  ┌────────────────────┐                                         ││
│  │  │    Data PDU        │ ◄───► MCS/Application data              ││
│  │  │ - EOT flag         │                                         ││
│  │  │ - Payload          │                                         ││
│  │  └────────────────────┘                                         ││
│  └─────────────────────────────────────────────────────────────────┘│
└───────────────────────────────┬─────────────────────────────────────┘
                                │
┌───────────────────────────────▼─────────────────────────────────────┐
│                        TPKT Layer                                    │
│                     (protocol/tpkt)                                  │
└─────────────────────────────────────────────────────────────────────┘
```

## Connection Request PDU

### Structure

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|     Length    |   CR Code     |    DST-REF    |    SRC-REF    |
|    Indicator  |    (0xE0)     |   (0x0000)    |   (0x0000)    |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|    Class      |  [Cookie...]  |    [RDP Neg Request...]       |
|    Options    |               |                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

### RDP Negotiation Request

```go
type NegotiationRequest struct {
    Type          uint8   // TYPE_RDP_NEG_REQ (0x01)
    Flags         uint8
    Length        uint16  // 8
    RequestedProto uint32  // Protocol flags
}
```

### Requested Protocols

| Flag | Value | Description |
|------|-------|-------------|
| PROTOCOL_RDP | 0x00000000 | Standard RDP |
| PROTOCOL_SSL | 0x00000001 | TLS 1.0 security |
| PROTOCOL_HYBRID | 0x00000002 | CredSSP + TLS (NLA) |
| PROTOCOL_RDSTLS | 0x00000004 | RDSTLS security |
| PROTOCOL_HYBRID_EX | 0x00000008 | CredSSP + Early User Auth |

## Connection Confirm PDU

### Structure

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|     Length    |   CC Code     |    DST-REF    |    SRC-REF    |
|    Indicator  |    (0xD0)     |   (0x0000)    |   (0x0000)    |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|    Class      |    [RDP Neg Response/Failure...]              |
|    Options    |                                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

### RDP Negotiation Response

```go
type NegotiationResponse struct {
    Type          uint8   // TYPE_RDP_NEG_RSP (0x02)
    Flags         uint8   // Server flags
    Length        uint16  // 8
    SelectedProto uint32  // Selected protocol
}
```

### Server Flags

| Flag | Value | Description |
|------|-------|-------------|
| EXTENDED_CLIENT_DATA_SUPPORTED | 0x01 | Extended GCC data |
| DYNVC_GFX_PROTOCOL_SUPPORTED | 0x02 | Graphics pipeline |
| NEGRSP_FLAG_RESERVED | 0x04 | Reserved |
| RESTRICTED_ADMIN_MODE_SUPPORTED | 0x08 | Restricted admin |
| REDIRECTED_AUTHENTICATION_MODE_SUPPORTED | 0x10 | Auth redirect |

## Data PDU

### Structure

```
+----------------+----------------+
| Length (1 byte)| Code (0xF0)    |
+----------------+----------------+
| EOT (0x80)     | Data...        |
+----------------+----------------+
```

```go
type Data struct {
    Header []byte  // Length, code (0xF0), EOT (0x80)
    Data   []byte  // Payload
}
```

## Protocol Interface

```go
type Protocol struct {
    tpkt *tpkt.Protocol
}

func NewProtocol(tpkt *tpkt.Protocol) *Protocol

// Connection phase
func (p *Protocol) SendConnectionRequest(protocols uint32, cookie string) error
func (p *Protocol) ReceiveConnectionConfirm() (*ConnectionConfirm, error)

// Data phase
func (p *Protocol) Send(data []byte) error
func (p *Protocol) Receive() (io.Reader, error)
```

## Connection Flow

```
Client                                  Server
   │                                      │
   │  X.224 Connection Request            │
   │  - Requested: SSL | HYBRID           │
   │  ────────────────────────────────►   │
   │                                      │
   │  X.224 Connection Confirm            │
   │  - Selected: HYBRID                  │
   │  - Flags: EXTENDED_CLIENT_DATA       │
   │  ◄────────────────────────────────   │
   │                                      │
   │  [TLS Handshake]                     │
   │  ◄────────────────────────────────►  │
   │                                      │
   │  [CredSSP/NLA if HYBRID]             │
   │  ◄────────────────────────────────►  │
   │                                      │
   │  X.224 Data PDUs (MCS)               │
   │  ◄────────────────────────────────►  │
```

## Usage

### Connection Negotiation

```go
x224 := x224.NewProtocol(tpktLayer)

// Request TLS or NLA
err := x224.SendConnectionRequest(
    x224.PROTOCOL_SSL | x224.PROTOCOL_HYBRID,
    "",  // optional cookie
)

// Get server response
confirm, err := x224.ReceiveConnectionConfirm()

switch confirm.SelectedProtocol {
case x224.PROTOCOL_SSL:
    // Upgrade to TLS
case x224.PROTOCOL_HYBRID:
    // Upgrade to TLS + CredSSP
}
```

### Data Transfer

```go
// Send MCS data
err := x224.Send(mcsData)

// Receive MCS data
reader, err := x224.Receive()
```

## Error Handling

### Negotiation Failure

```go
type NegotiationFailure struct {
    Type       uint8   // TYPE_RDP_NEG_FAILURE (0x03)
    Flags      uint8
    Length     uint16
    FailureCode uint32
}
```

| Failure Code | Description |
|--------------|-------------|
| 0x01 | SSL required by server |
| 0x02 | SSL not allowed by server |
| 0x03 | SSL cert not on server |
| 0x04 | Inconsistent flags |
| 0x05 | Hybrid required by server |
| 0x06 | SSL with user auth required |

## References

- **ISO 8073** - Connection-oriented transport protocol
- **MS-RDPBCGR** Section 2.2.1.1 - Client X.224 Connection Request
- **MS-RDPBCGR** Section 2.2.1.2 - Server X.224 Connection Confirm

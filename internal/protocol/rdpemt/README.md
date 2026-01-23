# internal/protocol/rdpemt

Multitransport Protocol Extension per MS-RDPEMT.

## Overview

This package implements the RDP Multitransport Extension, which allows:
- **Transport negotiation** - Switch from TCP to UDP for better performance
- **Tunnel management** - Create and manage additional transport tunnels
- **Security binding** - Bind UDP tunnels to the main TCP connection

RDPEMT enables UDP transport (MS-RDPEUDP) for RDP connections on lossy/high-latency networks.

## Specification Reference

- **MS-RDPEMT** - Remote Desktop Protocol: Multitransport Extension
  - https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-rdpemt/

## Files

| File | Purpose |
|------|---------|
| `rdpemt.go` | Protocol implementation and PDU definitions |
| `rdpemt_test.go` | Unit tests |

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                         RDP Client                                   │
└───────────┬─────────────────────────────────────────┬───────────────┘
            │ Main RDP Connection                     │ UDP Transport
            │ (TCP, TLS)                              │ (RDPEUDP, DTLS)
┌───────────▼─────────────────────────────────────────▼───────────────┐
│                     Multitransport Layer                             │
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │               Tunnel Header (4+ bytes)                          ││
│  │  ┌─────────┬─────────┬──────────────┬─────────────────────────┐ ││
│  │  │ Action  │ Flags   │ HeaderLength │ SubHeaders (optional)   │ ││
│  │  │ (4 bits)│ (4 bits)│ (8 bits)     │                         │ ││
│  │  └─────────┴─────────┴──────────────┴─────────────────────────┘ ││
│  └─────────────────────────────────────────────────────────────────┘│
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │                    Tunnel Data PDU                              ││
│  │           (CreateRequest, CreateResponse, Data)                 ││
│  └─────────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────────┘
```

## PDU Types

### Tunnel Actions

| Value | Action | Description |
|-------|--------|-------------|
| 0x00 | CYCLECHECK_ROUNDTRIP | Initiate auto-detect round trip |
| 0x01 | SET_ROUNDTRIP_MEASURE | Set round trip measure |
| 0x02 | TUNNEL_CREATE_REQUEST | Request tunnel creation |
| 0x03 | TUNNEL_CREATE_RESPONSE | Respond to tunnel creation |
| 0x04 | TUNNEL_DATA | Tunneled data |

### Tunnel Flags

| Value | Flag | Description |
|-------|------|-------------|
| 0x01 | SECURITY | Security parameters present |
| 0x02 | AUTODETECT | Auto-detect present |
| 0x04 | LOSSY | Lossy transport mode |

### Header Structure

```go
type TunnelHeader struct {
    // Byte 0: (Flags << 4) | (Action & 0x0F)
    Action       uint8   // Lower 4 bits
    Flags        uint8   // Upper 4 bits
    Reserved     uint8   // Reserved byte
    HeaderLength uint8   // Total header length (minimum 4)
    SubHeaders   []byte  // Optional sub-headers if HeaderLength > 4
}
```

### Create Request PDU

```go
type TunnelCreateRequestPDU struct {
    Header       TunnelHeader
    RequestID    uint32   // Request identifier
    Reserved     uint32   // Reserved
    SecurityCookie [16]byte  // Security binding cookie
}
```

### Create Response PDU

```go
type TunnelCreateResponsePDU struct {
    Header    TunnelHeader
    RequestID uint32   // Matching request ID
    HResult   uint32   // Result code
}
```

### Data PDU

```go
type TunnelDataPDU struct {
    Header     TunnelHeader
    HigherLayer []byte  // RDP PDU data
}
```

## Protocol Flow

### TCP-Based Multitransport Initiation

```
Client                              Server
   │                                   │
   │  (Main RDP connection over TCP)   │
   │                                   │
   │  Initiate Multitransport Request  │
   │  (via MCS channel)                │
   │  ◄────────────────────────────    │
   │                                   │
   │  (Client opens UDP socket)        │
   │                                   │
```

### UDP Tunnel Setup (via RDPEUDP)

```
Client                              Server
   │                                   │
   │  (UDP SYN via RDPEUDP)            │
   │  ────────────────────────────►    │
   │                                   │
   │  (UDP SYN+ACK)                    │
   │  ◄────────────────────────────    │
   │                                   │
   │  (UDP ACK - connection est.)      │
   │  ────────────────────────────►    │
   │                                   │
   │  TUNNEL_CREATE_REQUEST            │
   │  ────────────────────────────►    │
   │                                   │
   │  TUNNEL_CREATE_RESPONSE           │
   │  ◄────────────────────────────    │
   │                                   │
   │  (DTLS/TLS handshake)             │
   │                                   │
   │  TUNNEL_DATA (RDP PDUs)           │
   │  ◄────────────────────────────►   │
```

## Key Constants

### Protocol Versions

| Value | Version | Description |
|-------|---------|-------------|
| 0x0001 | V1 | Original multitransport |
| 0x0002 | V2 | Enhanced with auto-detect |

### HResult Codes

| Value | Code | Description |
|-------|------|-------------|
| 0x00000000 | S_OK | Success |
| 0x80004005 | E_FAIL | General failure |
| 0x80070005 | E_ACCESSDENIED | Access denied |
| 0x80004004 | E_ABORT | Operation aborted |

## Usage

### Parsing Tunnel Header

```go
data := receivedUDPPacket()

header, err := rdpemt.ParseTunnelHeader(data)
if err != nil {
    return err
}

switch header.Action {
case rdpemt.ActionTunnelCreateRequest:
    // Handle tunnel creation
case rdpemt.ActionTunnelData:
    // Process tunneled RDP data
}
```

### Creating Tunnel Response

```go
response := &rdpemt.TunnelCreateResponsePDU{
    Header: rdpemt.TunnelHeader{
        Action:       rdpemt.ActionTunnelCreateResponse,
        Flags:        0,
        HeaderLength: 4,
    },
    RequestID: request.RequestID,
    HResult:   rdpemt.S_OK,
}

data, err := response.Serialize()
SendToClient(data)
```

### Processing Tunnel Data

```go
func handleTunnelData(pdu *rdpemt.TunnelDataPDU) {
    // Extract the encapsulated RDP PDU
    rdpData := pdu.HigherLayer
    
    // Process as normal RDP PDU
    processRDPPDU(rdpData)
}
```

## Header Encoding

The tunnel header uses nibble encoding (4-bit fields):

```
Byte 0: [ Flags (4 bits) | Action (4 bits) ]
Byte 1: Reserved
Byte 2: Reserved  
Byte 3: HeaderLength (minimum 4)
Byte 4+: SubHeaders (if HeaderLength > 4)
```

**Important**: The original spec wording was ambiguous. Microsoft test suites confirm:
- Action occupies the **lower** 4 bits
- Flags occupy the **upper** 4 bits

## Test Coverage

Current coverage: **95.0%**

```bash
go test -cover ./internal/protocol/rdpemt/...
```

## References

- **MS-RDPEMT** - Multitransport Extension
  - https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-rdpemt/
- **MS-RDPEUDP** - UDP Transport Extension (transport layer)
- **MS-RDPBCGR** Section 2.2.15 - Initiate Multitransport Request

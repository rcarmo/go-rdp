# internal/protocol/rdpeudp

UDP Transport Extension per MS-RDPEUDP.

## Overview

This package implements the RDP UDP Transport Extension, which provides:
- **Reliable transport (RDP-UDP-R)** - Guaranteed delivery with retransmission
- **Lossy transport (RDP-UDP-L)** - Best-effort delivery for real-time data
- **Congestion control** - Network-aware rate adaptation
- **Forward Error Correction** - Recover from random packet loss

RDPEUDP improves RDP performance on high-latency and lossy networks (WANs, wireless).

## Specification Reference

- **MS-RDPEUDP** - Remote Desktop Protocol: UDP Transport Extension
  - https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-rdpeudp/
- **MS-RDPEUDP2** - UDP Transport Extension Version 2
  - https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-rdpeudp2/

## Files

| File | Purpose |
|------|---------|
| `rdpeudp.go` | Packet structures and serialization |
| `rdpeudp_test.go` | Unit tests |

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                    RDP Multitransport (MS-RDPEMT)                    │
└───────────────────────────────┬─────────────────────────────────────┘
                                │
┌───────────────────────────────▼─────────────────────────────────────┐
│                       DTLS/TLS Layer                                 │
│              (Reliable: TLS, Lossy: DTLS)                           │
└───────────────────────────────┬─────────────────────────────────────┘
                                │
┌───────────────────────────────▼─────────────────────────────────────┐
│                      RDPEUDP Protocol                                │
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │                    FEC Header (8 bytes)                         ││
│  │  snSourceAck(4) + uReceiveWindowSize(2) + uFlags(2)             ││
│  └─────────────────────────────────────────────────────────────────┘│
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │                   Optional Payloads                             ││
│  │  SynData | AckVector | SourcePayload | FECPayload | SynDataEx   ││
│  └─────────────────────────────────────────────────────────────────┘│
└───────────────────────────────┬─────────────────────────────────────┘
                                │
┌───────────────────────────────▼─────────────────────────────────────┐
│                            UDP Socket                                │
│                         (Port 3389)                                  │
└─────────────────────────────────────────────────────────────────────┘
```

## Packet Structure

### FEC Header (Always Present)

```go
type FECHeader struct {
    SnSourceAck              uint32  // Sequence number being acknowledged
    SourceAckReceiveWindowSize uint16  // Receive window size
    Flags                    uint16  // RDPUDP_FLAG_* values
}
```

### Flags

| Flag | Value | Description |
|------|-------|-------------|
| RDPUDP_FLAG_SYN | 0x0001 | Synchronization packet |
| RDPUDP_FLAG_FIN | 0x0002 | Finish (connection close) |
| RDPUDP_FLAG_ACK | 0x0004 | ACK_VECTOR present |
| RDPUDP_FLAG_DAT | 0x0008 | Source/FEC payload present |
| RDPUDP_FLAG_FEC | 0x0010 | FEC_PAYLOAD present |
| RDPUDP_FLAG_CN | 0x0020 | Congestion notification |
| RDPUDP_FLAG_CWR | 0x0040 | Congestion window reset |
| RDPUDP_FLAG_AOA | 0x0100 | Ack of Acks present |
| RDPUDP_FLAG_SYNLOSSY | 0x0200 | Lossy transport requested |
| RDPUDP_FLAG_ACKDELAYED | 0x0400 | ACK was delayed |
| RDPUDP_FLAG_CORRELATIONID | 0x0800 | Correlation ID present |
| RDPUDP_FLAG_SYNEX | 0x1000 | Extended SYN data present |

### SYN Data Payload

```go
type SynData struct {
    SnInitialSequenceNumber uint32  // Initial sequence number
    UpstreamMTU             uint16  // Client-to-server MTU (1132-1232)
    DownstreamMTU           uint16  // Server-to-client MTU (1132-1232)
}
```

### Extended SYN Data (Version 2+)

```go
type SynDataEx struct {
    Flags      uint16    // SYNEX flags
    Version    uint16    // Protocol version
    CookieHash [32]byte  // SHA-256 hash (Version 3 only)
}
```

### ACK Vector

```go
type AckVector struct {
    AckVectorSize     uint16   // Size in bytes (max 2048)
    AckVectorElements []uint8  // RLE-encoded states
}
```

Each ACK vector element encodes:
- **State (2 bits)**: 0=received, 3=not received
- **Length (6 bits)**: Run length (0-63)

### Source Payload Header

```go
type SourcePayloadHeader struct {
    SnCoded       uint32  // Coded packet sequence number
    SnSourceStart uint32  // Source packet sequence number
}
```

## Protocol Flow

### Connection Establishment

```
Client                              Server
   │                                   │
   │  SYN (snSourceAck=-1, MTU, ISN)   │
   │  ────────────────────────────►    │
   │                                   │
   │  SYN+ACK (ISN, ACK=client_ISN)    │
   │  ◄────────────────────────────    │
   │                                   │
   │  ACK (ACK=server_ISN)             │
   │  ────────────────────────────►    │
   │                                   │
   │  (Connection established)         │
```

### Data Transfer with Selective ACK

```
Client                              Server
   │                                   │
   │  DATA (seq=100)                   │
   │  ────────────────────────────►    │
   │                                   │
   │  DATA (seq=101) [lost]            │
   │  ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ►     │
   │                                   │
   │  DATA (seq=102)                   │
   │  ────────────────────────────►    │
   │                                   │
   │  ACK (snSourceAck=102, vector)    │
   │  ◄────────────────────────────    │
   │  (vector shows seq=101 missing)   │
   │                                   │
   │  DATA (seq=101) [retransmit]      │
   │  ────────────────────────────►    │
```

### Congestion Control

```
Client                              Server
   │                                   │
   │  (Detects packet loss)            │
   │                                   │
   │  ACK + RDPUDP_FLAG_CN             │
   │  ◄────────────────────────────    │
   │                                   │
   │  (Client reduces send rate)       │
   │                                   │
   │  DATA + RDPUDP_FLAG_CWR           │
   │  ────────────────────────────►    │
   │                                   │
   │  (Server stops sending CN)        │
```

## Protocol Versions

| Value | Version | Features |
|-------|---------|----------|
| 0x0001 | Version 1 | Basic reliable/lossy |
| 0x0002 | Version 2 | Extended SYN, faster retransmit |
| 0x0101 | Version 3 | Cookie hash for security binding |

## Timers

| Timer | Value | Description |
|-------|-------|-------------|
| Retransmit (V1) | 500ms | Minimum retransmit timeout |
| Retransmit (V2) | 300ms | Minimum retransmit timeout |
| Keepalive | 65s | Connection timeout |
| Delayed ACK | 200ms | Max delay before ACK |

## Usage

### Creating a SYN Packet

```go
// Create connection request
packet := rdpeudp.NewSYNPacket(
    generateRandomSeq(),  // Initial sequence number
    1232,                 // Upstream MTU
    1232,                 // Downstream MTU
    64,                   // Receive window
)

// Add version 2 negotiation
packet.Header.Flags |= rdpeudp.FlagSYNEX
packet.SynDataEx = &rdpeudp.SynDataEx{
    Flags:   rdpeudp.SynExFlagVersionInfoValid,
    Version: rdpeudp.ProtocolVersion2,
}

data, _ := packet.Serialize()
// Pad to MTU (1232 bytes) before sending
```

### Parsing Received Packet

```go
packet := &rdpeudp.Packet{}
err := packet.Deserialize(receivedData)
if err != nil {
    return err
}

if packet.Header.HasFlag(rdpeudp.FlagSYN) {
    // Handle SYN/SYN+ACK
    if packet.Header.HasFlag(rdpeudp.FlagACK) {
        // SYN+ACK from server
        handleSynAck(packet)
    } else {
        // SYN from client (server-side)
        handleSyn(packet)
    }
} else if packet.Header.HasFlag(rdpeudp.FlagDAT) {
    // Data packet
    handleData(packet)
} else if packet.Header.HasFlag(rdpeudp.FlagACK) {
    // Pure ACK
    handleAck(packet)
}
```

### Building ACK Vector

```go
// Create ACK vector showing selective acknowledgment
vector := &rdpeudp.AckVector{
    AckVectorElements: []uint8{
        (0 << 6) | 5,  // 6 packets received (state=0, length=5)
        (3 << 6) | 0,  // 1 packet missing (state=3, length=0)
        (0 << 6) | 2,  // 3 packets received
    },
}
vector.AckVectorSize = uint16(len(vector.AckVectorElements))
```

## Constraints

Per MS-RDPEUDP specification:

- **MTU range**: 1132-1232 bytes
- **SYN padding**: Must be zero-padded to MTU size
- **ACK vector max**: 2048 bytes
- **Max retransmits**: 3-5 attempts before connection close
- **SYN phase**: ACK_VECTOR not present even with FlagACK set

## Test Coverage

Current coverage: **78.4%**

```bash
go test -cover ./internal/protocol/rdpeudp/...
```

## References

- **MS-RDPEUDP** - UDP Transport Extension
  - https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-rdpeudp/
- **MS-RDPEUDP2** - UDP Transport Extension Version 2
  - https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-rdpeudp2/
- **MS-RDPEMT** - Multitransport Extension (higher layer)
- **Microsoft Protocol Test Suites** - Validation test cases
  - https://github.com/microsoft/WindowsProtocolTestSuites

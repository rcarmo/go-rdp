# internal/protocol/tpkt

TPKT framing layer for RDP over TCP.

## Overview

This package implements TPKT (RFC 1006), which provides framing for ISO 8073 X.224 PDUs over TCP/IP. TPKT adds a simple 4-byte header to each PDU for length-delimited message boundaries.

## Files

| File | Purpose |
|------|---------|
| `protocol.go` | Main Protocol struct |
| `send.go` | Sending TPKT frames |
| `receive.go` | Receiving TPKT frames |
| `tpkt_test.go` | Unit tests |

## TPKT Header Format

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|    Version    |   Reserved    |          Length               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                                                               |
|                       X.224 PDU Data                          |
|                                                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

| Field | Size | Value | Description |
|-------|------|-------|-------------|
| Version | 1 byte | 0x03 | TPKT version |
| Reserved | 1 byte | 0x00 | Must be zero |
| Length | 2 bytes | Big-endian | Total packet length (including header) |
| Data | Variable | - | X.224 PDU |

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                        X.224 Layer                                   │
│                   (protocol/x224)                                    │
└───────────────────────────────┬─────────────────────────────────────┘
                                │
┌───────────────────────────────▼─────────────────────────────────────┐
│                        TPKT Layer                                    │
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │                      Send()                                     ││
│  │   1. Calculate total length (4 + payload length)                ││
│  │   2. Write header [0x03, 0x00, length_hi, length_lo]            ││
│  │   3. Write payload data                                         ││
│  └─────────────────────────────────────────────────────────────────┘│
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │                     Receive()                                   ││
│  │   1. Read 4-byte header                                         ││
│  │   2. Verify version == 0x03                                     ││
│  │   3. Extract length                                             ││
│  │   4. Read payload (length - 4 bytes)                            ││
│  │   5. Return reader for payload                                  ││
│  └─────────────────────────────────────────────────────────────────┘│
└───────────────────────────────┬─────────────────────────────────────┘
                                │
┌───────────────────────────────▼─────────────────────────────────────┐
│                     TCP Connection                                   │
│                   (net.Conn / TLS)                                   │
└─────────────────────────────────────────────────────────────────────┘
```

## Protocol Interface

```go
type Protocol struct {
    conn io.ReadWriteCloser
}

func NewProtocol(conn io.ReadWriteCloser) *Protocol

func (p *Protocol) Send(data []byte) error
func (p *Protocol) Receive() (io.Reader, error)
func (p *Protocol) Close() error
```

## Usage

### Creating TPKT Layer

```go
// Wrap TCP connection
tpkt := tpkt.NewProtocol(tcpConn)

// Or wrap TLS connection
tpkt := tpkt.NewProtocol(tlsConn)
```

### Sending Data

```go
// X.224 PDU data
x224Data := buildX224PDU()

// Send with TPKT framing
err := tpkt.Send(x224Data)
```

### Receiving Data

```go
// Receive TPKT frame
reader, err := tpkt.Receive()
if err != nil {
    return err
}

// Parse X.224 PDU from reader
x224PDU, err := parseX224(reader)
```

## Wire Example

Sending a 100-byte X.224 PDU:

```
Sent bytes:
03 00 00 68    // TPKT header (version=3, reserved=0, length=104)
[100 bytes]    // X.224 PDU data
```

Breakdown:
- `03` - Version 3
- `00` - Reserved
- `00 68` - Length = 104 (0x0068 in big-endian) = 4 header + 100 data

## Error Handling

| Error | Description |
|-------|-------------|
| Invalid version | Header byte 0 is not 0x03 |
| Short read | Less than 4 bytes available |
| Invalid length | Length < 4 or exceeds buffer |

## FastPath Bypass

Note: RDP FastPath traffic bypasses TPKT entirely. FastPath packets start with a different header format and are handled separately by the `fastpath` package.

Detection:
```go
// TPKT: first byte is 0x03
// FastPath: first byte has bits 0-1 = 0 (action field)
if data[0] == 0x03 {
    // TPKT/X.224 slow-path
} else {
    // FastPath
}
```

## Why TPKT?

TCP is a stream protocol with no message boundaries. TPKT provides:
- Simple framing with 4-byte header
- Length-prefixed messages for reliable parsing
- Compatibility with ISO transport protocols
- Minimal overhead (4 bytes per PDU)

## References

- **RFC 1006** - ISO Transport Service on top of the TCP
- **ISO 8073** - Connection-oriented transport protocol specification
- **MS-RDPBCGR** - Uses TPKT for X.224 PDU framing

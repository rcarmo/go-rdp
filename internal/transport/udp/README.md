# internal/transport/udp

UDP Transport Layer implementing MS-RDPEUDP connection management.

## Overview

This package provides the transport-level UDP connection management, including:
- **Connection state machine** - Per MS-RDPEUDP Section 3.1.5
- **Retransmission** - Reliable delivery with configurable timeouts
- **Keepalive** - 65-second connection timeout with periodic ACKs
- **Congestion control** - CN/CWR flag handling
- **Selective ACK** - RLE-encoded ACK vector processing

This is the transport layer that uses the `rdpeudp` protocol package for packet encoding/decoding.

## Specification Reference

- **MS-RDPEUDP** - Remote Desktop Protocol: UDP Transport Extension
  - https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-rdpeudp/
- **Microsoft Protocol Test Suites** - Validation test cases
  - https://github.com/microsoft/WindowsProtocolTestSuites
  - `TestSuites/RDP/Client/docs/MS-RDPEUDP_ClientTestDesignSpecification.md`

## Files

| File | Purpose |
|------|---------|
| `connection.go` | Connection state machine and management |
| `connection_test.go` | Unit tests including MS Protocol Test Suite validation |

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                    RDP Client (internal/rdp)                         │
└───────────────────────────────┬─────────────────────────────────────┘
                                │
┌───────────────────────────────▼─────────────────────────────────────┐
│                    UDP Transport Layer                               │
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │                   Connection State Machine                      ││
│  │                                                                  ││
│  │  CLOSED ──► SYN_SENT ──► ESTABLISHED ──► CLOSED                 ││
│  │                              │                                   ││
│  │                      (data transfer)                             ││
│  └─────────────────────────────────────────────────────────────────┘│
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │                        Timer Management                         ││
│  │                                                                  ││
│  │  ┌──────────────┐  ┌───────────────┐  ┌──────────────────────┐  ││
│  │  │ Retransmit   │  │ Keepalive     │  │ Delayed ACK          │  ││
│  │  │ V1:500ms     │  │ Interval:30s  │  │ 200ms                │  ││
│  │  │ V2:300ms     │  │ Timeout:65s   │  │                      │  ││
│  │  └──────────────┘  └───────────────┘  └──────────────────────┘  ││
│  └─────────────────────────────────────────────────────────────────┘│
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │                     Send/Receive Buffers                        ││
│  │                                                                  ││
│  │  Send Buffer: Packets awaiting ACK (for retransmission)         ││
│  │  Recv Buffer: Out-of-order packets (for reordering)             ││
│  └─────────────────────────────────────────────────────────────────┘│
└───────────────────────────────┬─────────────────────────────────────┘
                                │
┌───────────────────────────────▼─────────────────────────────────────┐
│                 Protocol Layer (internal/protocol/rdpeudp)           │
│                    (Packet serialization/deserialization)            │
└───────────────────────────────┬─────────────────────────────────────┘
                                │
┌───────────────────────────────▼─────────────────────────────────────┐
│                           UDP Socket                                 │
│                         (net.UDPConn)                                │
└─────────────────────────────────────────────────────────────────────┘
```

## Connection States

| State | Description | Next States |
|-------|-------------|-------------|
| CLOSED | Initial/final state | SYN_SENT (client), LISTEN (server) |
| LISTEN | Server waiting for SYN | SYN_RECEIVED |
| SYN_SENT | Client sent SYN, awaiting SYN+ACK | ESTABLISHED, CLOSED |
| SYN_RECEIVED | Server sent SYN+ACK, awaiting ACK | ESTABLISHED, CLOSED |
| ESTABLISHED | Connection active, data transfer | CLOSED |

## Configuration

```go
type Config struct {
    LocalAddr         *net.UDPAddr  // Local bind address
    RemoteAddr        *net.UDPAddr  // Remote server address
    MTU               uint16        // MTU (1132-1232)
    ReceiveWindowSize uint16        // Receive buffer size
    Reliable          bool          // Reliable (true) or lossy (false)
    ProtocolVersion   uint16        // RDPEUDP version
    CookieHash        [32]byte      // Security cookie (V3)
}
```

## Constants

| Constant | Value | Description |
|----------|-------|-------------|
| DefaultMTU | 1232 | Default MTU |
| MinMTU | 1132 | Minimum MTU |
| MaxMTU | 1232 | Maximum MTU |
| RetransmitTimeoutV1 | 500ms | V1 minimum retransmit |
| RetransmitTimeoutV2 | 300ms | V2 minimum retransmit |
| KeepaliveTimeout | 65s | Connection timeout |
| KeepaliveInterval | 30s | Keepalive ACK interval |
| DelayedACKTimeout | 200ms | Max ACK delay |
| MaxRetransmitCount | 3 | SYN/SYN+ACK retries |
| MaxDataRetransmitCount | 5 | Data packet retries |

## Usage

### Creating a Connection

```go
config := &udp.Config{
    RemoteAddr: &net.UDPAddr{
        IP:   net.ParseIP("192.168.1.100"),
        Port: 3389,
    },
    MTU:             udp.DefaultMTU,
    ReceiveWindowSize: 64,
    Reliable:        true,
    ProtocolVersion: rdpeudp.ProtocolVersion2,
}

conn, err := udp.NewConnection(config)
if err != nil {
    return err
}
defer conn.Close()
```

### Connecting

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

err := conn.Connect(ctx)
if err != nil {
    if err == udp.ErrTimeout {
        log.Println("Connection timed out")
    }
    return err
}

log.Printf("Connected! State: %s", conn.State())
```

### Reading/Writing

```go
// Write data
n, err := conn.Write([]byte("Hello, RDP!"))

// Read data
buf := make([]byte, 4096)
n, err := conn.Read(buf)
data := buf[:n]
```

### Getting Statistics

```go
stats := conn.Stats()
fmt.Printf("Packets sent: %d\n", stats.PacketsSent)
fmt.Printf("Packets received: %d\n", stats.PacketsReceived)
fmt.Printf("Retransmits: %d\n", stats.Retransmits)
fmt.Printf("RTT: %v\n", stats.RTT)
fmt.Printf("Congestion events: %d\n", stats.CongestionEvents)
```

## Timer Management

### Retransmit Timer

Per Section 3.1.6.1:
- Fires at `max(minTimeout, 2*RTT)` after transmission
- V1: 500ms minimum, V2: 300ms minimum
- After 3-5 retransmits, connection is closed

### Keepalive Timer

Per Section 3.1.1.9:
- 65-second timeout if no data received
- 30-second interval for sending keepalive ACKs
- Prevents NAT timeout and detects dead connections

### Delayed ACK Timer

Per Section 3.1.6.3:
- 200ms maximum delay before sending pending ACK
- Allows batching of ACKs for efficiency

## Congestion Control

| Flag | Behavior |
|------|----------|
| CN (Congestion Notification) | Set when packet loss detected |
| CWR (Congestion Window Reset) | Acknowledges CN, stops notifications |

When CN is received:
- Congestion window is halved (multiplicative decrease)
- Stats.CongestionEvents is incremented

## Microsoft Protocol Test Suite Compliance

This implementation passes the following Microsoft test cases:

| Test Case | Description |
|-----------|-------------|
| S1_Connection_Initialization | SYN datagram validation |
| S1_Connection_Keepalive | 65/2 second keepalive interval |
| S2_DataTransfer_ClientReceiveData | ACK with ACK_VECTOR |
| S2_DataTransfer_AcknowlegeLossyPackage | Lost packet handling |
| S2_DataTransfer_SequenceNumberWrapAround | uint.maxValue-3 wrap |
| S2_DataTransfer_ClientAckDelay | RDPUDP_FLAG_ACKDELAYED |
| S2_DataTransfer_RetransmitTest | Retransmission on timeout |
| S2_DataTransfer_CongestionControlTest | CN/CWR handling |

## Test Coverage

Current coverage: **41.6%**

```bash
go test -cover ./internal/transport/udp/...
```

## Error Types

| Error | Description |
|-------|-------------|
| ErrClosed | Connection was closed |
| ErrTimeout | Operation timed out |
| ErrInvalidState | Invalid state for operation |
| ErrInvalidPacket | Malformed packet received |
| ErrConnectionFailed | Connection establishment failed |

## Optional Features Not Yet Implemented

- **DTLS integration** - Requires `github.com/pion/dtls/v2`
- **CORRELATION_ID** - Windows servers send this
- **FEC (Forward Error Correction)** - For lossy transport

## References

- **MS-RDPEUDP** - UDP Transport Extension
  - https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-rdpeudp/
- **MS-RDPEUDP2** - UDP Transport Extension Version 2
  - https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-rdpeudp2/
- **Microsoft Protocol Test Suites**
  - https://github.com/microsoft/WindowsProtocolTestSuites
- **Hardening Consulting - UDP support in FreeRDP**
  - https://www.hardening-consulting.com/en/posts/20230109-udp-support-2.html

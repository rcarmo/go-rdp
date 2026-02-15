# UDP Transport Implementation

> MS-RDPEUDP and MS-RDPEMT Protocol Support  
> Last Updated: January 23, 2026

## Overview

The RDP HTML5 client supports UDP transport as an optional, experimental feature for improved performance over high-latency or lossy networks. UDP transport provides lower latency compared to TCP by avoiding head-of-line blocking.

### Status

| Component | Status | Notes |
|-----------|--------|-------|
| MS-RDPEUDP protocol | ✅ Complete | SYN/ACK, ACK vector, FEC headers |
| MS-RDPEMT protocol | ✅ Complete | Tunnel create/response, error codes |
| Connection state machine | ✅ Complete | SYN_SENT, SYN_RECEIVED, CONNECTED |
| Reliable transport | ✅ Complete | Sequence numbers, retransmission |
| Lossy transport | ✅ Complete | Forward error correction hooks |
| TLS/DTLS secure tunnel | ✅ Complete | TLS for reliable, DTLS for lossy |
| Multitransport negotiation | ✅ Complete | Server request handling, accept/decline |
| Integration with RDP client | 🔧 Partial | Basic wiring, needs full data path |

### Enabling UDP Transport

UDP is disabled by default. Enable it via:

```bash
# Environment variable
export RDP_ENABLE_UDP=true

# Or command-line flag
./go-rdp -udp
```

When disabled, the client automatically declines server UDP transport requests with `E_ABORT` (0x80004004).

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          RDP Client                                      │
│  ┌───────────────────────────────────────────────────────────────────┐  │
│  │                  MultitransportHandler                             │  │
│  │  - Receives Initiate Multitransport Request PDU                   │  │
│  │  - Sends Initiate Multitransport Response PDU via MCS I/O channel │  │
│  │  - Manages TunnelManager for UDP connections                       │  │
│  └───────────────────────────────────────────────────────────────────┘  │
│                                  │                                       │
│  ┌───────────────────────────────▼───────────────────────────────────┐  │
│  │                     TunnelManager                                  │  │
│  │  - Creates Tunnel instances per server request                    │  │
│  │  - Handles tunnel lifecycle (create, data, close)                 │  │
│  │  - Routes data between tunnels and client                         │  │
│  └───────────────────────────────────────────────────────────────────┘  │
│                                  │                                       │
│  ┌───────────────────────────────▼───────────────────────────────────┐  │
│  │                       Tunnel                                       │  │
│  │  - Wraps SecureConnection (TLS/DTLS)                              │  │
│  │  - Sends Tunnel Create Request PDU on establishment               │  │
│  │  - Receives Tunnel Create Response PDU                             │  │
│  └───────────────────────────────────────────────────────────────────┘  │
│                                  │                                       │
│  ┌───────────────────────────────▼───────────────────────────────────┐  │
│  │                   SecureConnection                                 │  │
│  │  - TLS over reliable UDP transport                                │  │
│  │  - DTLS over lossy UDP transport                                  │  │
│  │  - Wraps data in RDPEUDP Tunnel Data PDU                          │  │
│  └───────────────────────────────────────────────────────────────────┘  │
│                                  │                                       │
│  ┌───────────────────────────────▼───────────────────────────────────┐  │
│  │                     Connection                                     │  │
│  │  - MS-RDPEUDP state machine (SYN_SENT → CONNECTED)                │  │
│  │  - ACK vector tracking for reliable delivery                      │  │
│  │  - Retransmission with exponential backoff                        │  │
│  │  - Timer management (keepalive, delayed ACK)                      │  │
│  └───────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────┘
                                   │
                                   │ UDP :3389
                                   ▼
                    ┌─────────────────────────────┐
                    │        RDP Server           │
                    │   (Windows Server 2012+)    │
                    └─────────────────────────────┘
```

---

## Protocol Flow

### 1. Multitransport Negotiation (MS-RDPEMT)

The server initiates UDP transport after TCP connection is established:

```
Server                                    Client
   │                                         │
   │◄─────── TCP Connection Established ────►│
   │                                         │
   │  Initiate Multitransport Request PDU   │
   │  (requestId, requestedProtocol, cookie) │
   │─────────────────────────────────────────►│
   │                                         │
   │                        [If UDP enabled] │
   │                                         │
   │                    UDP SYN ─────────────►│
   │◄───────────────── UDP SYN+ACK ──────────│
   │                    UDP ACK ─────────────►│
   │                                         │
   │◄──────── TLS/DTLS Handshake ───────────►│
   │                                         │
   │◄──── Tunnel Create Request PDU ─────────│
   │ ──── Tunnel Create Response PDU ────────►│
   │                                         │
   │  Initiate Multitransport Response PDU  │
   │  (requestId, hrResponse=S_OK)          │
   │◄─────────────────────────────────────────│
   │                                         │
   │       [If UDP disabled or failed]       │
   │                                         │
   │  Initiate Multitransport Response PDU  │
   │  (requestId, hrResponse=E_ABORT)       │
   │◄─────────────────────────────────────────│
```

### 2. Transport Types

| Protocol | MS-RDPEUDP Flag | Security | Use Case |
|----------|-----------------|----------|----------|
| Reliable | TRANSPORTTYPE_UDP_FECR (0x01) | TLS | Critical data (input, control) |
| Lossy | TRANSPORTTYPE_UDP_FECL (0x04) | DTLS | Bulk data (graphics, audio) |

### 3. Connection State Machine

```
         ┌─────────────────┐
         │   CLOSED        │
         └────────┬────────┘
                  │ Connect()
                  ▼
         ┌─────────────────┐
         │   SYN_SENT      │◄─────────────┐
         └────────┬────────┘              │
                  │ Recv SYN+ACK          │ Timeout
                  ▼                       │ (retry)
         ┌─────────────────┐              │
         │  SYN_RECEIVED   │──────────────┘
         └────────┬────────┘
                  │ Send ACK
                  ▼
         ┌─────────────────┐
         │   CONNECTED     │
         └────────┬────────┘
                  │ Close()
                  ▼
         ┌─────────────────┐
         │   CLOSED        │
         └─────────────────┘
```

---

## Key Components

### Connection (`internal/transport/udp/connection.go`)

Implements the MS-RDPEUDP protocol:

- **State machine**: SYN_SENT → SYN_RECEIVED → CONNECTED → CLOSED
- **Sequence tracking**: Outbound and inbound sequence numbers
- **ACK vectors**: Compact bitmap of received packets
- **Retransmission**: Tracks unacked packets with timestamps
- **Timers**: Keepalive (30s), delayed ACK (50ms), retransmit (200ms initial)

Key functions:
```go
func (c *Connection) Connect() error           // Initiates 3-way handshake
func (c *Connection) Read(b []byte) (int, error)   // Receives data
func (c *Connection) Write(b []byte) (int, error)  // Sends data
func (c *Connection) Close() error             // Closes connection
```

### SecureConnection (`internal/transport/udp/secure.go`)

Wraps Connection with TLS/DTLS:

```go
// For reliable transport (TLS)
secure := NewSecureConnection(conn, false)

// For lossy transport (DTLS)
secure := NewSecureConnection(conn, true)
```

Data is wrapped in RDPEUDP Tunnel Data PDU:
```
┌──────────────────────────────────────┐
│  Action (1 byte): 0x00 = DATA        │
│  Data Length (2 bytes)               │
│  Payload (variable)                  │
└──────────────────────────────────────┘
```

### TunnelManager (`internal/transport/udp/tunnel.go`)

Manages multiple tunnels for a client session:

```go
mgr, _ := NewTunnelManager(&TunnelManagerConfig{
    ServerAddr:     "server:3389",
    Enabled:        true,
    ConnectTimeout: 10 * time.Second,
})

// Handle server request
mgr.HandleMultitransportRequest(req)

// Send data on tunnel
mgr.SendData(requestID, data)

// Close all tunnels
mgr.Close()
```

### MultitransportHandler (`internal/rdp/multitransport.go`)

Integrates UDP transport with the RDP client:

```go
// Enable UDP support
client.EnableMultitransport(true)

// Set server address for UDP connections
handler.SetServerAddress("server", 3389)

// Set callback for when UDP tunnel is ready
handler.SetUDPReadyCallback(func(requestID uint32, cookie [16]byte, reliable bool) {
    log.Printf("UDP tunnel %d ready", requestID)
})
```

---

## MS-RDPEUDP Packet Formats

### SYN Packet (Version 2)

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|            Flags              |          uUpStreamMtu         |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|         uDownStreamMtu        |       snInitialSequenceNumber |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|   (continued)                 |    uReceiveWindowSize (16)    |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
| CorrId (16 bytes)                                             |
|                                                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
| SynDataPayload (optional)                                     |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

### ACK Packet

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|            Flags              |       snAckOfAcksSeqNum       |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|   (continued)                 |    uReceiveWindowSize         |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|            snSourceAck                                        |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
| AckVectorElement[] (variable)                                 |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

### DATA Packet

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|            Flags              |        snSourceAck            |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|   (continued)                 |       uReceiveWindowSize      |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|           snCoded                                             |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
| Payload Data (variable)                                       |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

---

## MS-RDPEMT Packet Formats

### Tunnel Header

All MS-RDPEMT messages start with a tunnel header:

```
 0       1       2       3
+-------+-------+-------+-------+
| Action| Flags | PayloadLength |
+-------+-------+-------+-------+
```

Actions:
- `0x00` - CYCR_CYCR_PKT (Tunnel Create Request)
- `0x01` - CYCR_CYCR_RESP_PKT (Tunnel Create Response)
- `0x02` - DATA_PKT (Tunnel Data)

### Multitransport Request PDU

```
+-------+-------+-------+-------+-------+-------+-------+-------+
|            requestId          |      requestedProtocol        |
+-------+-------+-------+-------+-------+-------+-------+-------+
| Reserved      |         securityCookie (16 bytes)             |
+-------+-------+-------+-------+-------+-------+-------+-------+
```

### Multitransport Response PDU

```
+-------+-------+-------+-------+-------+-------+-------+-------+
|            requestId          |            hrResponse         |
+-------+-------+-------+-------+-------+-------+-------+-------+
```

Response codes:
- `S_OK` (0x00000000) - Success
- `E_ABORT` (0x80004004) - Aborted/declined

---

## Error Handling

### When UDP is Disabled

```go
// In MultitransportHandler.HandleRequest()
if !enabled {
    return h.sendDecline(req.RequestID)  // Sends E_ABORT
}
```

The decline response is sent via the MCS I/O channel using `sendMultitransportResponse()`.

### Connection Failures

- **Timeout during handshake**: Retries with exponential backoff
- **Connection refused**: Falls back to TCP-only operation
- **TLS/DTLS failure**: Reports error, declines request if Soft-Sync
- **Network unreachable**: Logs error, continues with TCP

---

## Testing

Run UDP-specific tests:

```bash
# All UDP transport tests
go test -v ./internal/transport/udp/...

# Protocol packet tests
go test -v ./internal/protocol/rdpeudp/...
go test -v ./internal/protocol/rdpemt/...
```

Test coverage includes:
- Connection state machine transitions
- ACK vector building and parsing
- Packet serialization/deserialization
- Timer lifecycle management
- Secure connection wrapping
- Error handling paths

---

## Future Work

1. **Full data path integration**: Route graphics/audio data through UDP when available
2. **Soft-Sync support**: Migrate existing channels from TCP to UDP
3. **Performance tuning**: Optimize MTU discovery and congestion control
4. **FEC implementation**: Forward error correction for lossy transport
5. **Connection migration**: Handle network changes gracefully

---

## References

- [MS-RDPEUDP](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-rdpeudp/) - UDP Transport Extension
- [MS-RDPEUDP2](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-rdpeudp2/) - UDP Transport Version 2
- [MS-RDPEMT](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-rdpemt/) - Multitransport Extension
- [MS-RDPBCGR Section 3.2.5.15](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-rdpbcgr/) - Multitransport Processing

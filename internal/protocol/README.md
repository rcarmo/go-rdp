# internal/protocol

RDP protocol layer implementations.

## Overview

This directory contains the layered protocol stack for RDP communication. Each subdirectory implements a specific protocol layer according to Microsoft and ITU specifications.

## Protocol Stack

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Application Layer                             │
│                      (internal/rdp, internal/handler)                │
└───────────────────────────────┬─────────────────────────────────────┘
                                │
┌───────────────────────────────▼─────────────────────────────────────┐
│                         pdu/                                         │
│              Protocol Data Units (60+ message types)                 │
│         Capabilities, Connection, Data, Licensing, etc.              │
└───────────────────────────────┬─────────────────────────────────────┘
                                │
        ┌───────────────────────┴───────────────────────┐
        │                                               │
┌───────▼───────┐                               ┌───────▼───────┐
│   fastpath/   │                               │    audio/     │
│ High-speed    │                               │ Virtual Chan  │
│ Updates/Input │                               │ (rdpsnd)      │
└───────┬───────┘                               └───────┬───────┘
        │                                               │
        └───────────────────────┬───────────────────────┘
                                │
┌───────────────────────────────▼─────────────────────────────────────┐
│                           mcs/                                       │
│                Multi-Channel Service (ITU T.125)                     │
│              Channel multiplexing, user attachment                   │
└───────────────────────────────┬─────────────────────────────────────┘
                                │
                        ┌───────┴───────┐
                        │               │
                ┌───────▼───────┐       │
                │     gcc/      │       │
                │   T.124 GCC   │       │
                │ Initial setup │       │
                └───────────────┘       │
                                        │
┌───────────────────────────────────────▼─────────────────────────────┐
│                          x224/                                       │
│              Connection-Oriented Transport (ISO 8073)                │
│                    Connection negotiation                            │
└───────────────────────────────┬─────────────────────────────────────┘
                                │
┌───────────────────────────────▼─────────────────────────────────────┐
│                          tpkt/                                       │
│                    TPKT Framing (RFC 1006)                           │
│                   ISO PDUs over TCP/IP                               │
└───────────────────────────────┬─────────────────────────────────────┘
                                │
┌───────────────────────────────▼─────────────────────────────────────┐
│                        encoding/                                     │
│              BER/PER Encoding (ASN.1 serialization)                  │
│               Used by MCS, GCC, X.224 layers                         │
└─────────────────────────────────────────────────────────────────────┘
```

## Subdirectories

| Directory | Protocol | Specification | Purpose |
|-----------|----------|---------------|---------|
| `audio/` | RDPEA/RDPEAI | MS-RDPEA | Audio virtual channel |
| `encoding/` | BER/PER | ITU X.690/X.691 | ASN.1 serialization |
| `fastpath/` | FastPath | MS-RDPBCGR | Optimized data path |
| `gcc/` | T.124 GCC | ITU T.124 | Conference control |
| `mcs/` | T.125 MCS | ITU T.125 | Channel multiplexing |
| `pdu/` | RDP PDUs | MS-RDPBCGR | All RDP message types |
| `tpkt/` | TPKT | RFC 1006 | TCP framing |
| `x224/` | X.224 | ISO 8073 | Connection layer |

## Data Flow

### Connection Establishment

```
1. TCP Connect
2. TPKT: Frame X.224 Connection Request
3. X.224: Negotiate protocol (RDP/TLS/NLA)
4. MCS: Connect Initial (with GCC user data)
5. MCS: Erect Domain, Attach User, Join Channels
6. PDU: Secure Settings Exchange
7. PDU: Licensing
8. PDU: Capability Exchange
9. PDU: Connection Finalization
```

### Screen Updates (FastPath)

```
Server → TPKT Frame → FastPath Header → Bitmap Data → Client
```

### Input Events (FastPath)

```
Client → TPKT Frame → FastPath Header → Input Event → Server
```

### Virtual Channel (Audio)

```
Server → TPKT Frame → MCS → Audio PDU → Client
```

## Encoding Layer

The `encoding/` package provides:
- **BER** (Basic Encoding Rules) - For X.224, MCS
- **PER** (Packed Encoding Rules) - For GCC conference data

## References

- **MS-RDPBCGR** - RDP Basic Connectivity and Graphics Remoting
- **MS-RDPEA** - Audio Output Virtual Channel Extension
- **ITU T.124** - Generic Conference Control
- **ITU T.125** - Multi-Channel Service
- **ISO 8073** - Connection-Oriented Transport Protocol
- **RFC 1006** - ISO Transport over TCP

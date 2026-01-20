# internal/protocol/mcs

ITU T.125 Multi-Channel Service (MCS) implementation.

## Overview

This package implements the MCS protocol layer, which provides:
- **Channel multiplexing** - Multiple logical channels over one connection
- **Domain management** - User attachment and channel joining
- **Data routing** - PDU delivery to appropriate channels

MCS sits between the X.224 transport layer and higher-level RDP protocols.

## Files

| File | Purpose |
|------|---------|
| `protocol.go` | Main Protocol struct and interface |
| `connect.go` | Connection establishment |
| `domain.go` | Domain erection |
| `attach_user.go` | User attachment |
| `channel_join.go` | Channel joining |
| `send_data.go` | Data transmission |
| `receive_data.go` | Data reception |
| `types.go` | Type definitions |
| `pdu_*.go` | Individual PDU definitions |
| `*_test.go` | Unit tests |

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                        RDP Client                                    │
└───────────────────────────────┬─────────────────────────────────────┘
                                │
┌───────────────────────────────▼─────────────────────────────────────┐
│                          MCS Layer                                   │
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │                    Channel Router                               ││
│  │                                                                  ││
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────────────┐ ││
│  │  │ Global   │  │  User    │  │  rdpsnd  │  │  Other Virtual   │ ││
│  │  │ Channel  │  │ Channel  │  │ Channel  │  │  Channels        │ ││
│  │  │ (1003)   │  │ (1003+n) │  │          │  │                  │ ││
│  │  └──────────┘  └──────────┘  └──────────┘  └──────────────────┘ ││
│  └─────────────────────────────────────────────────────────────────┘│
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │                   Domain Controller                             ││
│  │            (User attachment, channel join)                      ││
│  └─────────────────────────────────────────────────────────────────┘│
└───────────────────────────────┬─────────────────────────────────────┘
                                │
┌───────────────────────────────▼─────────────────────────────────────┐
│                        X.224 Layer                                   │
└─────────────────────────────────────────────────────────────────────┘
```

## MCS Interface

```go
type MCSLayer interface {
    // Connection
    Connect(userData []byte) (io.Reader, error)
    
    // Domain management
    ErectDomain() error
    AttachUser() (uint16, error)
    JoinChannels(userID uint16, channelIDMap map[string]uint16) error
    
    // Data transfer
    Send(userID, channelID uint16, data []byte) error
    Receive() (channelID uint16, reader io.Reader, err error)
}
```

## Connection Flow

```
Client                                  Server
   │                                      │
   │  Connect-Initial (with GCC data)     │
   │  ────────────────────────────────►   │
   │                                      │
   │  Connect-Response (with GCC data)    │
   │  ◄────────────────────────────────   │
   │                                      │
   │  Erect-Domain-Request                │
   │  ────────────────────────────────►   │
   │                                      │
   │  Attach-User-Request                 │
   │  ────────────────────────────────►   │
   │                                      │
   │  Attach-User-Confirm (userID)        │
   │  ◄────────────────────────────────   │
   │                                      │
   │  Channel-Join-Request (global)       │
   │  ────────────────────────────────►   │
   │                                      │
   │  Channel-Join-Confirm                │
   │  ◄────────────────────────────────   │
   │                                      │
   │  Channel-Join-Request (user)         │
   │  ────────────────────────────────►   │
   │                                      │
   │  Channel-Join-Confirm                │
   │  ◄────────────────────────────────   │
   │                                      │
   │  [Join additional channels...]       │
```

## Key PDU Types

### Connect Initial

```go
type ClientConnectInitial struct {
    CallingDomainSelector  []byte
    CalledDomainSelector   []byte
    UpwardFlag             bool
    TargetParameters       DomainParameters
    MinimumParameters      DomainParameters
    MaximumParameters      DomainParameters
    UserData               []byte  // GCC Conference Create Request
}
```

### Connect Response

```go
type ServerConnectResponse struct {
    Result           uint8
    CalledConnectID  uint32
    DomainParameters DomainParameters
    UserData         []byte  // GCC Conference Create Response
}
```

### Domain Parameters

```go
type DomainParameters struct {
    MaxChannelIDs   uint32
    MaxUserIDs      uint32
    MaxTokenIDs     uint32
    NumPriorities   uint32
    MinThroughput   uint32
    MaxHeight       uint32
    MaxMCSPDUsize   uint32
    ProtocolVersion uint32
}
```

### Attach User Confirm

```go
type ServerAttachUserConfirm struct {
    Result uint8
    UserID uint16
}
```

### Send Data Request

```go
type ClientSendDataRequest struct {
    UserID      uint16
    ChannelID   uint16
    Priority    uint8
    Segmentation uint8
    UserData    []byte
}
```

## Channel Types

| Channel | ID | Description |
|---------|-----|-------------|
| Global | Server-assigned (typically 1003) | Primary control channel |
| User | userID + 1001 | Per-user data channel |
| I/O | 1004+ | Virtual channels (rdpsnd, etc.) |

## Usage

### Connection Setup

```go
mcs := mcs.NewProtocol(x224)

// Connect with GCC user data
serverData, err := mcs.Connect(clientUserData)

// Erect domain
err = mcs.ErectDomain()

// Attach user
userID, err := mcs.AttachUser()

// Join channels
channelMap := map[string]uint16{
    "global": globalChannelID,
    "user":   userID,
    "rdpsnd": audioChannelID,
}
err = mcs.JoinChannels(userID, channelMap)
```

### Sending Data

```go
// Send to global channel
err := mcs.Send(userID, globalChannelID, pduData)
```

### Receiving Data

```go
channelID, reader, err := mcs.Receive()
switch channelID {
case globalChannelID:
    handleGlobalPDU(reader)
case audioChannelID:
    handleAudioPDU(reader)
}
```

## BER Encoding

MCS uses BER (Basic Encoding Rules) with application tags:

| PDU | Tag |
|-----|-----|
| Connect-Initial | 0x65 (101) |
| Connect-Response | 0x66 (102) |
| Erect-Domain | 0x04 |
| Attach-User-Request | 0x28 (40) |
| Attach-User-Confirm | 0x2C (44) |
| Channel-Join-Request | 0x38 |
| Channel-Join-Confirm | 0x3C |
| Send-Data-Request | 0x64 |
| Send-Data-Indication | 0x68 |

## Error Handling

| Result Code | Description |
|-------------|-------------|
| 0 | Success |
| 1 | Domain not hierarchical |
| 2 | No such channel |
| 3 | No such domain |
| 4 | No such user |
| 5 | Not admitted |
| 6 | Other user ID |
| 7 | Parameters unacceptable |
| 8 | Token not available |
| 9 | Token not possessed |
| 10 | Too many channels |
| 11 | Too many tokens |
| 12 | Too many users |
| 13 | Unspecified failure |
| 14 | User rejected |

## References

- **ITU-T T.125** - Multipoint Communication Service Protocol
- **ITU-T T.122** - Multipoint Communication Service
- **MS-RDPBCGR** Section 2.2.1 - MCS Connection Sequence

# internal/protocol/drdynvc

Dynamic Virtual Channel (DVC) implementation per MS-RDPEDYC.

## Overview

This package implements the Dynamic Virtual Channel protocol, which allows:
- **Dynamic channel creation** - Create virtual channels after connection establishment
- **Soft-Sync capability** - Request soft-sync for seamless channel recovery
- **Channel data transport** - First and subsequent data fragments with compression

Dynamic channels are transported over the `drdynvc` static virtual channel.

## Specification Reference

- **MS-RDPEDYC** - Remote Desktop Protocol: Dynamic Channel Virtual Channel Extension
  - https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-rdpedyc/

## Files

| File | Purpose |
|------|---------|
| `drdynvc.go` | Protocol implementation and PDU definitions |
| `drdynvc_test.go` | Unit tests |

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                    Application (Display, Audio, etc.)               │
└───────────────────────────────┬─────────────────────────────────────┘
                                │
┌───────────────────────────────▼─────────────────────────────────────┐
│                      Dynamic Virtual Channel                         │
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │                     Channel Manager                             ││
│  │                                                                  ││
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────────────┐ ││
│  │  │ RDPEDISP │  │ Graphics │  │  Audio   │  │  Other DVC       │ ││
│  │  │ Display  │  │ Pipeline │  │ Input    │  │  Extensions      │ ││
│  │  └──────────┘  └──────────┘  └──────────┘  └──────────────────┘ ││
│  └─────────────────────────────────────────────────────────────────┘│
└───────────────────────────────┬─────────────────────────────────────┘
                                │
┌───────────────────────────────▼─────────────────────────────────────┐
│                    drdynvc Static Virtual Channel                    │
│                         (MCS Channel)                                │
└─────────────────────────────────────────────────────────────────────┘
```

## PDU Types

### Commands (Cmd field)

| Value | Command | Description |
|-------|---------|-------------|
| 0x01 | CREATE | Server creates a dynamic channel |
| 0x02 | DATA_FIRST | First data fragment |
| 0x03 | DATA | Subsequent data fragments |
| 0x04 | CLOSE | Close channel |
| 0x05 | CAPS | Capability exchange |
| 0x06 | DATA_FIRST_COMPRESSED | Compressed first fragment (V3) |
| 0x07 | DATA_COMPRESSED | Compressed subsequent fragment (V3) |
| 0x08 | SOFT_SYNC_REQUEST | Request soft-sync (V3) |
| 0x09 | SOFT_SYNC_RESPONSE | Soft-sync response (V3) |

### Capability Versions

| Version | Features |
|---------|----------|
| V1 (0x01) | Basic dynamic channels |
| V2 (0x02) | Priority levels |
| V3 (0x03) | Compression (RDP8), soft-sync support |

## Key Structures

### Header

```go
type Header struct {
    Cmd   uint8  // Command (4 bits) + Sp (2 bits) + CbChId (2 bits)
    // CbChId determines channel ID size: 0=1byte, 1=2bytes, 2=4bytes
}
```

### Capability PDU

```go
type CapsPDU struct {
    Version  uint16  // Protocol version (V1, V2, V3)
    Priority uint16  // Priority support (V2+)
}
```

### Create Request

```go
type CreateRequestPDU struct {
    ChannelID   uint32  // Dynamically assigned channel ID
    ChannelName string  // Channel name (e.g., "Microsoft::Windows::RDS::DisplayControl")
}
```

### Data PDU

```go
type DataPDU struct {
    ChannelID uint32
    Length    uint32  // Total uncompressed length (DATA_FIRST only)
    Data      []byte  // Fragment data
}
```

## Protocol Flow

### Capability Exchange

```
Client                              Server
   │                                   │
   │  CAPS (Version=V3)                │
   │  ◄────────────────────────────    │
   │                                   │
   │  CAPS (Version=min(V3,our_ver))   │
   │  ────────────────────────────►    │
```

### Channel Creation

```
Client                              Server
   │                                   │
   │  CREATE_REQ (channelID, name)     │
   │  ◄────────────────────────────    │
   │                                   │
   │  CREATE_RSP (channelID, result)   │
   │  ────────────────────────────►    │
```

### Data Transfer

```
Client                              Server
   │                                   │
   │  DATA_FIRST (channelID, len, data)│
   │  ◄────────────────────────────    │
   │                                   │
   │  DATA (channelID, data)           │
   │  ◄────────────────────────────    │
   │  ...                              │
   │                                   │
   │  DATA (channelID, final_data)     │
   │  ◄────────────────────────────    │
```

### Soft-Sync (V3)

```
Client                              Server
   │                                   │
   │  SOFT_SYNC_REQ (flags, channels)  │
   │  ────────────────────────────►    │
   │                                   │
   │  SOFT_SYNC_RSP (channels tunneled)│
   │  ◄────────────────────────────    │
```

## Usage

### Initializing

```go
// Create DVC manager
manager := drdynvc.NewManager()

// Process server capability
serverCaps := drdynvc.CapsPDU{Version: drdynvc.V3}
response := manager.HandleCaps(serverCaps)

// Respond with our capability
SendToServer(response.Serialize())
```

### Handling Channel Creation

```go
func handleDVCPDU(data []byte) {
    header := drdynvc.ParseHeader(data)
    
    switch header.Cmd {
    case drdynvc.CmdCreate:
        req := drdynvc.ParseCreateRequest(data)
        // Accept or reject channel
        resp := drdynvc.CreateResponsePDU{
            ChannelID: req.ChannelID,
            Result:    0, // Success
        }
        SendToServer(resp.Serialize())
        
    case drdynvc.CmdDataFirst:
        pdu := drdynvc.ParseDataFirst(data)
        // Start assembling fragmented data
        
    case drdynvc.CmdData:
        pdu := drdynvc.ParseData(data)
        // Continue assembling, deliver when complete
    }
}
```

## Well-Known Channel Names

| Channel Name | Purpose | Specification |
|--------------|---------|---------------|
| `Microsoft::Windows::RDS::DisplayControl` | Display resolution changes | MS-RDPEDISP |
| `Microsoft::Windows::RDS::Geometry::v08.01` | Graphics geometry | MS-RDPEGFX |
| `Microsoft::Windows::RDS::GraphicsPipeline` | Graphics pipeline | MS-RDPEGFX |
| `Microsoft::Windows::RDS::Input` | Input extensions | MS-RDPEI |
| `Microsoft::Windows::RDS::AudioInput` | Audio input capture | MS-RDPEAI |

## Error Handling

| Result Code | Description |
|-------------|-------------|
| 0x00000000 | Success |
| 0xC0000001 | General failure |
| 0xC0000002 | No listener |
| 0xC0000003 | Access denied |
| 0xC0000004 | Bad channel name |

## Test Coverage

Current coverage: **55.1%**

```bash
go test -cover ./internal/protocol/drdynvc/...
```

## References

- **MS-RDPEDYC** - Dynamic Channel Virtual Channel Extension
  - https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-rdpedyc/
- **MS-RDPEGFX** - Graphics Pipeline Extension (uses DVC)
- **MS-RDPEDISP** - Display Control (uses DVC)
- **MS-RDPEI** - Input Extension (uses DVC)

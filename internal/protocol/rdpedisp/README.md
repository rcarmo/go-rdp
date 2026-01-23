# internal/protocol/rdpedisp

Display Control Virtual Channel Extension per MS-RDPEDISP.

## Overview

This package implements the Display Control Virtual Channel Extension, which allows:
- **Dynamic resolution changes** - Client requests display resolution changes
- **Monitor configuration** - Multi-monitor layout negotiation
- **Orientation support** - Screen rotation (landscape/portrait)

RDPEDISP is transported over the Dynamic Virtual Channel (`drdynvc`).

## Specification Reference

- **MS-RDPEDISP** - Remote Desktop Protocol: Display Control Virtual Channel Extension
  - https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-rdpedisp/

## Files

| File | Purpose |
|------|---------|
| `rdpedisp.go` | Protocol implementation and PDU definitions |
| `rdpedisp_test.go` | Unit tests |

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Web Browser                                  │
│                    (window resize event)                             │
└───────────────────────────────┬─────────────────────────────────────┘
                                │
┌───────────────────────────────▼─────────────────────────────────────┐
│                       RDPEDISP Handler                               │
│                  (formats resize request PDU)                        │
└───────────────────────────────┬─────────────────────────────────────┘
                                │
┌───────────────────────────────▼─────────────────────────────────────┐
│                  Dynamic Virtual Channel (drdynvc)                   │
│              Channel: "Microsoft::Windows::RDS::DisplayControl"      │
└───────────────────────────────┬─────────────────────────────────────┘
                                │
┌───────────────────────────────▼─────────────────────────────────────┐
│                         RDP Server                                   │
│                (resizes desktop, sends updates)                      │
└─────────────────────────────────────────────────────────────────────┘
```

## PDU Types

### Message Types

| Value | Type | Description |
|-------|------|-------------|
| 0x00000005 | DISPLAYCONTROL_PDU_TYPE_CAPS | Capability exchange |
| 0x00000002 | DISPLAYCONTROL_PDU_TYPE_MONITOR_LAYOUT | Monitor layout request |

### Capability PDU

```go
type CapsPDU struct {
    Type                   uint32  // DISPLAYCONTROL_PDU_TYPE_CAPS (0x05)
    Length                 uint32  // PDU length
    MaxNumMonitors         uint32  // Max monitors supported
    MaxMonitorAreaFactorA  uint32  // Max area factor A
    MaxMonitorAreaFactorB  uint32  // Max area factor B
}
```

### Monitor Layout PDU

```go
type MonitorLayoutPDU struct {
    Type        uint32           // DISPLAYCONTROL_PDU_TYPE_MONITOR_LAYOUT (0x02)
    Length      uint32           // PDU length
    MonitorCount uint32          // Number of monitors
    Monitors    []MonitorDef     // Monitor definitions
}
```

### Monitor Definition

```go
type MonitorDef struct {
    Flags           uint32  // Monitor flags (primary, etc.)
    Left            int32   // Left position
    Top             int32   // Top position
    Width           uint32  // Width in pixels
    Height          uint32  // Height in pixels
    PhysicalWidth   uint32  // Physical width in mm
    PhysicalHeight  uint32  // Physical height in mm
    Orientation     uint32  // 0=landscape, 90/180/270=rotation
    DesktopScaleFactor  uint32  // Scale factor (100-500%)
    DeviceScaleFactor   uint32  // Device scale factor
}
```

## Protocol Flow

### Capability Exchange

```
Client                              Server
   │                                   │
   │  (Channel created via drdynvc)    │
   │                                   │
   │  CAPS PDU                         │
   │  ◄────────────────────────────    │
   │                                   │
   │  CAPS PDU (response)              │
   │  ────────────────────────────►    │
```

### Display Resize

```
Client                              Server
   │                                   │
   │  MONITOR_LAYOUT PDU               │
   │  ────────────────────────────►    │
   │                                   │
   │  (Server resizes desktop)         │
   │                                   │
   │  Graphics updates (new resolution)│
   │  ◄────────────────────────────    │
```

## Flags

### Monitor Flags

| Flag | Value | Description |
|------|-------|-------------|
| Primary | 0x00000001 | This is the primary monitor |

### Orientation Values

| Value | Orientation |
|-------|-------------|
| 0 | Landscape (0°) |
| 90 | Portrait (90° clockwise) |
| 180 | Landscape (180° rotation) |
| 270 | Portrait (90° counter-clockwise) |

## Usage

### Creating a Resize Request

```go
// Create single-monitor layout for 1920x1080
monitors := []rdpedisp.MonitorDef{
    {
        Flags:   rdpedisp.FlagPrimary,
        Left:    0,
        Top:     0,
        Width:   1920,
        Height:  1080,
        Orientation: 0,
        DesktopScaleFactor: 100,
        DeviceScaleFactor:  100,
    },
}

pdu := rdpedisp.NewMonitorLayoutPDU(monitors)
data, err := pdu.Serialize()
// Send via drdynvc channel
```

### Handling Server Capabilities

```go
func handleDisplayControlPDU(data []byte) {
    pduType := binary.LittleEndian.Uint32(data[0:4])
    
    switch pduType {
    case rdpedisp.PDUTypeCaps:
        caps, err := rdpedisp.ParseCapsPDU(data)
        if err != nil {
            return
        }
        // Store server capabilities
        maxMonitors = caps.MaxNumMonitors
        // Send our caps response
        response := rdpedisp.NewCapsPDU(1, 2560, 2560)
        SendToServer(response.Serialize())
    }
}
```

### Multi-Monitor Configuration

```go
// Create dual-monitor layout
monitors := []rdpedisp.MonitorDef{
    {
        Flags:   rdpedisp.FlagPrimary,
        Left:    0,
        Top:     0,
        Width:   1920,
        Height:  1080,
    },
    {
        Flags:   0, // Secondary
        Left:    1920,
        Top:     0,
        Width:   1920,
        Height:  1080,
    },
}
```

## Constraints

Per MS-RDPEDISP specification:

- **Maximum monitors**: Determined by server capability
- **Maximum area**: `MaxMonitorAreaFactorA * MaxMonitorAreaFactorB` pixels
- **Minimum size**: 200x200 pixels per monitor
- **Maximum size**: 8192x8192 pixels per monitor (typical)

## Test Coverage

Current coverage: **88.4%**

```bash
go test -cover ./internal/protocol/rdpedisp/...
```

## References

- **MS-RDPEDISP** - Display Control Virtual Channel Extension
  - https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-rdpedisp/
- **MS-RDPEDYC** - Dynamic Channel Virtual Channel Extension (transport layer)
- **MS-RDPBCGR** Section 2.2.1.3.6 - Monitor capability

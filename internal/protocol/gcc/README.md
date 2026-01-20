# internal/protocol/gcc

ITU T.124 Generic Conference Control (GCC) implementation.

## Overview

This package implements the GCC protocol layer used during RDP connection establishment. GCC provides a standardized mechanism for exchanging user data during MCS connection setup.

GCC is defined by ITU-T T.124 and uses PER (Packed Encoding Rules) for serialization.

## Files

| File | Purpose |
|------|---------|
| `conference_create_request.go` | Client conference request with user data |
| `conference_create_response.go` | Server conference response |
| `gcc_test.go` | Unit tests |

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                        MCS Connect Initial                           │
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │                  GCC Conference Create Request                   ││
│  │                                                                  ││
│  │  ┌─────────────────────────────────────────────────────────────┐││
│  │  │                     PER Encoded Data                        │││
│  │  │                                                             │││
│  │  │  - T.124 OID: 0.0.20.124.0.1                                │││
│  │  │  - H.221 Key: "Duca" (Microsoft)                            │││
│  │  │  - User Data Blocks:                                        │││
│  │  │    • Client Core Data                                       │││
│  │  │    • Client Security Data                                   │││
│  │  │    • Client Network Data                                    │││
│  │  │    • Client Cluster Data                                    │││
│  │  └─────────────────────────────────────────────────────────────┘││
│  └─────────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────────┘
```

## Protocol Flow

```
Client                                  Server
   │                                      │
   │  MCS Connect Initial                 │
   │  └── GCC Conference Create Request   │
   │      └── Client User Data            │
   │  ────────────────────────────────►   │
   │                                      │
   │  MCS Connect Response                │
   │  └── GCC Conference Create Response  │
   │      └── Server User Data            │
   │  ◄────────────────────────────────   │
```

## Conference Create Request

### Structure

```go
type ConferenceCreateRequest struct {
    // PER-encoded wrapper
    ObjectIdentifier []byte   // T.124 OID
    ConnectGCCPDU    []byte   // Encoded user data
}
```

### T.124 Object Identifier

```go
// OID: 0.0.20.124.0.1 (T.124 protocol identifier)
var T124OID = []byte{0x00, 0x00, 0x14, 0x7C, 0x00, 0x01}
```

### H.221 Non-Standard Key

Microsoft uses "Duca" as the H.221 manufacturer key:

```go
var H221Key = []byte{'D', 'u', 'c', 'a'}
```

### User Data Types

| Type | ID | Description |
|------|-----|-------------|
| CS_CORE | 0xC001 | Client Core Data |
| CS_SECURITY | 0xC002 | Security settings |
| CS_NET | 0xC003 | Network channel list |
| CS_CLUSTER | 0xC004 | Cluster settings |
| SC_CORE | 0x0C01 | Server Core Data |
| SC_SECURITY | 0x0C02 | Server security |
| SC_NET | 0x0C03 | Server network |

## Usage

### Creating Conference Request

```go
// Build user data blocks
coreData := BuildClientCoreData(width, height, colorDepth)
securityData := BuildClientSecurityData()
networkData := BuildClientNetworkData(channels)

// Combine user data
userData := append(coreData, securityData...)
userData = append(userData, networkData...)

// Create conference request
request := gcc.NewConferenceCreateRequest(userData)
encoded := request.Serialize()
```

### Parsing Conference Response

```go
// Parse server response
response, err := gcc.ParseConferenceCreateResponse(data)

// Extract server user data blocks
for _, block := range response.UserDataBlocks {
    switch block.Type {
    case SC_CORE:
        parseServerCoreData(block.Data)
    case SC_NET:
        parseServerNetworkData(block.Data)
    }
}
```

## PER Encoding Details

GCC uses ASN.1 PER (Packed Encoding Rules):

### Choice Encoding

```
┌────────────────┐
│ Choice index   │  (indicates which alternative)
│ (1 byte)       │
├────────────────┤
│ Encoded value  │
└────────────────┘
```

### Length Encoding

```
Length < 128:    [length]
Length < 16384:  [0x80 | high] [low]
Length >= 16384: Fragmented
```

### Object Identifier

```
┌────────────────┐
│ Length         │
├────────────────┤
│ OID components │
│ (arc encoding) │
└────────────────┘
```

## Wire Format

### Request PDU

```
00 05                     // ConnectData choice + length indicator
00 14 7c 00 01            // T.124 OID
00 01                     // Connect-Initial choice
00 01                     // Conference name length
00                        // Conference name
00 00 00 00 00 00 00 00   // Optional fields
44 75 63 61               // "Duca" H.221 key
xx xx                     // User data length
[user data...]            // Client user data blocks
```

### Response PDU

```
00 05                     // ConnectData choice
00 14 7c 00 01            // T.124 OID
14                        // Conference-Create-Response choice
7f                        // Result (success)
[optional tag]
00 01                     // User data set count
44 75 63 61               // "Duca" H.221 key
xx xx                     // User data length
[user data...]            // Server user data blocks
```

## Design Notes

### Why GCC?

GCC provides a standardized way to exchange arbitrary user data during connection. This allows Microsoft to embed proprietary RDP settings while maintaining ITU T.120 compliance.

### H.221 Key Purpose

The "Duca" key identifies Microsoft as the data format provider, allowing non-Microsoft implementations to skip unknown data blocks.

## References

- **ITU-T T.124** - Generic Conference Control
- **ITU-T X.691** - PER Encoding Rules
- **MS-RDPBCGR** Section 2.2.1 - Connection Sequence

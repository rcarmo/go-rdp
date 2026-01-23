# internal/protocol/encoding

ASN.1 encoding utilities for RDP protocol layers.

## Specification References

- [ITU-T X.690](https://www.itu.int/rec/T-REC-X.690) - ASN.1 BER/DER Encoding Rules
- [ITU-T X.691](https://www.itu.int/rec/T-REC-X.691) - ASN.1 PER Encoding Rules
- [MS-RDPBCGR Section 2.2.1.3](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-rdpbcgr/db6713ee-1c0e-4064-a3b3-0fac30b4037b) - PER-encoded GCC data

## Overview

This package implements BER (Basic Encoding Rules) and PER (Packed Encoding Rules) encoding as specified in ITU X.690 and X.691. These encodings are used by various RDP protocol layers:

- **BER** - Used by X.224, MCS for connection setup
- **PER** - Used by GCC for conference control data

## Files

| File | Purpose |
|------|---------|
| `asn1.go` | ASN.1 tag constants and definitions |
| `ber.go` | BER encoding/decoding functions |
| `per.go` | PER encoding/decoding functions |
| `encoding_test.go` | Unit tests |

## BER (Basic Encoding Rules)

### TLV Structure

BER uses Tag-Length-Value encoding:

```
┌─────────┬─────────┬─────────────────┐
│   Tag   │ Length  │     Value       │
│ (1+ B)  │ (1+ B)  │   (variable)    │
└─────────┴─────────┴─────────────────┘
```

### Tag Types

```go
const (
    BerTagBoolean         = 0x01
    BerTagInteger         = 0x02
    BerTagBitString       = 0x03
    BerTagOctetString     = 0x04
    BerTagObjectId        = 0x06
    BerTagEnumerated      = 0x0A
    BerTagSequence        = 0x30
    BerTagSequenceOf      = 0x30
)
```

### Length Encoding

| Form | Description |
|------|-------------|
| Short | Length < 128: single byte |
| Long | Length ≥ 128: first byte = 0x80 + N, followed by N length bytes |

```go
// Read BER length
length, err := BerReadLength(reader)

// Parse length bytes
if size&0x80 > 0 {
    numBytes := size &^ 0x80
    // Read numBytes for actual length
}
```

### Key Functions

```go
// Read application tag
tag, length, err := BerReadApplicationTag(reader, expectedTag)

// Read universal tag
tag, length, err := BerReadUniversalTag(reader, expectedTag, primitive)

// Read length field
length, err := BerReadLength(reader)

// Write functions use binary.Write with BigEndian
```

## PER (Packed Encoding Rules)

PER provides more compact encoding than BER, used for GCC T.124 data.

### Encoding Modes

| Mode | Description |
|------|-------------|
| Aligned | Byte-aligned encoding (used by RDP) |
| Unaligned | Bit-packed encoding |

### Key Functions

```go
// Read PER choice
choice, err := PerReadChoice(reader)

// Read PER integer with length
value, err := PerReadInteger(reader, minBytes)

// Read PER length determinant
length, err := PerReadLength(reader)

// Read PER object identifier
oid, err := PerReadObjectIdentifier(reader)

// Read PER octet string
data, err := PerReadOctetStream(reader, minLength)

// Read number of set items
count, err := PerReadNumberOfSet(reader)
```

### Object Identifiers

```go
// T.124 OID: 0.0.20.124.0.1
var T124OID = []byte{0x00, 0x00, 0x14, 0x7C, 0x00, 0x01}

// H.221 Non-Standard Key: Duca (Microsoft)
var H221Key = []byte{'D', 'u', 'c', 'a'}
```

### Length Determinant

PER length encoding:

| Length | Encoding |
|--------|----------|
| 0-127 | Single byte |
| 128-16383 | Two bytes, high bit set |
| ≥16384 | Fragmented |

```go
func PerReadLength(r io.Reader) (uint16, error) {
    var size uint8
    binary.Read(r, binary.BigEndian, &size)
    
    if size&0x80 > 0 {
        // Two-byte length
        var size2 uint8
        binary.Read(r, binary.BigEndian, &size2)
        return ((uint16(size) &^ 0x80) << 8) | uint16(size2), nil
    }
    return uint16(size), nil
}
```

## Usage in Protocol Layers

### X.224 Connection

```go
// Read X.224 data using BER
tag, len, _ := BerReadApplicationTag(r, X224Tag)
// ... process connection data
```

### MCS Connect Initial

```go
// Parse MCS PDU with BER
berTag, _ := BerReadApplicationTag(r, MCSConnectInitialTag)
// Read contained PER-encoded GCC data
```

### GCC Conference Create

```go
// Read GCC data with PER
choice, _ := PerReadChoice(r)
oid, _ := PerReadObjectIdentifier(r)
userData, _ := PerReadOctetStream(r, 0)
```

## Design Notes

### Why BER and PER?

RDP uses multiple ITU protocols that predate modern serialization formats:
- T.125 MCS uses BER for historical ITU compliance
- T.124 GCC uses PER for compactness
- Both are required for standards compliance

### Byte Order

- **BER**: Big-endian (network byte order)
- **PER**: Big-endian with bit-level packing

### Error Handling

Functions return errors for:
- Unexpected tags
- Invalid lengths
- Truncated data
- Malformed encodings

## References

- **ITU-T X.680** - ASN.1 Basic Notation
- **ITU-T X.690** - BER, CER, DER Encoding Rules
- **ITU-T X.691** - PER Encoding Rules
- **ITU-T T.124** - GCC Protocol (uses PER)
- **ITU-T T.125** - MCS Protocol (uses BER)

# Documentation

This folder contains the long-form documentation for the project.

## Contents

- [Architecture](ARCHITECTURE.md) - System design, data flow, and protocol details
- [Configuration](configuration.md) - Environment variables, command-line flags, and settings
- [Debugging](debugging.md) - Logging, capabilities, and troubleshooting guide
- [NSCodec](NSCODEC.md) - Bitmap codec implementation details
- [RemoteFX](REMOTEFX.md) - RemoteFX wavelet codec implementation
- [UDP Transport](UDP.md) - UDP transport extension (MS-RDPEUDP/MS-RDPEMT)
- [WebGL](webgl.md) - WebGL rendering implementation

## Package Documentation

Each Go package has its own README with implementation details:

### Core Packages

- `internal/auth/` - NTLM/CredSSP authentication
- `internal/codec/` - Bitmap compression (RLE, NSCodec)
- `internal/codec/rfx/` - RemoteFX wavelet codec
- `internal/config/` - Configuration management
- `internal/handler/` - WebSocket connection handling
- `internal/logging/` - Leveled logging system
- `internal/rdp/` - RDP client implementation

### Protocol Packages

- `internal/protocol/` - Protocol layer overview
- `internal/protocol/audio/` - Audio virtual channel (MS-RDPEA)
- `internal/protocol/drdynvc/` - Dynamic virtual channels (MS-RDPEDYC)
- `internal/protocol/encoding/` - BER/PER ASN.1 encoding
- `internal/protocol/fastpath/` - FastPath optimization
- `internal/protocol/gcc/` - Generic Conference Control (T.124)
- `internal/protocol/mcs/` - Multi-Channel Service (T.125)
- `internal/protocol/pdu/` - RDP Protocol Data Units
- `internal/protocol/rdpedisp/` - Display control (MS-RDPEDISP)
- `internal/protocol/rdpemt/` - Multitransport (MS-RDPEMT)
- `internal/protocol/rdpeudp/` - UDP transport packets (MS-RDPEUDP)
- `internal/protocol/tpkt/` - TPKT framing (RFC 1006)
- `internal/protocol/x224/` - Connection layer (ISO 8073)

### Transport Packages

- `internal/transport/udp/` - UDP transport layer

## JavaScript Modules

- `web/src/js/` - Browser client modules
  - `wasm.js` - WASM codec wrapper (WASMCodec, RFXDecoder)
  - `codec-fallback.js` - Pure JS fallback codecs
  - `graphics.js` - Canvas rendering
  - `audio.js` - Audio redirection
  - `session.js` - Connection management

## Protocol References

Microsoft Open Specifications:

| Protocol | Description | Link |
|----------|-------------|------|
| MS-RDPBCGR | Basic Connectivity and Graphics Remoting | [Spec](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-rdpbcgr/) |
| MS-RDPEA | Audio Output Virtual Channel Extension | [Spec](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-rdpea/) |
| MS-RDPEDYC | Dynamic Channel Virtual Channel Extension | [Spec](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-rdpedyc/) |
| MS-RDPEDISP | Display Control Virtual Channel Extension | [Spec](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-rdpedisp/) |
| MS-RDPEMT | Multitransport Extension | [Spec](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-rdpemt/) |
| MS-RDPEUDP | UDP Transport Extension | [Spec](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-rdpeudp/) |
| MS-RDPEUDP2 | UDP Transport Extension Version 2 | [Spec](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-rdpeudp2/) |

Other Standards:

- **ITU T.124** - Generic Conference Control
- **ITU T.125** - Multi-Channel Service Protocol
- **ISO 8073** - Connection-Oriented Transport Protocol (X.224)
- **RFC 1006** - ISO Transport Service on top of TCP

## Testing & Spec Compliance

The test suite includes Microsoft Protocol Test Suite validation tests based on:
- [WindowsProtocolTestSuites](https://github.com/microsoft/WindowsProtocolTestSuites/tree/main/TestSuites/RDP/Client/docs)

| Test Suite | Coverage |
|------------|----------|
| MS-RDPBCGR | S1_Connection, S4_SlowPathInput, S5_FastPathInput, S7_VirtualChannel, S10_FastPathOutput |
| MS-RDPEDISP | Monitor layout validation, orientation, resolution |
| MS-RDPRFX | Frame structures, tile encoding, RLGR modes |
| MS-RDPEUDP | SYN, ACK, MTU, retransmission |
| MS-RDPEMT | Tunnel header, request/response |
| MS-NLMP | NTLM message types, negotiate flags |

Run tests with: `make test`

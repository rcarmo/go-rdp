# internal/protocol/audio

RDP audio virtual channel implementation.

## Specification References

- [MS-RDPEA](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-rdpea/) - Remote Desktop Protocol: Audio Output Virtual Channel Extension
- [MS-RDPEAI](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-rdpeai/) - Remote Desktop Protocol: Audio Input Redirection Virtual Channel Extension

## Overview

This package implements the audio redirection virtual channel for RDP, supporting both:
- **MS-RDPEA** - Remote Desktop Protocol: Audio Output Virtual Channel Extension
- **MS-RDPEAI** - Remote Desktop Protocol: Audio Input Redirection Virtual Channel Extension

It enables remote desktop audio to be streamed to the client browser.

## Files

| File | Purpose |
|------|---------|
| `channel.go` | Virtual channel PDU framing and defragmentation |
| `channel_test.go` | Channel tests |
| `rdpsnd.go` | RDPSND protocol messages (audio formats, wave data) |
| `rdpsnd_test.go` | RDPSND tests |

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                         RDP Server                                   │
│                    (Audio Output Stream)                             │
└───────────────────────────────┬─────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    Virtual Channel Layer                             │
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │                 ChannelDefragmenter                             ││
│  │         (reassembles fragmented channel PDUs)                   ││
│  └─────────────────────────────────────────────────────────────────┘│
└───────────────────────────────┬─────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      RDPSND Protocol                                 │
│                                                                      │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐ │
│  │  Server     │  │   Client    │  │   Wave      │  │   Wave2     │ │
│  │  Formats    │  │   Formats   │  │   Info      │  │   PDU       │ │
│  │  (caps)     │  │   (ack)     │  │   (header)  │  │   (data)    │ │
│  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘ │
└───────────────────────────────┬─────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────────┐
│                       WebSocket Client                               │
│                    (Browser Audio API)                               │
└─────────────────────────────────────────────────────────────────────┘
```

## Virtual Channel Protocol

### Channel PDU Header

```go
type ChannelPDUHeader struct {
    Length uint32  // Total uncompressed length
    Flags  uint32  // CHANNEL_FLAG_* values
}
```

### Channel Flags

| Flag | Value | Description |
|------|-------|-------------|
| `CHANNEL_FLAG_FIRST` | 0x01 | First chunk in message |
| `CHANNEL_FLAG_LAST` | 0x02 | Last chunk in message |
| `CHANNEL_FLAG_ONLY` | 0x03 | Single-chunk message |
| `CHANNEL_FLAG_SHOW_PROTOCOL` | 0x10 | Show protocol header |

### Defragmentation

Large audio data is split across multiple PDUs:

```go
defrag := NewChannelDefragmenter()

for {
    chunk := receiveChannelChunk()
    complete, data := defrag.Process(chunk)
    if complete {
        processAudioPDU(data)
    }
}
```

## RDPSND Protocol

### Message Types

| Type | Value | Direction | Description |
|------|-------|-----------|-------------|
| `SNDC_FORMATS` | 0x07 | S→C | Server audio format list |
| `SNDC_TRAINING` | 0x06 | S→C | Training (latency test) |
| `SNDC_WAVE` | 0x02 | S→C | Audio data (deprecated) |
| `SNDC_WAVE2` | 0x0D | S→C | Audio data (current) |
| `SNDC_CLOSE` | 0x01 | S→C | Close audio channel |
| `SNDC_FORMATS` | 0x07 | C→S | Client format response |
| `SNDC_TRAINING_CONFIRM` | 0x06 | C→S | Training acknowledgment |
| `SNDC_WAVE_CONFIRM` | 0x05 | C→S | Wave playback confirm |

### PDU Header

```go
type PDUHeader struct {
    MsgType  uint8   // Message type
    Reserved uint8   // Padding
    BodySize uint16  // Body length
}
```

### Audio Format

```go
type AudioFormat struct {
    FormatTag      uint16  // WAVE_FORMAT_PCM = 1
    Channels       uint16  // 1 = mono, 2 = stereo
    SamplesPerSec  uint32  // 44100, 48000, etc.
    AvgBytesPerSec uint32  // SamplesPerSec * BlockAlign
    BlockAlign     uint16  // Channels * BitsPerSample / 8
    BitsPerSample  uint16  // 8, 16, etc.
    ExtraDataSize  uint16  // Codec-specific data length
    ExtraData      []byte  // Codec-specific data
}
```

### Server Audio Formats

```go
type ServerAudioFormats struct {
    Flags              uint32  // Capability flags
    Volume             uint32  // Initial volume
    Pitch              uint32  // Initial pitch
    DGramPort          uint16  // UDP port (unused)
    NumFormats         uint16  // Number of formats
    LastBlockConfirmed uint8   // Initial block confirmation counter
    Version            uint16  // Protocol version
    Pad                uint8   // Unused padding
    Formats            []AudioFormat
}
```

### Wave2 PDU (Audio Data)

```go
type Wave2PDU struct {
    Timestamp      uint16  // Playback timestamp
    FormatNo       uint16  // Format index
    BlockNo        uint8   // Block number
    AudioData      []byte  // PCM audio samples
}
```

## Audio Flow

### Format Negotiation

```
Server                              Client
   │                                   │
   │  SNDC_FORMATS (supported list)    │
   │  ─────────────────────────────►   │
   │                                   │
   │  SNDC_FORMATS (selected formats)  │
   │  ◄─────────────────────────────   │
   │                                   │
   │  SNDC_TRAINING                    │
   │  ─────────────────────────────►   │
   │                                   │
   │  SNDC_TRAINING_CONFIRM            │
   │  ◄─────────────────────────────   │
```

### Audio Streaming

```
Server                              Client
   │                                   │
   │  SNDC_WAVE2 (audio data)          │
   │  ─────────────────────────────►   │
   │                                   │
   │  SNDC_WAVE_CONFIRM                │
   │  ◄─────────────────────────────   │
   │        (optional)                 │
```

## Usage

### Parsing Server Formats

```go
var formats ServerAudioFormats
err := formats.Deserialize(reader)

for _, fmt := range formats.Formats {
    if fmt.FormatTag == WAVE_FORMAT_PCM {
        // PCM audio supported
    }
}
```

### Parsing Wave Data

```go
var wave Wave2PDU
err := wave.Deserialize(reader)

// Forward to audio output
playAudio(wave.AudioData, formats.Formats[wave.FormatNo])
```

## Supported Formats

| Format | Tag | Description |
|--------|-----|-------------|
| PCM | 0x0001 | Uncompressed audio |
| ADPCM | 0x0002 | Adaptive differential PCM |
| GSM | 0x0031 | GSM 6.10 |
| AAC | 0x00FF | Advanced Audio Coding |

Currently, the implementation focuses on PCM for maximum compatibility.

## References

- **MS-RDPEA** - Remote Desktop Protocol: Audio Output Virtual Channel Extension
- **MS-RDPEAI** - Remote Desktop Protocol: Audio Input Redirection Virtual Channel Extension

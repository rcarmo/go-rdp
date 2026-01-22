// Package audio implements RDP audio virtual channel protocols.
// MS-RDPEA: Remote Desktop Protocol Audio Output Virtual Channel Extension
// MS-RDPEAI: Remote Desktop Protocol Audio Input Virtual Channel Extension
package audio

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

// Channel names for audio
const (
	ChannelRDPSND = "rdpsnd" // Audio output (server -> client)
	ChannelAUDIN  = "audin"  // Audio input (client -> server) - dynamic virtual channel
)

// RDPSND message types (MS-RDPEA 2.2.2)
const (
	SND_CLOSE         = 0x01
	SND_WAVE          = 0x02
	SND_SET_VOLUME    = 0x03
	SND_SET_PITCH     = 0x04
	SND_WAVE_CONFIRM  = 0x05
	SND_TRAINING      = 0x06
	SND_FORMATS       = 0x07
	SND_CRYPT_KEY     = 0x08
	SND_WAVE_ENCRYPT  = 0x09
	SND_UDP_WAVE      = 0x0A
	SND_UDP_WAVE_LAST = 0x0B
	SND_QUALITYMODE   = 0x0C
	SND_WAVE2         = 0x0D
)

// Audio format tags (WAVE format identifiers)
const (
	WAVE_FORMAT_PCM        = 0x0001
	WAVE_FORMAT_ADPCM      = 0x0002
	WAVE_FORMAT_ALAW       = 0x0006
	WAVE_FORMAT_MULAW      = 0x0007
	WAVE_FORMAT_GSM610     = 0x0031
	WAVE_FORMAT_AAC        = 0x00FF
	WAVE_FORMAT_MPEGLAYER3 = 0x0055
)

// PDUHeader represents the RDPSND PDU header
type PDUHeader struct {
	MsgType  uint8
	Reserved uint8
	BodySize uint16
}

func (h *PDUHeader) Serialize() []byte {
	buf := make([]byte, 4)
	buf[0] = h.MsgType
	buf[1] = h.Reserved
	binary.LittleEndian.PutUint16(buf[2:4], h.BodySize)
	return buf
}

func (h *PDUHeader) Deserialize(r io.Reader) error {
	return binary.Read(r, binary.LittleEndian, h)
}

// AudioFormat represents an audio format descriptor
type AudioFormat struct {
	FormatTag        uint16
	Channels         uint16
	SamplesPerSec    uint32
	AvgBytesPerSec   uint32
	BlockAlign       uint16
	BitsPerSample    uint16
	ExtraDataSize    uint16
	ExtraData        []byte
}

func (f *AudioFormat) Serialize() []byte {
	size := 18 + len(f.ExtraData)
	buf := make([]byte, size)
	binary.LittleEndian.PutUint16(buf[0:2], f.FormatTag)
	binary.LittleEndian.PutUint16(buf[2:4], f.Channels)
	binary.LittleEndian.PutUint32(buf[4:8], f.SamplesPerSec)
	binary.LittleEndian.PutUint32(buf[8:12], f.AvgBytesPerSec)
	binary.LittleEndian.PutUint16(buf[12:14], f.BlockAlign)
	binary.LittleEndian.PutUint16(buf[14:16], f.BitsPerSample)
	binary.LittleEndian.PutUint16(buf[16:18], f.ExtraDataSize)
	if len(f.ExtraData) > 0 {
		copy(buf[18:], f.ExtraData)
	}
	return buf
}

func (f *AudioFormat) Deserialize(r io.Reader) error {
	if err := binary.Read(r, binary.LittleEndian, &f.FormatTag); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &f.Channels); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &f.SamplesPerSec); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &f.AvgBytesPerSec); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &f.BlockAlign); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &f.BitsPerSample); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &f.ExtraDataSize); err != nil {
		return err
	}
	if f.ExtraDataSize > 0 {
		f.ExtraData = make([]byte, f.ExtraDataSize)
		if _, err := io.ReadFull(r, f.ExtraData); err != nil {
			return err
		}
	}
	return nil
}

// String returns a human-readable format description
func (f *AudioFormat) String() string {
	var formatName string
	switch f.FormatTag {
	case WAVE_FORMAT_PCM:
		formatName = "PCM"
	case WAVE_FORMAT_ADPCM:
		formatName = "ADPCM"
	case WAVE_FORMAT_ALAW:
		formatName = "A-Law"
	case WAVE_FORMAT_MULAW:
		formatName = "µ-Law"
	case WAVE_FORMAT_AAC:
		formatName = "AAC"
	case WAVE_FORMAT_MPEGLAYER3:
		formatName = "MP3"
	default:
		formatName = fmt.Sprintf("0x%04X", f.FormatTag)
	}
	return fmt.Sprintf("%s %dHz %dch %dbit", formatName, f.SamplesPerSec, f.Channels, f.BitsPerSample)
}

// ServerAudioFormats represents the server's audio format list (SNDC_FORMATS)
type ServerAudioFormats struct {
	Flags              uint32
	Volume             uint32
	Pitch              uint32
	DGramPort          uint16
	NumFormats         uint16
	LastBlockConfirmed uint8
	Version            uint16
	Pad                uint8
	Formats            []AudioFormat
}

func (s *ServerAudioFormats) Deserialize(data []byte) error {
	if len(data) < 12 {
		return fmt.Errorf("server audio formats too short: %d", len(data))
	}
	r := bytes.NewReader(data)
	
	if err := binary.Read(r, binary.LittleEndian, &s.Flags); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &s.Volume); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &s.Pitch); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &s.DGramPort); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &s.NumFormats); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &s.LastBlockConfirmed); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &s.Version); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &s.Pad); err != nil {
		return err
	}
	
	s.Formats = make([]AudioFormat, s.NumFormats)
	for i := uint16(0); i < s.NumFormats; i++ {
		if err := s.Formats[i].Deserialize(r); err != nil {
			return fmt.Errorf("format %d: %w", i, err)
		}
	}
	
	return nil
}

// ClientAudioFormats represents the client's response with supported formats
type ClientAudioFormats struct {
	Flags              uint32
	Volume             uint32
	Pitch              uint32
	DGramPort          uint16
	NumFormats         uint16
	LastBlockConfirmed uint8
	Version            uint16
	Pad                uint8
	Formats            []AudioFormat
}

func (c *ClientAudioFormats) Serialize() []byte {
	var buf bytes.Buffer
	_ = binary.Write(&buf, binary.LittleEndian, c.Flags)
	_ = binary.Write(&buf, binary.LittleEndian, c.Volume)
	_ = binary.Write(&buf, binary.LittleEndian, c.Pitch)
	_ = binary.Write(&buf, binary.LittleEndian, c.DGramPort)
	_ = binary.Write(&buf, binary.LittleEndian, c.NumFormats)
	_ = binary.Write(&buf, binary.LittleEndian, c.LastBlockConfirmed)
	_ = binary.Write(&buf, binary.LittleEndian, c.Version)
	_ = binary.Write(&buf, binary.LittleEndian, c.Pad)

	for _, format := range c.Formats {
		buf.Write(format.Serialize())
	}

	return buf.Bytes()
}

// TrainingPDU represents SNDC_TRAINING
type TrainingPDU struct {
	Timestamp uint16
	PackSize  uint16
	Data      []byte
}

func (t *TrainingPDU) Deserialize(data []byte) error {
	if len(data) < 4 {
		return fmt.Errorf("training PDU too short")
	}
	r := bytes.NewReader(data)
	if err := binary.Read(r, binary.LittleEndian, &t.Timestamp); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &t.PackSize); err != nil {
		return err
	}
	if t.PackSize > 4 {
		t.Data = make([]byte, t.PackSize-4)
		if _, err := io.ReadFull(r, t.Data); err != nil {
			return err
		}
	}
	return nil
}

// TrainingConfirmPDU represents SNDC_TRAINING_CONFIRM (client response)
type TrainingConfirmPDU struct {
	Timestamp uint16
	PackSize  uint16
}

func (t *TrainingConfirmPDU) Serialize() []byte {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint16(buf[0:2], t.Timestamp)
	binary.LittleEndian.PutUint16(buf[2:4], t.PackSize)
	return buf
}

// WaveInfoPDU represents SNDC_WAVE (first part of audio data)
type WaveInfoPDU struct {
	Timestamp       uint16
	FormatNo        uint16
	BlockNo         uint8
	Padding         [3]byte
	InitialData     []byte // First 4 bytes of audio data
}

func (w *WaveInfoPDU) Deserialize(data []byte) error {
	if len(data) < 12 {
		return fmt.Errorf("wave info too short")
	}
	r := bytes.NewReader(data)
	if err := binary.Read(r, binary.LittleEndian, &w.Timestamp); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &w.FormatNo); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &w.BlockNo); err != nil {
		return err
	}
	if _, err := r.Read(w.Padding[:]); err != nil {
		return err
	}
	w.InitialData = make([]byte, 4)
	_, err := r.Read(w.InitialData)
	return err
}

// Wave2PDU represents SNDC_WAVE2 (simplified wave PDU)
type Wave2PDU struct {
	Timestamp uint16
	FormatNo  uint16
	BlockNo   uint8
	Padding   [3]byte
	DataSize  uint32
	Data      []byte
}

func (w *Wave2PDU) Deserialize(data []byte) error {
	if len(data) < 12 {
		return fmt.Errorf("wave2 PDU too short")
	}
	r := bytes.NewReader(data)
	if err := binary.Read(r, binary.LittleEndian, &w.Timestamp); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &w.FormatNo); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &w.BlockNo); err != nil {
		return err
	}
	if _, err := r.Read(w.Padding[:]); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &w.DataSize); err != nil {
		return err
	}

	remaining := len(data) - 12
	if remaining > 0 {
		w.Data = make([]byte, remaining)
		_, err := r.Read(w.Data)
		return err
	}
	return nil
}

// WaveConfirmPDU represents SNDC_WAVECONFIRM (client acknowledgment)
type WaveConfirmPDU struct {
	Timestamp     uint16
	ConfirmedBlock uint8
	Padding        uint8
}

func (w *WaveConfirmPDU) Serialize() []byte {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint16(buf[0:2], w.Timestamp)
	buf[2] = w.ConfirmedBlock
	buf[3] = w.Padding
	return buf
}

// ClosePDU represents SNDC_CLOSE
type ClosePDU struct {
}

// QualityModePDU represents SNDC_QUALITYMODE
type QualityModePDU struct {
	QualityMode uint16
}

func (q *QualityModePDU) Serialize() []byte {
	buf := make([]byte, 2)
	binary.LittleEndian.PutUint16(buf[0:2], q.QualityMode)
	return buf
}

// Quality mode constants
const (
	QualityModeDynamic = 0x0000
	QualityModeMedium  = 0x0001
	QualityModeHigh    = 0x0002
)

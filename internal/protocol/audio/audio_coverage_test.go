package audio

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAudioFormatDeserializeErrorPaths tests error paths in AudioFormat.Deserialize
func TestAudioFormatDeserializeErrorPaths(t *testing.T) {
	// Test truncated data at each field
	tests := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"only format tag", []byte{0x01, 0x00}},
		{"only format tag + channels", []byte{0x01, 0x00, 0x02, 0x00}},
		{"truncated samples", []byte{0x01, 0x00, 0x02, 0x00, 0x44, 0xAC}},
		{"missing avg bytes", []byte{0x01, 0x00, 0x02, 0x00, 0x44, 0xAC, 0x00, 0x00}},
		{"missing block align", []byte{0x01, 0x00, 0x02, 0x00, 0x44, 0xAC, 0x00, 0x00, 0x10, 0xB1, 0x02, 0x00}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			format := &AudioFormat{}
			err := format.Deserialize(bytes.NewReader(tt.data))
			assert.Error(t, err, "Expected error for %s", tt.name)
		})
	}
}

// TestAudioFormatStringAllFormats tests String method for all format types
func TestAudioFormatStringAllFormats(t *testing.T) {
	tests := []struct {
		formatTag uint16
		expected  string
	}{
		{WAVE_FORMAT_PCM, "PCM"},
		{WAVE_FORMAT_ADPCM, "ADPCM"},
		{WAVE_FORMAT_ALAW, "A-Law"},
		{WAVE_FORMAT_MULAW, "µ-Law"},
		{WAVE_FORMAT_AAC, "AAC"},
		{WAVE_FORMAT_MPEGLAYER3, "MP3"},
		{0xFFFF, "0xFFFF"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			format := &AudioFormat{
				FormatTag:     tt.formatTag,
				Channels:      2,
				SamplesPerSec: 44100,
				BitsPerSample: 16,
			}
			str := format.String()
			assert.Contains(t, str, tt.expected)
		})
	}
}

// TestChannelPDUHeaderDeserializeError tests error paths
func TestChannelPDUHeaderDeserializeError(t *testing.T) {
	// Empty data
	cd := &ChannelPDUHeader{}
	err := cd.Deserialize(bytes.NewReader([]byte{}))
	assert.Error(t, err)

	// Truncated data (only flags)
	err = cd.Deserialize(bytes.NewReader([]byte{0x01, 0x00, 0x00, 0x00}))
	assert.Error(t, err)
}

// TestTrainingPDUDeserializeError tests error paths
func TestTrainingPDUDeserializeError(t *testing.T) {
	// Empty data
	at := &TrainingPDU{}
	err := at.Deserialize([]byte{})
	assert.Error(t, err)

	// Truncated data (only wTimeStamp)
	err = at.Deserialize([]byte{0x01, 0x00})
	assert.Error(t, err)
}

// TestWave2PDUDeserializeError tests error paths
func TestWave2PDUDeserializeError(t *testing.T) {
	// Empty data
	wi := &Wave2PDU{}
	err := wi.Deserialize([]byte{})
	assert.Error(t, err)

	// Truncated data
	err = wi.Deserialize([]byte{0x01, 0x00})
	assert.Error(t, err)
}

// TestWaveInfoPDUDeserializeError tests error paths
func TestWaveInfoPDUDeserializeError(t *testing.T) {
	// Empty data
	wd := &WaveInfoPDU{}
	err := wd.Deserialize([]byte{})
	assert.Error(t, err)
}

// TestTrainingPDUDeserializeWithData tests training PDU with extra data
func TestTrainingPDUDeserializeWithData(t *testing.T) {
	// PackSize > 4 triggers data reading
	data := make([]byte, 12) 
	binary.LittleEndian.PutUint16(data[0:2], 1000) // Timestamp
	binary.LittleEndian.PutUint16(data[2:4], 8)    // PackSize = 8 (>4)
	// data[4:8] = extra data

	pdu := &TrainingPDU{}
	err := pdu.Deserialize(data)
	assert.NoError(t, err)
	assert.Equal(t, uint16(1000), pdu.Timestamp)
}

// TestTrainingPDUDeserializeTruncated tests truncated extra data
func TestTrainingPDUDeserializeTruncated(t *testing.T) {
	// PackSize > 4 but insufficient data
	data := make([]byte, 4)
	binary.LittleEndian.PutUint16(data[0:2], 1000) // Timestamp
	binary.LittleEndian.PutUint16(data[2:4], 100)  // PackSize = 100 (way more than data has)

	pdu := &TrainingPDU{}
	err := pdu.Deserialize(data)
	assert.Error(t, err)
}

// TestServerAudioFormatsDeserializeError tests error paths
func TestServerAudioFormatsDeserializeError(t *testing.T) {
	// Too short data
	saf := &ServerAudioFormats{}
	err := saf.Deserialize([]byte{0x01, 0x02})
	assert.Error(t, err)

	// Valid header but truncated format
	data := make([]byte, 12)
	data[8] = 1 // NumFormats = 1
	saf2 := &ServerAudioFormats{}
	err = saf2.Deserialize(data)
	assert.Error(t, err)
}

// TestAudioFormatDeserializeWithExtraData tests extradata path
func TestAudioFormatDeserializeWithExtraData(t *testing.T) {
	// Valid format with ExtraDataSize but truncated extra data
	data := []byte{
		0x01, 0x00, // FormatTag
		0x02, 0x00, // Channels
		0x44, 0xAC, 0x00, 0x00, // SamplesPerSec
		0x10, 0xB1, 0x02, 0x00, // AvgBytesPerSec
		0x04, 0x00, // BlockAlign
		0x10, 0x00, // BitsPerSample
		0x10, 0x00, // ExtraDataSize = 16
		// Missing extra data
	}
	format := &AudioFormat{}
	err := format.Deserialize(bytes.NewReader(data))
	assert.Error(t, err)
}

// TestWave2PDUDeserializeMore tests more error paths
func TestWave2PDUDeserializeMore(t *testing.T) {
	// Data with some fields but truncated
	data := []byte{
		0x01, 0x00, // wTimeStamp
		0x02, 0x00, // wFormatNo
		0x03,       // cBlockNo (truncated)
	}
	w := &Wave2PDU{}
	err := w.Deserialize(data)
	assert.Error(t, err)
}

// TestWaveInfoPDUDeserializeMore tests more error paths
func TestWaveInfoPDUDeserializeMore(t *testing.T) {
	// Data with some fields but truncated
	data := []byte{
		0x01, 0x00, // wTimeStamp
		0x02, 0x00, // wFormatNo
	}
	w := &WaveInfoPDU{}
	err := w.Deserialize(data)
	assert.Error(t, err)
}

// TestAudioFormatSerializeWithExtraData tests serialization with extra data
func TestAudioFormatSerializeWithExtraData(t *testing.T) {
	format := &AudioFormat{
		FormatTag:     WAVE_FORMAT_PCM,
		Channels:      2,
		SamplesPerSec: 44100,
		AvgBytesPerSec: 176400,
		BlockAlign:    4,
		BitsPerSample: 16,
		ExtraDataSize: 4,
		ExtraData:     []byte{0x01, 0x02, 0x03, 0x04},
	}
	data := format.Serialize()
	assert.Equal(t, 22, len(data)) // 18 + 4 extra
}


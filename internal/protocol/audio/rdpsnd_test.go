package audio

import (
	"bytes"
	"testing"
)

func TestPDUHeader_Serialize(t *testing.T) {
	tests := []struct {
		name     string
		header   PDUHeader
		expected []byte
	}{
		{
			name: "formats message",
			header: PDUHeader{
				MsgType:  SND_FORMATS,
				Reserved: 0,
				BodySize: 100,
			},
			expected: []byte{0x07, 0x00, 0x64, 0x00},
		},
		{
			name: "wave confirm",
			header: PDUHeader{
				MsgType:  SND_WAVE_CONFIRM,
				Reserved: 0,
				BodySize: 4,
			},
			expected: []byte{0x05, 0x00, 0x04, 0x00},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.header.Serialize()
			if !bytes.Equal(result, tt.expected) {
				t.Errorf("Serialize() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestPDUHeader_Deserialize(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		wantMsgType uint8
		wantSize    uint16
		wantErr     bool
	}{
		{
			name:        "valid header",
			data:        []byte{0x07, 0x00, 0x64, 0x00},
			wantMsgType: SND_FORMATS,
			wantSize:    100,
			wantErr:     false,
		},
		{
			name:    "too short",
			data:    []byte{0x04, 0x00},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var h PDUHeader
			err := h.Deserialize(bytes.NewReader(tt.data))
			if (err != nil) != tt.wantErr {
				t.Errorf("Deserialize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if h.MsgType != tt.wantMsgType {
					t.Errorf("MsgType = %v, want %v", h.MsgType, tt.wantMsgType)
				}
				if h.BodySize != tt.wantSize {
					t.Errorf("BodySize = %v, want %v", h.BodySize, tt.wantSize)
				}
			}
		})
	}
}

func TestAudioFormat_Serialize(t *testing.T) {
	format := AudioFormat{
		FormatTag:      WAVE_FORMAT_PCM,
		Channels:       2,
		SamplesPerSec:  44100,
		AvgBytesPerSec: 176400,
		BlockAlign:     4,
		BitsPerSample:  16,
		ExtraDataSize:  0,
	}

	result := format.Serialize()
	if len(result) != 18 {
		t.Errorf("Serialize() length = %d, want 18", len(result))
	}

	// Verify format tag (first 2 bytes)
	if result[0] != 0x01 || result[1] != 0x00 {
		t.Errorf("FormatTag = %v, want [0x01, 0x00]", result[0:2])
	}
}

func TestAudioFormat_Deserialize(t *testing.T) {
	// PCM 44100Hz stereo 16-bit
	data := []byte{
		0x01, 0x00, // FormatTag = PCM
		0x02, 0x00, // Channels = 2
		0x44, 0xAC, 0x00, 0x00, // SamplesPerSec = 44100
		0x10, 0xB1, 0x02, 0x00, // AvgBytesPerSec = 176400
		0x04, 0x00, // BlockAlign = 4
		0x10, 0x00, // BitsPerSample = 16
		0x00, 0x00, // ExtraDataSize = 0
	}

	var f AudioFormat
	err := f.Deserialize(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Deserialize() error = %v", err)
	}

	if f.FormatTag != WAVE_FORMAT_PCM {
		t.Errorf("FormatTag = %v, want %v", f.FormatTag, WAVE_FORMAT_PCM)
	}
	if f.Channels != 2 {
		t.Errorf("Channels = %v, want 2", f.Channels)
	}
	if f.SamplesPerSec != 44100 {
		t.Errorf("SamplesPerSec = %v, want 44100", f.SamplesPerSec)
	}
	if f.BitsPerSample != 16 {
		t.Errorf("BitsPerSample = %v, want 16", f.BitsPerSample)
	}
}

func TestAudioFormat_String(t *testing.T) {
	tests := []struct {
		name     string
		format   AudioFormat
		contains string
	}{
		{
			name: "PCM format",
			format: AudioFormat{
				FormatTag:     WAVE_FORMAT_PCM,
				Channels:      2,
				SamplesPerSec: 44100,
				BitsPerSample: 16,
			},
			contains: "PCM",
		},
		{
			name: "ADPCM format",
			format: AudioFormat{
				FormatTag:     WAVE_FORMAT_ADPCM,
				Channels:      1,
				SamplesPerSec: 22050,
				BitsPerSample: 4,
			},
			contains: "ADPCM",
		},
		{
			name: "Unknown format",
			format: AudioFormat{
				FormatTag:     0x9999,
				Channels:      1,
				SamplesPerSec: 8000,
				BitsPerSample: 8,
			},
			contains: "0x9999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.format.String()
			if !bytes.Contains([]byte(result), []byte(tt.contains)) {
				t.Errorf("String() = %v, want to contain %v", result, tt.contains)
			}
		})
	}
}

func TestServerAudioFormats_Deserialize(t *testing.T) {
	// Build a server audio formats packet
	data := []byte{
		0x00, 0x00, 0x00, 0x00, // Flags
		0x00, 0x00, 0x00, 0x00, // Volume
		0x00, 0x00, 0x00, 0x00, // Pitch
		0x00, 0x00, // DGramPort
		0x01, 0x00, // NumFormats = 1
		0x00,       // LastBlockConfirmed
		0x06, 0x00, // Version = 6
		0x00,       // Pad
		// Format 1: PCM 44100Hz stereo 16-bit
		0x01, 0x00, // FormatTag = PCM
		0x02, 0x00, // Channels = 2
		0x44, 0xAC, 0x00, 0x00, // SamplesPerSec = 44100
		0x10, 0xB1, 0x02, 0x00, // AvgBytesPerSec = 176400
		0x04, 0x00, // BlockAlign = 4
		0x10, 0x00, // BitsPerSample = 16
		0x00, 0x00, // ExtraDataSize = 0
	}

	var s ServerAudioFormats
	err := s.Deserialize(data)
	if err != nil {
		t.Fatalf("Deserialize() error = %v", err)
	}

	if s.Version != 6 {
		t.Errorf("Version = %v, want 6", s.Version)
	}
	if s.NumFormats != 1 {
		t.Errorf("NumFormats = %v, want 1", s.NumFormats)
	}
	if len(s.Formats) != 1 {
		t.Errorf("len(Formats) = %v, want 1", len(s.Formats))
	}
	if s.Formats[0].FormatTag != WAVE_FORMAT_PCM {
		t.Errorf("Formats[0].FormatTag = %v, want PCM", s.Formats[0].FormatTag)
	}
}

func TestServerAudioFormats_Deserialize_TooShort(t *testing.T) {
	data := []byte{0x06, 0x00, 0x00}

	var s ServerAudioFormats
	err := s.Deserialize(data)
	if err == nil {
		t.Error("Deserialize() should return error for short data")
	}
}

func TestClientAudioFormats_Serialize(t *testing.T) {
	c := ClientAudioFormats{
		Flags:              TSSNDCAPS_ALIVE,
		Volume:             0xFFFFFFFF,
		Pitch:              0x00010000,
		DGramPort:          0,
		NumFormats:         1,
		LastBlockConfirmed: 0,
		Version:            6,
		Pad:                0,
		Formats: []AudioFormat{
			{
				FormatTag:      WAVE_FORMAT_PCM,
				Channels:       2,
				SamplesPerSec:  44100,
				AvgBytesPerSec: 176400,
				BlockAlign:     4,
				BitsPerSample:  16,
				ExtraDataSize:  0,
			},
		},
	}

	result := c.Serialize()
	// Header (20 bytes) + 1 format (18 bytes) = 38 bytes
	if len(result) != 38 {
		t.Errorf("Serialize() length = %d, want 38", len(result))
	}
}

func TestTrainingPDU_Deserialize(t *testing.T) {
	data := []byte{
		0x10, 0x27, // Timestamp = 10000
		0x08, 0x00, // PackSize = 8
		0x01, 0x02, 0x03, 0x04, // Data
	}

	var tr TrainingPDU
	err := tr.Deserialize(data)
	if err != nil {
		t.Fatalf("Deserialize() error = %v", err)
	}

	if tr.Timestamp != 10000 {
		t.Errorf("Timestamp = %v, want 10000", tr.Timestamp)
	}
	if tr.PackSize != 8 {
		t.Errorf("PackSize = %v, want 8", tr.PackSize)
	}
}

func TestTrainingConfirmPDU_Serialize(t *testing.T) {
	tc := TrainingConfirmPDU{
		Timestamp: 10000,
		PackSize:  8,
	}

	result := tc.Serialize()
	expected := []byte{0x10, 0x27, 0x08, 0x00}
	if !bytes.Equal(result, expected) {
		t.Errorf("Serialize() = %v, want %v", result, expected)
	}
}

func TestWaveInfoPDU_Deserialize(t *testing.T) {
	data := []byte{
		0x10, 0x27, // Timestamp = 10000
		0x00, 0x00, // FormatNo = 0
		0x01,             // BlockNo = 1
		0x00, 0x00, 0x00, // Padding
		0xAA, 0xBB, 0xCC, 0xDD, // InitialData
	}

	var w WaveInfoPDU
	err := w.Deserialize(data)
	if err != nil {
		t.Fatalf("Deserialize() error = %v", err)
	}

	if w.Timestamp != 10000 {
		t.Errorf("Timestamp = %v, want 10000", w.Timestamp)
	}
	if w.FormatNo != 0 {
		t.Errorf("FormatNo = %v, want 0", w.FormatNo)
	}
	if w.BlockNo != 1 {
		t.Errorf("BlockNo = %v, want 1", w.BlockNo)
	}
}

func TestWave2PDU_Deserialize(t *testing.T) {
	data := []byte{
		0x10, 0x27, // Timestamp = 10000
		0x00, 0x00, // FormatNo = 0
		0x01,             // BlockNo = 1
		0x00, 0x00, 0x00, // Padding
		0x04, 0x00, 0x00, 0x00, // DataSize = 4
		0xAA, 0xBB, 0xCC, 0xDD, // Data
	}

	var w Wave2PDU
	err := w.Deserialize(data)
	if err != nil {
		t.Fatalf("Deserialize() error = %v", err)
	}

	if w.Timestamp != 10000 {
		t.Errorf("Timestamp = %v, want 10000", w.Timestamp)
	}
	if w.DataSize != 4 {
		t.Errorf("DataSize = %v, want 4", w.DataSize)
	}
	if len(w.Data) != 4 {
		t.Errorf("len(Data) = %v, want 4", len(w.Data))
	}
}

func TestWaveConfirmPDU_Serialize(t *testing.T) {
	wc := WaveConfirmPDU{
		Timestamp:      10000,
		ConfirmedBlock: 1,
		Padding:        0,
	}

	result := wc.Serialize()
	expected := []byte{0x10, 0x27, 0x01, 0x00}
	if !bytes.Equal(result, expected) {
		t.Errorf("Serialize() = %v, want %v", result, expected)
	}
}

func TestQualityModePDU_Serialize(t *testing.T) {
	tests := []struct {
		name     string
		mode     uint16
		expected []byte
	}{
		{"dynamic", QualityModeDynamic, []byte{0x00, 0x00}},
		{"medium", QualityModeMedium, []byte{0x01, 0x00}},
		{"high", QualityModeHigh, []byte{0x02, 0x00}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := QualityModePDU{QualityMode: tt.mode}
			result := q.Serialize()
			if !bytes.Equal(result, tt.expected) {
				t.Errorf("Serialize() = %v, want %v", result, tt.expected)
			}
		})
	}
}

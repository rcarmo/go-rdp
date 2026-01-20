package audio

import (
	"bytes"
	"testing"
)

func TestChannelPDUHeader_Serialize(t *testing.T) {
	h := ChannelPDUHeader{
		Length: 100,
		Flags:  ChannelFlagFirst | ChannelFlagLast,
	}

	result := h.Serialize()
	if len(result) != 8 {
		t.Errorf("Serialize() length = %d, want 8", len(result))
	}

	// Verify length (first 4 bytes, little endian)
	if result[0] != 0x64 || result[1] != 0x00 || result[2] != 0x00 || result[3] != 0x00 {
		t.Errorf("Length bytes = %v, want [0x64, 0x00, 0x00, 0x00]", result[0:4])
	}

	// Verify flags (next 4 bytes)
	if result[4] != 0x03 || result[5] != 0x00 || result[6] != 0x00 || result[7] != 0x00 {
		t.Errorf("Flags bytes = %v, want [0x03, 0x00, 0x00, 0x00]", result[4:8])
	}
}

func TestChannelPDUHeader_Deserialize(t *testing.T) {
	data := []byte{
		0x64, 0x00, 0x00, 0x00, // Length = 100
		0x03, 0x00, 0x00, 0x00, // Flags = First | Last
	}

	var h ChannelPDUHeader
	err := h.Deserialize(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Deserialize() error = %v", err)
	}

	if h.Length != 100 {
		t.Errorf("Length = %v, want 100", h.Length)
	}
	if h.Flags != 0x03 {
		t.Errorf("Flags = %v, want 0x03", h.Flags)
	}
}

func TestChannelPDUHeader_Flags(t *testing.T) {
	tests := []struct {
		name       string
		flags      uint32
		isFirst    bool
		isLast     bool
		isComplete bool
	}{
		{"first only", ChannelFlagFirst, true, false, false},
		{"last only", ChannelFlagLast, false, true, false},
		{"complete", ChannelFlagFirst | ChannelFlagLast, true, true, true},
		{"middle", 0, false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := ChannelPDUHeader{Flags: tt.flags}
			if h.IsFirst() != tt.isFirst {
				t.Errorf("IsFirst() = %v, want %v", h.IsFirst(), tt.isFirst)
			}
			if h.IsLast() != tt.isLast {
				t.Errorf("IsLast() = %v, want %v", h.IsLast(), tt.isLast)
			}
			if h.IsComplete() != tt.isComplete {
				t.Errorf("IsComplete() = %v, want %v", h.IsComplete(), tt.isComplete)
			}
		})
	}
}

func TestChannelDefragmenter_SinglePacket(t *testing.T) {
	d := ChannelDefragmenter{}

	chunk := &ChannelChunk{
		Header: ChannelPDUHeader{
			Length: 4,
			Flags:  ChannelFlagFirst | ChannelFlagLast,
		},
		Data: []byte{0x01, 0x02, 0x03, 0x04},
	}

	data, complete := d.Process(chunk)
	if !complete {
		t.Error("Process() should return complete=true for single packet")
	}
	if !bytes.Equal(data, []byte{0x01, 0x02, 0x03, 0x04}) {
		t.Errorf("Process() data = %v, want [0x01, 0x02, 0x03, 0x04]", data)
	}
}

func TestChannelDefragmenter_FragmentedPacket(t *testing.T) {
	d := ChannelDefragmenter{}

	// First chunk
	chunk1 := &ChannelChunk{
		Header: ChannelPDUHeader{
			Length: 8,
			Flags:  ChannelFlagFirst,
		},
		Data: []byte{0x01, 0x02, 0x03, 0x04},
	}

	data, complete := d.Process(chunk1)
	if complete {
		t.Error("Process() should return complete=false for first fragment")
	}
	if data != nil {
		t.Errorf("Process() data should be nil for incomplete, got %v", data)
	}

	// Middle chunk (no first/last flags)
	chunk2 := &ChannelChunk{
		Header: ChannelPDUHeader{
			Length: 8,
			Flags:  0,
		},
		Data: []byte{0x05, 0x06},
	}

	data, complete = d.Process(chunk2)
	_ = data // Not used until complete
	if complete {
		t.Error("Process() should return complete=false for middle fragment")
	}

	// Last chunk
	chunk3 := &ChannelChunk{
		Header: ChannelPDUHeader{
			Length: 8,
			Flags:  ChannelFlagLast,
		},
		Data: []byte{0x07, 0x08},
	}

	data, complete = d.Process(chunk3)
	if !complete {
		t.Error("Process() should return complete=true for last fragment")
	}
	expected := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	if !bytes.Equal(data, expected) {
		t.Errorf("Process() data = %v, want %v", data, expected)
	}
}

func TestParseChannelData(t *testing.T) {
	data := []byte{
		0x64, 0x00, 0x00, 0x00, // Length = 100
		0x03, 0x00, 0x00, 0x00, // Flags = First | Last
		0x01, 0x02, 0x03, 0x04, // Payload
	}

	chunk, err := ParseChannelData(data)
	if err != nil {
		t.Fatalf("ParseChannelData() error = %v", err)
	}

	if chunk.Header.Length != 100 {
		t.Errorf("Header.Length = %v, want 100", chunk.Header.Length)
	}
	if !bytes.Equal(chunk.Data, []byte{0x01, 0x02, 0x03, 0x04}) {
		t.Errorf("Data = %v, want [0x01, 0x02, 0x03, 0x04]", chunk.Data)
	}
}

func TestParseChannelData_TooShort(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03}

	_, err := ParseChannelData(data)
	if err == nil {
		t.Error("ParseChannelData() should return error for short data")
	}
}

func TestBuildChannelData(t *testing.T) {
	payload := []byte{0x01, 0x02, 0x03, 0x04}
	result := BuildChannelData(payload)

	if len(result) != 12 {
		t.Errorf("BuildChannelData() length = %d, want 12", len(result))
	}

	// Verify length field = 4
	if result[0] != 0x04 || result[1] != 0x00 || result[2] != 0x00 || result[3] != 0x00 {
		t.Errorf("Length bytes = %v, want [0x04, 0x00, 0x00, 0x00]", result[0:4])
	}

	// Verify flags = First | Last = 0x03
	if result[4] != 0x03 || result[5] != 0x00 || result[6] != 0x00 || result[7] != 0x00 {
		t.Errorf("Flags bytes = %v, want [0x03, 0x00, 0x00, 0x00]", result[4:8])
	}

	// Verify payload
	if !bytes.Equal(result[8:], payload) {
		t.Errorf("Payload = %v, want %v", result[8:], payload)
	}
}

func TestBuildChannelPDU(t *testing.T) {
	body := []byte{0x01, 0x02}
	result := BuildChannelPDU(SND_FORMATS, body)

	// Expected: channel header (8) + PDU header (4) + body (2) = 14 bytes
	if len(result) != 14 {
		t.Errorf("BuildChannelPDU() length = %d, want 14", len(result))
	}

	// Parse back and verify
	chunk, err := ParseChannelData(result)
	if err != nil {
		t.Fatalf("ParseChannelData() error = %v", err)
	}

	// PDU should be complete
	if !chunk.Header.IsComplete() {
		t.Error("BuildChannelPDU() should create complete PDU")
	}

	// First byte of payload should be message type
	if chunk.Data[0] != SND_FORMATS {
		t.Errorf("MsgType = %v, want %v", chunk.Data[0], SND_FORMATS)
	}
}

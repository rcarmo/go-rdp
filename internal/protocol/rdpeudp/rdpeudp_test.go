package rdpeudp

import (
	"bytes"
	"testing"
)

func TestFECHeader_RoundTrip(t *testing.T) {
	h := &FECHeader{
		SnSourceAck:              0x12345678,
		SourceAckReceiveWindowSize: 64,
		Flags:                    FlagSYN | FlagACK,
	}

	data := h.Serialize()
	if len(data) != h.Size() {
		t.Errorf("Expected %d bytes, got %d", h.Size(), len(data))
	}

	h2 := &FECHeader{}
	if err := h2.Deserialize(data); err != nil {
		t.Fatalf("Deserialize failed: %v", err)
	}

	if h2.SnSourceAck != h.SnSourceAck {
		t.Errorf("SnSourceAck mismatch: 0x%X vs 0x%X", h2.SnSourceAck, h.SnSourceAck)
	}
	if h2.SourceAckReceiveWindowSize != h.SourceAckReceiveWindowSize {
		t.Errorf("ReceiveWindow mismatch: %d vs %d", h2.SourceAckReceiveWindowSize, h.SourceAckReceiveWindowSize)
	}
	if h2.Flags != h.Flags {
		t.Errorf("Flags mismatch: 0x%X vs 0x%X", h2.Flags, h.Flags)
	}
}

func TestFECHeader_Deserialize_TooShort(t *testing.T) {
	h := &FECHeader{}
	err := h.Deserialize(make([]byte, 4))
	if err == nil {
		t.Error("Expected error for short data")
	}
}

func TestFECHeader_Flags(t *testing.T) {
	tests := []struct {
		name   string
		flags  uint16
		isSYN  bool
		isFIN  bool
		isACK  bool
		isData bool
	}{
		{"SYN", FlagSYN, true, false, false, false},
		{"FIN", FlagFIN, false, true, false, false},
		{"ACK", FlagACK, false, false, true, false},
		{"DAT", FlagDAT, false, false, false, true},
		{"SYN|ACK", FlagSYN | FlagACK, true, false, true, false},
		{"DAT|ACK", FlagDAT | FlagACK, false, false, true, true},
		{"None", 0, false, false, false, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := &FECHeader{Flags: tc.flags}
			if h.IsSYN() != tc.isSYN {
				t.Errorf("IsSYN: expected %v, got %v", tc.isSYN, h.IsSYN())
			}
			if h.IsFIN() != tc.isFIN {
				t.Errorf("IsFIN: expected %v, got %v", tc.isFIN, h.IsFIN())
			}
			if h.IsACK() != tc.isACK {
				t.Errorf("IsACK: expected %v, got %v", tc.isACK, h.IsACK())
			}
			if h.IsData() != tc.isData {
				t.Errorf("IsData: expected %v, got %v", tc.isData, h.IsData())
			}
		})
	}
}

func TestSynData_RoundTrip(t *testing.T) {
	s := &SynData{
		SnInitialSequenceNumber: 12345,
		UpstreamMTU:             1200,
		DownstreamMTU:           1200,
	}

	data := s.Serialize()
	if len(data) != SynDataSize {
		t.Errorf("Expected %d bytes, got %d", SynDataSize, len(data))
	}

	s2 := &SynData{}
	if err := s2.Deserialize(data); err != nil {
		t.Fatalf("Deserialize failed: %v", err)
	}

	if s2.SnInitialSequenceNumber != s.SnInitialSequenceNumber {
		t.Error("SnInitialSequenceNumber mismatch")
	}
	if s2.UpstreamMTU != s.UpstreamMTU {
		t.Error("UpstreamMTU mismatch")
	}
	if s2.DownstreamMTU != s.DownstreamMTU {
		t.Error("DownstreamMTU mismatch")
	}
}

func TestSynData_Deserialize_TooShort(t *testing.T) {
	s := &SynData{}
	err := s.Deserialize(make([]byte, 4))
	if err == nil {
		t.Error("Expected error for short data")
	}
}

func TestAckVector_RoundTrip(t *testing.T) {
	a := &AckVector{
		AckVectorSize:     4,
		Reserved:          0,
		AckVectorElements: []uint8{0xFF, 0x0F, 0xF0, 0x55},
	}

	data := a.Serialize()
	if len(data) != a.Size() {
		t.Errorf("Expected %d bytes, got %d", a.Size(), len(data))
	}

	a2 := &AckVector{}
	if err := a2.Deserialize(data); err != nil {
		t.Fatalf("Deserialize failed: %v", err)
	}

	if a2.AckVectorSize != a.AckVectorSize {
		t.Error("AckVectorSize mismatch")
	}
	if !bytes.Equal(a2.AckVectorElements, a.AckVectorElements) {
		t.Error("AckVectorElements mismatch")
	}
}

func TestAckVector_Deserialize_TooShort(t *testing.T) {
	a := &AckVector{}
	err := a.Deserialize(make([]byte, 1))
	if err == nil {
		t.Error("Expected error for short data")
	}
}

func TestAckVector_Deserialize_ElementsTruncated(t *testing.T) {
	data := []byte{10, 0} // Claims 10 elements but has none
	a := &AckVector{}
	err := a.Deserialize(data)
	if err == nil {
		t.Error("Expected error for truncated elements")
	}
}

func TestSourcePayloadHeader_NoFEC(t *testing.T) {
	h := &SourcePayloadHeader{
		SnSourceStart: 100,
	}

	data := h.Serialize(false)
	if len(data) != 4 {
		t.Errorf("Expected 4 bytes, got %d", len(data))
	}

	h2 := &SourcePayloadHeader{}
	if err := h2.Deserialize(data, false); err != nil {
		t.Fatalf("Deserialize failed: %v", err)
	}

	if h2.SnSourceStart != h.SnSourceStart {
		t.Error("SnSourceStart mismatch")
	}
}

func TestSourcePayloadHeader_WithFEC(t *testing.T) {
	h := &SourcePayloadHeader{
		SnSourceStart: 100,
		SnCoded:       50,
		FECMode:       FECReliable,
	}

	data := h.Serialize(true)
	if len(data) != 9 {
		t.Errorf("Expected 9 bytes, got %d", len(data))
	}

	h2 := &SourcePayloadHeader{}
	if err := h2.Deserialize(data, true); err != nil {
		t.Fatalf("Deserialize failed: %v", err)
	}

	if h2.SnSourceStart != h.SnSourceStart {
		t.Error("SnSourceStart mismatch")
	}
	if h2.SnCoded != h.SnCoded {
		t.Error("SnCoded mismatch")
	}
	if h2.FECMode != h.FECMode {
		t.Error("FECMode mismatch")
	}
}

func TestSourcePayloadHeader_Deserialize_TooShort(t *testing.T) {
	h := &SourcePayloadHeader{}

	// Too short for non-FEC
	err := h.Deserialize(make([]byte, 2), false)
	if err == nil {
		t.Error("Expected error for short non-FEC data")
	}

	// Too short for FEC
	err = h.Deserialize(make([]byte, 6), true)
	if err == nil {
		t.Error("Expected error for short FEC data")
	}
}

func TestPacket_SYN(t *testing.T) {
	p := NewSYNPacket(12345, DefaultMTU, DefaultMTU)

	if !p.Header.IsSYN() {
		t.Error("Expected SYN flag")
	}
	if p.Header.IsACK() {
		t.Error("Unexpected ACK flag")
	}
	if p.SynData == nil {
		t.Fatal("Expected SynData")
	}
	if p.SynData.SnInitialSequenceNumber != 12345 {
		t.Error("Wrong initial sequence number")
	}

	// Test round-trip
	data := p.Serialize()
	p2 := &Packet{}
	if err := p2.Deserialize(data); err != nil {
		t.Fatalf("Deserialize failed: %v", err)
	}

	if !p2.Header.IsSYN() {
		t.Error("Lost SYN flag in round-trip")
	}
	if p2.SynData.SnInitialSequenceNumber != 12345 {
		t.Error("Lost sequence number in round-trip")
	}
}

func TestPacket_SYNACK(t *testing.T) {
	p := NewSYNACKPacket(100, 50, DefaultMTU, DefaultMTU)

	if !p.Header.IsSYN() {
		t.Error("Expected SYN flag")
	}
	if !p.Header.IsACK() {
		t.Error("Expected ACK flag")
	}
	if p.Header.SnSourceAck != 50 {
		t.Errorf("Expected SnSourceAck 50, got %d", p.Header.SnSourceAck)
	}

	// Test round-trip
	data := p.Serialize()
	p2 := &Packet{}
	if err := p2.Deserialize(data); err != nil {
		t.Fatalf("Deserialize failed: %v", err)
	}

	if !p2.Header.IsSYN() || !p2.Header.IsACK() {
		t.Error("Lost flags in round-trip")
	}
}

func TestPacket_ACK(t *testing.T) {
	p := NewACKPacket(42, 64)

	if p.Header.IsSYN() {
		t.Error("Unexpected SYN flag")
	}
	if !p.Header.IsACK() {
		t.Error("Expected ACK flag")
	}
	if p.Header.SnSourceAck != 42 {
		t.Errorf("Expected SnSourceAck 42, got %d", p.Header.SnSourceAck)
	}
	if p.Header.SourceAckReceiveWindowSize != 64 {
		t.Errorf("Expected window 64, got %d", p.Header.SourceAckReceiveWindowSize)
	}

	// Test round-trip
	data := p.Serialize()
	p2 := &Packet{}
	if err := p2.Deserialize(data); err != nil {
		t.Fatalf("Deserialize failed: %v", err)
	}

	if !p2.Header.IsACK() {
		t.Error("Lost ACK flag in round-trip")
	}
}

func TestPacket_Data(t *testing.T) {
	payload := []byte("Hello, RDPEUDP!")
	p := NewDataPacket(100, 50, payload, 64)

	if !p.Header.IsData() {
		t.Error("Expected DAT flag")
	}
	if !p.Header.IsACK() {
		t.Error("Expected ACK flag")
	}
	if p.DataHeader == nil {
		t.Fatal("Expected DataHeader")
	}
	if p.DataHeader.SnSourceStart != 100 {
		t.Error("Wrong sequence number")
	}
	if !bytes.Equal(p.Payload, payload) {
		t.Error("Payload mismatch")
	}

	// Test round-trip
	data := p.Serialize()
	p2 := &Packet{}
	if err := p2.Deserialize(data); err != nil {
		t.Fatalf("Deserialize failed: %v", err)
	}

	if !p2.Header.IsData() {
		t.Error("Lost DAT flag in round-trip")
	}
	if !bytes.Equal(p2.Payload, payload) {
		t.Error("Lost payload in round-trip")
	}
}

func TestPacket_FIN(t *testing.T) {
	p := NewFINPacket(99)

	if !p.Header.IsFIN() {
		t.Error("Expected FIN flag")
	}
	if !p.Header.IsACK() {
		t.Error("Expected ACK flag")
	}
	if p.Header.SnSourceAck != 99 {
		t.Errorf("Expected SnSourceAck 99, got %d", p.Header.SnSourceAck)
	}

	// Test round-trip
	data := p.Serialize()
	p2 := &Packet{}
	if err := p2.Deserialize(data); err != nil {
		t.Fatalf("Deserialize failed: %v", err)
	}

	if !p2.Header.IsFIN() {
		t.Error("Lost FIN flag in round-trip")
	}
}

func TestPacket_Deserialize_HeaderTooShort(t *testing.T) {
	p := &Packet{}
	err := p.Deserialize(make([]byte, 4))
	if err == nil {
		t.Error("Expected error for short header")
	}
}

func TestPacket_WithAckVector(t *testing.T) {
	p := &Packet{
		Header: FECHeader{
			SnSourceAck:              100,
			SourceAckReceiveWindowSize: 64,
			Flags:                    FlagACK | FlagACKV,
		},
		AckVector: &AckVector{
			AckVectorSize:     3,
			AckVectorElements: []uint8{0xFF, 0x0F, 0xF0},
		},
	}

	data := p.Serialize()
	p2 := &Packet{}
	if err := p2.Deserialize(data); err != nil {
		t.Fatalf("Deserialize failed: %v", err)
	}

	if p2.AckVector == nil {
		t.Fatal("Expected AckVector")
	}
	if p2.AckVector.AckVectorSize != 3 {
		t.Errorf("Expected AckVectorSize 3, got %d", p2.AckVector.AckVectorSize)
	}
	if !bytes.Equal(p2.AckVector.AckVectorElements, []uint8{0xFF, 0x0F, 0xF0}) {
		t.Error("AckVectorElements mismatch")
	}
}

func TestFlagsString(t *testing.T) {
	tests := []struct {
		flags    uint16
		expected string
	}{
		{0, "NONE"},
		{FlagSYN, "[SYN]"},
		{FlagFIN, "[FIN]"},
		{FlagACK, "[ACK]"},
		{FlagDAT, "[DAT]"},
		{FlagFEC, "[FEC]"},
		{FlagSYN | FlagACK, "[SYN ACK]"},
		{FlagDAT | FlagACK | FlagFEC, "[ACK DAT FEC]"},
	}

	for _, tc := range tests {
		result := FlagsString(tc.flags)
		if result != tc.expected {
			t.Errorf("FlagsString(0x%04X): expected %q, got %q", tc.flags, tc.expected, result)
		}
	}
}

func TestPacket_ComplexScenario(t *testing.T) {
	// Create a complex packet with SYN data and data payload (unusual but valid)
	p := &Packet{
		Header: FECHeader{
			SnSourceAck:              50,
			SourceAckReceiveWindowSize: 64,
			Flags:                    FlagSYN | FlagACK | FlagDAT,
		},
		SynData: &SynData{
			SnInitialSequenceNumber: 100,
			UpstreamMTU:             1232,
			DownstreamMTU:           1232,
		},
		DataHeader: &SourcePayloadHeader{
			SnSourceStart: 100,
		},
		Payload: []byte("Initial data with SYN"),
	}

	data := p.Serialize()
	p2 := &Packet{}
	if err := p2.Deserialize(data); err != nil {
		t.Fatalf("Deserialize failed: %v", err)
	}

	// Verify all components
	if !p2.Header.IsSYN() || !p2.Header.IsACK() || !p2.Header.IsData() {
		t.Error("Lost flags in round-trip")
	}
	if p2.SynData == nil {
		t.Fatal("Lost SynData")
	}
	if p2.DataHeader == nil {
		t.Fatal("Lost DataHeader")
	}
	if !bytes.Equal(p2.Payload, p.Payload) {
		t.Error("Payload mismatch")
	}
}

func TestPacket_EmptyPayload(t *testing.T) {
	// Data packet with no payload (valid for ACK-only with DAT flag)
	p := NewDataPacket(100, 50, nil, 64)

	data := p.Serialize()
	p2 := &Packet{}
	if err := p2.Deserialize(data); err != nil {
		t.Fatalf("Deserialize failed: %v", err)
	}

	if len(p2.Payload) != 0 {
		t.Errorf("Expected empty payload, got %d bytes", len(p2.Payload))
	}
}

package rdpeudp

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// ============================================================================
// MS-RDPEUDP Specification Validation Tests
// Reference: https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-rdpeudp/
// ============================================================================

// TestFECHeader_Flags_PerSpec validates flag constants per Section 2.2.1.1
func TestFECHeader_Flags_PerSpec(t *testing.T) {
	// Per MS-RDPEUDP Section 2.2.1.1 (ceaae261-e538-40f0-8678-254e31621054)
	expectedFlags := map[string]uint16{
		"RDPUDP_FLAG_SYN":            0x0001,
		"RDPUDP_FLAG_FIN":            0x0002,
		"RDPUDP_FLAG_ACK":            0x0004,
		"RDPUDP_FLAG_DATA":           0x0008,
		"RDPUDP_FLAG_FEC":            0x0010,
		"RDPUDP_FLAG_CN":             0x0020,
		"RDPUDP_FLAG_CWR":            0x0040,
		"RDPUDP_FLAG_SACK_OPTION":    0x0080, // Not used
		"RDPUDP_FLAG_ACK_OF_ACKS":    0x0100,
		"RDPUDP_FLAG_SYNLOSSY":       0x0200,
		"RDPUDP_FLAG_ACKDELAYED":     0x0400,
		"RDPUDP_FLAG_CORRELATION_ID": 0x0800,
		"RDPUDP_FLAG_SYNEX":          0x1000,
	}

	// Verify our constants match spec
	if FlagSYN != expectedFlags["RDPUDP_FLAG_SYN"] {
		t.Errorf("FlagSYN = 0x%04X, spec says 0x%04X", FlagSYN, expectedFlags["RDPUDP_FLAG_SYN"])
	}
	if FlagFIN != expectedFlags["RDPUDP_FLAG_FIN"] {
		t.Errorf("FlagFIN = 0x%04X, spec says 0x%04X", FlagFIN, expectedFlags["RDPUDP_FLAG_FIN"])
	}
	if FlagACK != expectedFlags["RDPUDP_FLAG_ACK"] {
		t.Errorf("FlagACK = 0x%04X, spec says 0x%04X", FlagACK, expectedFlags["RDPUDP_FLAG_ACK"])
	}
	if FlagDAT != expectedFlags["RDPUDP_FLAG_DATA"] {
		t.Errorf("FlagDAT = 0x%04X, spec says 0x%04X", FlagDAT, expectedFlags["RDPUDP_FLAG_DATA"])
	}
	if FlagFEC != expectedFlags["RDPUDP_FLAG_FEC"] {
		t.Errorf("FlagFEC = 0x%04X, spec says 0x%04X", FlagFEC, expectedFlags["RDPUDP_FLAG_FEC"])
	}
	if FlagCN != expectedFlags["RDPUDP_FLAG_CN"] {
		t.Errorf("FlagCN = 0x%04X, spec says 0x%04X", FlagCN, expectedFlags["RDPUDP_FLAG_CN"])
	}
	if FlagCWR != expectedFlags["RDPUDP_FLAG_CWR"] {
		t.Errorf("FlagCWR = 0x%04X, spec says 0x%04X", FlagCWR, expectedFlags["RDPUDP_FLAG_CWR"])
	}
	if FlagAOA != expectedFlags["RDPUDP_FLAG_ACK_OF_ACKS"] {
		t.Errorf("FlagAOA = 0x%04X, spec says 0x%04X", FlagAOA, expectedFlags["RDPUDP_FLAG_ACK_OF_ACKS"])
	}
	if FlagSYNLOSSY != expectedFlags["RDPUDP_FLAG_SYNLOSSY"] {
		t.Errorf("FlagSYNLOSSY = 0x%04X, spec says 0x%04X", FlagSYNLOSSY, expectedFlags["RDPUDP_FLAG_SYNLOSSY"])
	}
	if FlagACKDelayed != expectedFlags["RDPUDP_FLAG_ACKDELAYED"] {
		t.Errorf("FlagACKDelayed = 0x%04X, spec says 0x%04X", FlagACKDelayed, expectedFlags["RDPUDP_FLAG_ACKDELAYED"])
	}
	if FlagCorrelationID != expectedFlags["RDPUDP_FLAG_CORRELATION_ID"] {
		t.Errorf("FlagCorrelationID = 0x%04X, spec says 0x%04X", FlagCorrelationID, expectedFlags["RDPUDP_FLAG_CORRELATION_ID"])
	}
	if FlagSYNEX != expectedFlags["RDPUDP_FLAG_SYNEX"] {
		t.Errorf("FlagSYNEX = 0x%04X, spec says 0x%04X", FlagSYNEX, expectedFlags["RDPUDP_FLAG_SYNEX"])
	}
}

// TestSourcePayloadHeader_FieldOrder validates field order per Section 2.2.2.4
// Per spec: snCoded (4 bytes) FIRST, snSourceStart (4 bytes) SECOND
func TestSourcePayloadHeader_FieldOrder(t *testing.T) {
	h := &SourcePayloadHeader{
		SnCoded:       0x11111111,
		SnSourceStart: 0x22222222,
	}

	data := h.Serialize()

	// Per spec (d98c71ec-945b-4d1c-8a03-c16818ae3f20):
	// snCoded is FIRST (bytes 0-3)
	// snSourceStart is SECOND (bytes 4-7)
	snCoded := binary.LittleEndian.Uint32(data[0:4])
	snSourceStart := binary.LittleEndian.Uint32(data[4:8])

	if snCoded != 0x11111111 {
		t.Errorf("snCoded at wrong position: got 0x%08X at bytes 0-3, want 0x11111111", snCoded)
	}
	if snSourceStart != 0x22222222 {
		t.Errorf("snSourceStart at wrong position: got 0x%08X at bytes 4-7, want 0x22222222", snSourceStart)
	}
}

// TestAckVector_MaxSize validates max ACK vector size per Section 2.2.2.7
// Per spec: "The maximum size of the ACK Vector is 2048 bytes"
func TestAckVector_MaxSize(t *testing.T) {
	// Create ACK vector at max size
	maxVector := make([]uint8, 2048)
	av := &AckVector{
		AckVectorSize:     2048,
		AckVectorElements: maxVector,
	}

	data := av.Serialize()

	// Should succeed at max size
	av2 := &AckVector{}
	if err := av2.Deserialize(data); err != nil {
		t.Errorf("Deserialize at max size failed: %v", err)
	}

	// Should fail if exceeding max
	oversizeData := make([]byte, 2+2049)
	binary.LittleEndian.PutUint16(oversizeData[0:2], 2049)

	av3 := &AckVector{}
	if err := av3.Deserialize(oversizeData); err == nil {
		t.Error("Should reject ACK vector > 2048 bytes")
	}
}

// TestAckVector_DWORDPadding validates DWORD padding per Section 2.2.2.7
// Per spec: "Padding (variable): such that this structure ends on a DWORD boundary"
func TestAckVector_DWORDPadding(t *testing.T) {
	tests := []struct {
		vectorSize   int
		expectedSize int // Total size including header and padding
	}{
		{1, 4},  // 2 + 1 = 3, padded to 4
		{2, 4},  // 2 + 2 = 4, no padding needed
		{3, 8},  // 2 + 3 = 5, padded to 8
		{4, 8},  // 2 + 4 = 6, padded to 8
		{5, 8},  // 2 + 5 = 7, padded to 8
		{6, 8},  // 2 + 6 = 8, no padding needed
		{7, 12}, // 2 + 7 = 9, padded to 12
	}

	for _, tc := range tests {
		av := &AckVector{
			AckVectorSize:     uint16(tc.vectorSize),
			AckVectorElements: make([]uint8, tc.vectorSize),
		}

		if av.Size() != tc.expectedSize {
			t.Errorf("AckVector size %d: Size() = %d, want %d (DWORD aligned)",
				tc.vectorSize, av.Size(), tc.expectedSize)
		}

		// Verify serialized data is actually padded
		data := av.Serialize()
		if len(data) != tc.expectedSize {
			t.Errorf("AckVector size %d: serialized = %d bytes, want %d",
				tc.vectorSize, len(data), tc.expectedSize)
		}
	}
}

// TestSynData_MTURange validates MTU range per Section 2.2.2.5
// Per spec: "This value MUST be >= 1132 and <= 1232"
func TestSynData_MTURange(t *testing.T) {
	validMTUs := []uint16{1132, 1150, 1200, 1232}
	for _, mtu := range validMTUs {
		s := &SynData{
			SnInitialSequenceNumber: 1,
			UpstreamMTU:             mtu,
			DownstreamMTU:           mtu,
		}
		data := s.Serialize()
		s2 := &SynData{}
		if err := s2.Deserialize(data); err != nil {
			t.Errorf("MTU %d should be valid: %v", mtu, err)
		}
	}
}

// TestSYNPacket_NoACKVector validates that SYN packets don't include ACK vector
// Per spec: During SYN phase, ACK_VECTOR_HEADER is NOT present
func TestSYNPacket_NoACKVector(t *testing.T) {
	// Create SYN+ACK packet (has ACK flag but is SYN phase)
	p := NewSYNACKPacket(100, 50, DefaultMTU, DefaultMTU)

	// Should have both SYN and ACK flags
	if !p.Header.HasFlag(FlagSYN) || !p.Header.HasFlag(FlagACK) {
		t.Error("SYN+ACK should have both flags")
	}

	// But should NOT have AckVector (SYN phase exemption)
	if p.AckVector != nil {
		t.Error("SYN+ACK should NOT include ACK_VECTOR (SYN phase)")
	}

	// Serialize and deserialize
	data, err := p.Serialize()
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	p2 := &Packet{}
	if err := p2.Deserialize(data); err != nil {
		t.Fatalf("Deserialize failed: %v", err)
	}

	// Should still not have AckVector after round-trip
	if p2.AckVector != nil {
		t.Error("Deserialized SYN+ACK should NOT have ACK_VECTOR")
	}
}

// ============================================================================
// Original Tests
// ============================================================================

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
		AckVectorElements: []uint8{0xFF, 0x0F, 0xF0, 0x55},
	}

	data := a.Serialize()
	// Size should be 2 (header) + 4 (elements) + 2 (padding to DWORD) = 8
	if len(data) != 8 {
		t.Errorf("Expected 8 bytes (with padding), got %d", len(data))
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
	data := []byte{10, 0} // Claims 10 elements (little endian: 0x000A) but has none
	a := &AckVector{}
	err := a.Deserialize(data)
	if err == nil {
		t.Error("Expected error for truncated elements")
	}
}

func TestAckVector_LargeSize(t *testing.T) {
	// Test that uint16 can handle sizes > 255
	elements := make([]uint8, 300)
	for i := range elements {
		elements[i] = uint8(i % 256)
	}
	a := &AckVector{
		AckVectorSize:     300,
		AckVectorElements: elements,
	}

	data := a.Serialize()
	a2 := &AckVector{}
	if err := a2.Deserialize(data); err != nil {
		t.Fatalf("Deserialize failed: %v", err)
	}

	if a2.AckVectorSize != 300 {
		t.Errorf("Expected size 300, got %d", a2.AckVectorSize)
	}
}

func TestAckVector_MaxSizeExceeded(t *testing.T) {
	// Test that sizes > 2048 are rejected
	data := make([]byte, 4)
	// Set size to 2049 (little endian)
	data[0] = 0x01
	data[1] = 0x08 // 0x0801 = 2049

	a := &AckVector{}
	err := a.Deserialize(data)
	if err == nil {
		t.Error("Expected error for size > 2048")
	}
}

func TestSourcePayloadHeader_RoundTrip(t *testing.T) {
	h := &SourcePayloadHeader{
		SnCoded:       50,  // Per spec, snCoded comes first
		SnSourceStart: 100,
	}

	data := h.Serialize()
	// Per spec, always 8 bytes
	if len(data) != 8 {
		t.Errorf("Expected 8 bytes, got %d", len(data))
	}

	h2 := &SourcePayloadHeader{}
	if err := h2.Deserialize(data); err != nil {
		t.Fatalf("Deserialize failed: %v", err)
	}

	if h2.SnCoded != h.SnCoded {
		t.Errorf("SnCoded mismatch: %d vs %d", h2.SnCoded, h.SnCoded)
	}
	if h2.SnSourceStart != h.SnSourceStart {
		t.Errorf("SnSourceStart mismatch: %d vs %d", h2.SnSourceStart, h.SnSourceStart)
	}
}

func TestSourcePayloadHeader_Deserialize_TooShort(t *testing.T) {
	h := &SourcePayloadHeader{}

	// Too short (need 8 bytes)
	err := h.Deserialize(make([]byte, 4))
	if err == nil {
		t.Error("Expected error for short data")
	}
}

func TestPacket_SYN(t *testing.T) {
	p := NewSYNPacket(12345, DefaultMTU, DefaultMTU, DefaultReceiveWindow)

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
	data, err := p.Serialize()
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}
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
	data, err := p.Serialize()
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}
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
	data, err := p.Serialize()
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}
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
	// NewDataPacket(codedSeq, sourceSeq, data)
	p := NewDataPacket(100, 100, payload)

	if !p.Header.IsData() {
		t.Error("Expected DAT flag")
	}
	// Data packets without ACK vector shouldn't have ACK flag
	if p.Header.IsACK() {
		t.Error("Unexpected ACK flag on data-only packet")
	}
	if p.SourcePayload == nil {
		t.Fatal("Expected SourcePayload")
	}
	if p.SourcePayload.SnSourceStart != 100 {
		t.Error("Wrong source sequence number")
	}
	if p.SourcePayload.SnCoded != 100 {
		t.Error("Wrong coded sequence number")
	}
	if !bytes.Equal(p.Data, payload) {
		t.Error("Payload mismatch")
	}

	// Test round-trip
	data, err := p.Serialize()
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}
	p2 := &Packet{}
	if err := p2.Deserialize(data); err != nil {
		t.Fatalf("Deserialize failed: %v", err)
	}

	if !p2.Header.IsData() {
		t.Error("Lost DAT flag in round-trip")
	}
	if !bytes.Equal(p2.Data, payload) {
		t.Error("Lost payload in round-trip")
	}
}

func TestPacket_FIN(t *testing.T) {
	p := NewFINPacket(99)

	if !p.Header.IsFIN() {
		t.Error("Expected FIN flag")
	}
	// FIN packets don't have ACK_VECTOR
	if p.Header.IsACK() {
		t.Error("Unexpected ACK flag on FIN packet")
	}
	if p.Header.SnSourceAck != 99 {
		t.Errorf("Expected SnSourceAck 99, got %d", p.Header.SnSourceAck)
	}

	// Test round-trip
	data, err := p.Serialize()
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}
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
			SnSourceAck:                100,
			SourceAckReceiveWindowSize: 64,
			Flags:                      FlagACK, // FlagACK means ACK_VECTOR_HEADER is present
		},
		AckVector: &AckVector{
			AckVectorSize:     3,
			AckVectorElements: []uint8{0xFF, 0x0F, 0xF0},
		},
	}

	data, err := p.Serialize()
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}
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

func TestNewDataPacketWithACK(t *testing.T) {
	payload := []byte("Data with ACK")
	ackVector := &AckVector{
		AckVectorSize:     2,
		AckVectorElements: []uint8{0xAA, 0x55},
	}
	// NewDataPacketWithACK(codedSeq, sourceSeq, ackSeq, data, receiveWindow, ackVector)
	p := NewDataPacketWithACK(100, 100, 50, payload, 64, ackVector)

	if !p.Header.IsData() {
		t.Error("Expected DAT flag")
	}
	if !p.Header.IsACK() {
		t.Error("Expected ACK flag")
	}
	if p.AckVector == nil {
		t.Fatal("Expected AckVector")
	}

	// Test round-trip
	data, err := p.Serialize()
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}
	p2 := &Packet{}
	if err := p2.Deserialize(data); err != nil {
		t.Fatalf("Deserialize failed: %v", err)
	}

	if p2.AckVector == nil {
		t.Fatal("Lost AckVector in round-trip")
	}
	if !bytes.Equal(p2.Data, payload) {
		t.Error("Lost payload in round-trip")
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
			SnSourceAck:                50,
			SourceAckReceiveWindowSize: 64,
			Flags:                      FlagSYN | FlagACK | FlagDAT,
		},
		SynData: &SynData{
			SnInitialSequenceNumber: 100,
			UpstreamMTU:             1232,
			DownstreamMTU:           1232,
		},
		SourcePayload: &SourcePayloadHeader{
			SnCoded:       100,
			SnSourceStart: 100,
		},
		Data: []byte("Initial data with SYN"),
	}

	data, err := p.Serialize()
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}
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
	if p2.SourcePayload == nil {
		t.Fatal("Lost SourcePayload")
	}
	if !bytes.Equal(p2.Data, p.Data) {
		t.Error("Payload mismatch")
	}
}

func TestPacket_EmptyPayload(t *testing.T) {
	// Data packet with no payload (valid for ACK-only with DAT flag)
	p := NewDataPacket(100, 100, nil)

	data, err := p.Serialize()
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}
	p2 := &Packet{}
	if err := p2.Deserialize(data); err != nil {
		t.Fatalf("Deserialize failed: %v", err)
	}

	if len(p2.Data) != 0 {
		t.Errorf("Expected empty payload, got %d bytes", len(p2.Data))
	}
}

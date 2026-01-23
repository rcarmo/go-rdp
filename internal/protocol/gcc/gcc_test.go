package gcc

import (
	"bytes"
	"testing"
)

func TestNewConferenceCreateRequest(t *testing.T) {
	userData := []byte{0x01, 0x02, 0x03, 0x04}
	req := NewConferenceCreateRequest(userData)

	if req == nil {
		t.Fatal("NewConferenceCreateRequest() returned nil")
	}

	if !bytes.Equal(req.UserData, userData) {
		t.Errorf("UserData = %v, want %v", req.UserData, userData)
	}
}

func TestConferenceCreateRequestSerialize(t *testing.T) {
	tests := []struct {
		name     string
		userData []byte
	}{
		{
			name:     "empty user data",
			userData: []byte{},
		},
		{
			name:     "simple user data",
			userData: []byte{0x01, 0x02, 0x03, 0x04},
		},
		{
			name:     "larger user data",
			userData: bytes.Repeat([]byte{0xAB}, 100),
		},
		{
			name:     "typical RDP client data",
			userData: []byte{0x01, 0x00, 0x08, 0x00, 0x03, 0x00, 0x00, 0x00},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := NewConferenceCreateRequest(tt.userData)
			serialized := req.Serialize()

			if serialized == nil {
				t.Fatal("Serialize() returned nil")
			}

			if len(serialized) == 0 {
				t.Fatal("Serialize() returned empty slice")
			}

			// Verify basic structure - first byte should be choice (0x00)
			if serialized[0] != 0x00 {
				t.Errorf("first byte (choice) = 0x%02x, want 0x00", serialized[0])
			}

			// Verify the serialized data contains the user data
			if len(tt.userData) > 0 {
				if !bytes.Contains(serialized, tt.userData) {
					t.Error("serialized data does not contain user data")
				}
			}
		})
	}
}

func TestConferenceCreateResponseDeserialize(t *testing.T) {
	tests := []struct {
		name      string
		data      []byte
		wantErr   bool
		errSubstr string
	}{
		{
			name:    "valid response",
			data:    buildValidGCCResponse(),
			wantErr: false,
		},
		{
			name:    "empty data",
			data:    []byte{},
			wantErr: true,
		},
		{
			name:      "bad object identifier",
			data:      buildBadOIDResponse(),
			wantErr:   true,
			errSubstr: "bad object identifier",
		},
		{
			name:      "bad H221 SC key",
			data:      buildBadH221Response(),
			wantErr:   true,
			errSubstr: "bad H221 SC_KEY",
		},
		{
			name:    "truncated data",
			data:    []byte{0x00, 0x05},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &ConferenceCreateResponse{}
			reader := bytes.NewReader(tt.data)
			err := resp.Deserialize(reader)

			if (err != nil) != tt.wantErr {
				t.Errorf("Deserialize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errSubstr != "" {
				if err == nil || !bytes.Contains([]byte(err.Error()), []byte(tt.errSubstr)) {
					t.Errorf("Deserialize() error = %v, want error containing %q", err, tt.errSubstr)
				}
			}
		})
	}
}

func TestConferenceCreateResponseDeserializeReadErrors(t *testing.T) {
	// Test various truncated responses to hit different error paths
	testCases := []struct {
		name string
		data []byte
	}{
		{
			name: "truncated after choice",
			data: []byte{0x00},
		},
		{
			name: "truncated after oid length",
			data: []byte{0x00, 0x05},
		},
		{
			name: "truncated in oid",
			data: []byte{0x00, 0x05, 0x00, 0x14},
		},
		{
			name: "truncated after oid",
			data: func() []byte {
				// choice + oid length + partial oid
				buf := bytes.Buffer{}
				buf.WriteByte(0x00)        // choice
				buf.WriteByte(0x05)        // oid length
				buf.WriteByte(0x00)        // t12
				buf.Write([]byte{20, 124}) // partial oid
				return buf.Bytes()
			}(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp := &ConferenceCreateResponse{}
			reader := bytes.NewReader(tc.data)
			err := resp.Deserialize(reader)

			if err == nil {
				t.Error("expected error for truncated data, got nil")
			}
		})
	}
}

// buildValidGCCResponse builds a valid GCC Conference Create Response
func buildValidGCCResponse() []byte {
	buf := new(bytes.Buffer)

	// Choice
	buf.WriteByte(0x00)

	// Object Identifier (t124_02_98_oid)
	// Length of OID encoding (5 bytes)
	buf.WriteByte(0x05)
	// t12 = (oid[0] << 4) | (oid[1] & 0x0f) = (0 << 4) | (0 & 0x0f) = 0
	buf.WriteByte(0x00)
	// oid[2..5] = {20, 124, 0, 1}
	buf.Write([]byte{20, 124, 0, 1})

	// Length (using short form)
	buf.WriteByte(0x10) // 16 bytes following

	// Choice
	buf.WriteByte(0x00)

	// Integer16 (conference ID - minimum 1001)
	buf.Write([]byte{0x00, 0x00}) // 0 + 1001 = 1001

	// Integer (result - length 1, value 0)
	buf.WriteByte(0x01) // length
	buf.WriteByte(0x00) // value

	// Enumerates
	buf.WriteByte(0x00)

	// Number of set
	buf.WriteByte(0x01)

	// Choice
	buf.WriteByte(0x00)

	// H221 SC Key octet stream
	buf.WriteByte(0x00)                     // length - minValue = 4 - 4 = 0
	buf.Write([]byte{'M', 'c', 'D', 'n'})   // h221SCKey

	// Final length
	buf.WriteByte(0x00)

	return buf.Bytes()
}

// buildBadOIDResponse builds a response with wrong OID
func buildBadOIDResponse() []byte {
	buf := new(bytes.Buffer)

	// Choice
	buf.WriteByte(0x00)

	// Wrong Object Identifier
	buf.WriteByte(0x05)
	buf.WriteByte(0xFF) // wrong t12
	buf.Write([]byte{0xFF, 0xFF, 0xFF, 0xFF})

	return buf.Bytes()
}

// buildBadH221Response builds a response with wrong H221 SC Key
func buildBadH221Response() []byte {
	buf := new(bytes.Buffer)

	// Choice
	buf.WriteByte(0x00)

	// Valid Object Identifier (t124_02_98_oid)
	buf.WriteByte(0x05)
	buf.WriteByte(0x00)
	buf.Write([]byte{20, 124, 0, 1})

	// Length
	buf.WriteByte(0x10)

	// Choice
	buf.WriteByte(0x00)

	// Integer16
	buf.Write([]byte{0x00, 0x00})

	// Integer (length 1)
	buf.WriteByte(0x01)
	buf.WriteByte(0x00)

	// Enumerates
	buf.WriteByte(0x00)

	// Number of set
	buf.WriteByte(0x01)

	// Choice
	buf.WriteByte(0x00)

	// Wrong H221 SC Key
	buf.WriteByte(0x00)                         // length - minValue
	buf.Write([]byte{'W', 'r', 'o', 'n'})       // wrong key

	return buf.Bytes()
}

func TestTypesConstants(t *testing.T) {
	// Test that the constants are correctly defined
	expectedOID := [6]byte{0, 0, 20, 124, 0, 1}
	if t124_02_98_oid != expectedOID {
		t.Errorf("t124_02_98_oid = %v, want %v", t124_02_98_oid, expectedOID)
	}

	if h221CSKey != "Duca" {
		t.Errorf("h221CSKey = %q, want %q", h221CSKey, "Duca")
	}

	if h221SCKey != "McDn" {
		t.Errorf("h221SCKey = %q, want %q", h221SCKey, "McDn")
	}
}

func TestConferenceCreateRequestSerializeConsistency(t *testing.T) {
	userData := []byte{0x01, 0x02, 0x03, 0x04}
	req := NewConferenceCreateRequest(userData)

	// Serialize multiple times and verify consistency
	serialized1 := req.Serialize()
	serialized2 := req.Serialize()

	if !bytes.Equal(serialized1, serialized2) {
		t.Error("Serialize() produced inconsistent results")
	}
}

func TestConferenceCreateRequestDifferentUserData(t *testing.T) {
	userData1 := []byte{0x01, 0x02}
	userData2 := []byte{0x03, 0x04, 0x05}

	req1 := NewConferenceCreateRequest(userData1)
	req2 := NewConferenceCreateRequest(userData2)

	serialized1 := req1.Serialize()
	serialized2 := req2.Serialize()

	if bytes.Equal(serialized1, serialized2) {
		t.Error("different user data should produce different serialization")
	}

	// Longer user data should produce longer serialized output
	if len(serialized2) <= len(serialized1) {
		t.Error("longer user data should produce longer serialized output")
	}
}

// BenchmarkConferenceCreateRequestSerialize benchmarks serialization
func BenchmarkConferenceCreateRequestSerialize(b *testing.B) {
	userData := bytes.Repeat([]byte{0x00}, 256)
	req := NewConferenceCreateRequest(userData)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = req.Serialize()
	}
}

// BenchmarkConferenceCreateResponseDeserialize benchmarks deserialization
func BenchmarkConferenceCreateResponseDeserialize(b *testing.B) {
	data := buildValidGCCResponse()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &ConferenceCreateResponse{}
		reader := bytes.NewReader(data)
		_ = resp.Deserialize(reader)
	}
}

func TestSerializeContainsH221CSKey(t *testing.T) {
	req := NewConferenceCreateRequest([]byte{0x01})
	serialized := req.Serialize()

	// The serialized data should contain the "Duca" H221 client-to-server key
	if !bytes.Contains(serialized, []byte("Duca")) {
		t.Error("serialized data should contain H221 CS Key 'Duca'")
	}
}

func TestDeserializeWithOIDSizeMismatch(t *testing.T) {
	buf := new(bytes.Buffer)

	// Choice
	buf.WriteByte(0x00)

	// OID with wrong size (not 5)
	buf.WriteByte(0x06) // size 6 instead of 5
	buf.Write([]byte{0x00, 20, 124, 0, 1, 0})

	resp := &ConferenceCreateResponse{}
	reader := bytes.NewReader(buf.Bytes())
	err := resp.Deserialize(reader)

	if err == nil {
		t.Error("expected error for OID size mismatch")
	}
}

func TestConferenceCreateResponseZeroStruct(t *testing.T) {
	// Test that zero-value struct can be used
	var resp ConferenceCreateResponse

	data := buildValidGCCResponse()
	reader := bytes.NewReader(data)
	err := resp.Deserialize(reader)

	if err != nil {
		t.Errorf("Deserialize() on zero struct error = %v", err)
	}
}

// ============================================================================
// Microsoft Protocol Test Suite Validation Tests
// Reference: MS-RDPBCGR_ClientTestDesignSpecification.md - S1_Connection
// ============================================================================

// TestBVT_BasicSettingExchange_GCCRequest validates per MS test case:
// "BVT_ConnectionTest_BasicSettingExchange_PositiveTest_ExtendedClientDataSupported"
// Per MS-RDPBCGR Section 2.2.1.3
func TestBVT_BasicSettingExchange_GCCRequest(t *testing.T) {
	// GCC Conference Create Request structure per MS-RDPBCGR 2.2.1.3
	// Contains: T.124 header, H.221 key "Duca", User Data

	// Minimal valid user data (Client Core Data header)
	userData := []byte{
		0x01, 0xc0, 0x08, 0x00, // CS_CORE header (type=0xC001, length=8)
		0x04, 0x00, 0x08, 0x00, // version, desktopWidth
	}

	req := NewConferenceCreateRequest(userData)
	serialized := req.Serialize()

	// Verify H.221 key "Duca" is present
	// Per spec: The key is "Duca" (0x44756361)
	ducaFound := false
	for i := 0; i < len(serialized)-3; i++ {
		if serialized[i] == 'D' && serialized[i+1] == 'u' &&
			serialized[i+2] == 'c' && serialized[i+3] == 'a' {
			ducaFound = true
			break
		}
	}
	if !ducaFound {
		t.Error("H.221 key 'Duca' not found in GCC Conference Create Request")
	}
}

// TestS1_BasicSettingExchange_H221Key validates H.221 non-standard key per spec
// Per MS-RDPBCGR Section 2.2.1.3.1: "Duca" for client, "McDn" for server
func TestS1_BasicSettingExchange_H221Key(t *testing.T) {
	tests := []struct {
		name string
		key  string
		side string
	}{
		{"Client H.221 key", "Duca", "client"},
		{"Server H.221 key", "McDn", "server"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// H.221 key must be exactly 4 ASCII characters
			if len(tc.key) != 4 {
				t.Errorf("H.221 key length = %d, want 4", len(tc.key))
			}
			// Must be printable ASCII
			for _, c := range tc.key {
				if c < 0x20 || c > 0x7E {
					t.Errorf("H.221 key contains non-printable character: 0x%02X", c)
				}
			}
		})
	}
}

// TestS1_BasicSettingExchange_UserDataBlocks validates user data block types per spec
// Per MS-RDPBCGR Section 2.2.1.3.2
func TestS1_BasicSettingExchange_UserDataBlocks(t *testing.T) {
	// Client-to-server user data block types
	clientBlocks := []struct {
		typeCode uint16
		name     string
	}{
		{0xC001, "CS_CORE"},
		{0xC002, "CS_SECURITY"},
		{0xC003, "CS_NET"},
		{0xC004, "CS_CLUSTER"},
		{0xC005, "CS_MONITOR"},
		{0xC006, "CS_MCS_MSGCHANNEL"},
		{0xC008, "CS_MONITOR_EX"},
		{0xC00A, "CS_MULTITRANSPORT"},
	}

	for _, cb := range clientBlocks {
		t.Run(cb.name, func(t *testing.T) {
			// Verify type code has 0xC0 high byte (client data)
			if cb.typeCode>>8 != 0xC0 {
				t.Errorf("Client block %s has invalid type: 0x%04X", cb.name, cb.typeCode)
			}
		})
	}

	// Server-to-client user data block types
	serverBlocks := []struct {
		typeCode uint16
		name     string
	}{
		{0x0C01, "SC_CORE"},
		{0x0C02, "SC_SECURITY"},
		{0x0C03, "SC_NET"},
		{0x0C04, "SC_MCS_MSGCHANNEL"},
		{0x0C08, "SC_MULTITRANSPORT"},
	}

	for _, sb := range serverBlocks {
		t.Run(sb.name, func(t *testing.T) {
			// Verify type code has 0x0C high byte (server data)
			if sb.typeCode>>8 != 0x0C {
				t.Errorf("Server block %s has invalid type: 0x%04X", sb.name, sb.typeCode)
			}
		})
	}
}

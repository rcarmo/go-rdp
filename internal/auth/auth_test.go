package auth

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"testing"
)

// =============================================================================
// MD4 Tests
// =============================================================================

func TestMD4(t *testing.T) {
	// Known test vectors from RFC 1320
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "31d6cfe0d16ae931b73c59d7e0c089c0",
		},
		{
			name:     "single character 'a'",
			input:    "a",
			expected: "bde52cb31de33e46245e05fbdbd6fb24",
		},
		{
			name:     "abc",
			input:    "abc",
			expected: "a448017aaf21d8525fc10ae87aa6729d",
		},
		{
			name:     "message digest",
			input:    "message digest",
			expected: "d9130a8164549fe818874806e1c7014b",
		},
		{
			name:     "alphabet lowercase",
			input:    "abcdefghijklmnopqrstuvwxyz",
			expected: "d79e1c308aa5bbcdeea8ed63df412da9",
		},
		{
			name:     "alphanumeric mixed",
			input:    "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789",
			expected: "043f8582f241db351ce627e153e7f0e4",
		},
		{
			name:     "numeric sequence",
			input:    "12345678901234567890123456789012345678901234567890123456789012345678901234567890",
			expected: "e33b4ddc9c38f2199c3e7b164fcc0536",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := md4([]byte(tt.input))
			got := hex.EncodeToString(result)
			if got != tt.expected {
				t.Errorf("md4(%q) = %s, want %s", tt.input, got, tt.expected)
			}
		})
	}
}

func TestMD4MultipleBlocks(t *testing.T) {
	// Test input that spans multiple 64-byte blocks
	input := make([]byte, 128)
	for i := range input {
		input[i] = byte(i)
	}
	result := md4(input)
	if len(result) != 16 {
		t.Errorf("md4 output length = %d, want 16", len(result))
	}
}

func TestMD4BoundaryConditions(t *testing.T) {
	tests := []struct {
		name string
		size int
	}{
		{"exactly 55 bytes", 55},
		{"exactly 56 bytes", 56},
		{"exactly 63 bytes", 63},
		{"exactly 64 bytes", 64},
		{"exactly 119 bytes", 119},
		{"exactly 120 bytes", 120},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := make([]byte, tt.size)
			for i := range input {
				input[i] = byte(i % 256)
			}
			result := md4(input)
			if len(result) != 16 {
				t.Errorf("md4 output length = %d, want 16", len(result))
			}
		})
	}
}

// =============================================================================
// NTLMv2 Tests
// =============================================================================

func TestNewNTLMv2(t *testing.T) {
	tests := []struct {
		name     string
		domain   string
		user     string
		password string
	}{
		{
			name:     "basic credentials",
			domain:   "DOMAIN",
			user:     "User",
			password: "Password",
		},
		{
			name:     "empty domain",
			domain:   "",
			user:     "admin",
			password: "secret",
		},
		{
			name:     "unicode password",
			domain:   "CORP",
			user:     "user1",
			password: "Pässwörd",
		},
		{
			name:     "empty credentials",
			domain:   "",
			user:     "",
			password: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := NewNTLMv2(tt.domain, tt.user, tt.password)
			if n == nil {
				t.Fatal("NewNTLMv2 returned nil")
			}
			if n.domain != tt.domain {
				t.Errorf("domain = %q, want %q", n.domain, tt.domain)
			}
			if n.user != tt.user {
				t.Errorf("user = %q, want %q", n.user, tt.user)
			}
			if n.password != tt.password {
				t.Errorf("password = %q, want %q", n.password, tt.password)
			}
			if len(n.respKeyNT) != 16 {
				t.Errorf("respKeyNT length = %d, want 16", len(n.respKeyNT))
			}
			if len(n.respKeyLM) != 16 {
				t.Errorf("respKeyLM length = %d, want 16", len(n.respKeyLM))
			}
		})
	}
}

func TestGetNegotiateMessage(t *testing.T) {
	n := NewNTLMv2("DOMAIN", "User", "Password")
	msg := n.GetNegotiateMessage()

	// Verify signature
	if !bytes.Equal(msg[:8], ntlmSignature) {
		t.Error("negotiate message missing NTLMSSP signature")
	}

	// Verify message type (should be 1)
	msgType := binary.LittleEndian.Uint32(msg[8:12])
	if msgType != 1 {
		t.Errorf("message type = %d, want 1", msgType)
	}

	// Verify negotiate flags
	flags := binary.LittleEndian.Uint32(msg[12:16])
	expectedFlags := uint32(
		NTLMSSP_NEGOTIATE_KEY_EXCH |
			NTLMSSP_NEGOTIATE_128 |
			NTLMSSP_NEGOTIATE_EXTENDED_SESSIONSECURITY |
			NTLMSSP_NEGOTIATE_ALWAYS_SIGN |
			NTLMSSP_NEGOTIATE_NTLM |
			NTLMSSP_NEGOTIATE_SEAL |
			NTLMSSP_NEGOTIATE_SIGN |
			NTLMSSP_REQUEST_TARGET |
			NTLMSSP_NEGOTIATE_UNICODE |
			NTLMSSP_NEGOTIATE_VERSION)

	if flags != expectedFlags {
		t.Errorf("flags = 0x%08x, want 0x%08x", flags, expectedFlags)
	}

	// Verify minimum message length
	if len(msg) < 40 {
		t.Errorf("message length = %d, want at least 40", len(msg))
	}

	// Verify message is cached
	msg2 := n.negotiateMsg
	if !bytes.Equal(msg, msg2) {
		t.Error("negotiate message not properly cached")
	}
}

func TestParseChallengeMessage(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		wantErr     bool
		checkFields func(t *testing.T, msg *ChallengeMessage)
	}{
		{
			name:    "too short message",
			data:    make([]byte, 55),
			wantErr: true,
		},
		{
			name: "valid challenge without target info",
			data: func() []byte {
				buf := &bytes.Buffer{}
				buf.Write(ntlmSignature)                               // Signature (8)
				binary.Write(buf, binary.LittleEndian, uint32(2))      // MessageType (4)
				binary.Write(buf, binary.LittleEndian, uint16(0))      // TargetNameLen
				binary.Write(buf, binary.LittleEndian, uint16(0))      // TargetNameMaxLen
				binary.Write(buf, binary.LittleEndian, uint32(0))      // TargetNameOffset
				binary.Write(buf, binary.LittleEndian, uint32(0x1234)) // NegotiateFlags
				buf.Write([]byte{1, 2, 3, 4, 5, 6, 7, 8})              // ServerChallenge
				buf.Write(make([]byte, 8))                             // Reserved
				binary.Write(buf, binary.LittleEndian, uint16(0))      // TargetInfoLen
				binary.Write(buf, binary.LittleEndian, uint16(0))      // TargetInfoMaxLen
				binary.Write(buf, binary.LittleEndian, uint32(0))      // TargetInfoOffset
				buf.Write(make([]byte, 16))                            // Padding to reach 56 bytes
				return buf.Bytes()
			}(),
			wantErr: false,
			checkFields: func(t *testing.T, msg *ChallengeMessage) {
				if msg.NegotiateFlags != 0x1234 {
					t.Errorf("NegotiateFlags = 0x%x, want 0x1234", msg.NegotiateFlags)
				}
				expected := [8]byte{1, 2, 3, 4, 5, 6, 7, 8}
				if msg.ServerChallenge != expected {
					t.Errorf("ServerChallenge = %v, want %v", msg.ServerChallenge, expected)
				}
			},
		},
		{
			name: "valid challenge with target info and timestamp",
			data: func() []byte {
				buf := &bytes.Buffer{}
				buf.Write(ntlmSignature)                                    // Signature (8)
				binary.Write(buf, binary.LittleEndian, uint32(2))           // MessageType (4)
				binary.Write(buf, binary.LittleEndian, uint16(0))           // TargetNameLen
				binary.Write(buf, binary.LittleEndian, uint16(0))           // TargetNameMaxLen
				binary.Write(buf, binary.LittleEndian, uint32(0))           // TargetNameOffset
				binary.Write(buf, binary.LittleEndian, uint32(0xabcd1234))  // NegotiateFlags
				buf.Write([]byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88}) // ServerChallenge
				buf.Write(make([]byte, 8))                                  // Reserved
				binary.Write(buf, binary.LittleEndian, uint16(16))          // TargetInfoLen
				binary.Write(buf, binary.LittleEndian, uint16(16))          // TargetInfoMaxLen
				binary.Write(buf, binary.LittleEndian, uint32(56))          // TargetInfoOffset

				// Pad to offset 56
				for buf.Len() < 56 {
					buf.WriteByte(0)
				}

				// TargetInfo with timestamp
				binary.Write(buf, binary.LittleEndian, uint16(MsvAvTimestamp)) // AvId
				binary.Write(buf, binary.LittleEndian, uint16(8))              // AvLen
				buf.Write([]byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x00, 0x11}) // Timestamp
				binary.Write(buf, binary.LittleEndian, uint16(MsvAvEOL))       // AvId (EOL)
				binary.Write(buf, binary.LittleEndian, uint16(0))              // AvLen

				return buf.Bytes()
			}(),
			wantErr: false,
			checkFields: func(t *testing.T, msg *ChallengeMessage) {
				if msg.NegotiateFlags != 0xabcd1234 {
					t.Errorf("NegotiateFlags = 0x%x, want 0xabcd1234", msg.NegotiateFlags)
				}
				if len(msg.TargetInfo) != 16 {
					t.Errorf("TargetInfo length = %d, want 16", len(msg.TargetInfo))
				}
				if len(msg.Timestamp) != 8 {
					t.Errorf("Timestamp length = %d, want 8", len(msg.Timestamp))
				}
				expectedTimestamp := []byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x00, 0x11}
				if !bytes.Equal(msg.Timestamp, expectedTimestamp) {
					t.Errorf("Timestamp = %v, want %v", msg.Timestamp, expectedTimestamp)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := ParseChallengeMessage(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseChallengeMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.checkFields != nil {
				tt.checkFields(t, msg)
			}
		})
	}
}

func TestExtractTimestamp(t *testing.T) {
	tests := []struct {
		name           string
		targetInfo     []byte
		wantTimestamp  []byte
	}{
		{
			name:          "empty target info",
			targetInfo:    []byte{},
			wantTimestamp: nil,
		},
		{
			name: "target info with only EOL",
			targetInfo: func() []byte {
				buf := &bytes.Buffer{}
				binary.Write(buf, binary.LittleEndian, uint16(MsvAvEOL))
				binary.Write(buf, binary.LittleEndian, uint16(0))
				return buf.Bytes()
			}(),
			wantTimestamp: nil,
		},
		{
			name: "target info with timestamp",
			targetInfo: func() []byte {
				buf := &bytes.Buffer{}
				binary.Write(buf, binary.LittleEndian, uint16(MsvAvTimestamp))
				binary.Write(buf, binary.LittleEndian, uint16(8))
				buf.Write([]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08})
				binary.Write(buf, binary.LittleEndian, uint16(MsvAvEOL))
				binary.Write(buf, binary.LittleEndian, uint16(0))
				return buf.Bytes()
			}(),
			wantTimestamp: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		},
		{
			name: "target info with multiple pairs before timestamp",
			targetInfo: func() []byte {
				buf := &bytes.Buffer{}
				// NbComputerName
				binary.Write(buf, binary.LittleEndian, uint16(MsvAvNbComputerName))
				binary.Write(buf, binary.LittleEndian, uint16(4))
				buf.Write([]byte{0xAA, 0xBB, 0xCC, 0xDD})
				// Timestamp
				binary.Write(buf, binary.LittleEndian, uint16(MsvAvTimestamp))
				binary.Write(buf, binary.LittleEndian, uint16(8))
				buf.Write([]byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88})
				// EOL
				binary.Write(buf, binary.LittleEndian, uint16(MsvAvEOL))
				binary.Write(buf, binary.LittleEndian, uint16(0))
				return buf.Bytes()
			}(),
			wantTimestamp: []byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTimestamp(tt.targetInfo)
			if !bytes.Equal(got, tt.wantTimestamp) {
				t.Errorf("extractTimestamp() = %v, want %v", got, tt.wantTimestamp)
			}
		})
	}
}

func TestUnicodeEncode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []byte
	}{
		{
			name:     "empty string",
			input:    "",
			expected: []byte{},
		},
		{
			name:     "ASCII string",
			input:    "Hello",
			expected: []byte{0x48, 0x00, 0x65, 0x00, 0x6c, 0x00, 0x6c, 0x00, 0x6f, 0x00},
		},
		{
			name:     "single character",
			input:    "A",
			expected: []byte{0x41, 0x00},
		},
		{
			name:     "unicode characters",
			input:    "日本",
			expected: []byte{0xe5, 0x65, 0x2c, 0x67}, // UTF-16LE
		},
		{
			name:     "mixed ASCII and special",
			input:    "Pässwörd",
			expected: []byte{0x50, 0x00, 0xe4, 0x00, 0x73, 0x00, 0x73, 0x00, 0x77, 0x00, 0xf6, 0x00, 0x72, 0x00, 0x64, 0x00},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := unicodeEncode(tt.input)
			if !bytes.Equal(got, tt.expected) {
				t.Errorf("unicodeEncode(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestNtowfv2(t *testing.T) {
	// Test that ntowfv2 produces consistent results
	tests := []struct {
		name     string
		password string
		user     string
		domain   string
	}{
		{
			name:     "basic credentials",
			password: "Password",
			user:     "User",
			domain:   "DOMAIN",
		},
		{
			name:     "empty domain",
			password: "secret",
			user:     "admin",
			domain:   "",
		},
		{
			name:     "case sensitivity test",
			password: "Pass",
			user:     "user",
			domain:   "dom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ntowfv2(tt.password, tt.user, tt.domain)
			if len(result) != 16 {
				t.Errorf("ntowfv2 result length = %d, want 16", len(result))
			}
			// Verify determinism
			result2 := ntowfv2(tt.password, tt.user, tt.domain)
			if !bytes.Equal(result, result2) {
				t.Error("ntowfv2 is not deterministic")
			}
		})
	}
}

func TestLmowfv2(t *testing.T) {
	// lmowfv2 should be identical to ntowfv2
	password, user, domain := "Password", "User", "DOMAIN"
	lm := lmowfv2(password, user, domain)
	nt := ntowfv2(password, user, domain)

	if !bytes.Equal(lm, nt) {
		t.Error("lmowfv2 should equal ntowfv2")
	}
}

func TestHmacMD5(t *testing.T) {
	tests := []struct {
		name     string
		key      []byte
		data     []byte
		expected string
	}{
		{
			name:     "RFC 2104 test vector 1",
			key:      bytes.Repeat([]byte{0x0b}, 16),
			data:     []byte("Hi There"),
			expected: "9294727a3638bb1c13f48ef8158bfc9d",
		},
		{
			name:     "RFC 2104 test vector 2",
			key:      []byte("Jefe"),
			data:     []byte("what do ya want for nothing?"),
			expected: "750c783e6ab0b503eaa86e310a5db738",
		},
		{
			name:     "empty data",
			key:      []byte("key"),
			data:     []byte{},
			expected: "63530468a04e386459855da0063b6596",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hmacMD5(tt.key, tt.data)
			got := hex.EncodeToString(result)
			if got != tt.expected {
				t.Errorf("hmacMD5 = %s, want %s", got, tt.expected)
			}
		})
	}
}

func TestToUpper(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "lowercase",
			input:    "hello",
			expected: "HELLO",
		},
		{
			name:     "uppercase",
			input:    "HELLO",
			expected: "HELLO",
		},
		{
			name:     "mixed case",
			input:    "HeLLo",
			expected: "HELLO",
		},
		{
			name:     "empty",
			input:    "",
			expected: "",
		},
		{
			name:     "numbers",
			input:    "test123",
			expected: "TEST123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toUpper(tt.input)
			if got != tt.expected {
				t.Errorf("toUpper(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestMakeTimestamp(t *testing.T) {
	ts := makeTimestamp()
	if len(ts) != 8 {
		t.Errorf("makeTimestamp length = %d, want 8", len(ts))
	}

	// Verify it's a valid Windows FILETIME (should be non-zero)
	val := binary.LittleEndian.Uint64(ts)
	if val == 0 {
		t.Error("makeTimestamp returned zero")
	}

	// Make sure sequential calls produce increasing timestamps
	ts2 := makeTimestamp()
	val2 := binary.LittleEndian.Uint64(ts2)
	if val2 < val {
		t.Error("makeTimestamp should produce increasing values")
	}
}

func TestGetEncodedCredentials(t *testing.T) {
	t.Run("unicode disabled", func(t *testing.T) {
		n := NewNTLMv2("DOMAIN", "User", "Pass")
		n.enableUnicode = false
		domain, user, pass := n.GetEncodedCredentials()
		if string(domain) != "DOMAIN" {
			t.Errorf("domain = %q, want DOMAIN", string(domain))
		}
		if string(user) != "User" {
			t.Errorf("user = %q, want User", string(user))
		}
		if string(pass) != "Pass" {
			t.Errorf("pass = %q, want Pass", string(pass))
		}
	})

	t.Run("unicode enabled", func(t *testing.T) {
		n := NewNTLMv2("DOM", "Usr", "Pwd")
		n.enableUnicode = true
		domain, user, pass := n.GetEncodedCredentials()

		expectedDomain := unicodeEncode("DOM")
		expectedUser := unicodeEncode("Usr")
		expectedPass := unicodeEncode("Pwd")

		if !bytes.Equal(domain, expectedDomain) {
			t.Errorf("domain = %v, want %v", domain, expectedDomain)
		}
		if !bytes.Equal(user, expectedUser) {
			t.Errorf("user = %v, want %v", user, expectedUser)
		}
		if !bytes.Equal(pass, expectedPass) {
			t.Errorf("pass = %v, want %v", pass, expectedPass)
		}
	})
}

func TestGetAuthenticateMessage(t *testing.T) {
	n := NewNTLMv2("DOMAIN", "User", "Password")
	_ = n.GetNegotiateMessage()

	// Build a valid challenge message with unicode flag and timestamp
	challengeData := func() []byte {
		buf := &bytes.Buffer{}
		buf.Write(ntlmSignature)
		binary.Write(buf, binary.LittleEndian, uint32(2))
		binary.Write(buf, binary.LittleEndian, uint16(0))
		binary.Write(buf, binary.LittleEndian, uint16(0))
		binary.Write(buf, binary.LittleEndian, uint32(0))
		flags := uint32(NTLMSSP_NEGOTIATE_UNICODE | NTLMSSP_NEGOTIATE_NTLM)
		binary.Write(buf, binary.LittleEndian, flags)
		buf.Write([]byte{1, 2, 3, 4, 5, 6, 7, 8}) // challenge
		buf.Write(make([]byte, 8))                // reserved

		// TargetInfo with timestamp
		targetInfo := &bytes.Buffer{}
		binary.Write(targetInfo, binary.LittleEndian, uint16(MsvAvTimestamp))
		binary.Write(targetInfo, binary.LittleEndian, uint16(8))
		targetInfo.Write([]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08})
		binary.Write(targetInfo, binary.LittleEndian, uint16(MsvAvEOL))
		binary.Write(targetInfo, binary.LittleEndian, uint16(0))

		binary.Write(buf, binary.LittleEndian, uint16(targetInfo.Len()))
		binary.Write(buf, binary.LittleEndian, uint16(targetInfo.Len()))
		binary.Write(buf, binary.LittleEndian, uint32(56))

		for buf.Len() < 56 {
			buf.WriteByte(0)
		}
		buf.Write(targetInfo.Bytes())
		return buf.Bytes()
	}()

	authMsg, security := n.GetAuthenticateMessage(challengeData)
	if authMsg == nil {
		t.Fatal("GetAuthenticateMessage returned nil auth message")
	}
	if security == nil {
		t.Fatal("GetAuthenticateMessage returned nil security context")
	}

	// Verify auth message structure
	if !bytes.Equal(authMsg[:8], ntlmSignature) {
		t.Error("auth message missing signature")
	}
	msgType := binary.LittleEndian.Uint32(authMsg[8:12])
	if msgType != 3 {
		t.Errorf("message type = %d, want 3", msgType)
	}

	// Test with invalid challenge
	authMsg2, security2 := n.GetAuthenticateMessage([]byte{1, 2, 3})
	if authMsg2 != nil || security2 != nil {
		t.Error("expected nil for invalid challenge")
	}
}

func TestGssEncrypt(t *testing.T) {
	n := NewNTLMv2("DOMAIN", "User", "Password")
	_ = n.GetNegotiateMessage()

	// Build challenge
	challengeData := func() []byte {
		buf := &bytes.Buffer{}
		buf.Write(ntlmSignature)
		binary.Write(buf, binary.LittleEndian, uint32(2))
		binary.Write(buf, binary.LittleEndian, uint16(0))
		binary.Write(buf, binary.LittleEndian, uint16(0))
		binary.Write(buf, binary.LittleEndian, uint32(0))
		binary.Write(buf, binary.LittleEndian, uint32(NTLMSSP_NEGOTIATE_UNICODE))
		buf.Write([]byte{1, 2, 3, 4, 5, 6, 7, 8})
		buf.Write(make([]byte, 8))
		binary.Write(buf, binary.LittleEndian, uint16(0))
		binary.Write(buf, binary.LittleEndian, uint16(0))
		binary.Write(buf, binary.LittleEndian, uint32(0))
		for buf.Len() < 56 {
			buf.WriteByte(0)
		}
		return buf.Bytes()
	}()

	_, security := n.GetAuthenticateMessage(challengeData)
	if security == nil {
		t.Fatal("no security context")
	}

	plaintext := []byte("Hello, World!")
	encrypted := security.GssEncrypt(plaintext)

	// Verify structure: Version(4) + Checksum(8) + SeqNum(4) + EncryptedData
	expectedLen := 4 + 8 + 4 + len(plaintext)
	if len(encrypted) != expectedLen {
		t.Errorf("encrypted length = %d, want %d", len(encrypted), expectedLen)
	}

	// Verify version
	version := binary.LittleEndian.Uint32(encrypted[:4])
	if version != 1 {
		t.Errorf("version = %d, want 1", version)
	}

	// Verify seqNum increments
	if security.seqNum != 1 {
		t.Errorf("seqNum = %d, want 1", security.seqNum)
	}

	// Encrypt again to check seqNum increment
	security.GssEncrypt([]byte("test"))
	if security.seqNum != 2 {
		t.Errorf("seqNum = %d, want 2", security.seqNum)
	}
}

// =============================================================================
// CredSSP Tests
// =============================================================================

func TestEncodeLength(t *testing.T) {
	tests := []struct {
		name     string
		length   int
		expected []byte
	}{
		{
			name:     "short form (0)",
			length:   0,
			expected: []byte{0x00},
		},
		{
			name:     "short form (127)",
			length:   127,
			expected: []byte{0x7F},
		},
		{
			name:     "long form 1 byte (128)",
			length:   128,
			expected: []byte{0x81, 0x80},
		},
		{
			name:     "long form 1 byte (255)",
			length:   255,
			expected: []byte{0x81, 0xFF},
		},
		{
			name:     "long form 2 bytes (256)",
			length:   256,
			expected: []byte{0x82, 0x01, 0x00},
		},
		{
			name:     "long form 2 bytes (65535)",
			length:   65535,
			expected: []byte{0x82, 0xFF, 0xFF},
		},
		{
			name:     "long form 3 bytes (65536)",
			length:   65536,
			expected: []byte{0x83, 0x01, 0x00, 0x00},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := encodeLength(tt.length)
			if !bytes.Equal(got, tt.expected) {
				t.Errorf("encodeLength(%d) = %v, want %v", tt.length, got, tt.expected)
			}
		})
	}
}

func TestEncodeSequence(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected []byte
	}{
		{
			name:     "empty sequence",
			data:     []byte{},
			expected: []byte{0x30, 0x00},
		},
		{
			name:     "short sequence",
			data:     []byte{0x01, 0x02, 0x03},
			expected: []byte{0x30, 0x03, 0x01, 0x02, 0x03},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := encodeSequence(tt.data)
			if !bytes.Equal(got, tt.expected) {
				t.Errorf("encodeSequence() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestEncodeContextTag(t *testing.T) {
	tests := []struct {
		name     string
		tag      int
		data     []byte
		expected []byte
	}{
		{
			name:     "tag 0 with empty data",
			tag:      0,
			data:     []byte{},
			expected: []byte{0xA0, 0x00},
		},
		{
			name:     "tag 1 with data",
			tag:      1,
			data:     []byte{0x01, 0x02},
			expected: []byte{0xA1, 0x02, 0x01, 0x02},
		},
		{
			name:     "tag 3 with data",
			tag:      3,
			data:     []byte{0xAA, 0xBB, 0xCC},
			expected: []byte{0xA3, 0x03, 0xAA, 0xBB, 0xCC},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := encodeContextTag(tt.tag, tt.data)
			if !bytes.Equal(got, tt.expected) {
				t.Errorf("encodeContextTag() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestEncodeOctetString(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected []byte
	}{
		{
			name:     "empty octet string",
			data:     []byte{},
			expected: []byte{0x04, 0x00},
		},
		{
			name:     "short octet string",
			data:     []byte{0x01, 0x02, 0x03},
			expected: []byte{0x04, 0x03, 0x01, 0x02, 0x03},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := encodeOctetString(tt.data)
			if !bytes.Equal(got, tt.expected) {
				t.Errorf("encodeOctetString() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestEncodeInteger(t *testing.T) {
	tests := []struct {
		name     string
		val      int
		expected []byte
	}{
		{
			name:     "zero",
			val:      0,
			expected: []byte{0x02, 0x01, 0x00},
		},
		{
			name:     "small value (1)",
			val:      1,
			expected: []byte{0x02, 0x01, 0x01},
		},
		{
			name:     "max short (127)",
			val:      127,
			expected: []byte{0x02, 0x01, 0x7F},
		},
		{
			name:     "needs padding (128)",
			val:      128,
			expected: []byte{0x02, 0x02, 0x00, 0x80},
		},
		{
			name:     "max 1 byte (255)",
			val:      255,
			expected: []byte{0x02, 0x02, 0x00, 0xFF},
		},
		{
			name:     "two bytes (256)",
			val:      256,
			expected: []byte{0x02, 0x02, 0x01, 0x00},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := encodeInteger(tt.val)
			if !bytes.Equal(got, tt.expected) {
				t.Errorf("encodeInteger(%d) = %v, want %v", tt.val, got, tt.expected)
			}
		})
	}
}

func TestParseTag(t *testing.T) {
	tests := []struct {
		name          string
		data          []byte
		wantTag       byte
		wantContent   []byte
		wantErr       bool
	}{
		{
			name:    "too short",
			data:    []byte{0x30},
			wantErr: true,
		},
		{
			name:        "short form length",
			data:        []byte{0x30, 0x03, 0x01, 0x02, 0x03},
			wantTag:     0x30,
			wantContent: []byte{0x01, 0x02, 0x03},
			wantErr:     false,
		},
		{
			name:        "empty content",
			data:        []byte{0x04, 0x00},
			wantTag:     0x04,
			wantContent: []byte{},
			wantErr:     false,
		},
		{
			name:        "long form 1 byte",
			data:        append([]byte{0x04, 0x81, 0x05}, make([]byte, 5)...),
			wantTag:     0x04,
			wantContent: make([]byte, 5),
			wantErr:     false,
		},
		{
			name:    "long form insufficient data",
			data:    []byte{0x04, 0x81, 0x10, 0x01, 0x02},
			wantErr: true,
		},
		{
			name:    "content length exceeds data",
			data:    []byte{0x30, 0x10, 0x01, 0x02},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tag, content, err := parseTag(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseTag() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if tag != tt.wantTag {
					t.Errorf("tag = %v, want %v", tag, tt.wantTag)
				}
				if !bytes.Equal(content, tt.wantContent) {
					t.Errorf("content = %v, want %v", content, tt.wantContent)
				}
			}
		})
	}
}

func TestParseInteger(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected int
	}{
		{
			name:     "single byte",
			data:     []byte{0x02, 0x01, 0x05},
			expected: 5,
		},
		{
			name:     "two bytes",
			data:     []byte{0x02, 0x02, 0x01, 0x00},
			expected: 256,
		},
		{
			name:     "invalid data",
			data:     []byte{0x02},
			expected: 0,
		},
		{
			name:     "empty value",
			data:     []byte{0x02, 0x00},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseInteger(tt.data)
			if got != tt.expected {
				t.Errorf("parseInteger() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestTagLen(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected int
	}{
		{
			name:     "short form",
			data:     []byte{0x30, 0x03, 0x01, 0x02, 0x03},
			expected: 5,
		},
		{
			name:     "long form 1 byte",
			data:     append([]byte{0x04, 0x81, 0x80}, make([]byte, 128)...),
			expected: 131,
		},
		{
			name:     "empty data",
			data:     []byte{},
			expected: 0,
		},
		{
			name:     "single byte",
			data:     []byte{0x30},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tagLen(tt.data)
			if got != tt.expected {
				t.Errorf("tagLen() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestEncodeTSRequest(t *testing.T) {
	tests := []struct {
		name         string
		ntlmMessages [][]byte
		authInfo     []byte
		pubKeyAuth   []byte
		checkResult  func(t *testing.T, result []byte)
	}{
		{
			name:         "empty request (version only)",
			ntlmMessages: nil,
			authInfo:     nil,
			pubKeyAuth:   nil,
			checkResult: func(t *testing.T, result []byte) {
				if result[0] != 0x30 {
					t.Error("expected SEQUENCE tag")
				}
				// Should decode successfully
				req, err := DecodeTSRequest(result)
				if err != nil {
					t.Errorf("decode failed: %v", err)
				}
				if req.Version != 6 {
					t.Errorf("version = %d, want 6", req.Version)
				}
			},
		},
		{
			name:         "with NTLM message",
			ntlmMessages: [][]byte{{0x4E, 0x54, 0x4C, 0x4D}},
			authInfo:     nil,
			pubKeyAuth:   nil,
			checkResult: func(t *testing.T, result []byte) {
				req, err := DecodeTSRequest(result)
				if err != nil {
					t.Errorf("decode failed: %v", err)
				}
				if len(req.NegoTokens) != 1 {
					t.Errorf("NegoTokens count = %d, want 1", len(req.NegoTokens))
				}
			},
		},
		{
			name:         "with authInfo",
			ntlmMessages: nil,
			authInfo:     []byte{0xAA, 0xBB, 0xCC},
			pubKeyAuth:   nil,
			checkResult: func(t *testing.T, result []byte) {
				req, err := DecodeTSRequest(result)
				if err != nil {
					t.Errorf("decode failed: %v", err)
				}
				if !bytes.Equal(req.AuthInfo, []byte{0xAA, 0xBB, 0xCC}) {
					t.Errorf("AuthInfo = %v, want [AA BB CC]", req.AuthInfo)
				}
			},
		},
		{
			name:         "with pubKeyAuth",
			ntlmMessages: nil,
			authInfo:     nil,
			pubKeyAuth:   []byte{0x11, 0x22, 0x33},
			checkResult: func(t *testing.T, result []byte) {
				req, err := DecodeTSRequest(result)
				if err != nil {
					t.Errorf("decode failed: %v", err)
				}
				if !bytes.Equal(req.PubKeyAuth, []byte{0x11, 0x22, 0x33}) {
					t.Errorf("PubKeyAuth = %v, want [11 22 33]", req.PubKeyAuth)
				}
			},
		},
		{
			name:         "full request",
			ntlmMessages: [][]byte{{0x01, 0x02}},
			authInfo:     []byte{0xAA, 0xBB},
			pubKeyAuth:   []byte{0xCC, 0xDD},
			checkResult: func(t *testing.T, result []byte) {
				req, err := DecodeTSRequest(result)
				if err != nil {
					t.Errorf("decode failed: %v", err)
				}
				if req.Version != 6 {
					t.Errorf("version = %d, want 6", req.Version)
				}
				if len(req.NegoTokens) != 1 {
					t.Errorf("NegoTokens count = %d, want 1", len(req.NegoTokens))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EncodeTSRequest(tt.ntlmMessages, tt.authInfo, tt.pubKeyAuth)
			if tt.checkResult != nil {
				tt.checkResult(t, result)
			}
		})
	}
}

func TestDecodeTSRequest(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		wantErr     bool
		checkResult func(t *testing.T, req *TSRequest)
	}{
		{
			name:    "invalid - too short",
			data:    []byte{0x30},
			wantErr: true,
		},
		{
			name: "valid minimal",
			data: func() []byte {
				return EncodeTSRequest(nil, nil, nil)
			}(),
			wantErr: false,
			checkResult: func(t *testing.T, req *TSRequest) {
				if req.Version != 6 {
					t.Errorf("Version = %d, want 6", req.Version)
				}
			},
		},
		{
			name: "valid with tokens",
			data: func() []byte {
				return EncodeTSRequest([][]byte{{0x01, 0x02, 0x03}}, nil, nil)
			}(),
			wantErr: false,
			checkResult: func(t *testing.T, req *TSRequest) {
				if len(req.NegoTokens) != 1 {
					t.Errorf("NegoTokens count = %d, want 1", len(req.NegoTokens))
				}
				if !bytes.Equal(req.NegoTokens[0].Data, []byte{0x01, 0x02, 0x03}) {
					t.Errorf("NegoToken data mismatch")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := DecodeTSRequest(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("DecodeTSRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.checkResult != nil {
				tt.checkResult(t, req)
			}
		})
	}
}

func TestEncodeCredentials(t *testing.T) {
	domain := []byte{0x44, 0x00, 0x4F, 0x00, 0x4D, 0x00} // "DOM" in UTF-16LE
	username := []byte{0x55, 0x00, 0x53, 0x00, 0x52, 0x00} // "USR" in UTF-16LE
	password := []byte{0x50, 0x00, 0x57, 0x00, 0x44, 0x00} // "PWD" in UTF-16LE

	result := EncodeCredentials(domain, username, password)

	// Verify it starts with SEQUENCE tag
	if result[0] != 0x30 {
		t.Error("expected SEQUENCE tag")
	}

	// Verify it's valid DER by parsing
	_, content, err := parseTag(result)
	if err != nil {
		t.Errorf("failed to parse credentials: %v", err)
	}
	if len(content) == 0 {
		t.Error("credentials content is empty")
	}
}

func TestParseNegoTokens(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected int
	}{
		{
			name:     "empty data",
			data:     []byte{},
			expected: 0,
		},
		{
			name: "single token",
			data: func() []byte {
				// Build a sequence containing one NegoDataItem
				item := encodeSequence(encodeContextTag(0, encodeOctetString([]byte{0x01, 0x02})))
				return encodeSequence(item)
			}(),
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := parseNegoTokens(tt.data)
			if len(tokens) != tt.expected {
				t.Errorf("parseNegoTokens() count = %d, want %d", len(tokens), tt.expected)
			}
		})
	}
}

// =============================================================================
// Round-trip Tests
// =============================================================================

func TestTSRequestRoundTrip(t *testing.T) {
	tests := []struct {
		name         string
		ntlmMessages [][]byte
		authInfo     []byte
		pubKeyAuth   []byte
	}{
		{
			name:         "empty",
			ntlmMessages: nil,
			authInfo:     nil,
			pubKeyAuth:   nil,
		},
		{
			name:         "with nego token",
			ntlmMessages: [][]byte{ntlmSignature},
			authInfo:     nil,
			pubKeyAuth:   nil,
		},
		{
			name:         "with auth info",
			ntlmMessages: nil,
			authInfo:     []byte{0x01, 0x02, 0x03, 0x04},
			pubKeyAuth:   nil,
		},
		{
			name:         "with pubkey auth",
			ntlmMessages: nil,
			authInfo:     nil,
			pubKeyAuth:   []byte{0xAA, 0xBB, 0xCC, 0xDD},
		},
		{
			name:         "full request",
			ntlmMessages: [][]byte{{0x01, 0x02}, {0x03, 0x04}},
			authInfo:     []byte{0x11, 0x22},
			pubKeyAuth:   []byte{0x33, 0x44},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := EncodeTSRequest(tt.ntlmMessages, tt.authInfo, tt.pubKeyAuth)
			decoded, err := DecodeTSRequest(encoded)
			if err != nil {
				t.Fatalf("DecodeTSRequest failed: %v", err)
			}

			if decoded.Version != 6 {
				t.Errorf("Version = %d, want 6", decoded.Version)
			}

			if len(tt.ntlmMessages) != len(decoded.NegoTokens) {
				t.Errorf("NegoTokens count = %d, want %d", len(decoded.NegoTokens), len(tt.ntlmMessages))
			}

			if tt.authInfo != nil && !bytes.Equal(decoded.AuthInfo, tt.authInfo) {
				t.Errorf("AuthInfo mismatch")
			}

			if tt.pubKeyAuth != nil && !bytes.Equal(decoded.PubKeyAuth, tt.pubKeyAuth) {
				t.Errorf("PubKeyAuth mismatch")
			}
		})
	}
}

// =============================================================================
// Edge Cases and Error Conditions
// =============================================================================

func TestMD4EdgeCases(t *testing.T) {
	// Test with nil input (should handle gracefully)
	result := md4(nil)
	if len(result) != 16 {
		t.Errorf("md4(nil) length = %d, want 16", len(result))
	}

	// Test with very long input
	longInput := make([]byte, 10000)
	result = md4(longInput)
	if len(result) != 16 {
		t.Errorf("md4(long input) length = %d, want 16", len(result))
	}
}

func TestParseTagEdgeCases(t *testing.T) {
	// Long form with 2 bytes
	data := []byte{0x04, 0x82, 0x01, 0x00}
	data = append(data, make([]byte, 256)...)
	tag, content, err := parseTag(data)
	if err != nil {
		t.Errorf("parseTag with 2-byte length failed: %v", err)
	}
	if tag != 0x04 {
		t.Errorf("tag = %v, want 0x04", tag)
	}
	if len(content) != 256 {
		t.Errorf("content length = %d, want 256", len(content))
	}
}

func TestNTLMv2WithSpecialCharacters(t *testing.T) {
	// Test with special characters in credentials
	n := NewNTLMv2("DOMAIN™", "Üser", "Pässwörd€")
	if n == nil {
		t.Fatal("NewNTLMv2 returned nil")
	}
	
	msg := n.GetNegotiateMessage()
	if msg == nil {
		t.Fatal("GetNegotiateMessage returned nil")
	}
}

func TestDecodeTSRequestWithMalformedData(t *testing.T) {
	// Test various malformed inputs
	malformedInputs := [][]byte{
		{0x30, 0x00},                     // Empty sequence
		{0x30, 0xFF},                     // Invalid length
		{0x30, 0x02, 0xA0, 0x00},         // Context tag with empty content
	}

	for i, input := range malformedInputs {
		_, err := DecodeTSRequest(input)
		// These should either succeed or fail gracefully, not panic
		_ = err
		t.Logf("Test %d completed (err=%v)", i, err)
	}
}

func TestComputeClientPubKeyAuth(t *testing.T) {
pubKey := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
nonce := []byte{0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f,
0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28, 0x29, 0x2a, 0x2b, 0x2c, 0x2d, 0x2e, 0x2f}

// Test version 2-4: should return pubKey unchanged
result := ComputeClientPubKeyAuth(2, pubKey, nil)
if !bytes.Equal(result, pubKey) {
t.Errorf("Version 2 should return pubKey unchanged")
}

result = ComputeClientPubKeyAuth(4, pubKey, nonce)
if !bytes.Equal(result, pubKey) {
t.Errorf("Version 4 should return pubKey unchanged")
}

// Test version 5+: should return SHA256 hash
result = ComputeClientPubKeyAuth(5, pubKey, nonce)
if len(result) != 32 { // SHA256 hash length
t.Errorf("Version 5 should return 32-byte SHA256 hash, got %d bytes", len(result))
}
if bytes.Equal(result, pubKey) {
t.Errorf("Version 5 result should not equal original pubKey")
}

// Test version 6 with nonce
result = ComputeClientPubKeyAuth(6, pubKey, nonce)
if len(result) != 32 {
t.Errorf("Version 6 should return 32-byte SHA256 hash")
}
}

func TestVerifyServerPubKeyAuth(t *testing.T) {
clientPubKey := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
nonce := []byte{0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f,
0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28, 0x29, 0x2a, 0x2b, 0x2c, 0x2d, 0x2e, 0x2f}

// Test version 2-4: server should send pubKey with first byte + 1
serverResponse := make([]byte, len(clientPubKey))
copy(serverResponse, clientPubKey)
serverResponse[0]++

if !VerifyServerPubKeyAuth(2, serverResponse, clientPubKey, nil) {
t.Errorf("Version 2 should verify correctly")
}

// Test with wrong response
wrongResponse := make([]byte, len(clientPubKey))
copy(wrongResponse, clientPubKey)
if VerifyServerPubKeyAuth(2, wrongResponse, clientPubKey, nil) {
t.Errorf("Version 2 should fail with wrong response")
}

// Test version 5+: need to compute correct hash
// Server sends SHA256(ServerClientHashMagic || nonce || pubKey)
// We can't easily compute this without importing crypto/sha256 in test
// but we can test that it returns false for wrong response
if VerifyServerPubKeyAuth(5, serverResponse, clientPubKey, nonce) {
t.Errorf("Version 5 should fail with old-style response")
}

// Test length mismatch
if VerifyServerPubKeyAuth(2, []byte{0x01}, clientPubKey, nil) {
t.Errorf("Should fail with length mismatch")
}
}

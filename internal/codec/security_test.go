package codec

import (
	"bytes"
	"testing"
)

func TestWrapSecurityFlag(t *testing.T) {
	flag := uint16(0x1234)
	data := []byte{0xAA, 0xBB, 0xCC}

	result := WrapSecurityFlag(flag, data)

	// Expected: flag (2 bytes LE) + flagsHi (2 bytes) + data
	if len(result) != 7 {
		t.Errorf("WrapSecurityFlag() len = %d, want 7", len(result))
	}

	// Check flag (little endian)
	if result[0] != 0x34 || result[1] != 0x12 {
		t.Errorf("WrapSecurityFlag() flag bytes = %v, want [0x34, 0x12]", result[0:2])
	}

	// Check flagsHi (should be 0x0000)
	if result[2] != 0x00 || result[3] != 0x00 {
		t.Errorf("WrapSecurityFlag() flagsHi = %v, want [0x00, 0x00]", result[2:4])
	}

	// Check data
	if !bytes.Equal(result[4:], data) {
		t.Errorf("WrapSecurityFlag() data = %v, want %v", result[4:], data)
	}
}

func TestUnwrapSecurityFlag(t *testing.T) {
	// Build test data: flag (0x1234) + flagsHi (0x0000)
	data := []byte{0x34, 0x12, 0x00, 0x00, 0xAA, 0xBB}

	flag, err := UnwrapSecurityFlag(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("UnwrapSecurityFlag() error = %v", err)
	}

	if flag != 0x1234 {
		t.Errorf("UnwrapSecurityFlag() = 0x%04X, want 0x1234", flag)
	}
}

func TestUnwrapSecurityFlag_TooShort(t *testing.T) {
	// Only 2 bytes, need at least 4
	data := []byte{0x34, 0x12}

	_, err := UnwrapSecurityFlag(bytes.NewReader(data))
	if err == nil {
		t.Error("UnwrapSecurityFlag() should return error for short data")
	}
}

func TestUnwrapSecurityFlag_Empty(t *testing.T) {
	_, err := UnwrapSecurityFlag(bytes.NewReader([]byte{}))
	if err == nil {
		t.Error("UnwrapSecurityFlag() should return error for empty data")
	}
}

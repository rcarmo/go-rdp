package encoding

import (
	"bytes"
	"io"
	"testing"
)

// ============================================================================
// BER Tests
// ============================================================================

func TestBerReadLength(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		want    uint16
		wantErr bool
	}{
		// Short form (0x00-0x7F)
		{name: "short form zero", input: []byte{0x00}, want: 0},
		{name: "short form 1", input: []byte{0x01}, want: 1},
		{name: "short form 127", input: []byte{0x7F}, want: 127},

		// Long form: 1 byte (0x81 + 1 byte)
		{name: "long form 1 byte - 128", input: []byte{0x81, 0x80}, want: 128},
		{name: "long form 1 byte - 255", input: []byte{0x81, 0xFF}, want: 255},

		// Long form: 2 bytes (0x82 + 2 bytes)
		{name: "long form 2 bytes - 256", input: []byte{0x82, 0x01, 0x00}, want: 256},
		{name: "long form 2 bytes - 0xFFFF", input: []byte{0x82, 0xFF, 0xFF}, want: 0xFFFF},
		{name: "long form 2 bytes - 1000", input: []byte{0x82, 0x03, 0xE8}, want: 1000},

		// Error cases
		{name: "empty input", input: []byte{}, wantErr: true},
		{name: "invalid long form size 3", input: []byte{0x83}, wantErr: true},
		{name: "truncated long form 1 byte", input: []byte{0x81}, wantErr: true},
		{name: "truncated long form 2 bytes", input: []byte{0x82, 0x01}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewReader(tt.input)
			got, err := BerReadLength(r)
			if (err != nil) != tt.wantErr {
				t.Errorf("BerReadLength() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("BerReadLength() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBerWriteLength(t *testing.T) {
	tests := []struct {
		name string
		size int
		want []byte
	}{
		// Short form (0x00-0x7F)
		{name: "short form zero", size: 0, want: []byte{0x00}},
		{name: "short form 1", size: 1, want: []byte{0x01}},
		{name: "short form 127", size: 127, want: []byte{0x7F}},

		// Long form: 1 byte (0x81 + 1 byte)
		{name: "long form 1 byte - 128", size: 128, want: []byte{0x81, 0x80}},
		{name: "long form 1 byte - 255", size: 255, want: []byte{0x81, 0xFF}},

		// Long form: 2 bytes (0x82 + 2 bytes)
		{name: "long form 2 bytes - 256", size: 256, want: []byte{0x82, 0x01, 0x00}},
		{name: "long form 2 bytes - 0xFFFF", size: 0xFFFF, want: []byte{0x82, 0xFF, 0xFF}},
		{name: "long form 2 bytes - 1000", size: 1000, want: []byte{0x82, 0x03, 0xE8}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			BerWriteLength(tt.size, &buf)
			if !bytes.Equal(buf.Bytes(), tt.want) {
				t.Errorf("BerWriteLength() = %v, want %v", buf.Bytes(), tt.want)
			}
		})
	}
}

func TestBerReadWriteLengthRoundTrip(t *testing.T) {
	sizes := []int{0, 1, 50, 127, 128, 200, 255, 256, 1000, 0x7FFF, 0xFFFF}

	for _, size := range sizes {
		var buf bytes.Buffer
		BerWriteLength(size, &buf)
		got, err := BerReadLength(bytes.NewReader(buf.Bytes()))
		if err != nil {
			t.Errorf("Round trip failed for size %d: %v", size, err)
			continue
		}
		if int(got) != size {
			t.Errorf("Round trip size %d: got %d", size, got)
		}
	}
}

func TestBerReadInteger(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		want    int
		wantErr bool
	}{
		// 1-byte integers
		{name: "1 byte - zero", input: []byte{0x02, 0x01, 0x00}, want: 0},
		{name: "1 byte - 1", input: []byte{0x02, 0x01, 0x01}, want: 1},
		{name: "1 byte - 255", input: []byte{0x02, 0x01, 0xFF}, want: 255},

		// 2-byte integers
		{name: "2 bytes - 256", input: []byte{0x02, 0x02, 0x01, 0x00}, want: 256},
		{name: "2 bytes - 0xFFFF", input: []byte{0x02, 0x02, 0xFF, 0xFF}, want: 0xFFFF},

		// 3-byte integers
		{name: "3 bytes - 0x010203", input: []byte{0x02, 0x03, 0x01, 0x02, 0x03}, want: 0x010203},

		// 4-byte integers
		{name: "4 bytes - 0x01020304", input: []byte{0x02, 0x04, 0x01, 0x02, 0x03, 0x04}, want: 0x01020304},

		// Error cases
		{name: "empty input", input: []byte{}, wantErr: true},
		{name: "wrong tag", input: []byte{0x03, 0x01, 0x00}, wantErr: true},
		{name: "truncated", input: []byte{0x02, 0x02, 0x01}, wantErr: true},
		{name: "invalid size 5", input: []byte{0x02, 0x05, 0x01, 0x02, 0x03, 0x04, 0x05}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewReader(tt.input)
			got, err := BerReadInteger(r)
			if (err != nil) != tt.wantErr {
				t.Errorf("BerReadInteger() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("BerReadInteger() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBerWriteInteger(t *testing.T) {
	tests := []struct {
		name string
		n    int
		want []byte
	}{
		{name: "zero", n: 0, want: []byte{0x02, 0x01, 0x00}},
		{name: "1", n: 1, want: []byte{0x02, 0x01, 0x01}},
		{name: "255", n: 255, want: []byte{0x02, 0x01, 0xFF}},
		{name: "256", n: 256, want: []byte{0x02, 0x02, 0x01, 0x00}},
		{name: "0xFFFF", n: 0xFFFF, want: []byte{0x02, 0x02, 0xFF, 0xFF}},
		{name: "0x10000", n: 0x10000, want: []byte{0x02, 0x04, 0x00, 0x01, 0x00, 0x00}},
		{name: "large", n: 0x12345678, want: []byte{0x02, 0x04, 0x12, 0x34, 0x56, 0x78}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			BerWriteInteger(tt.n, &buf)
			if !bytes.Equal(buf.Bytes(), tt.want) {
				t.Errorf("BerWriteInteger(%d) = %v, want %v", tt.n, buf.Bytes(), tt.want)
			}
		})
	}
}

func TestBerReadOctetString(t *testing.T) {
	// BerReadOctetString doesn't exist in the package, but BerWriteOctetString does
	// We test write only
}

func TestBerWriteOctetString(t *testing.T) {
	tests := []struct {
		name string
		str  []byte
		want []byte
	}{
		{name: "empty", str: []byte{}, want: []byte{0x04, 0x00}},
		{name: "single byte", str: []byte{0x41}, want: []byte{0x04, 0x01, 0x41}},
		{name: "hello", str: []byte("hello"), want: []byte{0x04, 0x05, 'h', 'e', 'l', 'l', 'o'}},
		{name: "binary data", str: []byte{0x00, 0xFF, 0x80}, want: []byte{0x04, 0x03, 0x00, 0xFF, 0x80}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			BerWriteOctetString(tt.str, &buf)
			if !bytes.Equal(buf.Bytes(), tt.want) {
				t.Errorf("BerWriteOctetString() = %v, want %v", buf.Bytes(), tt.want)
			}
		})
	}
}

func TestBerWriteOctetStringLongForm(t *testing.T) {
	// Test with data requiring long form length
	data := make([]byte, 200)
	for i := range data {
		data[i] = byte(i)
	}

	var buf bytes.Buffer
	BerWriteOctetString(data, &buf)

	result := buf.Bytes()
	// Should have tag 0x04, long form length 0x81 0xC8, then 200 bytes of data
	if result[0] != 0x04 {
		t.Errorf("Expected tag 0x04, got %v", result[0])
	}
	if result[1] != 0x81 || result[2] != 0xC8 {
		t.Errorf("Expected long form length [0x81, 0xC8], got [%v, %v]", result[1], result[2])
	}
	if !bytes.Equal(result[3:], data) {
		t.Error("Data mismatch in long form octet string")
	}
}

func TestBerReadApplicationTag(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		want    uint8
		wantErr bool
	}{
		{name: "valid tag 0x01", input: []byte{0x7F, 0x01}, want: 0x01},
		{name: "valid tag 0x65", input: []byte{0x7F, 0x65}, want: 0x65},
		{name: "valid tag 0xFF", input: []byte{0x7F, 0xFF}, want: 0xFF},

		// Error cases
		{name: "empty input", input: []byte{}, wantErr: true},
		{name: "wrong identifier", input: []byte{0x00, 0x01}, wantErr: true},
		{name: "truncated", input: []byte{0x7F}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewReader(tt.input)
			got, err := BerReadApplicationTag(r)
			if (err != nil) != tt.wantErr {
				t.Errorf("BerReadApplicationTag() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("BerReadApplicationTag() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBerWriteApplicationTag(t *testing.T) {
	tests := []struct {
		name     string
		tag      uint8
		size     int
		wantHead []byte // Just check the header bytes
	}{
		// Tags <= 30 use short form
		{name: "tag 0", tag: 0, size: 10, wantHead: []byte{0x00, 0x0A}},
		{name: "tag 1", tag: 1, size: 5, wantHead: []byte{0x01, 0x05}},
		{name: "tag 30", tag: 30, size: 100, wantHead: []byte{0x1E, 0x64}},

		// Tags > 30 use long form with 0x7F prefix
		{name: "tag 31", tag: 31, size: 10, wantHead: []byte{0x7F, 0x1F, 0x0A}},
		{name: "tag 50", tag: 50, size: 20, wantHead: []byte{0x7F, 0x32, 0x14}},
		{name: "tag 255", tag: 255, size: 50, wantHead: []byte{0x7F, 0xFF, 0x32}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			BerWriteApplicationTag(tt.tag, tt.size, &buf)
			result := buf.Bytes()
			if !bytes.HasPrefix(result, tt.wantHead) {
				t.Errorf("BerWriteApplicationTag(%d, %d) = %v, want prefix %v", tt.tag, tt.size, result, tt.wantHead)
			}
		})
	}
}

func TestBerWriteApplicationTagLongFormSize(t *testing.T) {
	// Test with size requiring long form length
	var buf bytes.Buffer
	BerWriteApplicationTag(50, 300, &buf)
	result := buf.Bytes()

	// Should be: 0x7F, tag, 0x82, high byte, low byte
	if result[0] != 0x7F || result[1] != 50 {
		t.Errorf("Expected [0x7F, 0x32], got [%v, %v]", result[0], result[1])
	}
	if result[2] != 0x82 {
		t.Errorf("Expected long form length indicator 0x82, got %v", result[2])
	}
}

func TestBerReadUniversalTag(t *testing.T) {
	tests := []struct {
		name    string
		tag     uint8
		pc      bool
		input   []byte
		want    bool
		wantErr bool
	}{
		// Primitive tags
		{name: "integer primitive match", tag: TagInteger, pc: false, input: []byte{0x02}, want: true},
		{name: "boolean primitive match", tag: TagBoolean, pc: false, input: []byte{0x01}, want: true},
		{name: "octet string primitive", tag: TagOctetString, pc: false, input: []byte{0x04}, want: true},
		{name: "enumerated primitive", tag: TagEnumerated, pc: false, input: []byte{0x0A}, want: true},

		// Constructed tags
		{name: "sequence constructed match", tag: TagSequence, pc: true, input: []byte{0x30}, want: true},

		// Non-matches
		{name: "tag mismatch", tag: TagInteger, pc: false, input: []byte{0x03}, want: false},
		{name: "pc mismatch", tag: TagSequence, pc: false, input: []byte{0x30}, want: false},

		// Error cases
		{name: "empty input", tag: TagInteger, pc: false, input: []byte{}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewReader(tt.input)
			got, err := BerReadUniversalTag(tt.tag, tt.pc, r)
			if (err != nil) != tt.wantErr {
				t.Errorf("BerReadUniversalTag() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("BerReadUniversalTag() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBerWriteSequence(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want []byte
	}{
		{name: "empty sequence", data: []byte{}, want: []byte{0x30, 0x00}},
		{name: "single byte", data: []byte{0x01}, want: []byte{0x30, 0x01, 0x01}},
		{name: "multiple bytes", data: []byte{0x01, 0x02, 0x03}, want: []byte{0x30, 0x03, 0x01, 0x02, 0x03}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			BerWriteSequence(tt.data, &buf)
			if !bytes.Equal(buf.Bytes(), tt.want) {
				t.Errorf("BerWriteSequence() = %v, want %v", buf.Bytes(), tt.want)
			}
		})
	}
}

func TestBerWriteSequenceLongForm(t *testing.T) {
	// Test with data requiring long form length
	data := make([]byte, 150)

	var buf bytes.Buffer
	BerWriteSequence(data, &buf)

	result := buf.Bytes()
	if result[0] != 0x30 {
		t.Errorf("Expected sequence tag 0x30, got %v", result[0])
	}
	if result[1] != 0x81 || result[2] != 150 {
		t.Errorf("Expected long form length [0x81, 0x96], got [%v, %v]", result[1], result[2])
	}
}

func TestBerWriteBoolean(t *testing.T) {
	tests := []struct {
		name string
		b    bool
		want []byte
	}{
		{name: "false", b: false, want: []byte{0x01, 0x01, 0x00}},
		{name: "true", b: true, want: []byte{0x01, 0x01, 0xFF}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			BerWriteBoolean(tt.b, &buf)
			if !bytes.Equal(buf.Bytes(), tt.want) {
				t.Errorf("BerWriteBoolean(%v) = %v, want %v", tt.b, buf.Bytes(), tt.want)
			}
		})
	}
}

func TestBerReadEnumerated(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		want    uint8
		wantErr bool
	}{
		{name: "enumerated 0", input: []byte{0x0A, 0x01, 0x00}, want: 0},
		{name: "enumerated 1", input: []byte{0x0A, 0x01, 0x01}, want: 1},
		{name: "enumerated 255", input: []byte{0x0A, 0x01, 0xFF}, want: 255},

		// Error cases
		{name: "wrong tag", input: []byte{0x02, 0x01, 0x00}, wantErr: true},
		{name: "wrong length", input: []byte{0x0A, 0x02, 0x00, 0x00}, wantErr: true},
		{name: "empty", input: []byte{}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewReader(tt.input)
			got, err := BerReadEnumerated(r)
			if (err != nil) != tt.wantErr {
				t.Errorf("BerReadEnumerated() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("BerReadEnumerated() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBerWriteInteger16(t *testing.T) {
	tests := []struct {
		name string
		n    uint16
		want []byte
	}{
		{name: "zero", n: 0, want: []byte{0x02, 0x02, 0x00, 0x00}},
		{name: "1", n: 1, want: []byte{0x02, 0x02, 0x00, 0x01}},
		{name: "256", n: 256, want: []byte{0x02, 0x02, 0x01, 0x00}},
		{name: "0xFFFF", n: 0xFFFF, want: []byte{0x02, 0x02, 0xFF, 0xFF}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			BerWriteInteger16(tt.n, &buf)
			if !bytes.Equal(buf.Bytes(), tt.want) {
				t.Errorf("BerWriteInteger16(%d) = %v, want %v", tt.n, buf.Bytes(), tt.want)
			}
		})
	}
}

func TestBerReadInteger16(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		want    uint16
		wantErr bool
	}{
		{name: "zero", input: []byte{0x02, 0x02, 0x00, 0x00}, want: 0},
		{name: "1", input: []byte{0x02, 0x02, 0x00, 0x01}, want: 1},
		{name: "256", input: []byte{0x02, 0x02, 0x01, 0x00}, want: 256},
		{name: "0xFFFF", input: []byte{0x02, 0x02, 0xFF, 0xFF}, want: 0xFFFF},

		// Error cases
		{name: "wrong tag", input: []byte{0x03, 0x02, 0x00, 0x00}, wantErr: true},
		{name: "wrong size", input: []byte{0x02, 0x01, 0x00}, wantErr: true},
		{name: "empty", input: []byte{}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewReader(tt.input)
			got, err := BerReadInteger16(r)
			if (err != nil) != tt.wantErr {
				t.Errorf("BerReadInteger16() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("BerReadInteger16() = %v, want %v", got, tt.want)
			}
		})
	}
}

// ============================================================================
// PER Tests
// ============================================================================

func TestPerReadChoice(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		want    uint8
		wantErr bool
	}{
		{name: "choice 0", input: []byte{0x00}, want: 0},
		{name: "choice 1", input: []byte{0x01}, want: 1},
		{name: "choice 255", input: []byte{0xFF}, want: 255},

		// Error cases
		{name: "empty", input: []byte{}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewReader(tt.input)
			got, err := PerReadChoice(r)
			if (err != nil) != tt.wantErr {
				t.Errorf("PerReadChoice() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("PerReadChoice() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPerWriteChoice(t *testing.T) {
	tests := []struct {
		name   string
		choice uint8
		want   []byte
	}{
		{name: "choice 0", choice: 0, want: []byte{0x00}},
		{name: "choice 1", choice: 1, want: []byte{0x01}},
		{name: "choice 255", choice: 255, want: []byte{0xFF}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			PerWriteChoice(tt.choice, &buf)
			if !bytes.Equal(buf.Bytes(), tt.want) {
				t.Errorf("PerWriteChoice(%d) = %v, want %v", tt.choice, buf.Bytes(), tt.want)
			}
		})
	}
}

func TestPerReadLength(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		want    int
		wantErr bool
	}{
		// Short form (0x00-0x7F)
		{name: "short form 0", input: []byte{0x00}, want: 0},
		{name: "short form 1", input: []byte{0x01}, want: 1},
		{name: "short form 127", input: []byte{0x7F}, want: 127},

		// Long form (0x80 + 2 bytes)
		{name: "long form 128", input: []byte{0x80, 0x80}, want: 128},
		{name: "long form 255", input: []byte{0x80, 0xFF}, want: 255},
		{name: "long form 256", input: []byte{0x81, 0x00}, want: 256},
		{name: "long form 1000", input: []byte{0x83, 0xE8}, want: 1000},
		{name: "long form 0x7FFF", input: []byte{0xFF, 0xFF}, want: 0x7FFF},

		// Error cases
		{name: "empty", input: []byte{}, wantErr: true},
		{name: "truncated long form", input: []byte{0x80}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewReader(tt.input)
			got, err := PerReadLength(r)
			if (err != nil) != tt.wantErr {
				t.Errorf("PerReadLength() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("PerReadLength() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPerWriteLength(t *testing.T) {
	tests := []struct {
		name  string
		value uint16
		want  []byte
	}{
		// Short form (0x00-0x7F)
		{name: "short form 0", value: 0, want: []byte{0x00}},
		{name: "short form 1", value: 1, want: []byte{0x01}},
		{name: "short form 127", value: 127, want: []byte{0x7F}},

		// Long form (value | 0x8000)
		{name: "long form 128", value: 128, want: []byte{0x80, 0x80}},
		{name: "long form 255", value: 255, want: []byte{0x80, 0xFF}},
		{name: "long form 256", value: 256, want: []byte{0x81, 0x00}},
		{name: "long form 1000", value: 1000, want: []byte{0x83, 0xE8}},
		{name: "long form 0x7FFF", value: 0x7FFF, want: []byte{0xFF, 0xFF}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			PerWriteLength(tt.value, &buf)
			if !bytes.Equal(buf.Bytes(), tt.want) {
				t.Errorf("PerWriteLength(%d) = %v, want %v", tt.value, buf.Bytes(), tt.want)
			}
		})
	}
}

func TestPerReadWriteLengthRoundTrip(t *testing.T) {
	sizes := []uint16{0, 1, 50, 127, 128, 200, 255, 256, 1000, 0x7FFF}

	for _, size := range sizes {
		var buf bytes.Buffer
		PerWriteLength(size, &buf)
		got, err := PerReadLength(bytes.NewReader(buf.Bytes()))
		if err != nil {
			t.Errorf("Round trip failed for size %d: %v", size, err)
			continue
		}
		if uint16(got) != size {
			t.Errorf("Round trip size %d: got %d", size, got)
		}
	}
}

func TestPerReadInteger(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		want    int
		wantErr bool
	}{
		// 1-byte integers
		{name: "1 byte - 0", input: []byte{0x01, 0x00}, want: 0},
		{name: "1 byte - 1", input: []byte{0x01, 0x01}, want: 1},
		{name: "1 byte - 255", input: []byte{0x01, 0xFF}, want: 255},

		// 2-byte integers
		{name: "2 bytes - 256", input: []byte{0x02, 0x01, 0x00}, want: 256},
		{name: "2 bytes - 0xFFFF", input: []byte{0x02, 0xFF, 0xFF}, want: 0xFFFF},

		// 4-byte integers
		{name: "4 bytes - 0x10000", input: []byte{0x04, 0x00, 0x01, 0x00, 0x00}, want: 0x10000},

		// Error cases
		{name: "empty", input: []byte{}, wantErr: true},
		{name: "invalid size 3", input: []byte{0x03, 0x01, 0x02, 0x03}, wantErr: true},
		{name: "truncated", input: []byte{0x02, 0x01}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewReader(tt.input)
			got, err := PerReadInteger(r)
			if (err != nil) != tt.wantErr {
				t.Errorf("PerReadInteger() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("PerReadInteger() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPerWriteInteger(t *testing.T) {
	tests := []struct {
		name  string
		value int
		want  []byte
	}{
		{name: "zero", value: 0, want: []byte{0x01, 0x00}},
		{name: "1", value: 1, want: []byte{0x01, 0x01}},
		{name: "255", value: 255, want: []byte{0x01, 0xFF}},
		{name: "256", value: 256, want: []byte{0x02, 0x01, 0x00}},
		{name: "0xFFFE", value: 0xFFFE, want: []byte{0x02, 0xFF, 0xFE}},
		{name: "0xFFFF", value: 0xFFFF, want: []byte{0x02, 0xFF, 0xFF}},
		{name: "0x10000", value: 0x10000, want: []byte{0x04, 0x00, 0x01, 0x00, 0x00}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			PerWriteInteger(tt.value, &buf)
			if !bytes.Equal(buf.Bytes(), tt.want) {
				t.Errorf("PerWriteInteger(%d) = %v, want %v", tt.value, buf.Bytes(), tt.want)
			}
		})
	}
}

func TestPerReadInteger16(t *testing.T) {
	tests := []struct {
		name    string
		minimum uint16
		input   []byte
		want    uint16
		wantErr bool
	}{
		{name: "zero with min 0", minimum: 0, input: []byte{0x00, 0x00}, want: 0},
		{name: "1 with min 0", minimum: 0, input: []byte{0x00, 0x01}, want: 1},
		{name: "100 with min 50", minimum: 50, input: []byte{0x00, 0x32}, want: 100},
		{name: "0xFFFF with min 0", minimum: 0, input: []byte{0xFF, 0xFF}, want: 0xFFFF},

		// Error cases
		{name: "empty", minimum: 0, input: []byte{}, wantErr: true},
		{name: "truncated", minimum: 0, input: []byte{0x00}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewReader(tt.input)
			got, err := PerReadInteger16(tt.minimum, r)
			if (err != nil) != tt.wantErr {
				t.Errorf("PerReadInteger16() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("PerReadInteger16() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPerWriteInteger16(t *testing.T) {
	tests := []struct {
		name    string
		value   uint16
		minimum uint16
		want    []byte
	}{
		{name: "0 with min 0", value: 0, minimum: 0, want: []byte{0x00, 0x00}},
		{name: "100 with min 0", value: 100, minimum: 0, want: []byte{0x00, 0x64}},
		{name: "100 with min 50", value: 100, minimum: 50, want: []byte{0x00, 0x32}},
		{name: "0xFFFF with min 0", value: 0xFFFF, minimum: 0, want: []byte{0xFF, 0xFF}},
		{name: "1000 with min 500", value: 1000, minimum: 500, want: []byte{0x01, 0xF4}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			PerWriteInteger16(tt.value, tt.minimum, &buf)
			if !bytes.Equal(buf.Bytes(), tt.want) {
				t.Errorf("PerWriteInteger16(%d, %d) = %v, want %v", tt.value, tt.minimum, buf.Bytes(), tt.want)
			}
		})
	}
}

func TestPerReadInteger16WriteRoundTrip(t *testing.T) {
	testCases := []struct {
		value   uint16
		minimum uint16
	}{
		{0, 0},
		{100, 0},
		{100, 50},
		{1000, 500},
		{0xFFFF, 0},
	}

	for _, tc := range testCases {
		var buf bytes.Buffer
		PerWriteInteger16(tc.value, tc.minimum, &buf)
		got, err := PerReadInteger16(tc.minimum, bytes.NewReader(buf.Bytes()))
		if err != nil {
			t.Errorf("Round trip failed for value=%d, min=%d: %v", tc.value, tc.minimum, err)
			continue
		}
		if got != tc.value {
			t.Errorf("Round trip value=%d, min=%d: got %d", tc.value, tc.minimum, got)
		}
	}
}

func TestPerReadObjectIdentifier(t *testing.T) {
	tests := []struct {
		name    string
		oid     [6]byte
		input   []byte
		want    bool
		wantErr bool
	}{
		// Valid OID match
		{
			name:  "matching OID",
			oid:   [6]byte{0x00, 0x05, 0x00, 0x14, 0x7C, 0x00},
			input: []byte{0x05, 0x05, 0x00, 0x14, 0x7C, 0x00}, // length 5, then t12 combined, then 4 bytes
			want:  true,
		},
		// Non-matching OID
		{
			name:  "non-matching OID",
			oid:   [6]byte{0x00, 0x05, 0x00, 0x14, 0x7C, 0x00},
			input: []byte{0x05, 0x05, 0x00, 0x14, 0x7C, 0x01}, // last byte different
			want:  false,
		},
		// Wrong size
		{
			name:  "wrong size",
			oid:   [6]byte{0x00, 0x05, 0x00, 0x14, 0x7C, 0x00},
			input: []byte{0x04, 0x00, 0x00, 0x00, 0x00}, // size 4 instead of 5
			want:  false,
		},

		// Error cases
		{name: "empty", oid: [6]byte{}, input: []byte{}, wantErr: true},
		{name: "truncated", oid: [6]byte{}, input: []byte{0x05}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewReader(tt.input)
			got, err := PerReadObjectIdentifier(tt.oid, r)
			if (err != nil) != tt.wantErr {
				t.Errorf("PerReadObjectIdentifier() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("PerReadObjectIdentifier() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPerWriteObjectIdentifier(t *testing.T) {
	tests := []struct {
		name string
		oid  [6]byte
	}{
		{name: "standard OID", oid: [6]byte{0x00, 0x05, 0x00, 0x14, 0x7C, 0x00}},
		{name: "all zeros", oid: [6]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00}},
		{name: "all ones", oid: [6]byte{0x0F, 0x0F, 0xFF, 0xFF, 0xFF, 0xFF}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			PerWriteObjectIdentifier(tt.oid, &buf)

			result := buf.Bytes()
			// Should be: length (5), combined byte, 4 more bytes
			if len(result) != 6 {
				t.Errorf("PerWriteObjectIdentifier() length = %d, want 6", len(result))
			}
			if result[0] != 0x05 {
				t.Errorf("PerWriteObjectIdentifier() length byte = %v, want 0x05", result[0])
			}
		})
	}
}

func TestPerReadEnumerates(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		want    uint8
		wantErr bool
	}{
		{name: "enum 0", input: []byte{0x00}, want: 0},
		{name: "enum 1", input: []byte{0x01}, want: 1},
		{name: "enum 255", input: []byte{0xFF}, want: 255},

		{name: "empty", input: []byte{}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewReader(tt.input)
			got, err := PerReadEnumerates(r)
			if (err != nil) != tt.wantErr {
				t.Errorf("PerReadEnumerates() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("PerReadEnumerates() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPerReadNumberOfSet(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		want    uint8
		wantErr bool
	}{
		{name: "set 0", input: []byte{0x00}, want: 0},
		{name: "set 1", input: []byte{0x01}, want: 1},
		{name: "set 255", input: []byte{0xFF}, want: 255},

		{name: "empty", input: []byte{}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewReader(tt.input)
			got, err := PerReadNumberOfSet(r)
			if (err != nil) != tt.wantErr {
				t.Errorf("PerReadNumberOfSet() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("PerReadNumberOfSet() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPerWriteSelection(t *testing.T) {
	tests := []struct {
		name      string
		selection uint8
		want      []byte
	}{
		{name: "selection 0", selection: 0, want: []byte{0x00}},
		{name: "selection 1", selection: 1, want: []byte{0x01}},
		{name: "selection 255", selection: 255, want: []byte{0xFF}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			PerWriteSelection(tt.selection, &buf)
			if !bytes.Equal(buf.Bytes(), tt.want) {
				t.Errorf("PerWriteSelection(%d) = %v, want %v", tt.selection, buf.Bytes(), tt.want)
			}
		})
	}
}

func TestPerWriteNumberOfSet(t *testing.T) {
	tests := []struct {
		name        string
		numberOfSet uint8
		want        []byte
	}{
		{name: "numberOfSet 0", numberOfSet: 0, want: []byte{0x00}},
		{name: "numberOfSet 1", numberOfSet: 1, want: []byte{0x01}},
		{name: "numberOfSet 255", numberOfSet: 255, want: []byte{0xFF}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			PerWriteNumberOfSet(tt.numberOfSet, &buf)
			if !bytes.Equal(buf.Bytes(), tt.want) {
				t.Errorf("PerWriteNumberOfSet(%d) = %v, want %v", tt.numberOfSet, buf.Bytes(), tt.want)
			}
		})
	}
}

func TestPerWritePadding(t *testing.T) {
	tests := []struct {
		name   string
		length int
		want   []byte
	}{
		{name: "padding 0", length: 0, want: []byte{}},
		{name: "padding 1", length: 1, want: []byte{0x00}},
		{name: "padding 5", length: 5, want: []byte{0x00, 0x00, 0x00, 0x00, 0x00}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			PerWritePadding(tt.length, &buf)
			if !bytes.Equal(buf.Bytes(), tt.want) {
				t.Errorf("PerWritePadding(%d) = %v, want %v", tt.length, buf.Bytes(), tt.want)
			}
		})
	}
}

func TestPerWriteNumericString(t *testing.T) {
	tests := []struct {
		name     string
		nStr     string
		minValue int
		want     []byte
	}{
		{name: "empty", nStr: "", minValue: 0, want: []byte{0x00}},
		{name: "single digit", nStr: "1", minValue: 0, want: []byte{0x01, 0x10}},
		{name: "two digits", nStr: "12", minValue: 0, want: []byte{0x02, 0x12}},
		{name: "four digits", nStr: "1234", minValue: 0, want: []byte{0x04, 0x12, 0x34}},
		{name: "with minValue", nStr: "1234", minValue: 2, want: []byte{0x02, 0x12, 0x34}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			PerWriteNumericString(tt.nStr, tt.minValue, &buf)
			if !bytes.Equal(buf.Bytes(), tt.want) {
				t.Errorf("PerWriteNumericString(%q, %d) = %v, want %v", tt.nStr, tt.minValue, buf.Bytes(), tt.want)
			}
		})
	}
}

func TestPerWriteOctetStream(t *testing.T) {
	tests := []struct {
		name     string
		oStr     string
		minValue int
		want     []byte
	}{
		{name: "empty", oStr: "", minValue: 0, want: []byte{0x00}},
		{name: "hello", oStr: "hello", minValue: 0, want: []byte{0x05, 'h', 'e', 'l', 'l', 'o'}},
		{name: "with minValue", oStr: "hello", minValue: 2, want: []byte{0x03, 'h', 'e', 'l', 'l', 'o'}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			PerWriteOctetStream(tt.oStr, tt.minValue, &buf)
			if !bytes.Equal(buf.Bytes(), tt.want) {
				t.Errorf("PerWriteOctetStream(%q, %d) = %v, want %v", tt.oStr, tt.minValue, buf.Bytes(), tt.want)
			}
		})
	}
}

func TestPerReadOctetStream(t *testing.T) {
	tests := []struct {
		name        string
		octetStream []byte
		minValue    int
		input       []byte
		want        bool
		wantErr     bool
	}{
		{
			name:        "matching stream",
			octetStream: []byte("hello"),
			minValue:    0,
			input:       []byte{0x05, 'h', 'e', 'l', 'l', 'o'},
			want:        true,
		},
		{
			name:        "matching with minValue",
			octetStream: []byte("hello"),
			minValue:    2,
			input:       []byte{0x03, 'h', 'e', 'l', 'l', 'o'},
			want:        true,
		},
		{
			name:        "non-matching content",
			octetStream: []byte("hello"),
			minValue:    0,
			input:       []byte{0x05, 'h', 'e', 'l', 'l', 'x'},
			want:        false,
		},
		{
			name:        "size mismatch",
			octetStream: []byte("hello"),
			minValue:    0,
			input:       []byte{0x04, 'h', 'e', 'l', 'l'},
			want:        false,
		},

		// Error cases
		{name: "empty", octetStream: []byte{}, minValue: 0, input: []byte{}, wantErr: true},
		{name: "truncated", octetStream: []byte("hello"), minValue: 0, input: []byte{0x05, 'h'}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewReader(tt.input)
			got, err := PerReadOctetStream(tt.octetStream, tt.minValue, r)
			if (err != nil) != tt.wantErr {
				t.Errorf("PerReadOctetStream() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("PerReadOctetStream() = %v, want %v", got, tt.want)
			}
		})
	}
}

// ============================================================================
// Edge Case and Integration Tests
// ============================================================================

func TestBerReadLengthEdgeCases(t *testing.T) {
	// Test reading from an empty reader
	emptyReader := bytes.NewReader([]byte{})
	_, err := BerReadLength(emptyReader)
	if err != io.EOF {
		t.Errorf("Expected EOF for empty reader, got %v", err)
	}

	// Test boundary between short and long form
	var buf bytes.Buffer
	BerWriteLength(0x7F, &buf)
	if buf.Len() != 1 {
		t.Errorf("0x7F should use short form (1 byte), got %d bytes", buf.Len())
	}

	buf.Reset()
	BerWriteLength(0x80, &buf)
	if buf.Len() != 2 {
		t.Errorf("0x80 should use long form (2 bytes), got %d bytes", buf.Len())
	}
}

func TestPerReadLengthEdgeCases(t *testing.T) {
	// Test reading from an empty reader
	emptyReader := bytes.NewReader([]byte{})
	_, err := PerReadLength(emptyReader)
	if err != io.EOF {
		t.Errorf("Expected EOF for empty reader, got %v", err)
	}

	// Test boundary between short and long form
	var buf bytes.Buffer
	PerWriteLength(0x7F, &buf)
	if buf.Len() != 1 {
		t.Errorf("0x7F should use short form (1 byte), got %d bytes", buf.Len())
	}

	buf.Reset()
	PerWriteLength(0x80, &buf)
	if buf.Len() != 2 {
		t.Errorf("0x80 should use long form (2 bytes), got %d bytes", buf.Len())
	}
}

func TestBerIntegerBoundaryValues(t *testing.T) {
	// Test boundary values for BerWriteInteger
	boundaries := []struct {
		value       int
		expectedLen int
	}{
		{0, 3},       // tag + length + 1 byte
		{0xFF, 3},    // tag + length + 1 byte
		{0x100, 4},   // tag + length + 2 bytes
		{0xFFFF, 4},  // tag + length + 2 bytes
		{0x10000, 6}, // tag + length + 4 bytes
	}

	for _, b := range boundaries {
		var buf bytes.Buffer
		BerWriteInteger(b.value, &buf)
		if buf.Len() != b.expectedLen {
			t.Errorf("BerWriteInteger(%d) length = %d, want %d", b.value, buf.Len(), b.expectedLen)
		}
	}
}

func TestPerIntegerBoundaryValues(t *testing.T) {
	// Test boundary values for PerWriteInteger
	boundaries := []struct {
		value       int
		expectedLen int
	}{
		{0, 2},       // length + 1 byte
		{0xFF, 2},    // length + 1 byte
		{0x100, 3},   // length + 2 bytes
		{0xFFFE, 3},  // length + 2 bytes
		{0xFFFF, 3},  // length + 2 bytes
		{0x10000, 5}, // length + 4 bytes
	}

	for _, b := range boundaries {
		var buf bytes.Buffer
		PerWriteInteger(b.value, &buf)
		if buf.Len() != b.expectedLen {
			t.Errorf("PerWriteInteger(%d) length = %d, want %d", b.value, buf.Len(), b.expectedLen)
		}
	}
}

// Test berPC helper function
func TestBerPC(t *testing.T) {
	// Test through BerReadUniversalTag
	var buf bytes.Buffer

	// Primitive tag
	buf.Write([]byte{ClassUniversal | PCPrimitive | TagInteger})
	match, err := BerReadUniversalTag(TagInteger, false, &buf)
	if err != nil || !match {
		t.Errorf("berPC(false) should match PCPrimitive")
	}

	// Constructed tag
	buf.Reset()
	buf.Write([]byte{ClassUniversal | PCConstruct | TagSequence})
	match, err = BerReadUniversalTag(TagSequence, true, &buf)
	if err != nil || !match {
		t.Errorf("berPC(true) should match PCConstruct")
	}
}

// Test ASN.1 constants are used correctly
func TestASN1Constants(t *testing.T) {
	// Verify constants are correct
	if ClassUniversal != 0x00 {
		t.Errorf("ClassUniversal = %v, want 0x00", ClassUniversal)
	}
	if ClassApplication != 0x40 {
		t.Errorf("ClassApplication = %v, want 0x40", ClassApplication)
	}
	if PCPrimitive != 0x00 {
		t.Errorf("PCPrimitive = %v, want 0x00", PCPrimitive)
	}
	if PCConstruct != 0x20 {
		t.Errorf("PCConstruct = %v, want 0x20", PCConstruct)
	}
	if TagMask != 0x1F {
		t.Errorf("TagMask = %v, want 0x1F", TagMask)
	}
	if TagInteger != 0x02 {
		t.Errorf("TagInteger = %v, want 0x02", TagInteger)
	}
	if TagBoolean != 0x01 {
		t.Errorf("TagBoolean = %v, want 0x01", TagBoolean)
	}
	if TagSequence != 0x10 {
		t.Errorf("TagSequence = %v, want 0x10", TagSequence)
	}
}

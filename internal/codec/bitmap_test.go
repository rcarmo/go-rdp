package codec

import (
	"bytes"
	"testing"
)

func TestFlipVertical(t *testing.T) {
	tests := []struct {
		name          string
		data          []byte
		width         int
		height        int
		bytesPerPixel int
		expected      []byte
	}{
		{
			name:          "2x2 image 1 bpp",
			data:          []byte{0x01, 0x02, 0x03, 0x04},
			width:         2,
			height:        2,
			bytesPerPixel: 1,
			expected:      []byte{0x03, 0x04, 0x01, 0x02},
		},
		{
			name:          "2x3 image 1 bpp",
			data:          []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06},
			width:         2,
			height:        3,
			bytesPerPixel: 1,
			expected:      []byte{0x05, 0x06, 0x03, 0x04, 0x01, 0x02},
		},
		{
			name:          "single row unchanged",
			data:          []byte{0x01, 0x02, 0x03, 0x04},
			width:         4,
			height:        1,
			bytesPerPixel: 1,
			expected:      []byte{0x01, 0x02, 0x03, 0x04},
		},
		{
			name:          "2x2 image 4 bpp (RGBA)",
			data:          []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18},
			width:         2,
			height:        2,
			bytesPerPixel: 4,
			expected:      []byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := make([]byte, len(tt.data))
			copy(data, tt.data)
			FlipVertical(data, tt.width, tt.height, tt.bytesPerPixel)
			if !bytes.Equal(data, tt.expected) {
				t.Errorf("FlipVertical() = %v, want %v", data, tt.expected)
			}
		})
	}
}

func TestFlipVertical_EdgeCases(t *testing.T) {
	// Empty data
	data := []byte{}
	FlipVertical(data, 0, 0, 1)

	// Invalid dimensions
	data = []byte{0x01, 0x02}
	FlipVertical(data, 0, 2, 1) // zero width

	// Data too short
	data = []byte{0x01}
	FlipVertical(data, 2, 2, 1) // expects 4 bytes
}

func TestSetPalette(t *testing.T) {
	// RGB palette data (3 bytes per color)
	paletteData := []byte{
		0xFF, 0x00, 0x00, // Red
		0x00, 0xFF, 0x00, // Green
		0x00, 0x00, 0xFF, // Blue
	}

	SetPalette(paletteData, 3)

	// Verify palette was set (use Palette8ToRGBA to check)
	src := []byte{0, 1, 2}
	dst := make([]byte, 12)
	Palette8ToRGBA(src, dst)

	// Check red
	if dst[0] != 0xFF || dst[1] != 0x00 || dst[2] != 0x00 || dst[3] != 0xFF {
		t.Errorf("Palette[0] incorrect: got %v", dst[0:4])
	}
	// Check green
	if dst[4] != 0x00 || dst[5] != 0xFF || dst[6] != 0x00 || dst[7] != 0xFF {
		t.Errorf("Palette[1] incorrect: got %v", dst[4:8])
	}
	// Check blue
	if dst[8] != 0x00 || dst[9] != 0x00 || dst[10] != 0xFF || dst[11] != 0xFF {
		t.Errorf("Palette[2] incorrect: got %v", dst[8:12])
	}
}

func TestSetPalette_MoreThan256(t *testing.T) {
	// Should cap at 256
	paletteData := make([]byte, 300*3)
	SetPalette(paletteData, 300) // Should not panic
}

func TestPalette8ToRGBA(t *testing.T) {
	// Use default palette (index 0 is black, 255 is white)
	src := []byte{0, 255}
	dst := make([]byte, 8)

	// Reset to default palette first
	SetPalette([]byte{0, 0, 0, 255, 255, 255}, 2)

	Palette8ToRGBA(src, dst)

	// Index 0 should be black (after our SetPalette)
	if dst[0] != 0x00 || dst[1] != 0x00 || dst[2] != 0x00 || dst[3] != 0xFF {
		t.Errorf("Palette8ToRGBA index 0: got %v, want black", dst[0:4])
	}
}

func TestRGB555ToRGBA(t *testing.T) {
	// RGB555: 5 bits R, 5 bits G, 5 bits B
	// Full red: 0x7C00 (01111100 00000000)
	// Full green: 0x03E0 (00000011 11100000)
	// Full blue: 0x001F (00000000 00011111)

	tests := []struct {
		name   string
		src    []byte
		expect []byte
	}{
		{
			name:   "black",
			src:    []byte{0x00, 0x00},
			expect: []byte{0x00, 0x00, 0x00, 0xFF},
		},
		{
			name:   "white",
			src:    []byte{0xFF, 0x7F}, // 0x7FFF
			expect: []byte{0xFF, 0xFF, 0xFF, 0xFF},
		},
		{
			name:   "red",
			src:    []byte{0x00, 0x7C}, // 0x7C00
			expect: []byte{0xFF, 0x00, 0x00, 0xFF},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dst := make([]byte, 4)
			RGB555ToRGBA(tt.src, dst)
			if !bytes.Equal(dst, tt.expect) {
				t.Errorf("RGB555ToRGBA() = %v, want %v", dst, tt.expect)
			}
		})
	}
}

func TestRGB565ToRGBA(t *testing.T) {
	// RGB565: 5 bits R, 6 bits G, 5 bits B
	tests := []struct {
		name   string
		src    []byte
		expect []byte
	}{
		{
			name:   "black",
			src:    []byte{0x00, 0x00},
			expect: []byte{0x00, 0x00, 0x00, 0xFF},
		},
		{
			name:   "white",
			src:    []byte{0xFF, 0xFF},
			expect: []byte{0xFF, 0xFF, 0xFF, 0xFF},
		},
		{
			name:   "red",
			src:    []byte{0x00, 0xF8}, // 0xF800
			expect: []byte{0xFF, 0x00, 0x00, 0xFF},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dst := make([]byte, 4)
			RGB565ToRGBA(tt.src, dst)
			if !bytes.Equal(dst, tt.expect) {
				t.Errorf("RGB565ToRGBA() = %v, want %v", dst, tt.expect)
			}
		})
	}
}

func TestBGR24ToRGBA(t *testing.T) {
	// BGR order: Blue, Green, Red
	src := []byte{0x00, 0x00, 0xFF} // Blue=0, Green=0, Red=255
	dst := make([]byte, 4)

	BGR24ToRGBA(src, dst)

	// Should be RGBA: R=255, G=0, B=0, A=255
	expected := []byte{0xFF, 0x00, 0x00, 0xFF}
	if !bytes.Equal(dst, expected) {
		t.Errorf("BGR24ToRGBA() = %v, want %v", dst, expected)
	}
}

func TestBGRA32ToRGBA(t *testing.T) {
	// BGRA order: Blue, Green, Red, Alpha
	src := []byte{0x00, 0x00, 0xFF, 0x80} // Blue=0, Green=0, Red=255, Alpha=128
	dst := make([]byte, 4)

	BGRA32ToRGBA(src, dst)

	// Should be RGBA: R=255, G=0, B=0, A=255 (alpha forced to 255)
	expected := []byte{0xFF, 0x00, 0x00, 0xFF}
	if !bytes.Equal(dst, expected) {
		t.Errorf("BGRA32ToRGBA() = %v, want %v", dst, expected)
	}
}

func TestProcessBitmap_Uncompressed(t *testing.T) {
	tests := []struct {
		name   string
		src    []byte
		width  int
		height int
		bpp    int
	}{
		{
			name:   "8-bit 2x1",
			src:    []byte{0, 1},
			width:  2,
			height: 1,
			bpp:    8,
		},
		{
			name:   "16-bit 1x1",
			src:    []byte{0xFF, 0xFF}, // White in RGB565
			width:  1,
			height: 1,
			bpp:    16,
		},
		{
			name:   "24-bit 1x1",
			src:    []byte{0xFF, 0x00, 0x00}, // Blue in BGR
			width:  1,
			height: 1,
			bpp:    24,
		},
		{
			name:   "32-bit 1x1",
			src:    []byte{0xFF, 0x00, 0x00, 0xFF}, // Blue in BGRA
			width:  1,
			height: 1,
			bpp:    32,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ProcessBitmap(tt.src, tt.width, tt.height, tt.bpp, false, 0, false)
			if result == nil {
				t.Error("ProcessBitmap() returned nil")
			}
			expectedLen := tt.width * tt.height * 4
			if len(result) != expectedLen {
				t.Errorf("ProcessBitmap() len = %d, want %d", len(result), expectedLen)
			}
		})
	}
}

func TestProcessBitmap_UnsupportedBpp(t *testing.T) {
	src := []byte{0x00}
	result := ProcessBitmap(src, 1, 1, 7, false, 0, false)
	if result != nil {
		t.Error("ProcessBitmap() should return nil for unsupported bpp")
	}
}

func TestProcessBitmap_15bit(t *testing.T) {
	// Test 15-bit color depth (RGB555)
	src := []byte{0xFF, 0x7F} // White in RGB555
	result := ProcessBitmap(src, 1, 1, 15, false, 0, false)
	if result == nil {
		t.Fatal("ProcessBitmap() returned nil for 15-bit")
	}
	// Should be white RGBA
	if result[0] != 0xFF || result[1] != 0xFF || result[2] != 0xFF {
		t.Errorf("ProcessBitmap() 15-bit white = %v, want white", result[:3])
	}
}

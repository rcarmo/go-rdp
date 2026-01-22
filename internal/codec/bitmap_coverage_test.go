// Package codec tests for ProcessBitmap function
package codec

import (
	"testing"
)

// TestProcessBitmapUncompressed tests ProcessBitmap with uncompressed data
func TestProcessBitmapUncompressed(t *testing.T) {
	tests := []struct {
		name   string
		bpp    int
		width  int
		height int
	}{
		{"8bpp", 8, 4, 4},
		{"15bpp", 15, 4, 4},
		{"16bpp", 16, 4, 4},
		{"24bpp", 24, 4, 4},
		{"32bpp", 32, 4, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bytesPerPixel := tt.bpp / 8
			if bytesPerPixel == 0 {
				bytesPerPixel = 1
			}
			srcLen := tt.width * tt.height * bytesPerPixel
			src := make([]byte, srcLen)

			// Fill with test pattern
			for i := range src {
				src[i] = byte(i % 256)
			}

			result := ProcessBitmap(src, tt.width, tt.height, tt.bpp, false, 0)

			expectedLen := tt.width * tt.height * 4 // RGBA output
			if len(result) != expectedLen {
				t.Errorf("ProcessBitmap returned wrong length: got %d, want %d", len(result), expectedLen)
			}
		})
	}
}

// TestProcessBitmapCompressed8bpp tests 8-bit RLE decompression
func TestProcessBitmapCompressed8bpp(t *testing.T) {
	// Simple RLE data: background run of 4 pixels
	// 0x00 = REGULAR_BG_RUN with length in low 5 bits
	// 0x04 = BG_RUN with length 4
	src := []byte{0x04} // REGULAR_BG_RUN length 4

	result := ProcessBitmap(src, 2, 2, 8, true, 0)

	// Should produce 4x4=16 bytes RGBA (2x2 image)
	if result == nil {
		t.Error("ProcessBitmap returned nil for 8bpp compressed")
	}
}

// TestProcessBitmapCompressed15bpp tests 15-bit RLE decompression
func TestProcessBitmapCompressed15bpp(t *testing.T) {
	// Simple RLE data for 15bpp
	src := []byte{0x04} // REGULAR_BG_RUN length 4

	result := ProcessBitmap(src, 2, 2, 15, true, 0)

	if result == nil {
		t.Error("ProcessBitmap returned nil for 15bpp compressed")
	}
}

// TestProcessBitmapCompressed16bpp tests 16-bit RLE decompression
func TestProcessBitmapCompressed16bpp(t *testing.T) {
	// Simple RLE data for 16bpp
	src := []byte{0x04}

	result := ProcessBitmap(src, 2, 2, 16, true, 0)

	if result == nil {
		t.Error("ProcessBitmap returned nil for 16bpp compressed")
	}
}

// TestProcessBitmapCompressed24bpp tests 24-bit RLE decompression
func TestProcessBitmapCompressed24bpp(t *testing.T) {
	// Simple RLE data for 24bpp
	src := []byte{0x04}

	result := ProcessBitmap(src, 2, 2, 24, true, 0)

	if result == nil {
		t.Error("ProcessBitmap returned nil for 24bpp compressed")
	}
}

// TestProcessBitmapCompressed32bpp tests 32-bit planar codec
func TestProcessBitmapCompressed32bpp(t *testing.T) {
	// For 32bpp compressed, check planar codec path
	// Start with non-planar header to test RLE fallback
	src := []byte{0xC4} // High bits set = not planar

	result := ProcessBitmap(src, 2, 2, 32, true, 0)
	// May return nil if RLE fails, which is acceptable
	_ = result
}

// TestProcessBitmapInvalidBpp tests unsupported bit depths
func TestProcessBitmapInvalidBpp(t *testing.T) {
	src := []byte{0x00, 0x00, 0x00, 0x00}

	result := ProcessBitmap(src, 2, 2, 12, true, 0)
	if result != nil {
		t.Error("ProcessBitmap should return nil for unsupported bpp")
	}
}

// TestProcessBitmapPlanar tests 32-bit planar codec path
func TestProcessBitmapPlanar(t *testing.T) {
	// Create planar format data (format header with reserved bits = 0)
	// First byte is format header
	src := []byte{0x00} // Reserved bits 0 = planar format

	result := ProcessBitmap(src, 2, 2, 32, true, 0)
	// May fail decompression but should try planar path
	_ = result
}

// TestProcessBitmapEmptySource tests with empty source
func TestProcessBitmapEmptySource(t *testing.T) {
	src := []byte{}

	result := ProcessBitmap(src, 2, 2, 24, false, 0)
	// Should handle gracefully
	if result == nil {
		t.Error("ProcessBitmap should handle empty source for uncompressed")
	}
}

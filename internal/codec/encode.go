package codec

import (
	"bytes"
	"encoding/binary"
	"unicode/utf16"
)

// Encode converts a string to UTF-16LE encoded bytes.
func Encode(s string) []byte {
	buf := new(bytes.Buffer)

	for _, ch := range utf16.Encode([]rune(s)) {
		_ = binary.Write(buf, binary.LittleEndian, ch)
	}

	return buf.Bytes()
}

package codec

import (
	"bytes"
	"encoding/binary"
	"io"
)

// WrapSecurityFlag wraps data with an RDP security header containing the specified flag.
func WrapSecurityFlag(flag uint16, data []byte) []byte {
	buf := new(bytes.Buffer)

	_ = binary.Write(buf, binary.LittleEndian, flag)
	buf.Write([]byte{0x00, 0x00}) // flagsHi

	buf.Write(data)

	return buf.Bytes()
}

// UnwrapSecurityFlag reads and returns the security flag from an RDP security header.
func UnwrapSecurityFlag(wire io.Reader) (uint16, error) {
	var (
		flags   uint16
		flagsHi uint16
		err     error
	)

	err = binary.Read(wire, binary.LittleEndian, &flags)
	if err != nil {
		return 0, err
	}

	err = binary.Read(wire, binary.LittleEndian, &flagsHi)
	if err != nil {
		return 0, err
	}

	return flags, nil
}

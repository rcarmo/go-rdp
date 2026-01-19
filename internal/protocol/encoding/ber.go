package encoding

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// BER reading functions

func BerReadApplicationTag(r io.Reader) (uint8, error) {
	var (
		identifier uint8
		tag        uint8
		err        error
	)

	err = binary.Read(r, binary.BigEndian, &identifier)
	if err != nil {
		return 0, err
	}

	if identifier != (ClassApplication|PCConstruct)|TagMask {
		return 0, errors.New("ReadApplicationTag invalid data")
	}

	err = binary.Read(r, binary.BigEndian, &tag)
	if err != nil {
		return 0, err
	}

	return tag, nil
}

func BerReadLength(r io.Reader) (uint16, error) {
	var (
		ret  uint16
		size uint8
		err  error
	)

	err = binary.Read(r, binary.BigEndian, &size)
	if err != nil {
		return 0, err
	}

	if size&0x80 > 0 {
		size = size &^ 0x80

		if size == 1 {
			err = binary.Read(r, binary.BigEndian, &size)
			if err != nil {
				return 0, err
			}

			ret = uint16(size)
		} else if size == 2 {
			err = binary.Read(r, binary.BigEndian, &ret)
			if err != nil {
				return 0, err
			}
		} else {
			return 0, errors.New("BER length may be 1 or 2")
		}
	} else {
		ret = uint16(size)
	}

	return ret, nil
}

func berPC(pc bool) uint8 {
	if pc {
		return PCConstruct
	}
	return PCPrimitive
}

func BerReadUniversalTag(tag uint8, pc bool, r io.Reader) (bool, error) {
	var bb uint8

	err := binary.Read(r, binary.BigEndian, &bb)
	if err != nil {
		return false, err
	}

	return bb == (ClassUniversal|berPC(pc))|(TagMask&tag), nil
}

func BerReadEnumerated(r io.Reader) (uint8, error) {
	universalTag, err := BerReadUniversalTag(TagEnumerated, false, r)
	if err != nil {
		return 0, err
	}

	if !universalTag {
		return 0, errors.New("invalid ber tag")
	}

	length, err := BerReadLength(r)
	if err != nil {
		return 0, err
	}

	if length != 1 {
		return 0, fmt.Errorf("enumerate size is wrong, get %v, expect 1", length)
	}

	var enumerated uint8

	return enumerated, binary.Read(r, binary.BigEndian, &enumerated)
}

func BerReadInteger(r io.Reader) (int, error) {
	universalTag, err := BerReadUniversalTag(TagInteger, false, r)
	if err != nil {
		return 0, err
	}

	if !universalTag {
		return 0, errors.New("Bad integer tag")
	}

	size, err := BerReadLength(r)
	if err != nil {
		return 0, err
	}

	switch size {
	case 1:
		var num uint8

		return int(num), binary.Read(r, binary.BigEndian, &num)
	case 2:
		var num uint16

		return int(num), binary.Read(r, binary.BigEndian, &num)
	case 3:
		var (
			int1 uint8
			int2 uint16
		)

		err = binary.Read(r, binary.BigEndian, &int1)
		if err != nil {
			return 0, err
		}

		err = binary.Read(r, binary.BigEndian, &int2)
		if err != nil {
			return 0, err
		}

		return int(int1)<<0x10 + int(int2), nil
	case 4:
		var num uint32

		return int(num), binary.Read(r, binary.BigEndian, &num)
	default:
		return 0, errors.New("wrong size")
	}
}

// BER writing functions

func BerWriteBoolean(b bool, w io.Writer) {
	bb := uint8(0)
	if b {
		bb = uint8(0xff)
	}
	w.Write([]byte{0x01}) // tag boolean
	BerWriteLength(1, w)
	w.Write([]byte{bb})
}

func BerWriteInteger(n int, w io.Writer) {
	w.Write([]byte{0x02}) // tag integer
	if n <= 0xff {
		BerWriteLength(1, w)
		w.Write([]byte{uint8(n)})
	} else if n <= 0xffff {
		BerWriteLength(2, w)
		binary.Write(w, binary.BigEndian, uint16(n))
	} else {
		BerWriteLength(4, w)
		binary.Write(w, binary.BigEndian, uint32(n))
	}
}

func BerWriteOctetString(str []byte, w io.Writer) {
	w.Write([]byte{0x04}) // tag octet string
	BerWriteLength(len(str), w)
	w.Write(str)
}

func BerWriteSequence(data []byte, w io.Writer) {
	w.Write([]byte{0x30}) // tag sequence
	BerWriteLength(len(data), w)
	w.Write(data)
}

func BerWriteApplicationTag(tag uint8, size int, w io.Writer) {
	if tag > 30 {
		w.Write([]byte{
			0x7f, // leading octet for tags with number greater than or equal to 31
			tag,
		})
		BerWriteLength(size, w)
	} else {
		w.Write([]byte{tag})
		BerWriteLength(size, w)
	}
}

func BerWriteLength(size int, w io.Writer) {
	if size > 0xff {
		// Long form: 0x82 means 2 bytes follow
		w.Write([]byte{0x82})
		binary.Write(w, binary.BigEndian, uint16(size))
	} else if size > 0x7f {
		// Long form: 0x81 means 1 byte follows
		w.Write([]byte{0x81, uint8(size)})
	} else {
		// Short form: size directly in length octet
		w.Write([]byte{uint8(size)})
	}
}

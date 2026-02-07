package encoding

import (
	"encoding/binary"
	"errors"
	"io"
)

// PER reading functions

func PerReadChoice(r io.Reader) (uint8, error) {
	var choice uint8

	return choice, binary.Read(r, binary.BigEndian, &choice)
}

func PerReadLength(r io.Reader) (int, error) {
	var (
		octet uint8
		size  int
		err   error
	)

	if err = binary.Read(r, binary.BigEndian, &octet); err != nil {
		return 0, err
	}

	if octet&0x80 != 0x80 {
		return int(octet), nil
	}

	octet &^= 0x80
	size = int(octet) << 8

	if err = binary.Read(r, binary.BigEndian, &octet); err != nil {
		return 0, err
	}

	size += int(octet)

	return size, nil
}

func PerReadObjectIdentifier(oid [6]byte, r io.Reader) (bool, error) {
	size, err := PerReadLength(r)
	if err != nil {
		return false, err
	}

	if size != 5 {
		return false, nil
	}

	var t12 uint8
	err = binary.Read(r, binary.BigEndian, &t12)
	if err != nil {
		return false, err
	}

	aOid := make([]byte, 6)
	aOid[0] = t12 >> 4
	aOid[1] = t12 & 0x0f

	for i := 2; i <= 5; i++ {
		err = binary.Read(r, binary.BigEndian, &aOid[i])
		if err != nil {
			return false, err
		}
	}

	for i := 0; i < len(aOid); i++ {
		if oid[i] != aOid[i] {
			return false, nil
		}
	}

	return true, nil
}

func PerReadInteger16(minimum uint16, r io.Reader) (uint16, error) {
	var num uint16

	if err := binary.Read(r, binary.BigEndian, &num); err != nil {
		return 0, err
	}

	num += minimum

	return num, nil
}

func PerReadInteger(r io.Reader) (int, error) {
	size, err := PerReadLength(r)
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
	case 4:
		var num uint32

		return int(num), binary.Read(r, binary.BigEndian, &num)
	default:
		return 0, errors.New("bad integer length")
	}
}

func PerReadEnumerates(r io.Reader) (uint8, error) {
	var num uint8

	return num, binary.Read(r, binary.BigEndian, &num)
}

func PerReadNumberOfSet(r io.Reader) (uint8, error) {
	var num uint8

	return num, binary.Read(r, binary.BigEndian, &num)
}

func PerReadOctetStream(octetStream []byte, minValue int, r io.Reader) (bool, error) {
	length, err := PerReadLength(r)
	if err != nil {
		return false, err
	}

	size := length + minValue
	if size != len(octetStream) {
		return false, nil
	}

	var c uint8

	for i := 0; i < size; i++ {
		if err = binary.Read(r, binary.BigEndian, &c); err != nil {
			return false, err
		}

		if octetStream[i] != c {
			return false, nil
		}
	}

	return true, nil
}

// PER writing functions

func PerWriteChoice(choice uint8, w io.Writer) {
	_, _ = w.Write([]byte{
		choice,
	})
}

func PerWriteObjectIdentifier(oid [6]byte, w io.Writer) {
	PerWriteLength(5, w)

	_, _ = w.Write([]byte{
		(oid[0] << 4) | (oid[1] & 0x0f),
		oid[2],
		oid[3],
		oid[4],
		oid[5],
	})
}

func PerWriteLength(value uint16, w io.Writer) {
	if value > 0x7f {
		_ = binary.Write(w, binary.BigEndian, value|0x8000)
		return
	}

	_, _ = w.Write([]byte{uint8(value)})
}

func PerWriteSelection(selection uint8, w io.Writer) {
	_, _ = w.Write([]byte{
		selection,
	})
}

func PerWriteNumericString(nStr string, minValue int, w io.Writer) {
	length := len(nStr)
	mLength := minValue

	if length-minValue >= 0 {
		mLength = length - minValue
	}

	result := make([]byte, 0, mLength)

	for i := 0; i < length; i += 2 {
		c1 := nStr[i]
		c2 := byte(0x30)

		if i+1 < length {
			c2 = nStr[i+1]
		}

		c1 = (c1 - 0x30) % 10
		c2 = (c2 - 0x30) % 10

		result = append(result, (c1<<4)|c2)
	}

	PerWriteLength(uint16(mLength), w) // #nosec G115
	_, _ = w.Write(result)
}

func PerWritePadding(length int, w io.Writer) {
	_, _ = w.Write(make([]byte, length))
}

func PerWriteNumberOfSet(numberOfSet uint8, w io.Writer) {
	_, _ = w.Write([]byte{numberOfSet})
}

func PerWriteOctetStream(oStr string, minValue int, w io.Writer) {
	length := len(oStr)
	mLength := minValue

	if length-minValue >= 0 {
		mLength = length - minValue
	}

	result := make([]byte, 0, mLength)
	for i := 0; i < length; i++ {
		result = append(result, oStr[i])
	}

	PerWriteLength(uint16(mLength), w) // #nosec G115
	_, _ = w.Write(result)
}

func PerWriteInteger(value int, w io.Writer) {
	if value <= 0xff {
		PerWriteLength(1, w)
		_, _ = w.Write([]byte{uint8(value)}) // #nosec G115

		return
	}

	if value <= 0xffff {
		PerWriteLength(2, w)
		_ = binary.Write(w, binary.BigEndian, uint16(value)) // #nosec G115

		return
	}

	PerWriteLength(4, w)
	_ = binary.Write(w, binary.BigEndian, uint32(value)) // #nosec G115
}

func PerWriteInteger16(value, minimum uint16, w io.Writer) {
	value -= minimum

	_ = binary.Write(w, binary.BigEndian, value)
}

// BerWriteInteger16 writes a 16-bit integer in BER format
func BerWriteInteger16(n uint16, w io.Writer) {
	_, _ = w.Write([]byte{0x02}) // tag integer
	BerWriteLength(2, w)
	_ = binary.Write(w, binary.BigEndian, n)
}

// BerReadInteger16 reads a 16-bit integer in BER format
func BerReadInteger16(r io.Reader) (uint16, error) {
	universalTag, err := BerReadUniversalTag(TagInteger, false, r)
	if err != nil {
		return 0, err
	}

	if !universalTag {
		return 0, errors.New("bad integer tag")
	}

	size, err := BerReadLength(r)
	if err != nil {
		return 0, err
	}

	if size != 2 {
		return 0, errors.New("expected 2-byte integer")
	}

	var num uint16
	return num, binary.Read(r, binary.BigEndian, &num)
}

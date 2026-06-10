package codec

import (
	"encoding/binary"
	"errors"
)

const (
	H264MaxAccessUnitLen  = 1024 * 1024
	H264MaxAccessUnits    = 4
	AVCMaxRegionRects     = 256
	AVCMaxBitmapStreamLen = 1024 * 1024
)

var ErrAVCInvalidInput = errors.New("avc: invalid input")

// H264AccessUnit is one encoded H.264/AVC access unit prepared by a caller.
type H264AccessUnit struct {
	PresentationTimeUS int64
	KeyFrame           bool
	CodecConfig        bool
	Data               []byte
}

// AVC444Input describes base and auxiliary H.264 access units for AVC444/AVC444v2 bitmap streams.
type AVC444Input struct {
	Width, Height int
	BaseLayer     H264AccessUnit
	AuxLayer      H264AccessUnit
	RegionRects   []ProgressiveRect
	UseV2         bool
}

// ValidateH264AccessUnit validates bounded H.264 access unit metadata and bytes.
func ValidateH264AccessUnit(unit H264AccessUnit) error {
	if len(unit.Data) == 0 || len(unit.Data) > H264MaxAccessUnitLen || unit.PresentationTimeUS < 0 {
		return ErrAVCInvalidInput
	}
	return nil
}

// NormalizeH264AnnexB returns an Annex-B H.264 access unit, converting 4-byte length-prefixed NAL units when needed.
func NormalizeH264AnnexB(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, ErrAVCInvalidInput
	}
	if H264HasStartCode(data) {
		return data, nil
	}
	out, ok := h264LengthPrefixedToAnnexB(data)
	if !ok {
		return nil, ErrAVCInvalidInput
	}
	return out, nil
}

// H264HasStartCode reports whether data begins with a 3- or 4-byte Annex-B start code.
func H264HasStartCode(data []byte) bool {
	return len(data) >= 4 && data[0] == 0 && data[1] == 0 && (data[2] == 1 || data[2] == 0 && data[3] == 1)
}

// H264AnnexBContainsNALType reports whether Annex-B data contains a NAL unit of nalType.
func H264AnnexBContainsNALType(data []byte, nalType byte) bool {
	for offset := 0; offset < len(data); {
		start, prefixLen, ok := h264FindStartCode(data, offset)
		if !ok {
			return false
		}
		nalStart := start + prefixLen
		if nalStart < len(data) && data[nalStart]&0x1f == nalType {
			return true
		}
		offset = nalStart + 1
	}
	return false
}

// ValidateH264AccessUnitBatch validates a bounded set of access units.
func ValidateH264AccessUnitBatch(units []H264AccessUnit) error {
	if len(units) == 0 || len(units) > H264MaxAccessUnits {
		return ErrAVCInvalidInput
	}
	for _, unit := range units {
		if err := ValidateH264AccessUnit(unit); err != nil {
			return err
		}
	}
	return nil
}

// BuildAVC420BitmapStream builds an MS-RDPEGFX AVC420 bitmap stream for one frame region.
func BuildAVC420BitmapStream(accessUnit []byte, width, height uint16) ([]byte, error) {
	if width == 0 || height == 0 || len(accessUnit) == 0 || len(accessUnit) > H264MaxAccessUnitLen {
		return nil, ErrAVCInvalidInput
	}
	payload := make([]byte, 0, 4+8+2+len(accessUnit))
	payload = binary.LittleEndian.AppendUint32(payload, 1)
	payload = binary.LittleEndian.AppendUint16(payload, 0)
	payload = binary.LittleEndian.AppendUint16(payload, 0)
	payload = binary.LittleEndian.AppendUint16(payload, width)
	payload = binary.LittleEndian.AppendUint16(payload, height)
	payload = append(payload, 0, 0)
	payload = append(payload, accessUnit...)
	return payload, nil
}

// BuildAVC420WireToSurface builds an AVC420 WireToSurface_1 PDU.
func BuildAVC420WireToSurface(surfaceID uint16, pixelFormat uint8, dest Rect, accessUnit []byte, width, height uint16) ([]byte, error) {
	payload, err := BuildAVC420BitmapStream(accessUnit, width, height)
	if err != nil {
		return nil, err
	}
	return BuildRDPGFXWireToSurface1(surfaceID, RDPGFXCodecAVC420, pixelFormat, dest, payload)
}

// ValidateAVC444Input validates dimensions, access units, keyframe alignment, and regions.
func ValidateAVC444Input(in AVC444Input) error {
	if in.Width <= 0 || in.Height <= 0 || in.Width > 8192 || in.Height > 8192 {
		return ErrAVCInvalidInput
	}
	if err := ValidateH264AccessUnit(in.BaseLayer); err != nil || !in.BaseLayer.KeyFrame {
		return ErrAVCInvalidInput
	}
	if err := ValidateH264AccessUnit(in.AuxLayer); err != nil {
		return ErrAVCInvalidInput
	}
	if len(in.RegionRects) == 0 || len(in.RegionRects) > AVCMaxRegionRects {
		return ErrAVCInvalidInput
	}
	for _, r := range in.RegionRects {
		if r.Right <= r.Left || r.Bottom <= r.Top || int(r.Right) > in.Width || int(r.Bottom) > in.Height {
			return ErrAVCInvalidInput
		}
	}
	return nil
}

// BuildAVC444BitmapStream builds an AVC444 or AVC444v2 bitmap stream.
func BuildAVC444BitmapStream(in AVC444Input) ([]byte, error) {
	if err := ValidateAVC444Input(in); err != nil {
		return nil, err
	}
	total := avc444BitmapStreamLen(in)
	if total <= 0 || total > AVCMaxBitmapStreamLen {
		return nil, ErrAVCInvalidInput
	}
	out := make([]byte, total)
	if !writeAVC444BitmapStream(out, in) {
		return nil, ErrAVCInvalidInput
	}
	return out, nil
}

// BuildAVC444WireToSurface builds an AVC444/AVC444v2 WireToSurface_1 PDU.
func BuildAVC444WireToSurface(surfaceID uint16, pixelFormat uint8, dest Rect, in AVC444Input) ([]byte, error) {
	payload, err := BuildAVC444BitmapStream(in)
	if err != nil {
		return nil, err
	}
	codecID := RDPGFXCodecAVC444
	if in.UseV2 {
		codecID = RDPGFXCodecAVC444v2
	}
	return BuildRDPGFXWireToSurface1(surfaceID, codecID, pixelFormat, dest, payload)
}

func h264LengthPrefixedToAnnexB(data []byte) ([]byte, bool) {
	outLen := 0
	for offset := 0; offset < len(data); {
		if len(data)-offset < 4 {
			return nil, false
		}
		nalLen := int(binary.BigEndian.Uint32(data[offset : offset+4]))
		offset += 4
		if nalLen <= 0 || nalLen > len(data)-offset || outLen > H264MaxAccessUnitLen-4-nalLen {
			return nil, false
		}
		outLen += 4 + nalLen
		offset += nalLen
	}
	if outLen == 0 {
		return nil, false
	}
	out := make([]byte, outLen)
	writeOff := 0
	for offset := 0; offset < len(data); {
		nalLen := int(binary.BigEndian.Uint32(data[offset : offset+4]))
		offset += 4
		out[writeOff+0], out[writeOff+1], out[writeOff+2], out[writeOff+3] = 0, 0, 0, 1
		writeOff += 4
		writeOff += copy(out[writeOff:], data[offset:offset+nalLen])
		offset += nalLen
	}
	return out, true
}

func h264FindStartCode(data []byte, offset int) (start int, prefixLen int, ok bool) {
	for i := offset; i+3 <= len(data); i++ {
		if data[i] == 0 && data[i+1] == 0 {
			if data[i+2] == 1 {
				return i, 3, true
			}
			if i+4 <= len(data) && data[i+2] == 0 && data[i+3] == 1 {
				return i, 4, true
			}
		}
	}
	return 0, 0, false
}

func avc444BitmapStreamLen(in AVC444Input) int {
	return 4 + len(in.RegionRects)*8 + 4 + 4 + 2 + 2 + len(in.BaseLayer.Data) + len(in.AuxLayer.Data)
}

func writeAVC444BitmapStream(out []byte, in AVC444Input) bool {
	total := avc444BitmapStreamLen(in)
	if total <= 0 || total > len(out) {
		return false
	}
	off := 0
	binary.LittleEndian.PutUint32(out[off:off+4], uint32(len(in.RegionRects)))
	off += 4
	for _, r := range in.RegionRects {
		binary.LittleEndian.PutUint16(out[off:off+2], r.Left)
		binary.LittleEndian.PutUint16(out[off+2:off+4], r.Top)
		binary.LittleEndian.PutUint16(out[off+4:off+6], r.Right)
		binary.LittleEndian.PutUint16(out[off+6:off+8], r.Bottom)
		off += 8
	}
	binary.LittleEndian.PutUint32(out[off:off+4], uint32(len(in.BaseLayer.Data)))
	off += 4
	binary.LittleEndian.PutUint32(out[off:off+4], uint32(len(in.AuxLayer.Data)))
	off += 4
	var flags uint16
	if in.UseV2 {
		flags = 1
	}
	binary.LittleEndian.PutUint16(out[off:off+2], flags)
	off += 2
	binary.LittleEndian.PutUint16(out[off:off+2], 0)
	off += 2
	off += copy(out[off:], in.BaseLayer.Data)
	copy(out[off:], in.AuxLayer.Data)
	return true
}

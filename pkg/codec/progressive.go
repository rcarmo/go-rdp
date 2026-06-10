package codec

import (
	"encoding/binary"
	"errors"
)

const (
	ProgressiveMaxRects      = 256
	ProgressiveMaxPayloadLen = 1024 * 1024
)

var ErrProgressivePayloadInvalid = errors.New("progressive: invalid payload")

// ProgressiveRect is one CAProgressive/CAProgressiveV2 region rectangle.
type ProgressiveRect struct {
	Left   uint16
	Top    uint16
	Right  uint16
	Bottom uint16
}

// ProgressivePayload is a bounded, reusable representation of the Progressive
// and ProgressiveV2 payload subset used by this package. EncodedData is copied
// by MarshalProgressivePayload and ParseProgressivePayload; ParseProgressivePayloadAlias
// returns an alias into the input buffer for zero-copy inspection.
type ProgressivePayload struct {
	Width       int
	Height      int
	LayerCount  uint8
	Quant       uint8
	RegionRects []ProgressiveRect
	EncodedData []byte
}

// ValidateProgressivePayload validates dimensions, layers, regions, and encoded-data bounds.
func ValidateProgressivePayload(in ProgressivePayload) error {
	if in.Width <= 0 || in.Height <= 0 || in.Width > 8192 || in.Height > 8192 {
		return ErrProgressivePayloadInvalid
	}
	if in.LayerCount == 0 || in.LayerCount > 8 {
		return ErrProgressivePayloadInvalid
	}
	if len(in.RegionRects) == 0 || len(in.RegionRects) > ProgressiveMaxRects {
		return ErrProgressivePayloadInvalid
	}
	for _, r := range in.RegionRects {
		if r.Right <= r.Left || r.Bottom <= r.Top {
			return ErrProgressivePayloadInvalid
		}
		if int(r.Right) > in.Width || int(r.Bottom) > in.Height {
			return ErrProgressivePayloadInvalid
		}
	}
	if len(in.EncodedData) == 0 || len(in.EncodedData) > ProgressiveMaxPayloadLen {
		return ErrProgressivePayloadInvalid
	}
	return nil
}

// MarshalProgressivePayload marshals a CAProgressive/CAProgressiveV2 payload.
func MarshalProgressivePayload(in ProgressivePayload) ([]byte, error) {
	if err := ValidateProgressivePayload(in); err != nil {
		return nil, err
	}
	const headerLen = 2 + 2 + 1 + 1 + 2
	rectsLen := len(in.RegionRects) * 8
	total := headerLen + rectsLen + 4 + len(in.EncodedData)
	if total <= 0 || total > ProgressiveMaxPayloadLen {
		return nil, ErrProgressivePayloadInvalid
	}
	out := make([]byte, total)
	off := 0
	binary.LittleEndian.PutUint16(out[off:off+2], uint16(in.Width))
	off += 2
	binary.LittleEndian.PutUint16(out[off:off+2], uint16(in.Height))
	off += 2
	out[off] = in.LayerCount
	off++
	out[off] = in.Quant
	off++
	binary.LittleEndian.PutUint16(out[off:off+2], uint16(len(in.RegionRects)))
	off += 2
	for _, r := range in.RegionRects {
		binary.LittleEndian.PutUint16(out[off:off+2], r.Left)
		binary.LittleEndian.PutUint16(out[off+2:off+4], r.Top)
		binary.LittleEndian.PutUint16(out[off+4:off+6], r.Right)
		binary.LittleEndian.PutUint16(out[off+6:off+8], r.Bottom)
		off += 8
	}
	binary.LittleEndian.PutUint32(out[off:off+4], uint32(len(in.EncodedData)))
	off += 4
	copy(out[off:], in.EncodedData)
	return out, nil
}

// ParseProgressivePayload parses and copies a bounded Progressive payload.
func ParseProgressivePayload(data []byte) (ProgressivePayload, error) {
	out, err := ParseProgressivePayloadAlias(data)
	if err != nil {
		return ProgressivePayload{}, err
	}
	out.RegionRects = append([]ProgressiveRect(nil), out.RegionRects...)
	out.EncodedData = append([]byte(nil), out.EncodedData...)
	return out, nil
}

// ParseProgressivePayloadAlias parses a bounded Progressive payload and returns
// EncodedData as an alias into data. RegionRects are always newly allocated.
func ParseProgressivePayloadAlias(data []byte) (ProgressivePayload, error) {
	var out ProgressivePayload
	if len(data) < 12 || len(data) > ProgressiveMaxPayloadLen {
		return out, ErrProgressivePayloadInvalid
	}
	off := 0
	out.Width = int(binary.LittleEndian.Uint16(data[off : off+2]))
	off += 2
	out.Height = int(binary.LittleEndian.Uint16(data[off : off+2]))
	off += 2
	out.LayerCount = data[off]
	off++
	out.Quant = data[off]
	off++
	rectCount := int(binary.LittleEndian.Uint16(data[off : off+2]))
	off += 2
	if rectCount <= 0 || rectCount > ProgressiveMaxRects || off+rectCount*8+4 > len(data) {
		return out, ErrProgressivePayloadInvalid
	}
	out.RegionRects = make([]ProgressiveRect, rectCount)
	for i := 0; i < rectCount; i++ {
		out.RegionRects[i] = ProgressiveRect{
			Left:   binary.LittleEndian.Uint16(data[off : off+2]),
			Top:    binary.LittleEndian.Uint16(data[off+2 : off+4]),
			Right:  binary.LittleEndian.Uint16(data[off+4 : off+6]),
			Bottom: binary.LittleEndian.Uint16(data[off+6 : off+8]),
		}
		off += 8
	}
	dataLen := int(binary.LittleEndian.Uint32(data[off : off+4]))
	off += 4
	if dataLen <= 0 || dataLen > len(data)-off {
		return out, ErrProgressivePayloadInvalid
	}
	out.EncodedData = data[off : off+dataLen]
	if err := ValidateProgressivePayload(out); err != nil {
		return ProgressivePayload{}, err
	}
	return out, nil
}

// BuildProgressiveWireToSurface marshals a Progressive payload into a WireToSurface_1 PDU.
func BuildProgressiveWireToSurface(surfaceID uint16, pixelFormat uint8, dest Rect, in ProgressivePayload, v2 bool) ([]byte, error) {
	payload, err := MarshalProgressivePayload(in)
	if err != nil {
		return nil, err
	}
	codecID := RDPGFXCodecCAProgressive
	if v2 {
		codecID = RDPGFXCodecCAProgressiveV2
	}
	return BuildRDPGFXWireToSurface1(surfaceID, codecID, pixelFormat, dest, payload)
}

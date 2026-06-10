package codec

import "errors"

// RFX tile constants from MS-RDPRFX.
const (
	RFXTileSize   = rfxTileSize
	RFXTilePixels = rfxTilePixels
)

var (
	ErrRFXInvalidInput      = errors.New("rfx: invalid input")
	ErrRFXInvalidQuant      = errors.New("rfx: invalid quantization payload")
	ErrRFXEncodeFailed      = errors.New("rfx: encode failed")
	ErrRFXComponentTooShort = errors.New("rfx: component buffer must contain 4096 coefficients")
)

// RFXQuant holds RemoteFX quantization values in the encoder's MS-RDPRFX order:
// LL3, HL3, LH3, HH3, HL2, LH2, HH2, HL1, LH1, HH1.
type RFXQuant struct {
	Values [10]uint8
}

// DefaultRFXQuant returns the conservative default quantization table used by
// the reusable RemoteFX encoder.
func DefaultRFXQuant() RFXQuant { return RFXQuant{Values: rfxDefaultQuant.Values} }

// DefaultRFXQuantBytes returns the packed 5-byte representation carried in a
// RemoteFX tileset block.
func DefaultRFXQuantBytes() []byte {
	out := make([]byte, len(defaultRFXQuantBytes))
	copy(out, defaultRFXQuantBytes[:])
	return out
}

// ParseRFXQuant parses the packed 5-byte RemoteFX quantization table used in
// TS_RFX_TILESET.
func ParseRFXQuant(raw []byte) (RFXQuant, error) {
	q, ok := parseRFXQuant(raw)
	if !ok {
		return RFXQuant{}, ErrRFXInvalidQuant
	}
	return RFXQuant{Values: q.Values}, nil
}

// RFXRGBToYCoCg converts one RGB pixel to the reversible YCoCg representation
// used by the RemoteFX encoder.
func RFXRGBToYCoCg(r, g, b byte) (y, co, cg int16) {
	ri, gi, bi := int16(r), int16(g), int16(b)
	co = ri - bi
	t := bi + (co >> 1)
	cg = gi - t
	y = t + (cg >> 1)
	return y, co, cg
}

// ForwardRFXDWT53 applies the three-level 5/3 forward DWT in-place to one 64x64
// component. The coefficient layout remains spatial until SerializeRFXComponentForRLGR.
func ForwardRFXDWT53(coeff []int16) error {
	if len(coeff) < rfxTilePixels {
		return ErrRFXComponentTooShort
	}
	var tmp [rfxTilePixels]int16
	copy(tmp[:], coeff[:rfxTilePixels])
	forwardRFXDWT53(&tmp)
	copy(coeff[:rfxTilePixels], tmp[:])
	return nil
}

// QuantizeRFXComponent applies subband-scoped quantization to a DWT component.
func QuantizeRFXComponent(coeff []int16, quant RFXQuant) error {
	if len(coeff) < rfxTilePixels {
		return ErrRFXComponentTooShort
	}
	var tmp [rfxTilePixels]int16
	copy(tmp[:], coeff[:rfxTilePixels])
	quantizeRFXComponent(&tmp, rfxQuant{Values: quant.Values})
	copy(coeff[:rfxTilePixels], tmp[:])
	return nil
}

// SerializeRFXComponentForRLGR packs DWT subbands into the order consumed by
// RemoteFX RLGR coding and differentially encodes LL3.
func SerializeRFXComponentForRLGR(coeff []int16) ([]int16, error) {
	if len(coeff) < rfxTilePixels {
		return nil, ErrRFXComponentTooShort
	}
	var tmp [rfxTilePixels]int16
	copy(tmp[:], coeff[:rfxTilePixels])
	packed := serializeRFXComponentForRLGR(&tmp)
	out := make([]int16, rfxTilePixels)
	copy(out, packed[:])
	return out, nil
}

// EncodeRFXRLGR encodes a 4096-coefficient component using RLGR1 or RLGR3.
func EncodeRFXRLGR(coeff []int16, mode int) ([]byte, error) {
	var rmode rfxRLGRMode
	switch mode {
	case 1:
		rmode = rfxRLGR1
	case 3:
		rmode = rfxRLGR3
	default:
		return nil, ErrRFXInvalidInput
	}
	out, ok := encodeRFXRLGR(coeff, rmode)
	if !ok {
		return nil, ErrRFXEncodeFailed
	}
	return out, nil
}

// EncodeRFXTileComponents returns encoded Y, Co and Cg component streams for one
// 64x64 tile at a tile-aligned pixel origin.
func EncodeRFXTileComponents(input BitmapInput, tileX, tileY int, quantRaw []byte) ([]byte, []byte, []byte, error) {
	y, co, cg, ok := buildRFXEncodedTileParts(input, tileX, tileY, quantRaw)
	if !ok {
		return nil, nil, nil, ErrRFXEncodeFailed
	}
	return y, co, cg, nil
}

// BuildRFXTileBlock builds a TS_RFX_TILE block from encoded component streams.
func BuildRFXTileBlock(tileX, tileY uint16, yData, coData, cgData []byte) ([]byte, error) {
	out, ok := buildRFXTileBlock(tileX, tileY, yData, coData, cgData)
	if !ok {
		return nil, ErrRFXInvalidInput
	}
	return out, nil
}

// EncodeRFXSingleTileFrame builds a complete single-tile RemoteFX frame message.
func EncodeRFXSingleTileFrame(input BitmapInput, width, height int, frameID uint32, tileX, tileY int, quantRaw []byte) ([]byte, error) {
	out, ok := buildRFXMessageSingleTile(input, width, height, frameID, tileX, tileY, quantRaw)
	if !ok {
		return nil, ErrRFXEncodeFailed
	}
	return out, nil
}

// BuildRFXSetSurfaceBits wraps a RemoteFX message in a Set Surface Bits command.
// codecID must be the negotiated RemoteFX or RemoteFX Image bitmap codec ID.
func BuildRFXSetSurfaceBits(dest Rect, codecID uint8, input BitmapInput, frameID uint32, quantRaw []byte) ([]byte, error) {
	payload, err := EncodeRFXSingleTileFrame(input, input.Width, input.Height, frameID, 0, 0, quantRaw)
	if err != nil {
		return nil, err
	}
	return BuildSetSurfaceBits(dest, 32, codecID, uint16(input.Width), uint16(input.Height), payload) // #nosec G115 RFX frame builder validates 16-bit dimensions.
}

package codec

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestBuildRFXYCoCgTileRGBAAndBGRA(t *testing.T) {
	rgba := BitmapInput{
		Width:  rfxTileSize,
		Height: rfxTileSize,
		Stride: rfxTileSize * 4,
		Format: PixelFormatRGBA,
		Pixels: make([]byte, rfxTileSize*rfxTileSize*4),
	}
	bgra := rgba
	bgra.Format = PixelFormatBGRA
	bgra.Pixels = make([]byte, len(rgba.Pixels))

	// Pixel(0,0): R=100,G=50,B=10.
	rgba.Pixels[0] = 100
	rgba.Pixels[1] = 50
	rgba.Pixels[2] = 10
	rgba.Pixels[3] = 0xff

	bgra.Pixels[0] = 10
	bgra.Pixels[1] = 50
	bgra.Pixels[2] = 100
	bgra.Pixels[3] = 0xff

	rgbaTile, ok := buildRFXYCoCgTile(rgba, 0, 0)
	if !ok {
		t.Fatalf("buildRFXYCoCgTile RGBA = false")
	}
	bgraTile, ok := buildRFXYCoCgTile(bgra, 0, 0)
	if !ok {
		t.Fatalf("buildRFXYCoCgTile BGRA = false")
	}

	// co = r-b = 90
	// t = b + (co>>1) = 10 + 45 = 55
	// cg = g-t = -5
	// y = t + (cg>>1) = 55 + (-3) = 52
	if rgbaTile.Co[0] != 90 || rgbaTile.Cg[0] != -5 || rgbaTile.Y[0] != 52 {
		t.Fatalf("RGBA tile[0] = Y/Co/Cg %d/%d/%d, want 52/90/-5", rgbaTile.Y[0], rgbaTile.Co[0], rgbaTile.Cg[0])
	}
	if bgraTile.Co[0] != rgbaTile.Co[0] || bgraTile.Cg[0] != rgbaTile.Cg[0] || bgraTile.Y[0] != rgbaTile.Y[0] {
		t.Fatalf("BGRA tile[0] mismatch Y/Co/Cg got %d/%d/%d want %d/%d/%d", bgraTile.Y[0], bgraTile.Co[0], bgraTile.Cg[0], rgbaTile.Y[0], rgbaTile.Co[0], rgbaTile.Cg[0])
	}
}

func TestParseRFXQuant(t *testing.T) {
	q, ok := parseRFXQuant([]byte{0x12, 0x34, 0x56, 0x78, 0x9a})
	if !ok {
		t.Fatalf("parseRFXQuant = false")
	}
	want := [10]uint8{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	if q.Values != want {
		t.Fatalf("parseRFXQuant values = %v, want %v", q.Values, want)
	}
	if _, ok := parseRFXQuant([]byte{0x12}); ok {
		t.Fatalf("parseRFXQuant accepted short input")
	}
}

func TestForwardRFXDWT53ConstantProducesOnlyLL3Energy(t *testing.T) {
	var coeff [rfxTilePixels]int16
	for i := range coeff {
		coeff[i] = 64
	}
	forwardRFXDWT53(&coeff)

	for y := 0; y < rfxTileSize; y++ {
		for x := 0; x < rfxTileSize; x++ {
			if x < 8 && y < 8 {
				continue
			}
			if v := coeff[y*rfxTileSize+x]; v != 0 {
				t.Fatalf("highpass coeff[%d,%d]=%d, want 0", x, y, v)
			}
		}
	}
}

func TestQuantizeRFXComponentSubbandScoped(t *testing.T) {
	var coeff [rfxTilePixels]int16
	for y := 0; y < rfxTileSize; y++ {
		for x := 0; x < rfxTileSize; x++ {
			coeff[y*rfxTileSize+x] = 256
		}
	}
	quantizeRFXComponent(&coeff, rfxQuant{Values: [10]uint8{1, 2, 3, 4, 1, 1, 1, 1, 1, 1}})

	if got := coeff[0]; got != 128 { // LL3 shifted by 1
		t.Fatalf("LL3 coeff shift got %d, want 128", got)
	}
	if got := coeff[8]; got != 64 { // HL3 shifted by 2
		t.Fatalf("HL3 coeff shift got %d, want 64", got)
	}
	if got := coeff[8*rfxTileSize+0]; got != 32 { // LH3 shifted by 3
		t.Fatalf("LH3 coeff shift got %d, want 32", got)
	}
	if got := coeff[8*rfxTileSize+8]; got != 16 { // HH3 shifted by 4
		t.Fatalf("HH3 coeff shift got %d, want 16", got)
	}
	if got := coeff[63*rfxTileSize+63]; got != 128 { // HH1 shifted by 1
		t.Fatalf("HH1 coeff shift got %d, want 128", got)
	}
}

func TestSerializeRFXComponentForRLGRSubbandOrderAndLL3Differential(t *testing.T) {
	var coeff [rfxTilePixels]int16

	fillBand := func(x0, y0, size int, start int16) {
		v := start
		for y := 0; y < size; y++ {
			for x := 0; x < size; x++ {
				coeff[(y0+y)*rfxTileSize+(x0+x)] = v
				v++
			}
		}
	}

	fillBand(32, 0, 32, 1000)  // HL1
	fillBand(0, 32, 32, 2000)  // LH1
	fillBand(32, 32, 32, 3000) // HH1
	fillBand(16, 0, 16, 4000)  // HL2
	fillBand(0, 16, 16, 5000)  // LH2
	fillBand(16, 16, 16, 6000) // HH2
	fillBand(8, 0, 8, 7000)    // HL3
	fillBand(0, 8, 8, 8000)    // LH3
	fillBand(8, 8, 8, 9000)    // HH3
	fillBand(0, 0, 8, 100)     // LL3

	packed := serializeRFXComponentForRLGR(&coeff)

	if packed[rfxOffsetHL1] != 1000 || packed[rfxOffsetLH1] != 2000 || packed[rfxOffsetHH1] != 3000 {
		t.Fatalf("L1 offsets mismatch HL1/LH1/HH1 = %d/%d/%d", packed[rfxOffsetHL1], packed[rfxOffsetLH1], packed[rfxOffsetHH1])
	}
	if packed[rfxOffsetHL2] != 4000 || packed[rfxOffsetLH2] != 5000 || packed[rfxOffsetHH2] != 6000 {
		t.Fatalf("L2 offsets mismatch HL2/LH2/HH2 = %d/%d/%d", packed[rfxOffsetHL2], packed[rfxOffsetLH2], packed[rfxOffsetHH2])
	}
	if packed[rfxOffsetHL3] != 7000 || packed[rfxOffsetLH3] != 8000 || packed[rfxOffsetHH3] != 9000 {
		t.Fatalf("L3 offsets mismatch HL3/LH3/HH3 = %d/%d/%d", packed[rfxOffsetHL3], packed[rfxOffsetLH3], packed[rfxOffsetHH3])
	}
	if packed[rfxOffsetLL3] != 100 {
		t.Fatalf("LL3 first coefficient = %d, want 100", packed[rfxOffsetLL3])
	}
	if packed[rfxOffsetLL3+1] != 1 {
		t.Fatalf("LL3 differential second coefficient = %d, want 1", packed[rfxOffsetLL3+1])
	}
}

func TestEncodeRFXRLGRBoundedAndDeterministic(t *testing.T) {
	coeff := make([]int16, rfxTilePixels)
	for i := 0; i < len(coeff); i++ {
		if i%97 == 0 {
			coeff[i] = int16((i % 15) - 7)
		}
	}
	y1, ok := encodeRFXRLGR(coeff, rfxRLGR1)
	if !ok || len(y1) == 0 {
		t.Fatalf("encodeRFXRLGR RLGR1 failed len=%d ok=%t", len(y1), ok)
	}
	y2, ok := encodeRFXRLGR(coeff, rfxRLGR1)
	if !ok || string(y1) != string(y2) {
		t.Fatalf("encodeRFXRLGR RLGR1 not deterministic")
	}
	c1, ok := encodeRFXRLGR(coeff, rfxRLGR3)
	if !ok || len(c1) == 0 {
		t.Fatalf("encodeRFXRLGR RLGR3 failed len=%d ok=%t", len(c1), ok)
	}
	if _, ok := encodeRFXRLGR(coeff[:16], rfxRLGR1); ok {
		t.Fatalf("encodeRFXRLGR accepted short coefficients")
	}
}

func TestBuildRFXTileBlockBounds(t *testing.T) {
	tile, ok := buildRFXTileBlock(1, 2, []byte{0xaa}, []byte{0xbb}, []byte{0xcc})
	if !ok || len(tile) != 22 {
		t.Fatalf("buildRFXTileBlock = len %d ok=%t", len(tile), ok)
	}
	if got := binary.LittleEndian.Uint16(tile[0:2]); got != rfxBlockTypeTile {
		t.Fatalf("tile block type = 0x%04x, want 0x%04x", got, rfxBlockTypeTile)
	}
	if got := binary.LittleEndian.Uint16(tile[9:11]); got != 1 {
		t.Fatalf("tile x index = %d, want 1", got)
	}
	if got := binary.LittleEndian.Uint16(tile[11:13]); got != 2 {
		t.Fatalf("tile y index = %d, want 2", got)
	}
	if _, ok := buildRFXTileBlock(0, 0, nil, []byte{1}, []byte{1}); ok {
		t.Fatalf("buildRFXTileBlock accepted empty component")
	}
	big := make([]byte, 65536)
	if _, ok := buildRFXTileBlock(0, 0, big, []byte{1}, []byte{1}); ok {
		t.Fatalf("buildRFXTileBlock accepted oversized component")
	}
}

func TestBuildRFXEncodedTileDeterministicAndBounded(t *testing.T) {
	src := BitmapInput{
		Width:  rfxTileSize,
		Height: rfxTileSize,
		Stride: rfxTileSize * 4,
		Format: PixelFormatRGBA,
		Pixels: make([]byte, rfxTileSize*rfxTileSize*4),
	}
	for y := 0; y < rfxTileSize; y++ {
		for x := 0; x < rfxTileSize; x++ {
			i := (y*rfxTileSize + x) * 4
			src.Pixels[i+0] = byte(x)
			src.Pixels[i+1] = byte(y)
			src.Pixels[i+2] = byte((x + y) & 0xff)
			src.Pixels[i+3] = 0xff
		}
	}

	quantRaw := []byte{0x66, 0x66, 0x77, 0x88, 0x89}
	tileA, ok := buildRFXEncodedTile(src, 0, 0, quantRaw)
	if !ok || len(tileA) <= 19 {
		t.Fatalf("buildRFXEncodedTile first encode len=%d ok=%t", len(tileA), ok)
	}
	tileB, ok := buildRFXEncodedTile(src, 0, 0, quantRaw)
	if !ok || !bytes.Equal(tileA, tileB) {
		t.Fatalf("buildRFXEncodedTile not deterministic")
	}
	if got := binary.LittleEndian.Uint16(tileA[0:2]); got != rfxBlockTypeTile {
		t.Fatalf("tile block type = 0x%04x, want 0x%04x", got, rfxBlockTypeTile)
	}
	if got := binary.LittleEndian.Uint16(tileA[9:11]); got != 0 {
		t.Fatalf("tile x index = %d, want 0", got)
	}
	if got := binary.LittleEndian.Uint16(tileA[11:13]); got != 0 {
		t.Fatalf("tile y index = %d, want 0", got)
	}

	if _, ok := buildRFXEncodedTile(src, 1, 0, quantRaw); ok {
		t.Fatalf("buildRFXEncodedTile accepted non-tile-aligned tileX")
	}
	if _, ok := buildRFXEncodedTile(src, 0, 1, quantRaw); ok {
		t.Fatalf("buildRFXEncodedTile accepted non-tile-aligned tileY")
	}
	if _, ok := buildRFXEncodedTile(src, 0, 0, []byte{0x66}); ok {
		t.Fatalf("buildRFXEncodedTile accepted malformed quant payload")
	}
}

func TestBuildRFXMessageSingleTileHasExpectedBlockSequence(t *testing.T) {
	src := BitmapInput{
		Width:  rfxTileSize,
		Height: rfxTileSize,
		Stride: rfxTileSize * 4,
		Format: PixelFormatRGBA,
		Pixels: make([]byte, rfxTileSize*rfxTileSize*4),
	}
	for i := 0; i < len(src.Pixels); i += 4 {
		src.Pixels[i+0] = 0x20
		src.Pixels[i+1] = 0x40
		src.Pixels[i+2] = 0x60
		src.Pixels[i+3] = 0xff
	}
	msg, ok := buildRFXMessageSingleTile(src, rfxTileSize, rfxTileSize, 7, 0, 0, nil)
	if !ok || len(msg) == 0 {
		t.Fatalf("buildRFXMessageSingleTile len=%d ok=%t", len(msg), ok)
	}

	var gotTypes []uint16
	off := 0
	for off+6 <= len(msg) {
		bt := binary.LittleEndian.Uint16(msg[off : off+2])
		bl := int(binary.LittleEndian.Uint32(msg[off+2 : off+6]))
		if bl < 6 || off+bl > len(msg) {
			t.Fatalf("invalid block length=%d at off=%d", bl, off)
		}
		gotTypes = append(gotTypes, bt)
		off += bl
	}
	wantTypes := []uint16{
		rfxBlockTypeSync,
		rfxBlockTypeCodecVersions,
		rfxBlockTypeChannels,
		rfxBlockTypeContext,
		rfxBlockTypeFrameBegin,
		rfxBlockTypeRegion,
		rfxBlockTypeTileset,
		rfxBlockTypeFrameEnd,
	}
	if len(gotTypes) != len(wantTypes) {
		t.Fatalf("block count=%d want=%d types=%v", len(gotTypes), len(wantTypes), gotTypes)
	}
	for i := range wantTypes {
		if gotTypes[i] != wantTypes[i] {
			t.Fatalf("block[%d]=0x%04x want=0x%04x", i, gotTypes[i], wantTypes[i])
		}
	}
}

func TestBuildRFXMessageSingleTileRejectsInvalidDimensions(t *testing.T) {
	src := BitmapInput{
		Width:  rfxTileSize,
		Height: rfxTileSize,
		Stride: rfxTileSize * 4,
		Format: PixelFormatRGBA,
		Pixels: make([]byte, rfxTileSize*rfxTileSize*4),
	}
	if _, ok := buildRFXMessageSingleTile(src, 0, rfxTileSize, 1, 0, 0, nil); ok {
		t.Fatalf("accepted zero width")
	}
	if _, ok := buildRFXMessageSingleTile(src, rfxTileSize, 0, 1, 0, 0, nil); ok {
		t.Fatalf("accepted zero height")
	}
	if _, ok := buildRFXMessageSingleTile(src, 1<<16, rfxTileSize, 1, 0, 0, nil); ok {
		t.Fatalf("accepted oversized width")
	}
	if _, ok := buildRFXMessageSingleTile(src, rfxTileSize, 1<<16, 1, 0, 0, nil); ok {
		t.Fatalf("accepted oversized height")
	}
	if _, ok := buildRFXMessageSingleTile(src, rfxTileSize, rfxTileSize, 1, rfxTileSize, 0, nil); ok {
		t.Fatalf("accepted out-of-bounds tile origin")
	}
}

func TestBuildRFXYCoCgTileRejectsInvalidInputs(t *testing.T) {
	valid := BitmapInput{
		Width:  rfxTileSize,
		Height: rfxTileSize,
		Stride: rfxTileSize * 4,
		Format: PixelFormatRGBA,
		Pixels: make([]byte, rfxTileSize*rfxTileSize*4),
	}

	if _, ok := buildRFXYCoCgTile(valid, -1, 0); ok {
		t.Fatalf("negative tileX unexpectedly accepted")
	}
	if _, ok := buildRFXYCoCgTile(valid, 0, -1); ok {
		t.Fatalf("negative tileY unexpectedly accepted")
	}
	if _, ok := buildRFXYCoCgTile(valid, 1, 0); ok {
		t.Fatalf("tile outside width unexpectedly accepted")
	}
	if _, ok := buildRFXYCoCgTile(valid, 0, 1); ok {
		t.Fatalf("tile outside height unexpectedly accepted")
	}

	badFormat := valid
	badFormat.Format = PixelFormat(99)
	if _, ok := buildRFXYCoCgTile(badFormat, 0, 0); ok {
		t.Fatalf("unsupported format unexpectedly accepted")
	}

	badStride := valid
	badStride.Stride = 1
	if _, ok := buildRFXYCoCgTile(badStride, 0, 0); ok {
		t.Fatalf("bad stride unexpectedly accepted")
	}
}

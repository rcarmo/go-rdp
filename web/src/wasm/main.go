//go:build js && wasm

// Package main provides WebAssembly bindings for the RDP codec functions.
// This file contains only JavaScript glue code - all actual codec logic
// is in the internal/codec package.
package main

import (
	"syscall/js"

	"github.com/rcarmo/go-rdp/internal/codec"
	"github.com/rcarmo/go-rdp/internal/codec/rfx"
)

// jsDecompressRLE16 is the JS wrapper for RLEDecompress16
func jsDecompressRLE16(this js.Value, args []js.Value) interface{} {
	if len(args) < 4 {
		return false
	}

	srcArray := args[0]
	dstArray := args[1]
	_ = args[2].Int() // width (unused, rowDelta is enough)
	rowDelta := args[3].Int()

	srcLen := srcArray.Get("length").Int()
	src := make([]byte, srcLen)
	js.CopyBytesToGo(src, srcArray)

	dstLen := dstArray.Get("length").Int()
	dst := make([]byte, dstLen)

	if !codec.RLEDecompress16(src, dst, rowDelta) {
		return false
	}

	js.CopyBytesToJS(dstArray, dst)
	return true
}

// jsFlipVertical is the JS wrapper for FlipVertical
func jsFlipVertical(this js.Value, args []js.Value) interface{} {
    if len(args) < 4 {
        return false
    }

	dataArray := args[0]
	width := args[1].Int()
	height := args[2].Int()
	bytesPerPixel := args[3].Int()

	dataLen := dataArray.Get("length").Int()
	data := make([]byte, dataLen)
	js.CopyBytesToGo(data, dataArray)

    codec.FlipVertical(data, width, height, bytesPerPixel)

    js.CopyBytesToJS(dataArray, data)
    return true
}

// jsRGB565toRGBA is the JS wrapper for RGB565ToRGBA
func jsRGB565toRGBA(this js.Value, args []js.Value) interface{} {
    if len(args) < 2 {
        return false
    }

	srcArray := args[0]
	dstArray := args[1]

	srcLen := srcArray.Get("length").Int()
	src := make([]byte, srcLen)
	js.CopyBytesToGo(src, srcArray)

	dstLen := dstArray.Get("length").Int()
	dst := make([]byte, dstLen)

    codec.RGB565ToRGBA(src, dst)

    js.CopyBytesToJS(dstArray, dst)
    return true
}

// jsBGR24toRGBA is the JS wrapper for BGR24ToRGBA
func jsBGR24toRGBA(this js.Value, args []js.Value) interface{} {
    if len(args) < 2 {
        return false
    }

	srcArray := args[0]
	dstArray := args[1]

	srcLen := srcArray.Get("length").Int()
	src := make([]byte, srcLen)
	js.CopyBytesToGo(src, srcArray)

	dstLen := dstArray.Get("length").Int()
	dst := make([]byte, dstLen)

    codec.BGR24ToRGBA(src, dst)

    js.CopyBytesToJS(dstArray, dst)
    return true
}

// jsBGRA32toRGBA is the JS wrapper for BGRA32ToRGBA
func jsBGRA32toRGBA(this js.Value, args []js.Value) interface{} {
    if len(args) < 2 {
        return false
    }

	srcArray := args[0]
	dstArray := args[1]

	srcLen := srcArray.Get("length").Int()
	src := make([]byte, srcLen)
	js.CopyBytesToGo(src, srcArray)

	dstLen := dstArray.Get("length").Int()
	dst := make([]byte, dstLen)

    codec.BGRA32ToRGBA(src, dst)

    js.CopyBytesToJS(dstArray, dst)
    return true
}

// jsProcessBitmap handles decompression, flip, and color conversion in one call
func jsProcessBitmap(this js.Value, args []js.Value) interface{} {
	if len(args) < 7 {
		return false
	}

	srcArray := args[0]
	width := args[1].Int()
	height := args[2].Int()
	bpp := args[3].Int()
	isCompressed := args[4].Bool()
	dstArray := args[5]
	rowDelta := args[6].Int()

	// 8th arg: NO_BITMAP_COMPRESSION_HDR flag (RDP6/Planar for 32bpp)
	noHdr := false
	if len(args) >= 8 {
		noHdr = args[7].Bool()
	}

	srcLen := srcArray.Get("length").Int()
	src := make([]byte, srcLen)
	js.CopyBytesToGo(src, srcArray)

	rgba := codec.ProcessBitmap(src, width, height, bpp, isCompressed, rowDelta, noHdr)
	if rgba == nil {
		return false
	}

	js.CopyBytesToJS(dstArray, rgba)
	return true
}

// jsDecodeNSCodec is the JS wrapper for DecodeNSCodecToRGBA
func jsDecodeNSCodec(this js.Value, args []js.Value) interface{} {
	if len(args) < 4 {
		return false
	}

	srcArray := args[0]
	width := args[1].Int()
	height := args[2].Int()
	dstArray := args[3]

	srcLen := srcArray.Get("length").Int()
	src := make([]byte, srcLen)
	js.CopyBytesToGo(src, srcArray)

	rgba := codec.DecodeNSCodecToRGBA(src, width, height)
	if rgba == nil {
		return false
	}

	js.CopyBytesToJS(dstArray, rgba)
	return true
}

// jsSetPalette updates the 256-color palette from server data
func jsSetPalette(this js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		return false
	}

	dataArray := args[0]
	numColors := args[1].Int()

	dataLen := dataArray.Get("length").Int()
	data := make([]byte, dataLen)
	js.CopyBytesToGo(data, dataArray)

	codec.SetPalette(data, numColors)
	return true
}

// RFX buffers (reused to avoid allocations)
var (
	rfxInputBuffer  = make([]byte, 65536)          // Max compressed tile size
	rfxOutputBuffer = make([]byte, rfx.TileRGBASize) // 16384 bytes
	rfxQuantBuffer  = make([]byte, 15)             // 3 quant tables × 5 bytes

	// Coefficient buffers
	rfxYCoeff  = make([]int16, rfx.TilePixels)
	rfxCbCoeff = make([]int16, rfx.TilePixels)
	rfxCrCoeff = make([]int16, rfx.TilePixels)
)

// jsDecodeRFXTile decodes a single RemoteFX tile
// Returns: { x: pixelX, y: pixelY, width: 64, height: 64 } on success
// Returns: null on error
func jsDecodeRFXTile(this js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		return nil
	}

	srcArray := args[0]
	dstArray := args[1]

	srcLen := srcArray.Get("length").Int()
	if srcLen > len(rfxInputBuffer) {
		return nil
	}

	js.CopyBytesToGo(rfxInputBuffer[:srcLen], srcArray)

	// Parse quant values from buffer (set separately via setRFXQuant)
	quantY, _ := rfx.ParseQuantValues(rfxQuantBuffer[0:5])
	quantCb, _ := rfx.ParseQuantValues(rfxQuantBuffer[5:10])
	quantCr, _ := rfx.ParseQuantValues(rfxQuantBuffer[10:15])

	if quantY == nil {
		quantY = rfx.DefaultQuant()
	}
	if quantCb == nil {
		quantCb = rfx.DefaultQuant()
	}
	if quantCr == nil {
		quantCr = rfx.DefaultQuant()
	}

	// Decode using pre-allocated buffers
	xIdx, yIdx, err := rfx.DecodeTileWithBuffers(
		rfxInputBuffer[:srcLen],
		quantY, quantCb, quantCr,
		rfxYCoeff, rfxCbCoeff, rfxCrCoeff,
		rfxOutputBuffer,
	)

	if err != nil {
		return nil
	}

	js.CopyBytesToJS(dstArray, rfxOutputBuffer)

	// Return result as array [x, y, width, height] - more efficient than map
	return []interface{}{
		int(xIdx) * rfx.TileSize,
		int(yIdx) * rfx.TileSize,
		rfx.TileSize,
		rfx.TileSize,
	}
}

// jsSetRFXQuant sets quantization values for subsequent decodes
func jsSetRFXQuant(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return false
	}

	quantArray := args[0]
	quantLen := quantArray.Get("length").Int()

	if quantLen > len(rfxQuantBuffer) {
		quantLen = len(rfxQuantBuffer)
	}

	js.CopyBytesToGo(rfxQuantBuffer[:quantLen], quantArray)
	return true
}

func main() {
	c := make(chan struct{}, 0)

	// Register functions
	js.Global().Set("goRLE", js.ValueOf(map[string]interface{}{
		"decompressRLE16": js.FuncOf(jsDecompressRLE16),
		"flipVertical":    js.FuncOf(jsFlipVertical),
		"rgb565toRGBA":    js.FuncOf(jsRGB565toRGBA),
		"bgr24toRGBA":     js.FuncOf(jsBGR24toRGBA),
		"bgra32toRGBA":    js.FuncOf(jsBGRA32toRGBA),
		"processBitmap":   js.FuncOf(jsProcessBitmap),
		"decodeNSCodec":   js.FuncOf(jsDecodeNSCodec),
		"setPalette":      js.FuncOf(jsSetPalette),
		"decodeRFXTile":   js.FuncOf(jsDecodeRFXTile),
		"setRFXQuant":     js.FuncOf(jsSetRFXQuant),
	}))

	println("Go WASM RLE module loaded (with RFX support)")

	<-c
}

package rfx

import (
	"encoding/binary"
	"fmt"
)

// ParseRFXMessage parses a complete RFX message and returns decoded tiles.
func ParseRFXMessage(data []byte, ctx *Context) (*Frame, error) {
	if len(data) < 6 {
		return nil, ErrInvalidBlockLength
	}

	frame := &Frame{
		Tiles: make([]*Tile, 0),
	}

	offset := 0

	for offset < len(data) {
		if offset+6 > len(data) {
			break
		}

		blockType := binary.LittleEndian.Uint16(data[offset:])
		blockLen := int(binary.LittleEndian.Uint32(data[offset+2:]))

		if blockLen < 6 || offset+blockLen > len(data) {
			return nil, fmt.Errorf("%w: block at offset %d", ErrInvalidBlockLength, offset)
		}

		blockData := data[offset : offset+blockLen]

		switch blockType {
		case WBT_SYNC:
			if err := parseSyncBlock(blockData); err != nil {
				return nil, err
			}

		case WBT_CODEC_VERSIONS:
			// Contains codec version info, usually just verification

		case WBT_CHANNELS:
			// Contains channel info (usually just channel 0)

		case WBT_CONTEXT:
			if err := parseContextBlock(blockData, ctx); err != nil {
				return nil, err
			}

		case WBT_FRAME_BEGIN:
			frameIdx, err := parseFrameBegin(blockData)
			if err != nil {
				return nil, err
			}
			frame.FrameIdx = frameIdx

		case WBT_REGION:
			rects, err := parseRegionBlock(blockData)
			if err != nil {
				return nil, err
			}
			frame.Rects = rects

		case WBT_TILESET:
			tiles, err := parseTilesetBlock(blockData, ctx)
			if err != nil {
				return nil, err
			}
			frame.Tiles = append(frame.Tiles, tiles...)

		case WBT_FRAME_END:
			// Frame complete

		case WBT_EXTENSION:
			// Extension block, skip for now
		}

		offset += blockLen
	}

	return frame, nil
}

func parseSyncBlock(data []byte) error {
	if len(data) < 12 {
		return ErrInvalidBlockLength
	}
	// magic := binary.LittleEndian.Uint32(data[6:])
	// version := binary.LittleEndian.Uint16(data[10:])
	// Verify magic == 0xCACCACCA and version == 0x0100
	return nil
}

func parseContextBlock(data []byte, ctx *Context) error {
	if len(data) < 13 {
		return ErrInvalidBlockLength
	}

	offset := 6 // Skip block header

	// ctxId := data[offset]
	offset++

	// tileSize := binary.LittleEndian.Uint16(data[offset:])
	offset += 2

	ctx.Width = binary.LittleEndian.Uint16(data[offset:])
	offset += 2

	ctx.Height = binary.LittleEndian.Uint16(data[offset:])

	return nil
}

func parseFrameBegin(data []byte) (uint32, error) {
	if len(data) < 14 {
		return 0, ErrInvalidBlockLength
	}

	frameIdx := binary.LittleEndian.Uint32(data[6:])
	// numRegions := binary.LittleEndian.Uint16(data[10:])

	return frameIdx, nil
}

func parseRegionBlock(data []byte) ([]Rect, error) {
	if len(data) < 15 {
		return nil, ErrInvalidBlockLength
	}

	offset := 6 // Skip block header

	// regionFlags := data[offset]
	offset++

	numRects := binary.LittleEndian.Uint16(data[offset:])
	offset += 2

	rects := make([]Rect, numRects)

	for i := uint16(0); i < numRects && offset+8 <= len(data); i++ {
		rects[i] = Rect{
			X:      binary.LittleEndian.Uint16(data[offset:]),
			Y:      binary.LittleEndian.Uint16(data[offset+2:]),
			Width:  binary.LittleEndian.Uint16(data[offset+4:]),
			Height: binary.LittleEndian.Uint16(data[offset+6:]),
		}
		offset += 8
	}

	return rects, nil
}

func parseTilesetBlock(data []byte, ctx *Context) ([]*Tile, error) {
	if len(data) < 22 {
		return nil, ErrInvalidBlockLength
	}

	offset := 6 // Skip block header

	// subtype := binary.LittleEndian.Uint16(data[offset:])
	offset += 2

	// idx := binary.LittleEndian.Uint16(data[offset:])
	offset += 2

	// flags := binary.LittleEndian.Uint16(data[offset:])
	offset += 2

	numQuant := data[offset]
	offset++

	// tileSize := data[offset]
	offset++

	numTiles := binary.LittleEndian.Uint16(data[offset:])
	offset += 2

	// tileDataSize := binary.LittleEndian.Uint32(data[offset:])
	offset += 4

	// Parse quantization tables
	quantTables := make([]*SubbandQuant, numQuant)
	for i := uint8(0); i < numQuant && offset+5 <= len(data); i++ {
		quant, err := ParseQuantValues(data[offset:])
		if err != nil {
			return nil, err
		}
		quantTables[i] = quant
		offset += 5
	}

	// Parse tiles
	tiles := make([]*Tile, 0, numTiles)

	for i := uint16(0); i < numTiles && offset < len(data); i++ {
		if offset+6 > len(data) {
			break
		}

		tileBlockType := binary.LittleEndian.Uint16(data[offset:])
		if tileBlockType != CBT_TILE {
			break
		}

		tileBlockLen := int(binary.LittleEndian.Uint32(data[offset+2:]))
		if offset+tileBlockLen > len(data) {
			break
		}

		// Get quant indices from tile header
		quantIdxY := data[offset+6]
		quantIdxCb := data[offset+7]
		quantIdxCr := data[offset+8]

		// Get quant tables (with bounds checking)
		quantY := DefaultQuant()
		quantCb := DefaultQuant()
		quantCr := DefaultQuant()

		if int(quantIdxY) < len(quantTables) && quantTables[quantIdxY] != nil {
			quantY = quantTables[quantIdxY]
		}
		if int(quantIdxCb) < len(quantTables) && quantTables[quantIdxCb] != nil {
			quantCb = quantTables[quantIdxCb]
		}
		if int(quantIdxCr) < len(quantTables) && quantTables[quantIdxCr] != nil {
			quantCr = quantTables[quantIdxCr]
		}

		tile, err := DecodeTile(data[offset:offset+tileBlockLen], quantY, quantCb, quantCr)
		if err != nil {
			// Log error but continue with other tiles
			offset += tileBlockLen
			continue
		}

		tiles = append(tiles, tile)
		offset += tileBlockLen
	}

	return tiles, nil
}

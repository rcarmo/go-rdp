package fastpath

import (
	"encoding/binary"
	"io"
)

// Surface command types
const (
	CmdTypeSurfaceBits       uint16 = 0x0001
	CmdTypeFrameMarker       uint16 = 0x0004
	CmdTypeStreamSurfaceBits uint16 = 0x0006
)

// Frame marker action
const (
	FrameStart uint16 = 0x0000
	FrameEnd   uint16 = 0x0001
)

// SurfaceCommand represents a surface command PDU
type SurfaceCommand struct {
	CmdType uint16
	Data    []byte
}

// SetSurfaceBitsCommand represents CMDTYPE_SET_SURFACE_BITS
type SetSurfaceBitsCommand struct {
	DestLeft   uint16
	DestTop    uint16
	DestRight  uint16
	DestBottom uint16
	BPP        uint8
	Flags      uint8
	Reserved   uint8
	CodecID    uint8
	Width      uint16
	Height     uint16
	BitmapData []byte
}

// FrameMarkerCommand represents CMDTYPE_FRAME_MARKER
type FrameMarkerCommand struct {
	FrameAction uint16
	FrameID     uint32
}

// ParseSurfaceCommands parses surface commands from the data section of a fastpath update
func ParseSurfaceCommands(data []byte) ([]SurfaceCommand, error) {
	var commands []SurfaceCommand
	offset := 0

	for offset < len(data) {
		if offset+2 > len(data) {
			break
		}

		cmdType := binary.LittleEndian.Uint16(data[offset:])
		offset += 2

		var cmd SurfaceCommand
		cmd.CmdType = cmdType

		switch cmdType {
		case CmdTypeSurfaceBits, CmdTypeStreamSurfaceBits:
			// SetSurfaceBits/StreamSurfaceBits structure:
			// cmdType (2) + destLeft (2) + destTop (2) + destRight (2) + destBottom (2)
			// + bpp (1) + flags (1) + reserved (1) + codecID (1) + width (2) + height (2)
			// + bitmapDataLength (4) + bitmapData (variable)
			if offset+20 > len(data) {
				break
			}

			// Skip to bitmapDataLength (offset + 16 bytes)
			bitmapDataLengthOffset := offset + 16
			if bitmapDataLengthOffset+4 > len(data) {
				break
			}
			bitmapDataLength := binary.LittleEndian.Uint32(data[bitmapDataLengthOffset:])

			// Total command data size (excluding cmdType)
			cmdDataSize := 20 + int(bitmapDataLength)
			if offset+cmdDataSize > len(data) {
				break
			}

			cmd.Data = data[offset : offset+cmdDataSize]
			offset += cmdDataSize

		case CmdTypeFrameMarker:
			// FrameMarker structure: cmdType (2) + frameAction (2) + frameId (4)
			if offset+6 > len(data) {
				break
			}
			cmd.Data = data[offset : offset+6]
			offset += 6

		default:
			// Unknown command, skip remaining data
			cmd.Data = data[offset:]
			offset = len(data)
		}

		commands = append(commands, cmd)
	}

	return commands, nil
}

// ParseSetSurfaceBits parses a SetSurfaceBits command
func ParseSetSurfaceBits(data []byte) (*SetSurfaceBitsCommand, error) {
	if len(data) < 20 {
		return nil, io.ErrUnexpectedEOF
	}

	cmd := &SetSurfaceBitsCommand{
		DestLeft:   binary.LittleEndian.Uint16(data[0:2]),
		DestTop:    binary.LittleEndian.Uint16(data[2:4]),
		DestRight:  binary.LittleEndian.Uint16(data[4:6]),
		DestBottom: binary.LittleEndian.Uint16(data[6:8]),
		BPP:        data[8],
		Flags:      data[9],
		Reserved:   data[10],
		CodecID:    data[11],
		Width:      binary.LittleEndian.Uint16(data[12:14]),
		Height:     binary.LittleEndian.Uint16(data[14:16]),
	}

	bitmapDataLength := binary.LittleEndian.Uint32(data[16:20])
	if len(data) < 20+int(bitmapDataLength) {
		return nil, io.ErrUnexpectedEOF
	}

	cmd.BitmapData = data[20 : 20+bitmapDataLength]
	return cmd, nil
}

// ParseFrameMarker parses a FrameMarker command
func ParseFrameMarker(data []byte) (*FrameMarkerCommand, error) {
	if len(data) < 6 {
		return nil, io.ErrUnexpectedEOF
	}

	return &FrameMarkerCommand{
		FrameAction: binary.LittleEndian.Uint16(data[0:2]),
		FrameID:     binary.LittleEndian.Uint32(data[2:6]),
	}, nil
}

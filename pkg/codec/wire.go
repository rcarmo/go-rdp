package codec

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// Wire-format helpers in this file follow the public Microsoft protocol
// specifications used by RDP graphics paths:
//   - MS-RDPBCGR 2.2.7.2.10 Bitmap Codecs Capability Set
//   - MS-RDPBCGR 2.2.9.2.1.2.1 Set Surface Bits Command
//   - MS-RDPEGFX 2.2.2 capability and frame/SurfaceToWire PDUs
//
// They intentionally only marshal/parse protocol bytes. Server policy such as
// codec preference, Android capture handling, environment gates, and release
// defaults belongs in callers.

const (
	// Surface command types from MS-RDPBCGR Surface Commands.
	SurfaceCmdSetSurfaceBits    uint16 = 0x0001
	SurfaceCmdFrameMarker       uint16 = 0x0004
	SurfaceCmdStreamSurfaceBits uint16 = 0x0006

	// RDPGFX command IDs from MS-RDPEGFX.
	RDPGFXCmdWireToSurface1     uint16 = 0x0001
	RDPGFXCmdCreateSurface      uint16 = 0x0009
	RDPGFXCmdStartFrame         uint16 = 0x000b
	RDPGFXCmdEndFrame           uint16 = 0x000c
	RDPGFXCmdResetGraphics      uint16 = 0x000e
	RDPGFXCmdMapSurfaceToOutput uint16 = 0x000f
	RDPGFXCmdCapsAdvertise      uint16 = 0x0012
	RDPGFXCmdCapsConfirm        uint16 = 0x0013

	// RDPGFX pixel formats used by WireToSurface PDUs.
	RDPGFXPixelFormatXRGB8888 uint8 = 0x20
	RDPGFXPixelFormatARGB8888 uint8 = 0x21

	// RDPGFX AVC capability flags.
	RDPGFXCapsFlagAVC420Enabled uint32 = 0x00000010
	RDPGFXCapsFlagAVCDisabled   uint32 = 0x00000020

	// RDPGFX capability versions observed in MS-RDPEGFX.
	RDPGFXCapsVersion8   uint32 = 0x00080004
	RDPGFXCapsVersion81  uint32 = 0x00080105
	RDPGFXCapsVersion10  uint32 = 0x000A0002
	RDPGFXCapsVersion102 uint32 = 0x000A0200
	RDPGFXCapsVersion103 uint32 = 0x000A0301
	RDPGFXCapsVersion104 uint32 = 0x000A0400
	RDPGFXCapsVersion105 uint32 = 0x000A0502
	RDPGFXCapsVersion106 uint32 = 0x000A0600
)

const (
	maxBitmapCodecCount = 64
	maxRDPGFXCapsCount  = 64
	maxRDPGFXCapsData   = 64
	maxRDPGFXPDUSize    = 1024 * 1024
)

// BitmapCodecCapability is one entry in an MS-RDPBCGR Bitmap Codecs Capability Set.
type BitmapCodecCapability struct {
	GUID       [16]byte
	ID         uint8
	Name       string
	Properties []byte
}

// ParseBitmapCodecCapabilities parses a Bitmap Codecs Capability Set payload
// without the surrounding capabilitySetType/lengthCapability header.
func ParseBitmapCodecCapabilities(payload []byte) ([]BitmapCodecCapability, error) {
	if len(payload) < 1 {
		return nil, errors.New("short Bitmap Codecs capability")
	}
	count := int(payload[0])
	if count > maxBitmapCodecCount {
		return nil, fmt.Errorf("Bitmap Codecs count %d exceeds maximum %d", count, maxBitmapCodecCount)
	}
	payload = payload[1:]
	caps := make([]BitmapCodecCapability, 0, count)
	for i := 0; i < count; i++ {
		if len(payload) < 19 {
			return nil, fmt.Errorf("short Bitmap Codecs entry %d", i)
		}
		var guid [16]byte
		copy(guid[:], payload[:16])
		id := payload[16]
		propLen := int(binary.LittleEndian.Uint16(payload[17:19]))
		payload = payload[19:]
		if propLen > len(payload) {
			return nil, fmt.Errorf("Bitmap Codecs entry %d properties length %d exceeds remaining %d", i, propLen, len(payload))
		}
		props := append([]byte(nil), payload[:propLen]...)
		payload = payload[propLen:]
		caps = append(caps, BitmapCodecCapability{GUID: guid, ID: id, Name: BitmapCodecGUIDName(guid), Properties: props})
	}
	if len(payload) != 0 {
		return nil, fmt.Errorf("Bitmap Codecs capability has %d trailing bytes", len(payload))
	}
	return caps, nil
}

// FindBitmapCodec returns the first advertised bitmap codec matching guid.
func FindBitmapCodec(caps []BitmapCodecCapability, guid [16]byte) (BitmapCodecCapability, bool) {
	for _, cap := range caps {
		if cap.GUID == guid {
			return cap, true
		}
	}
	return BitmapCodecCapability{}, false
}

// RDPGFXCapabilitySet is one capability set inside an RDPGFX Caps Advertise PDU.
type RDPGFXCapabilitySet struct {
	Version        uint32
	CapsDataLength uint32
	Flags          uint32
	Data           []byte
}

// ParseRDPGFXCapsAdvertise parses an MS-RDPEGFX Caps Advertise payload, without
// the 8-byte RDPGFX command header.
func ParseRDPGFXCapsAdvertise(payload []byte) ([]RDPGFXCapabilitySet, error) {
	if len(payload) < 2 {
		return nil, errors.New("short RDPGFX Caps Advertise")
	}
	count := int(binary.LittleEndian.Uint16(payload[:2]))
	if count == 0 {
		return nil, errors.New("RDPGFX Caps Advertise contains no capability sets")
	}
	if count > maxRDPGFXCapsCount {
		return nil, fmt.Errorf("RDPGFX capability count %d exceeds maximum %d", count, maxRDPGFXCapsCount)
	}
	payload = payload[2:]
	caps := make([]RDPGFXCapabilitySet, 0, count)
	for i := 0; i < count; i++ {
		if len(payload) < 8 {
			return nil, fmt.Errorf("short RDPGFX capability set header %d", i)
		}
		version := binary.LittleEndian.Uint32(payload[:4])
		dataLen := binary.LittleEndian.Uint32(payload[4:8])
		payload = payload[8:]
		if dataLen > maxRDPGFXCapsData || int(dataLen) > len(payload) {
			return nil, fmt.Errorf("invalid RDPGFX capability set %d data length %d", i, dataLen)
		}
		data := payload[:int(dataLen)]
		payload = payload[int(dataLen):]
		var flags uint32
		if len(data) >= 4 {
			flags = binary.LittleEndian.Uint32(data[:4])
		}
		caps = append(caps, RDPGFXCapabilitySet{Version: version, CapsDataLength: dataLen, Flags: flags, Data: append([]byte(nil), data...)})
	}
	if len(payload) != 0 {
		return nil, fmt.Errorf("RDPGFX Caps Advertise has %d trailing bytes", len(payload))
	}
	return caps, nil
}

// RDPGFXAVCFlagNames returns stable symbolic names for known AVC flags.
func RDPGFXAVCFlagNames(flags uint32) []string {
	var names []string
	if flags&RDPGFXCapsFlagAVC420Enabled != 0 {
		names = append(names, "AVC420_ENABLED")
	}
	if flags&RDPGFXCapsFlagAVCDisabled != 0 {
		names = append(names, "AVC_DISABLED")
	}
	return names
}

// RDPGFXAVCDisabled reports whether the capability flags disable AVC use.
func RDPGFXAVCDisabled(flags uint32) bool { return flags&RDPGFXCapsFlagAVCDisabled != 0 }

// RDPGFXAVC420Enabled reports whether the capability flags explicitly enable AVC420.
func RDPGFXAVC420Enabled(flags uint32) bool { return flags&RDPGFXCapsFlagAVC420Enabled != 0 }

// Rect is an inclusive RDP destination rectangle.
type Rect struct {
	Left, Top, Right, Bottom uint16
}

// BuildSetSurfaceBits builds a Surface Commands Set Surface Bits command.
func BuildSetSurfaceBits(dest Rect, bpp uint8, codecID uint8, width, height uint16, bitmapData []byte) ([]byte, error) {
	if codecID == 0 {
		return nil, errors.New("codecID must be non-zero")
	}
	if width == 0 || height == 0 {
		return nil, errors.New("width and height must be non-zero")
	}
	if len(bitmapData) == 0 {
		return nil, errors.New("bitmapData must be non-empty")
	}
	if len(bitmapData) > maxRDPGFXPDUSize {
		return nil, fmt.Errorf("bitmapData length %d exceeds maximum %d", len(bitmapData), maxRDPGFXPDUSize)
	}
	out := make([]byte, 22+len(bitmapData))
	binary.LittleEndian.PutUint16(out[0:2], SurfaceCmdSetSurfaceBits)
	binary.LittleEndian.PutUint16(out[2:4], dest.Left)
	binary.LittleEndian.PutUint16(out[4:6], dest.Top)
	binary.LittleEndian.PutUint16(out[6:8], dest.Right)
	binary.LittleEndian.PutUint16(out[8:10], dest.Bottom)
	out[10] = bpp
	out[11] = 0 // flags
	out[12] = 0 // reserved
	out[13] = codecID
	binary.LittleEndian.PutUint16(out[14:16], width)
	binary.LittleEndian.PutUint16(out[16:18], height)
	binary.LittleEndian.PutUint32(out[18:22], uint32(len(bitmapData))) // #nosec G115 bounded above.
	copy(out[22:], bitmapData)
	return out, nil
}

// BuildRDPGFXPDU builds a generic RDPGFX PDU with the 8-byte command header.
func BuildRDPGFXPDU(cmdID, flags uint16, payload []byte) ([]byte, error) {
	if len(payload)+8 > maxRDPGFXPDUSize {
		return nil, fmt.Errorf("RDPGFX PDU length %d exceeds maximum %d", len(payload)+8, maxRDPGFXPDUSize)
	}
	out := make([]byte, 8+len(payload))
	binary.LittleEndian.PutUint16(out[0:2], cmdID)
	binary.LittleEndian.PutUint16(out[2:4], flags)
	binary.LittleEndian.PutUint32(out[4:8], uint32(len(out))) // #nosec G115 bounded above.
	copy(out[8:], payload)
	return out, nil
}

// BuildRDPGFXStartFrame builds an RDPGFX StartFrame PDU.
func BuildRDPGFXStartFrame(frameID uint32) ([]byte, error) {
	payload := make([]byte, 8)
	// timestamp is intentionally zero; callers needing a clock can use BuildRDPGFXPDU directly.
	binary.LittleEndian.PutUint32(payload[4:8], frameID)
	return BuildRDPGFXPDU(RDPGFXCmdStartFrame, 0, payload)
}

// BuildRDPGFXEndFrame builds an RDPGFX EndFrame PDU.
func BuildRDPGFXEndFrame(frameID uint32) ([]byte, error) {
	payload := make([]byte, 4)
	binary.LittleEndian.PutUint32(payload[0:4], frameID)
	return BuildRDPGFXPDU(RDPGFXCmdEndFrame, 0, payload)
}

// BuildRDPGFXCreateSurface builds an RDPGFX CreateSurface PDU.
func BuildRDPGFXCreateSurface(surfaceID uint16, width, height uint16, pixelFormat uint8) ([]byte, error) {
	if width == 0 || height == 0 {
		return nil, errors.New("width and height must be non-zero")
	}
	payload := make([]byte, 7)
	binary.LittleEndian.PutUint16(payload[0:2], surfaceID)
	binary.LittleEndian.PutUint16(payload[2:4], width)
	binary.LittleEndian.PutUint16(payload[4:6], height)
	payload[6] = pixelFormat
	return BuildRDPGFXPDU(RDPGFXCmdCreateSurface, 0, payload)
}

// BuildRDPGFXMapSurfaceToOutput builds an RDPGFX MapSurfaceToOutput PDU.
func BuildRDPGFXMapSurfaceToOutput(surfaceID uint16, originX, originY uint32) ([]byte, error) {
	payload := make([]byte, 12)
	binary.LittleEndian.PutUint16(payload[0:2], surfaceID)
	binary.LittleEndian.PutUint16(payload[2:4], 0) // reserved
	binary.LittleEndian.PutUint32(payload[4:8], originX)
	binary.LittleEndian.PutUint32(payload[8:12], originY)
	return BuildRDPGFXPDU(RDPGFXCmdMapSurfaceToOutput, 0, payload)
}

// BuildRDPGFXWireToSurface1 builds an RDPGFX WireToSurface_1 PDU.
func BuildRDPGFXWireToSurface1(surfaceID uint16, codecID uint16, pixelFormat uint8, dest Rect, bitmapData []byte) ([]byte, error) {
	if len(bitmapData) == 0 {
		return nil, errors.New("bitmapData must be non-empty")
	}
	if len(bitmapData)+25 > maxRDPGFXPDUSize {
		return nil, fmt.Errorf("WireToSurface_1 length %d exceeds maximum %d", len(bitmapData)+25, maxRDPGFXPDUSize)
	}
	payload := make([]byte, 17+len(bitmapData))
	binary.LittleEndian.PutUint16(payload[0:2], surfaceID)
	binary.LittleEndian.PutUint16(payload[2:4], codecID)
	payload[4] = pixelFormat
	binary.LittleEndian.PutUint16(payload[5:7], dest.Left)
	binary.LittleEndian.PutUint16(payload[7:9], dest.Top)
	binary.LittleEndian.PutUint16(payload[9:11], dest.Right)
	binary.LittleEndian.PutUint16(payload[11:13], dest.Bottom)
	binary.LittleEndian.PutUint32(payload[13:17], uint32(len(bitmapData))) // #nosec G115 bounded above.
	copy(payload[17:], bitmapData)
	return BuildRDPGFXPDU(RDPGFXCmdWireToSurface1, 0, payload)
}

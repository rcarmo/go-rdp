// Package codec exposes reusable RDP codec and graphics wire helpers.
//
// The exported capability parsers and PDU builders are aligned with the
// Microsoft protocol reference material used by this repository:
//   - MS-RDPBCGR 2.2.7.2.10 Bitmap Codecs Capability Set
//   - MS-RDPBCGR 2.2.9.1.1.3.1.2 Bitmap Update Data
//   - MS-RDPBCGR 2.2.9.1.1.3.1.2.4 Interleaved RLE Bitmap Compression
//   - MS-RDPBCGR 2.2.9.2.1.2.1 Set Surface Bits Command
//   - MS-RDPBCGR Bitmap Codecs JPEG capability/property and SetSurfaceBits usage
//   - MS-RDPEGFX 2.2.2 RDPGFX capability and frame/SurfaceToWire PDUs
//   - MS-RDPEGFX uncompressed codec ID and XRGB payload expectations
//   - MS-RDPEGFX Planar codec payload semantics for no-alpha RLE frames
//   - MS-RDPEGFX ClearCodec operations supported by the documented minimal subset
//   - MS-RDPEGFX Progressive and ProgressiveV2 payload layout helpers
//   - MS-RDPRFX RemoteFX tile/message structures, YCoCg, DWT 5/3, quantization, and RLGR
//
// Helpers in this package are deliberately protocol-level data-in/data-out
// functions. Product policy such as codec preference, Android capture format,
// environment-variable gates, client allowlists, and release defaults should
// live in applications that consume go-rdp rather than in this package.
package codec

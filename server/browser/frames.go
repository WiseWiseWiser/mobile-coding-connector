package browser

import "encoding/binary"

// Frame wire format (server -> client), 16-byte header then one JPEG:
//
//	[0]      magic 0x05 (browser frame; 0x01..0x02 are the screen mirror)
//	[1]      flags, reserved 0
//	[2..3]   reserved
//	[4..7]   sequence, big endian, wraps
//	[8..11]  capture timestamp in unix milliseconds
//	[12..15] jpeg length
//
// A JPEG length instead of a full frame length lets the client slice the image
// without knowing the header size, and keeps room for a second plane later.
const (
	frameMagic      = 0x05
	frameHeaderSize = 16

	// captureQuality keeps a 1440x900 window near 45 KB per frame, which is the
	// measured sweet spot for cellular: q80 costs ~13% more bytes for no visible
	// gain on text-heavy pages at this size.
	captureQuality = 60
)

func encodeFrame(seq uint32, tsMillis int64, jpeg []byte) []byte {
	buf := make([]byte, frameHeaderSize+len(jpeg))
	buf[0] = frameMagic
	binary.BigEndian.PutUint32(buf[4:8], seq)
	binary.BigEndian.PutUint32(buf[8:12], uint32(tsMillis))
	binary.BigEndian.PutUint32(buf[12:16], uint32(len(jpeg)))
	copy(buf[frameHeaderSize:], jpeg)
	return buf
}

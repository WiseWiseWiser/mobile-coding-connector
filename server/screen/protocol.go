package screen

import (
	"encoding/binary"
	"errors"
)

// Binary message types. High-rate traffic (frames, pointer) is binary so the
// client can hand the JPEG payloads straight to createImageBitmap without
// parsing base64. Low-rate control traffic is JSON text.
const (
	// server -> client
	msgFrame  byte = 0x01
	msgCursor byte = 0x02

	// client -> server
	msgPointer    byte = 0x81
	msgKey        byte = 0x82
	msgText       byte = 0x83
	msgReleaseAll byte = 0x84
)

// Pointer actions.
const (
	actionMove  byte = 1
	actionDown  byte = 2
	actionUp    byte = 3
	actionWheel byte = 4
)

// Mouse buttons as they appear in PointerEvent.button offset by one so 0 means
// "no button".
const (
	buttonNone   byte = 0
	buttonLeft   byte = 1
	buttonMiddle byte = 2
	buttonRight  byte = 3
)

// Cursor shapes the client can draw.
const (
	shapeArrow  byte = 0
	shapeIBeam  byte = 1
	shapeHand   byte = 2
	shapeResize byte = 3
)

// frameFixedSize is the header length written before tile payloads.
const (
	frameHeaderSize = 22
	tileHeaderSize  = 8
)

// Tile is one changed rectangle of the current frame.
type Tile struct {
	X int
	Y int
	// JPEG is a complete baseline JPEG for the tile.
	JPEG []byte
}

// encodeFrame packs a frame header plus the given tiles.
//
// Layout (little endian):
//
//	u8 type | u8 flags | u16 display | u32 seq | u32 tsMs
//	u16 imgW | u16 imgH | u16 tileW | u16 tileH | u16 nTiles
//	n * (u16 x | u16 y | u32 len | bytes)
func encodeFrame(seq uint32, tsMs uint32, imgW, imgH, tileW, tileH int, tiles []Tile, full bool) []byte {
	size := frameHeaderSize
	for _, t := range tiles {
		size += tileHeaderSize + len(t.JPEG)
	}
	buf := make([]byte, frameHeaderSize, size)
	buf[0] = msgFrame
	if full {
		buf[1] = 1
	}
	binary.LittleEndian.PutUint16(buf[2:], 0)
	binary.LittleEndian.PutUint32(buf[4:], seq)
	binary.LittleEndian.PutUint32(buf[8:], tsMs)
	binary.LittleEndian.PutUint16(buf[12:], uint16(imgW))
	binary.LittleEndian.PutUint16(buf[14:], uint16(imgH))
	binary.LittleEndian.PutUint16(buf[16:], uint16(tileW))
	binary.LittleEndian.PutUint16(buf[18:], uint16(tileH))
	binary.LittleEndian.PutUint16(buf[20:], uint16(len(tiles)))
	var hdr [tileHeaderSize]byte
	for _, t := range tiles {
		binary.LittleEndian.PutUint16(hdr[0:], uint16(t.X))
		binary.LittleEndian.PutUint16(hdr[2:], uint16(t.Y))
		binary.LittleEndian.PutUint32(hdr[4:], uint32(len(t.JPEG)))
		buf = append(buf, hdr[:]...)
		buf = append(buf, t.JPEG...)
	}
	return buf
}

// encodeCursor packs a cursor update: where the host pointer is, in image
// pixels, plus enough state for the client to draw it.
//
// Layout: u8 type | u8 visible | u8 shape | u8 buttons | u16 x | u16 y
func encodeCursor(x, y int, shape byte, buttons byte, visible bool) []byte {
	buf := make([]byte, 8)
	buf[0] = msgCursor
	if visible {
		buf[1] = 1
	}
	buf[2] = shape
	buf[3] = buttons
	binary.LittleEndian.PutUint16(buf[4:], uint16(clampU16(x)))
	binary.LittleEndian.PutUint16(buf[6:], uint16(clampU16(y)))
	return buf
}

func clampU16(v int) int {
	if v < 0 {
		return 0
	}
	if v > 0xffff {
		return 0xffff
	}
	return v
}

// pointerEvent is a decoded client pointer message.
//
// Layout: u8 type | u8 action | u8 button | u8 mods | u16 x | u16 y | i16 aux
type pointerEvent struct {
	Action byte
	Button byte
	Mods   uint8
	X      int
	Y      int
	Aux    int
}

const pointerEventSize = 10

var errShortMessage = errors.New("screen: message too short")

func decodePointer(b []byte) (pointerEvent, error) {
	if len(b) < pointerEventSize {
		return pointerEvent{}, errShortMessage
	}
	return pointerEvent{
		Action: b[1],
		Button: b[2],
		Mods:   b[3],
		X:      int(binary.LittleEndian.Uint16(b[4:])),
		Y:      int(binary.LittleEndian.Uint16(b[6:])),
		Aux:    int(int16(binary.LittleEndian.Uint16(b[8:]))),
	}, nil
}

// keyEvent is a decoded client key message carrying a DOM KeyboardEvent.code
// (for example "KeyA", "ShiftLeft"), so the server can translate to native
// keycodes and honour the host layout.
//
// Layout: u8 type | u8 down | u16 len | bytes code
type keyEvent struct {
	Down bool
	Code string
}

func decodeKey(b []byte) (keyEvent, error) {
	if len(b) < 4 {
		return keyEvent{}, errShortMessage
	}
	n := int(binary.LittleEndian.Uint16(b[2:]))
	if len(b) < 4+n {
		return keyEvent{}, errShortMessage
	}
	return keyEvent{Down: b[1] == 1, Code: string(b[4 : 4+n])}, nil
}

// textMessage carries a UTF-8 string to type on the host.
//
// Layout: u8 type | u16 len | bytes utf8
func decodeText(b []byte) (string, error) {
	if len(b) < 3 {
		return "", errShortMessage
	}
	n := int(binary.LittleEndian.Uint16(b[1:]))
	if len(b) < 3+n {
		return "", errShortMessage
	}
	return string(b[3 : 3+n]), nil
}

// eventHeaderSize is the leading bytes shared by every client message.
func eventType(b []byte) (byte, bool) {
	if len(b) == 0 {
		return 0, false
	}
	return b[0], true
}

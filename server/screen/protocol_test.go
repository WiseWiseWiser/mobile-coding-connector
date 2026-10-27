package screen

import (
	"encoding/binary"
	"testing"
)

func TestDecodePointer(t *testing.T) {
	buf := make([]byte, pointerEventSize)
	buf[0] = msgPointer
	buf[1] = actionDown
	buf[2] = buttonRight
	buf[3] = 0x01
	binary.LittleEndian.PutUint16(buf[4:], 512)
	binary.LittleEndian.PutUint16(buf[6:], 300)
	binary.LittleEndian.PutUint16(buf[8:], 0xfffd)

	ev, err := decodePointer(buf)
	if err != nil {
		t.Fatalf("decodePointer: %v", err)
	}
	if ev.Action != actionDown || ev.Button != buttonRight || ev.Mods != 0x01 {
		t.Fatalf("unexpected header: %+v", ev)
	}
	if ev.X != 512 || ev.Y != 300 {
		t.Fatalf("unexpected coords: %+v", ev)
	}
	if ev.Aux != -3 {
		t.Fatalf("aux = %d, want -3", ev.Aux)
	}
}

func TestDecodePointerRejectsShort(t *testing.T) {
	if _, err := decodePointer(make([]byte, pointerEventSize-1)); err == nil {
		t.Fatal("expected error for short pointer message")
	}
}

func TestDecodeKey(t *testing.T) {
	code := "ShiftLeft"
	buf := make([]byte, 4+len(code))
	buf[0] = msgKey
	buf[1] = 1
	binary.LittleEndian.PutUint16(buf[2:], uint16(len(code)))
	copy(buf[4:], code)

	ev, err := decodeKey(buf)
	if err != nil {
		t.Fatalf("decodeKey: %v", err)
	}
	if !ev.Down || ev.Code != code {
		t.Fatalf("unexpected key event: %+v", ev)
	}
	// Truncated payload must not panic or be accepted.
	if _, err := decodeKey(buf[:len(buf)-1]); err == nil {
		t.Fatal("expected error for truncated key message")
	}
}

func TestDecodeText(t *testing.T) {
	text := "héllo wörld"
	buf := make([]byte, 3+len(text))
	buf[0] = msgText
	binary.LittleEndian.PutUint16(buf[1:], uint16(len(text)))
	copy(buf[3:], text)

	got, err := decodeText(buf)
	if err != nil {
		t.Fatalf("decodeText: %v", err)
	}
	if got != text {
		t.Fatalf("text = %q, want %q", got, text)
	}
	if _, err := decodeText(buf[:len(buf)-1]); err == nil {
		t.Fatal("expected error for truncated text message")
	}
	if _, err := decodeText([]byte{msgText, 0}); err == nil {
		t.Fatal("expected error for 2-byte text message")
	}
}

func TestEventType(t *testing.T) {
	if _, ok := eventType(nil); ok {
		t.Fatal("empty message must not decode")
	}
	if typ, ok := eventType([]byte{msgPointer, 0}); !ok || typ != msgPointer {
		t.Fatalf("eventType = %v,%v", typ, ok)
	}
}

func TestEncodeFrameLayout(t *testing.T) {
	tiles := []Tile{
		{X: 0, Y: 0, JPEG: []byte{1, 2, 3}},
		{X: 128, Y: 256, JPEG: []byte{9}},
	}
	frame := encodeFrame(7, 1234, 640, 480, TileSize, TileSize, tiles, true)

	if len(frame) != frameHeaderSize+2*tileHeaderSize+4 {
		t.Fatalf("frame length = %d", len(frame))
	}
	if frame[0] != msgFrame || frame[1] != 1 {
		t.Fatalf("bad header: %v", frame[:2])
	}
	if got := binary.LittleEndian.Uint32(frame[4:]); got != 7 {
		t.Fatalf("seq = %d", got)
	}
	if got := binary.LittleEndian.Uint16(frame[16:]); got != TileSize {
		t.Fatalf("tile width = %d", got)
	}
	if got := binary.LittleEndian.Uint16(frame[20:]); got != 2 {
		t.Fatalf("tile count = %d", got)
	}

	off := frameHeaderSize
	x := binary.LittleEndian.Uint16(frame[off:])
	y := binary.LittleEndian.Uint16(frame[off+2:])
	n := binary.LittleEndian.Uint32(frame[off+4:])
	if x != 0 || y != 0 || n != 3 {
		t.Fatalf("tile0 = %d,%d,%d", x, y, n)
	}
	second := off + tileHeaderSize + int(n)
	if got := binary.LittleEndian.Uint16(frame[second:]); got != 128 {
		t.Fatalf("tile1 x = %d", got)
	}
}

func TestEncodeCursor(t *testing.T) {
	buf := encodeCursor(300, 700, shapeHand, 1, true)
	if len(buf) != 8 || buf[0] != msgCursor {
		t.Fatalf("bad cursor message: %v", buf)
	}
	if buf[1] != 1 || buf[2] != shapeHand || buf[3] != 1 {
		t.Fatalf("bad cursor flags: %v", buf[:4])
	}
	if got := binary.LittleEndian.Uint16(buf[4:]); got != 300 {
		t.Fatalf("x = %d", got)
	}
	if got := binary.LittleEndian.Uint16(buf[6:]); got != 700 {
		t.Fatalf("y = %d", got)
	}
	// Negative coordinates clamp instead of wrapping around.
	neg := encodeCursor(-5, -9, shapeArrow, 0, false)
	if got := binary.LittleEndian.Uint16(neg[4:]); got != 0 {
		t.Fatalf("clamped x = %d", got)
	}
	if neg[1] != 0 {
		t.Fatal("cursor should be hidden")
	}
}

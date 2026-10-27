package screen

import (
	"encoding/binary"
	"image"
	"image/color"
	"testing"
)

// staticFrame builds a frame whose pixels never change, so publishing it twice
// leaves the tiler with nothing dirty to send.
func staticFrame(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, color.RGBA{R: uint8(x % 200), G: uint8(y % 200), B: 90, A: 255})
		}
	}
	return img
}

// frames drains the queued image frames of a viewer, ignoring cursor updates and
// JSON status messages.
func frames(v *viewer) [][]byte {
	var out [][]byte
	for {
		select {
		case msg := <-v.send:
			if !msg.text && len(msg.data) > 1 && msg.data[0] == msgFrame {
				out = append(out, msg.data)
			}
		default:
			return out
		}
	}
}

func frameIsFull(data []byte) bool { return data[1]&1 == 1 }

func frameTileCount(data []byte) int { return int(binary.LittleEndian.Uint16(data[20:])) }

func fullTileCount(m *Manager) int {
	cols := (m.imgW + TileSize - 1) / TileSize
	rows := (m.imgH + TileSize - 1) / TileSize
	return cols * rows
}

// A viewer that joins a stream whose tiler is already primed has missed every
// tile sent so far. On a static screen no tile is ever dirty again, so the join
// has to replay the last image instead of waiting for a frame that never comes.
func TestLateViewerGetsSnapshotOnStaticScreen(t *testing.T) {
	m, _, first := testManager(t)
	m.cfg.Enabled = false // keep the capture loop out of this test
	// startCaptureLocked owns the tiler in production; this test publishes
	// directly, so it builds the same one the capture path would.
	m.tiler = newTiler(TileSize, m.cfg.Quality)
	frame := staticFrame(m.imgW, m.imgH)

	m.publish(m.gen, frame)

	got := frames(first)
	if len(got) != 1 {
		t.Fatalf("first viewer got %d frames, want 1", len(got))
	}
	if !frameIsFull(got[0]) || frameTileCount(got[0]) != fullTileCount(m) {
		t.Fatalf("first frame full=%v tiles=%d, want full with %d tiles",
			frameIsFull(got[0]), frameTileCount(got[0]), fullTileCount(m))
	}

	// addViewer is what the websocket handler calls; nil is safe here because
	// capture is disabled and the socket is only used by the write loop.
	late := m.addViewer(nil)
	got = frames(late)
	if len(got) != 1 {
		t.Fatalf("late viewer got %d frames on join, want 1 keyframe", len(got))
	}
	if !frameIsFull(got[0]) || frameTileCount(got[0]) != fullTileCount(m) {
		t.Fatalf("late frame full=%v tiles=%d, want full with %d tiles",
			frameIsFull(got[0]), frameTileCount(got[0]), fullTileCount(m))
	}
	if late.needsKeyframe.Load() {
		t.Fatal("keyframe request survived the snapshot it was sent for")
	}

	// And once served, a static screen goes quiet again.
	m.publish(m.gen, frame)
	if extra := frames(late); len(extra) != 0 {
		t.Fatalf("static screen still sent %d frames", len(extra))
	}
}

// Before the first frame there is nothing to replay, so the join keeps asking
// for a keyframe and the first capture answers it.
func TestLateViewerBeforeFirstFrameKeepsKeyframeRequest(t *testing.T) {
	m, _, _ := testManager(t)
	m.cfg.Enabled = false
	m.tiler = newTiler(TileSize, m.cfg.Quality)

	late := m.addViewer(nil)
	if !late.needsKeyframe.Load() {
		t.Fatal("late viewer did not ask for a keyframe")
	}
	if got := frames(late); len(got) != 0 {
		t.Fatalf("late viewer got %d frames before any capture, want 0", len(got))
	}

	m.publish(m.gen, staticFrame(m.imgW, m.imgH))
	got := frames(late)
	if len(got) != 1 || !frameIsFull(got[0]) || frameTileCount(got[0]) != fullTileCount(m) {
		t.Fatalf("first frame after join: got %d frames, want 1 full with %d tiles", len(got), fullTileCount(m))
	}
	if late.needsKeyframe.Load() {
		t.Fatal("keyframe request was not cleared after the frame was sent")
	}
}

// A viewer with a full queue drops the frame. The hub has to keep that viewer's
// keyframe request alive; clearing every viewer after the fan-out would erase
// exactly the retry and leave the slow viewer frozen on a stale canvas.
func TestDroppedFrameKeepsKeyframeRequest(t *testing.T) {
	m, _, slow := testManager(t)
	m.cfg.Enabled = false
	m.tiler = newTiler(TileSize, m.cfg.Quality)
	// One slot that is already occupied: every frame publish drops.
	slow.send = make(chan outMessage, 1)
	slow.send <- outMessage{data: []byte{0xff}}

	m.publish(m.gen, staticFrame(m.imgW, m.imgH))
	if !slow.needsKeyframe.Load() {
		t.Fatal("dropped frame left no keyframe request")
	}

	changed := staticFrame(m.imgW, m.imgH)
	changed.SetRGBA(10, 10, color.RGBA{R: 255, A: 255})
	m.publish(m.gen, changed)
	if !slow.needsKeyframe.Load() {
		t.Fatal("keyframe request did not survive the next publish")
	}

	// The viewer catches up: the next frame it accepts clears the request.
	<-slow.send
	m.publish(m.gen, changed)
	if slow.needsKeyframe.Load() {
		t.Fatal("keyframe request was not cleared by a delivered frame")
	}
	got := frames(slow)
	if len(got) != 1 || !frameIsFull(got[0]) {
		t.Fatalf("recovered viewer got %d frames (full=%v), want one full frame", len(got), len(got) == 1 && frameIsFull(got[0]))
	}
}

// Pointer updates are coalesced into a single slot so that a fast moving cursor
// can never fill the queue and starve the frames behind it.
func TestCursorUpdatesCoalesce(t *testing.T) {
	v := newTestViewer("v-cursor")
	v.cursor = make(chan outMessage, 1)
	for i := 1; i <= 5; i++ {
		v.enqueueCursor(outMessage{data: []byte{byte(i)}})
	}
	if len(v.cursor) != 1 {
		t.Fatalf("cursor queue holds %d messages, want 1", len(v.cursor))
	}
	if msg := <-v.cursor; len(msg.data) != 1 || msg.data[0] != 5 {
		t.Fatalf("kept cursor %v, want the newest one (5)", msg.data)
	}
	if v.needsKeyframe.Load() {
		t.Fatal("a coalesced cursor update must not request a keyframe")
	}
}

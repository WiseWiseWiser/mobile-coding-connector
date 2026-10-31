package browser

import (
	"net/http"
	"strconv"
	"time"
)

// HTTP fallbacks for a viewer whose WebSocket cannot connect: the frame poll
// serves the newest capture over plain GET (304 when nothing changed, so a
// slow link pays zero bytes for a still page), and the input endpoint wraps
// the same enqueueInput path the websocket reader feeds. RegisterAPI mounts
// both beside /api/browser/ws; the server's auth middleware covers them like
// every other /api route.

func (m *Manager) handleFrame(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	after := uint32(0)
	if raw := r.URL.Query().Get("after"); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid after sequence")
			return
		}
		after = uint32(parsed)
	}
	// First touch may find capture stopped: start it so the poll has a frame
	// to hand out instead of 304ing forever on a stale cache.
	m.hub.touchPoll()
	defer m.hub.pollIdle()

	// The capture loop publishes at ~1 fps in shot mode; a poll that arrives
	// right after start would answer with an empty cache. Give the first frame
	// one kick and a short wait before deciding there is nothing yet.
	frame, seq := m.hub.frameFor(after)
	if frame == nil && after == 0 {
		deadline := time.Now().Add(2500 * time.Millisecond)
		for frame == nil && time.Now().Before(deadline) {
			time.Sleep(150 * time.Millisecond)
			frame, seq = m.hub.frameFor(after)
		}
	}
	if frame == nil {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Frame-Seq", strconv.FormatUint(uint64(seq), 10))
	w.Header().Set("X-Frame-Max-Seq", strconv.FormatUint(uint64(m.hub.seqValue()), 10))
	_, _ = w.Write(frame)
}

func (m *Manager) handleInput(w http.ResponseWriter, r *http.Request) {
	var body InputEvent
	if !decodeBody(w, r, &body) {
		return
	}
	switch body.Kind {
	case InputTap, InputDown, InputMove, InputUp, InputWheel, InputKey, InputText:
	default:
		writeError(w, http.StatusBadRequest, "unknown input type")
		return
	}
	if body.X < 0 || body.X > 1 || body.Y < 0 || body.Y > 1 {
		writeError(w, http.StatusBadRequest, "x and y are normalized 0..1")
		return
	}
	if body.Button == 0 && (body.Kind == InputTap || body.Kind == InputDown || body.Kind == InputUp || body.Kind == InputMove) {
		body.Button = ButtonLeft
	}
	m.hub.enqueueInput(body)
	writeJSON(w, http.StatusOK, m.Snapshot())
}

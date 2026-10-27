package browser

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/websocket"

	"github.com/xhd2015/ai-critic/server/wskeepalive"
)

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
	// Frames reach ~50 KB; a bigger write buffer avoids a syscall per frame.
	ReadBufferSize:  4096,
	WriteBufferSize: 256 * 1024,
}

const wsReadLimit = 1 << 20

// clientMessage is one viewer command. Coordinates are normalized [0,1] so the
// page does not need to know the achieved CSS viewport.
//
// The socket is one-way for page input only. Session and tab operations need a
// response (the new tab list, a 409 for the last tab) and so go through REST;
// adding them here would duplicate the error handling the page already has.
type clientMessage struct {
	Type      string  `json:"type"`
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	Button    int     `json:"button"`
	DeltaX    float64 `json:"delta_x"`
	DeltaY    float64 `json:"delta_y"`
	Key       string  `json:"key"`
	Code      string  `json:"code"`
	VK        int     `json:"vk"`
	Modifiers int     `json:"modifiers"`
	Text      string  `json:"text"`
	URL       string  `json:"url"`
	Preset    string  `json:"preset"`
	Mode      string  `json:"mode"`
}

func (m *Manager) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	conn.SetReadLimit(wsReadLimit)

	v := &viewer{conn: conn}
	m.hub.addViewer(v)
	_ = v.writeJSON(map[string]any{"type": "hello", "session": m.sessionIDValue(), "status": m.Snapshot()})

	// Attach in the background: the page can paint its shell and show progress
	// instead of staring at a connecting socket for 25 seconds.
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		if err := m.Ensure(ctx); err != nil {
			_ = v.writeJSON(map[string]any{"type": "error", "message": err.Error()})
			return
		}
		m.hub.kick()
		_ = v.writeJSON(map[string]any{"type": "status", "status": m.Snapshot()})
	}()

	stopKeepalive := wskeepalive.Start(conn, wskeepalive.DefaultInterval)
	m.readLoop(v)
	stopKeepalive()
	v.close()
	m.hub.removeViewer(v)
}

func (m *Manager) readLoop(v *viewer) {
	for {
		_, data, err := v.conn.ReadMessage()
		if err != nil {
			return
		}
		var msg clientMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			_ = v.writeJSON(map[string]any{"type": "error", "message": "bad message: " + err.Error()})
			continue
		}
		switch msg.Type {
		case InputTap, InputDown, InputMove, InputUp:
			m.hub.enqueueInput(InputEvent{
				Kind:      msg.Type,
				X:         clamp01(msg.X),
				Y:         clamp01(msg.Y),
				Button:    normalizeButton(msg.Button),
				Modifiers: msg.Modifiers,
			})
		case InputWheel:
			m.hub.enqueueInput(InputEvent{
				Kind:   InputWheel,
				X:      clamp01(msg.X),
				Y:      clamp01(msg.Y),
				DeltaX: msg.DeltaX,
				DeltaY: msg.DeltaY,
			})
		case InputKey:
			m.hub.enqueueInput(InputEvent{Kind: InputKey, Key: msg.Key, Code: msg.Code, VK: msg.VK, Modifiers: msg.Modifiers})
		case InputText:
			m.hub.enqueueInput(InputEvent{Kind: InputText, Text: msg.Text})
		case "navigate", "preset", "mode", "refresh", "status":
			go m.handleControl(v, msg)
		}
	}
}

// handleControl runs the slow commands (navigation, resize) off the read goroutine
// so pointer input keeps flowing while they work.
func (m *Manager) handleControl(v *viewer, msg clientMessage) {
	var err error
	switch msg.Type {
	case "navigate":
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		err = m.Navigate(ctx, msg.URL)
		cancel()
	case "preset":
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		err = m.SetPreset(ctx, msg.Preset)
		cancel()
	case "mode":
		err = m.SetMode(msg.Mode)
	case "refresh":
		m.hub.kick()
	}
	if err != nil {
		_ = v.writeJSON(map[string]any{"type": "error", "message": err.Error()})
		return
	}
	m.hub.broadcastJSON(map[string]any{"type": "status", "status": m.Snapshot()})
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func normalizeButton(b int) int {
	switch b {
	case ButtonNone, ButtonLeft, ButtonMiddle, ButtonRight:
		return b
	}
	return ButtonLeft
}

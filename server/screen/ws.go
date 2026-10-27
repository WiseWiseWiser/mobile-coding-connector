package screen

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gorilla/websocket"

	"github.com/xhd2015/ai-critic/server/wskeepalive"
)

var wsUpgrader = websocket.Upgrader{
	CheckOrigin:     func(r *http.Request) bool { return true },
	ReadBufferSize:  4096,
	WriteBufferSize: 128 * 1024,
}

const wsReadLimit = 1 << 20

// handleWS upgrades the connection and supervises its lifetime.
func (m *Manager) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	conn.SetReadLimit(wsReadLimit)

	v := m.addViewer(conn)
	if label := r.URL.Query().Get("label"); label != "" {
		m.mu.Lock()
		v.label = label
		m.mu.Unlock()
	}
	go v.writeLoop()
	stopKeepalive := wskeepalive.Start(conn, wskeepalive.DefaultInterval)

	m.readLoop(v)

	stopKeepalive()
	m.removeViewer(v)
}

func (m *Manager) readLoop(v *viewer) {
	for {
		kind, data, err := v.conn.ReadMessage()
		if err != nil {
			return
		}
		switch kind {
		case websocket.BinaryMessage:
			m.handleBinary(v, data)
		case websocket.TextMessage:
			m.handleText(v, data)
		}
	}
}

type clientMessage struct {
	Type string `json:"type"`
}

func (m *Manager) handleText(v *viewer, data []byte) {
	var msg clientMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return
	}
	switch msg.Type {
	case "request_control":
		granted, reason := m.requestControl(v)
		if granted {
			return
		}
		payload, _ := json.Marshal(map[string]any{
			"type":    "control",
			"granted": false,
			"reason":  reason,
			"holder":  m.ControlHolderID(),
		})
		v.enqueue(outMessage{text: true, data: payload})
	case "release_control":
		m.releaseControlFor(v, "released by client")
	case "release_all":
		m.mu.Lock()
		if m.holder == v {
			m.backend.releaseAll()
		}
		m.mu.Unlock()
	case "status":
		m.mu.Lock()
		payload, _ := json.Marshal(map[string]any{"type": "status", "status": m.statusLocked()})
		m.mu.Unlock()
		v.enqueue(outMessage{text: true, data: payload})
	}
}

// errNotSupported is returned by platform hooks that do not exist yet.
var errNotSupported = errors.New("not supported on this platform")

// pingAfter is the delay used by tests to assert the connection stays healthy.
const pingAfter = 30 * time.Second

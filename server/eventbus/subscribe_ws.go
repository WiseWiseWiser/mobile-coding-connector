package eventbus

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

// SubscribeWSPath is the main-mux path for WebSocket event subscription.
const SubscribeWSPath = "/api/event-bus/ws"

var subscribeUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	// Keep handshake short; tests dial with 2s timeout.
	HandshakeTimeout: 5 * time.Second,
}

// RegisterSubscribeWS registers GET /api/event-bus/ws on mux.
// Each upgraded connection receives JSON Event frames for subsequent hub publishes.
// Optional query ?replay=N sends up to N recent ring events (oldest first) before live.
func RegisterSubscribeWS(mux *http.ServeMux, hub *Hub) {
	if mux == nil || hub == nil {
		return
	}
	mux.HandleFunc(SubscribeWSPath, func(w http.ResponseWriter, r *http.Request) {
		handleSubscribeWS(w, r, hub)
	})
}

func handleSubscribeWS(w http.ResponseWriter, r *http.Request, hub *Hub) {
	conn, err := subscribeUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	// Subscribe first so live events published during replay flush are not lost.
	ch, cancel := hub.Subscribe()
	defer cancel()

	// Optional replay of recent ring events before live frames.
	if n := parseReplayQuery(r); n > 0 {
		for _, ev := range hub.Recent(n) {
			data, err := json.Marshal(ev)
			if err != nil {
				return
			}
			_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}
		}
	}

	// Close the WS when the client disconnects / context ends so Publish fan-out
	// does not keep a dead subscriber indefinitely. A reader goroutine detects
	// client close while we block on channel sends.
	clientDone := make(chan struct{})
	go func() {
		defer close(clientDone)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	for {
		select {
		case <-clientDone:
			return
		case <-r.Context().Done():
			return
		case ev, ok := <-ch:
			if !ok {
				return
			}
			data, err := json.Marshal(ev)
			if err != nil {
				return
			}
			_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}
		}
	}
}

func parseReplayQuery(r *http.Request) int {
	if r == nil {
		return 0
	}
	q := r.URL.Query().Get("replay")
	if q == "" {
		return 0
	}
	n, err := strconv.Atoi(q)
	if err != nil || n <= 0 {
		return 0
	}
	return n
}

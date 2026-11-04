package services

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/xhd2015/agent-pro/agent/streaming/sse"
)

// Valid system-operation actions for the stream endpoint.
var systemStreamActions = map[string]bool{
	SystemActionStart:   true,
	SystemActionStop:    true,
	SystemActionRestart: true,
}

// registerSystemStreamAPI mounts the unified operation-stream endpoint for
// system services. Controllers that implement SystemOperationStreamer run the
// action inside the stream; everything else degrades to a short "no
// operation log" stream so the card's log area still gets a response.
func registerSystemStreamAPI(mux *http.ServeMux, m *Manager) {
	mux.HandleFunc("GET /api/services/system/stream", func(w http.ResponseWriter, r *http.Request) {
		m.handleSystemOperationStream(w, r)
	})
}

func (m *Manager) handleSystemOperationStream(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	action := r.URL.Query().Get("action")
	if !systemStreamActions[action] {
		http.Error(w, "action must be start, stop or restart", http.StatusBadRequest)
		return
	}
	svc, ok := m.systemService(id)
	if !ok {
		http.Error(w, fmt.Sprintf("system service %q not found", id), http.StatusNotFound)
		return
	}

	// Preferred path: the controller streams its own operation.
	if streamer, ok := svc.Controller.(SystemOperationStreamer); ok {
		if err := streamer.StreamSystemOperation(w, action); err != nil {
			// Headers may already be written; log server-side only.
			fmt.Printf("[services] system stream %s %s: %v\n", svc.ID, action, err)
		}
		return
	}

	// Fallback: run the action through the plain path, emitting a compact
	// before/after so the card's log area still shows what happened.
	sw := sse.NewWriter(w)
	if sw == nil {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	sw.SendLog(fmt.Sprintf("%s %s…", action, svc.Name))
	var err error
	started := time.Now()
	switch action {
	case SystemActionStart:
		_, err = m.Start(id)
	case SystemActionStop:
		err = m.Stop(id)
	case SystemActionRestart:
		err = m.Restart(id)
	}
	if err != nil {
		sw.SendError(err.Error())
		sw.SendDone(map[string]string{"status": "error", "action": action})
		return
	}
	sw.SendLog(fmt.Sprintf("%s completed in %s", action, time.Since(started).Round(time.Millisecond)))
	sw.SendDone(map[string]string{"status": "ok", "action": action})
}

// statusPort extracts the listen port from a status snapshot, for stream done events.
func statusPort(m *Manager, id string) string {
	if st, ok := m.statusByID(id); ok && st.Port > 0 {
		return strconv.Itoa(st.Port)
	}
	return ""
}

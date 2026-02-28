package acp

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// RegisterAgentAPI registers HTTP endpoints for a given ACP agent at the specified prefix.
// For example, RegisterAgentAPI(mux, "/api/cursor-acp", cursorAgent) registers:
//   - GET  /api/cursor-acp/status
//   - POST /api/cursor-acp/connect
//   - POST /api/cursor-acp/disconnect
//   - POST /api/cursor-acp/prompt
//   - POST /api/cursor-acp/cancel
func RegisterAgentAPI(mux *http.ServeMux, prefix string, agent Agent) {
	mux.HandleFunc(prefix+"/session/messages", func(w http.ResponseWriter, r *http.Request) {
		handleSessionMessages(w, r, agent)
	})
	mux.HandleFunc(prefix+"/status", func(w http.ResponseWriter, r *http.Request) {
		handleAgentStatus(w, r, agent)
	})
	mux.HandleFunc(prefix+"/sessions", func(w http.ResponseWriter, r *http.Request) {
		handleAgentSessions(w, r, agent)
	})
	mux.HandleFunc(prefix+"/models", func(w http.ResponseWriter, r *http.Request) {
		handleAgentModels(w, r, agent)
	})
	mux.HandleFunc(prefix+"/session/model", func(w http.ResponseWriter, r *http.Request) {
		handleUpdateSessionModel(w, r, agent)
	})
	mux.HandleFunc(prefix+"/connect", func(w http.ResponseWriter, r *http.Request) {
		handleAgentConnect(w, r, agent)
	})
	mux.HandleFunc(prefix+"/disconnect", func(w http.ResponseWriter, r *http.Request) {
		handleAgentDisconnect(w, r, agent)
	})
	mux.HandleFunc(prefix+"/prompt", func(w http.ResponseWriter, r *http.Request) {
		handleAgentPrompt(w, r, agent)
	})
	mux.HandleFunc(prefix+"/cancel", func(w http.ResponseWriter, r *http.Request) {
		handleAgentCancel(w, r, agent)
	})
}

func handleAgentStatus(w http.ResponseWriter, _ *http.Request, agent Agent) {
	status := agent.Status()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func handleAgentSessions(w http.ResponseWriter, _ *http.Request, agent Agent) {
	sessions := agent.Sessions()
	if sessions == nil {
		sessions = []SessionEntry{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sessions)
}

func handleSessionMessages(w http.ResponseWriter, r *http.Request, agent Agent) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		sessionID := r.URL.Query().Get("sessionId")
		if sessionID == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "sessionId required"})
			return
		}
		data, err := agent.GetMessages(sessionID)
		if err != nil {
			json.NewEncoder(w).Encode(json.RawMessage("[]"))
			return
		}
		w.Write(data)

	case http.MethodPost:
		var body struct {
			SessionID string          `json:"sessionId"`
			Messages  json.RawMessage `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.SessionID == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid request"})
			return
		}
		if err := agent.SaveMessages(body.SessionID, body.Messages); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"message": "OK"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleAgentModels(w http.ResponseWriter, _ *http.Request, agent Agent) {
	models, err := agent.Models()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
		return
	}
	if models == nil {
		models = []ModelInfo{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models)
}

func handleUpdateSessionModel(w http.ResponseWriter, r *http.Request, agent Agent) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		SessionID string `json:"sessionId"`
		Model     string `json:"model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.SessionID == "" || body.Model == "" {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	agent.UpdateSessionModel(body.SessionID, body.Model)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "OK"})
}

func handleAgentConnect(w http.ResponseWriter, r *http.Request, agent Agent) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if agent.IsConnected() {
		agent.Disconnect()
	}

	var body struct {
		CWD       string `json:"cwd"`
		SessionID string `json:"sessionId"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	cwdSource := "request"
	cwd := body.CWD
	if cwd == "" {
		status := agent.Status()
		cwd = status.CWD
		cwdSource = "status-fallback"
	}
	fmt.Printf("DEBUG handleAgentConnect cwd=%q source=%s sessionId=%q\n", cwd, cwdSource, body.SessionID)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	logFn := func(message string) {
		writeSSE(w, flusher, SessionUpdate{Type: "log", Message: message})
	}

	sessionID, err := agent.Connect(cwd, body.SessionID, logFn)
	if err != nil {
		writeSSE(w, flusher, SessionUpdate{Type: "error", Message: err.Error()})
		fmt.Fprintf(w, "data: [DONE]\n\n")
		flusher.Flush()
		return
	}

	status := agent.Status()
	writeSSE(w, flusher, SessionUpdate{
		Type:    "connected",
		Message: sessionID,
		Model:   status.Model,
	})
	fmt.Fprintf(w, "data: [DONE]\n\n")
	flusher.Flush()
}

func handleAgentDisconnect(w http.ResponseWriter, r *http.Request, agent Agent) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	agent.Disconnect()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Disconnected"})
}

func handleAgentPrompt(w http.ResponseWriter, r *http.Request, agent Agent) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		SessionID string `json:"sessionId"`
		Prompt    string `json:"prompt"`
		Model     string `json:"model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if !agent.IsConnected() {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"message": "Not connected"})
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	type promptDone struct {
		result *PromptResult
		err    error
	}
	doneCh := make(chan promptDone, 1)
	go func() {
		result, err := agent.SendPrompt(body.SessionID, body.Prompt, body.Model)
		doneCh <- promptDone{result: result, err: err}
	}()

	updates := agent.Updates()
	for {
		select {
		case update, ok := <-updates:
			if !ok {
				return
			}
			writeSSE(w, flusher, update)
		case pd := <-doneCh:
			// Drain remaining updates before writing final status
			for {
				select {
				case update := <-updates:
					writeSSE(w, flusher, update)
				default:
					goto drained
				}
			}
		drained:
			if pd.err != nil {
				writeSSE(w, flusher, SessionUpdate{Type: "error", Message: pd.err.Error()})
			} else {
				writeSSE(w, flusher, SessionUpdate{Type: "done", Message: pd.result.StopReason})
			}
			fmt.Fprintf(w, "data: [DONE]\n\n")
			flusher.Flush()
			return
		case <-r.Context().Done():
			return
		}
	}
}

func handleAgentCancel(w http.ResponseWriter, r *http.Request, agent Agent) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		SessionID string `json:"sessionId"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	if err := agent.Cancel(body.SessionID); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Cancelled"})
}

func writeSSE(w http.ResponseWriter, flusher http.Flusher, update SessionUpdate) {
	data, err := json.Marshal(update)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}

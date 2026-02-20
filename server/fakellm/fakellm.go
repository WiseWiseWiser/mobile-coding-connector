package fakellm

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

var (
	mu           sync.Mutex
	activeStream *StreamSession
)

type ProcessRequest struct {
	Prompt      string `json:"prompt"`
	ProjectPath string `json:"project_path"`
}

type ProcessResponse struct {
	Output   string `json:"output"`
	Status   string `json:"status"`
	Duration int    `json:"duration_ms"`
}

type StreamEvent struct {
	Type      string `json:"type"`
	Step      string `json:"step,omitempty"`
	Message   string `json:"message,omitempty"`
	Progress  int    `json:"progress,omitempty"`
	Output    string `json:"output,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

type StreamSession struct {
	ID        string
	Events    []StreamEvent
	Done      bool
	CreatedAt time.Time
}

var stepResponses = []struct {
	step    string
	message string
}{
	{"understanding", "Parsing feature request and identifying key requirements..."},
	{"understanding", "Found 6 key requirements: authentication, CRUD, upvoting, WebSocket, push notifications, admin dashboard"},
	{"clarifying", "Consulting Architect Agent for design decisions..."},
	{"clarifying", "Database: PostgreSQL with Prisma ORM selected"},
	{"clarifying", "API: RESTful with JSON response format"},
	{"clarifying", "Real-time: Socket.io for WebSocket communication"},
	{"implementing", "Delegating to Coder Agent..."},
	{"implementing", "Creating database schema for feature requests..."},
	{"implementing", "Implementing REST API endpoints..."},
	{"implementing", "Adding WebSocket integration..."},
	{"implementing", "Building frontend components..."},
	{"verifying", "Running unit tests..."},
	{"verifying", "Running integration tests..."},
	{"verifying", "Checking code coverage: 87%"},
	{"verifying", "All checks passed!"},
}

func RegisterAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/fakellm/process", handleProcess)
	mux.HandleFunc("/api/fakellm/stream", handleStream)
	mux.HandleFunc("/api/fakellm/stop", handleStop)
	mux.HandleFunc("/api/fakellm/status", handleStatus)
}

func handleProcess(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ProcessRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	time.Sleep(time.Duration(rand.Intn(300)+100) * time.Millisecond)

	responses := []string{
		"Processing complete. Generated implementation plan with 4 steps.",
		"Analysis finished. Identified 12 files to modify.",
		"Task completed successfully. Ready to proceed with implementation.",
		"Feature request parsed. Creating architecture proposal...",
	}

	resp := ProcessResponse{
		Output:   responses[rand.Intn(len(responses))],
		Status:   "completed",
		Duration: rand.Intn(500) + 100,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ProcessRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	sessionID := randomID()
	session := &StreamSession{
		ID:        sessionID,
		Events:    []StreamEvent{},
		CreatedAt: time.Now(),
	}

	mu.Lock()
	activeStream = session
	mu.Unlock()

	sendEvent := func(event StreamEvent) {
		event.Timestamp = time.Now().UnixMilli()
		data, _ := json.Marshal(event)
		w.Write([]byte("data: "))
		w.Write(data)
		w.Write([]byte("\n\n"))
		flusher.Flush()
	}

	sendEvent(StreamEvent{
		Type:    "start",
		Message: "Starting driver agent...",
	})

	for i, step := range stepResponses {
		mu.Lock()
		if activeStream == nil || activeStream.ID != sessionID {
			mu.Unlock()
			sendEvent(StreamEvent{
				Type:    "aborted",
				Message: "Stream stopped by user",
			})
			return
		}
		mu.Unlock()

		progress := ((i + 1) * 100) / len(stepResponses)
		sendEvent(StreamEvent{
			Type:     "step",
			Step:     step.step,
			Message:  step.message,
			Progress: progress,
		})

		time.Sleep(time.Duration(rand.Intn(400)+200) * time.Millisecond)
	}

	sendEvent(StreamEvent{
		Type:    "done",
		Message: "Feature implementation complete!",
		Output:  "Successfully implemented: authentication, CRUD operations, upvoting system, WebSocket real-time updates, push notifications, and admin dashboard.",
	})

	mu.Lock()
	if activeStream != nil && activeStream.ID == sessionID {
		activeStream.Done = true
	}
	mu.Unlock()
}

func handleStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	mu.Lock()
	activeStream = nil
	mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "stopped"})
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	mu.Lock()
	defer mu.Unlock()

	if activeStream != nil && !activeStream.Done {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "running",
			"sessionID": activeStream.ID,
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
}

func randomID() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 16)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

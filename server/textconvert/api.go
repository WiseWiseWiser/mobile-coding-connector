package textconvert

import (
	"encoding/json"
	"net/http"
	"strings"
)

const (
	// ConvertPath is POST for converting adhoc editor text.
	ConvertPath = "/api/local/text/convert"
)

// ConvertRequest is the POST ConvertPath body.
type ConvertRequest struct {
	Text      string `json:"text"`
	Converter string `json:"converter,omitempty"`
}

// ConvertResponse is the POST ConvertPath body.
type ConvertResponse struct {
	Text      string `json:"text"`
	Converter string `json:"converter"`
}

// Handler serves text conversion endpoints.
type Handler struct{}

// Register mounts ConvertPath on mux.
func Register(mux *http.ServeMux, h *Handler) {
	if h == nil {
		h = &Handler{}
	}
	mux.HandleFunc(ConvertPath, h.handleConvert)
}

func (h *Handler) handleConvert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req ConvertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	id := strings.TrimSpace(req.Converter)
	if id == "" {
		id = DefaultConverter
	}
	out, err := Convert(req.Text, id)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, ConvertResponse{Text: out, Converter: id})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"error": msg})
}

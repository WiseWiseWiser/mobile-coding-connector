package cursor

import (
	_ "embed"
	"encoding/json"
	"net/http"
)

//go:embed WHATS_NEXT.md
var whatsNextTemplate string

// Template represents a predefined message template.
type Template struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Content string `json:"content"`
}

// builtinTemplates returns all available templates.
func builtinTemplates() []Template {
	return []Template{
		{
			ID:      "whats_next",
			Name:    "Follow-up with whats_next",
			Content: whatsNextTemplate,
		},
	}
}

func (a *Adapter) handleListTemplates(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(builtinTemplates())
}

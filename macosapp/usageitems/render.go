package usageitems

import (
	"strings"

	"github.com/xhd2015/ai-critic/macosapp/menubar"
)

// MaxLabelLen caps the menu-bar title length in runes.
const MaxLabelLen = 40

// Item fetch states used in rendered views.
const (
	StatusLoading = "loading"
	StatusReady   = "ready"
	StatusError   = "error"
)

// ItemView is one registry item plus its rendered menu text.
type ItemView struct {
	Item
	Status    string `json:"status"`
	Title     string `json:"title"`
	Dropdown  string `json:"dropdown"`
	Detail    string `json:"detail,omitempty"`
	UsageURL  string `json:"usage_url,omitempty"`
	Error     string `json:"error,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

// ListResponse is the GET /api/usage/items payload: every item with rendered
// text, plus the menu-bar selection state.
type ListResponse struct {
	Version int        `json:"version"`
	Default string     `json:"default"`
	Rotate  bool       `json:"rotate"`
	Items   []ItemView `json:"items"`
}

// FormatTitle renders the menu-bar title for one item.
func FormatTitle(label, status, percent, errorMsg string) string {
	var title string
	switch status {
	case StatusReady:
		if strings.TrimSpace(percent) == "" {
			title = label
		} else {
			title = label + " " + percent
		}
	case StatusError:
		title = label + " err"
	default:
		title = label + " ..."
	}
	return menubar.TruncateRunes(title, MaxLabelLen)
}

// FormatDropdown renders the menu dropdown line for one item from a body that
// already excludes the label prefix.
func FormatDropdown(label, status, body, errorMsg string) string {
	switch status {
	case StatusReady:
		return label + ": " + body
	case StatusError:
		return label + ": Error: " + errorMsg
	default:
		return label + ": Loading..."
	}
}

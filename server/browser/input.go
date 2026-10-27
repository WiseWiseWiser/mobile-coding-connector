package browser

import (
	"encoding/base64"
	"fmt"
	"math"
	"strings"
	"unicode/utf8"
)

// Input kinds accepted from the viewer.
const (
	InputTap   = "tap"
	InputDown  = "down"
	InputMove  = "move"
	InputUp    = "up"
	InputWheel = "wheel"
	InputKey   = "key"
	InputText  = "text"
)

// Pointer buttons, matching the screen protocol so the two pages feel alike.
const (
	ButtonNone   = 0
	ButtonLeft   = 1
	ButtonMiddle = 2
	ButtonRight  = 3
)

// CDP modifier bits.
const (
	ModAlt   = 1
	ModCtrl  = 2
	ModMeta  = 4
	ModShift = 8
)

// InputEvent is one viewer action. X and Y are normalized to [0,1] so the
// client never has to know the CSS viewport size the window ended up with.
type InputEvent struct {
	Kind      string  `json:"type"`
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
}

func buttonName(button int) string {
	switch button {
	case ButtonMiddle:
		return "middle"
	case ButtonRight:
		return "right"
	default:
		return "left"
	}
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

// virtualKey maps the key names the on-screen keyboard sends onto Windows
// virtual key codes, which is what CDP wants for non-text keys.
func virtualKey(key string) int {
	if len(key) == 1 {
		r, _ := utf8.DecodeRuneInString(strings.ToUpper(key))
		return int(r)
	}
	switch key {
	case "Enter":
		return 13
	case "Tab":
		return 9
	case "Backspace":
		return 8
	case "Escape", "Esc":
		return 27
	case "Space", " ":
		return 32
	case "ArrowLeft":
		return 37
	case "ArrowUp":
		return 38
	case "ArrowRight":
		return 39
	case "ArrowDown":
		return 40
	case "Delete":
		return 46
	case "Home":
		return 36
	case "End":
		return 35
	case "PageUp":
		return 33
	case "PageDown":
		return 34
	}
	return 0
}

// keyText returns the text a key press should insert, or "" when the key only
// has a control meaning.
func keyText(key string, modifiers int) string {
	if modifiers != 0 && modifiers != ModShift {
		return ""
	}
	switch key {
	case "Enter":
		return "\r"
	case "Tab":
		return "\t"
	case "Space", " ":
		return " "
	}
	if utf8.RuneCountInString(key) == 1 {
		return key
	}
	return ""
}

func cloneParams(in map[string]any) map[string]any {
	out := make(map[string]any, len(in)+2)
	for k, v := range in {
		out[k] = v
	}
	return out
}

// decodeBase64 accepts both bare and data-URL base64, which is how the extension
// hands back a screenshot.
func decodeBase64(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if i := strings.Index(s, "base64,"); i >= 0 {
		s = s[i+len("base64,"):]
	}
	if s == "" {
		return nil, nil
	}
	raw, err := base64.StdEncoding.DecodeString(s)
	if err == nil {
		return raw, nil
	}
	raw, err = base64.RawStdEncoding.DecodeString(strings.TrimRight(s, "="))
	if err != nil {
		return nil, fmt.Errorf("invalid base64: %w", err)
	}
	return raw, nil
}

package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/xhd2015/ai-critic/server/localiterm2"
)

// ITermInventory is GET /api/local/iterm2/inventory (and stream last frame).
type ITermInventory = localiterm2.Inventory

// GetITermInventory returns the current iTerm inventory.
// When refresh is true, forces a full recapture via ?refresh=1.
func (c *Client) GetITermInventory(refresh bool) (*ITermInventory, error) {
	path := "/api/local/iterm2/inventory"
	if refresh {
		path += "?refresh=1"
	}
	var out ITermInventory
	if err := c.getJSON(path, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// StreamITermInventory GETs /api/local/iterm2/inventory/stream and returns
// the last inventory frame (cold seed+capture, or warm last-good+increment).
func (c *Client) StreamITermInventory() (*ITermInventory, error) {
	var last ITermInventory
	var have bool
	_, err := c.ConsumeStream(http.MethodGet, "/api/local/iterm2/inventory/stream", nil, func(ev StreamEvent, raw map[string]any) error {
		if ev.Type == "error" {
			msg := ev.Message
			if msg == "" {
				msg = "inventory stream failed"
			}
			return errors.New(msg)
		}
		if ev.Type != "inventory" {
			return nil
		}
		rawInv, ok := raw["inventory"]
		if !ok || rawInv == nil {
			return nil
		}
		b, err := json.Marshal(rawInv)
		if err != nil {
			return err
		}
		var inv ITermInventory
		if err := json.Unmarshal(b, &inv); err != nil {
			return fmt.Errorf("decode inventory frame: %w", err)
		}
		last = inv
		have = true
		return nil
	})
	if err != nil {
		return nil, err
	}
	if !have {
		return nil, fmt.Errorf("inventory stream produced no inventory")
	}
	return &last, nil
}

// ITermFocusResult is POST /api/local/iterm2/focus.
type ITermFocusResult struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

// FocusITermSession POSTs /api/local/iterm2/focus for the given session id.
func (c *Client) FocusITermSession(sessionID string) (*ITermFocusResult, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, fmt.Errorf("session_id is required")
	}
	var out ITermFocusResult
	if err := c.postJSON("/api/local/iterm2/focus", map[string]string{"session_id": sessionID}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// SwitchITermSpace POSTs /api/local/iterm2/switch-space for a CGS space id.
func (c *Client) SwitchITermSpace(spaceID uint64) error {
	if spaceID == 0 {
		return fmt.Errorf("space_id is required")
	}
	var out ITermFocusResult
	if err := c.postJSON("/api/local/iterm2/switch-space", map[string]any{"space_id": spaceID}, &out); err != nil {
		return err
	}
	return nil
}

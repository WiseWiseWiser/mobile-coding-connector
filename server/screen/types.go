// Package screen serves a live, interactive mirror of the host desktop over a
// WebSocket: it captures the display without the pointer, streams only changed
// 128px tiles, broadcasts the real pointer position so the browser can draw a
// smooth synthetic cursor, and injects remote mouse/keyboard input back into the
// host.
//
// Capture and injection are split behind small interfaces (source, backend) so
// the wire protocol, tiling and control-lock logic stay platform independent and
// testable without a display.
package screen

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/xhd2015/ai-critic/server/config"
)

// Stream states reported in Status.
const (
	StateStopped   = "stopped"
	StateStarting  = "starting"
	StateStreaming = "streaming"
	StateError     = "error"
)

// Control states reported per viewer.
const (
	ControlView   = "view"
	ControlActive = "active"
	ControlLocked = "locked"
)

// TileSize is the fixed edge of one dirty-rect tile in image pixels.
const TileSize = 128

// Defaults for Config.
const (
	defaultMaxWidth   = 1280
	defaultFPS        = 12
	defaultIdleFPS    = 3
	defaultQuality    = 55
	defaultMaxViewers = 4
	activeWindow      = 2 * time.Second
)

// Config is the persisted screen settings (screen-config.json).
type Config struct {
	Enabled      bool `json:"enabled"`
	AllowControl bool `json:"allow_control"`
	MaxWidth     int  `json:"max_width"`
	FPS          int  `json:"fps"`
	IdleFPS      int  `json:"idle_fps"`
	Quality      int  `json:"quality"`
	MaxViewers   int  `json:"max_viewers"`
}

// DefaultConfig returns the built-in settings.
func DefaultConfig() Config {
	return Config{
		Enabled:      true,
		AllowControl: true,
		MaxWidth:     defaultMaxWidth,
		FPS:          defaultFPS,
		IdleFPS:      defaultIdleFPS,
		Quality:      defaultQuality,
		MaxViewers:   defaultMaxViewers,
	}
}

// normalize clamps persisted values into a range the capture loop can use.
func (c *Config) normalize() {
	if c.MaxWidth < 320 {
		c.MaxWidth = defaultMaxWidth
	}
	if c.MaxWidth > 2560 {
		c.MaxWidth = 2560
	}
	if c.FPS <= 0 || c.FPS > 30 {
		c.FPS = defaultFPS
	}
	if c.IdleFPS <= 0 || c.IdleFPS > c.FPS {
		c.IdleFPS = defaultIdleFPS
	}
	if c.Quality <= 0 || c.Quality > 90 {
		c.Quality = defaultQuality
	}
	if c.MaxViewers <= 0 {
		c.MaxViewers = defaultMaxViewers
	}
}

// LoadConfig reads the screen config, returning defaults when absent.
func LoadConfig() (*Config, error) {
	cfg := DefaultConfig()
	data, err := os.ReadFile(config.ScreenConfFile)
	if err != nil {
		if os.IsNotExist(err) {
			return &cfg, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	cfg.normalize()
	return &cfg, nil
}

// SaveConfig writes the screen config, creating the data directory on first use
// so settings changed from the UI work on a fresh install.
func SaveConfig(cfg *Config) error {
	cfg.normalize()
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if dir := filepath.Dir(config.ScreenConfFile); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return os.WriteFile(config.ScreenConfFile, data, 0644)
}

// Display describes one capturable desktop in host coordinates. W/H are in the
// host's native unit: points on macOS, pixels on X11.
type Display struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	X       int     `json:"x"`
	Y       int     `json:"y"`
	W       int     `json:"w"`
	H       int     `json:"h"`
	Scale   float64 `json:"scale"`
	Primary bool    `json:"primary"`
}

// CaptureInfo describes what the encoder is producing.
type CaptureInfo struct {
	Tool     string `json:"tool"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	FPS      int    `json:"fps"`
	IdleFPS  int    `json:"idle_fps"`
	TileSize int    `json:"tile_size"`
	// Decoded and Published separate "the source stopped sending" from "the
	// screen stopped changing" when a remote viewer reports a frozen picture.
	Decoded   uint64 `json:"decoded"`
	Published uint64 `json:"published"`
}

// ControlInfo describes who currently owns input.
type ControlInfo struct {
	Enabled bool   `json:"enabled"`
	Holder  string `json:"holder"`
	Since   int64  `json:"since"`
}

// InputInfo describes whether input injection is usable right now.
type InputInfo struct {
	Available  bool   `json:"available"`
	Reason     string `json:"reason,omitempty"`
	Permission string `json:"permission,omitempty"`
	CanPrompt  bool   `json:"can_prompt"`
	Backend    string `json:"backend"`
	// Locked is set while the host login window owns the display. Input is
	// refused then: injected keys would go to the unlock prompt.
	Locked bool `json:"locked,omitempty"`
	// AXTrusted and PostEventAccess report the two macOS trust checks
	// separately. Available is their disjunction; a denial is far easier to
	// diagnose when the status says which half the OS refused.
	AXTrusted       bool `json:"ax_trusted"`
	PostEventAccess bool `json:"post_event_access"`
}

// Status is the JSON document returned by GET /api/screen/status and pushed on
// the WebSocket as a text message.
// Session reports one connected viewer. Frames and Dropped make transport
// congestion visible from the server; Keyframe marks a viewer that still owes a
// repaint (a fresh join or a dropped frame).
type Session struct {
	ID       string `json:"id"`
	Label    string `json:"label,omitempty"`
	Frames   uint64 `json:"frames"`
	Dropped  uint64 `json:"dropped"`
	Keyframe bool   `json:"keyframe"`
}

type Status struct {
	State    string      `json:"state"`
	Error    string      `json:"error,omitempty"`
	Platform string      `json:"platform"`
	Displays []Display   `json:"displays"`
	Capture  CaptureInfo `json:"capture"`
	Control  ControlInfo `json:"control"`
	Input    InputInfo   `json:"input"`
	Viewers  int         `json:"viewers"`
	Sessions []Session   `json:"sessions,omitempty"`
}

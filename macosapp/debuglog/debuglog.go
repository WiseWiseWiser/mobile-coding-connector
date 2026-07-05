package debuglog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	agentdebug "github.com/xhd2015/agent-pro/agent/debuglog"
)

const (
	// LogPath is the append-only JSONL debug log file.
	LogPath = "/tmp/debug-ai-critic.log"

	settingsFileName = "debug-settings.json"
)

type settingsFile struct {
	DebugLogEnabled bool `json:"debug_log_enabled"`
}

var (
	enabled atomic.Bool
	fileMu  sync.Mutex
)

// Init loads persisted settings and registers the agent-pro debug bridge.
func Init() error {
	enabled.Store(false)
	if err := loadSettings(); err != nil {
		return err
	}
	RegisterAgentProBridge()
	return nil
}

// RegisterAgentProBridge wires agent-pro debug hooks to this package.
func RegisterAgentProBridge() {
	agentdebug.SetLogger(func(e agentdebug.Entry) {
		Write(Entry{
			Event:  e.Event,
			Labels: e.Labels,
			Fields: e.Fields,
		})
	})
}

// Entry is a JSONL debug record written when logging is enabled.
type Entry struct {
	Event  string            `json:"event"`
	Labels map[string]string `json:"labels"`
	Fields map[string]any    `json:"fields,omitempty"`
}

// Enabled reports whether debug logging is active (cached, dynamic).
func Enabled() bool {
	return enabled.Load()
}

// Path returns the debug log file path.
func Path() string {
	return LogPath
}

// GetSettings returns current debug settings for API/UI.
func GetSettings() (bool, string) {
	return enabled.Load(), LogPath
}

// SetEnabled updates the cached flag and persists to disk.
func SetEnabled(on bool) error {
	enabled.Store(on)
	return saveSettings()
}

func loadSettings() error {
	path, err := settingsPath()
	if err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var s settingsFile
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	enabled.Store(s.DebugLogEnabled)
	return nil
}

func saveSettings() error {
	path, err := settingsPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(settingsFile{DebugLogEnabled: enabled.Load()})
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func settingsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".ai-critic", settingsFileName), nil
}

// Write appends one JSONL record when logging is enabled.
func Write(e Entry) {
	if !enabled.Load() {
		return
	}
	if e.Labels == nil {
		e.Labels = map[string]string{}
	}
	line := map[string]any{
		"ts":     time.Now().UTC().Format(time.RFC3339Nano),
		"event":  e.Event,
		"labels": e.Labels,
	}
	if len(e.Fields) > 0 {
		line["fields"] = e.Fields
	}
	payload, err := json.Marshal(line)
	if err != nil {
		return
	}
	payload = append(payload, '\n')

	fileMu.Lock()
	defer fileMu.Unlock()
	f, err := os.OpenFile(LogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.Write(payload)
}
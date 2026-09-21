package gomodrelay

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Paths under $HOME/.ai-critic/go/.
const (
	DirName        = "go"
	ConfigFileName = "mod-proxy-relay.json"
	LogFileName    = "mod-proxy-relay.log"
	SocketFileName = "mod-proxy-relay.sock"
)

// BaseDir returns the directory holding relay config/log/socket.
func BaseDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".ai-critic", DirName), nil
}

// ConfigPath / LogPath / SocketPath resolve under BaseDir (dir override for tests).
func ConfigPath(base string) string { return filepath.Join(base, ConfigFileName) }
func LogPath(base string) string    { return filepath.Join(base, LogFileName) }
func SocketPath(base string) string { return filepath.Join(base, SocketFileName) }

// LoadConfig reads the config file; missing file yields a zero config.
func LoadConfig(path string) (Config, error) {
	var cfg Config
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// SaveConfig atomically writes the config file.
func SaveConfig(path string, cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// logWriter is a tiny append-only logger safe for concurrent use.
type logWriter struct {
	mu   sync.Mutex
	path string
}

func newLogWriter(path string) *logWriter {
	_ = os.MkdirAll(filepath.Dir(path), 0o700)
	return &logWriter{path: path}
}

func (l *logWriter) logf(format string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	f, err := os.OpenFile(l.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, time.Now().Format("2006-01-02 15:04:05")+" "+format+"\n", args...)
}

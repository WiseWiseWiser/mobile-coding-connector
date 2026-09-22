package cloudflareproxy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Mapping is a persisted hostname publication.
type Mapping struct {
	ID        string    `json:"id"`
	Hostname  string    `json:"hostname"`
	DialToken string    `json:"dial_token"`
	CreatedAt time.Time `json:"created_at"`
}

// MappingView is the public list/add payload (no dial token).
type MappingView struct {
	ID       string `json:"id"`
	Hostname string `json:"hostname"`
	WS       int    `json:"ws"`
	URL      string `json:"url"`
	DialURL  string `json:"dial_url,omitempty"`
}

type liveMapping struct {
	Mapping
	mu   sync.Mutex
	idle []*websocket.Conn
}

func (m *liveMapping) wsCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.idle)
}

func (m *liveMapping) take() *websocket.Conn {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := len(m.idle)
	if n == 0 {
		return nil
	}
	c := m.idle[n-1]
	m.idle = m.idle[:n-1]
	return c
}

func (m *liveMapping) put(c *websocket.Conn) {
	if c == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.idle = append(m.idle, c)
}

func (m *liveMapping) closeAll() {
	m.mu.Lock()
	conns := m.idle
	m.idle = nil
	m.mu.Unlock()
	for _, c := range conns {
		_ = c.Close()
	}
}

func normalizeHostname(h string) string {
	h = strings.ToLower(strings.TrimSpace(h))
	h = strings.TrimSuffix(h, ".")
	if i := strings.IndexByte(h, ':'); i >= 0 {
		h = h[:i]
	}
	return h
}

func publicURL(hostname string) string {
	if hostname == "" {
		return ""
	}
	return "https://" + hostname
}

func loadMappingsFile(path string) ([]Mapping, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var list []Mapping
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, err
	}
	return list, nil
}

func saveMappingsFile(path string, list []Mapping) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

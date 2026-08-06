// Package synccmd implements remote-agent sync unison pair store, CRUD, profile
// emit, and in-process CLI (P1) — without real Unison, SSH, or network.
package synccmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Pair is one named local↔remote Unison sync definition.
type Pair struct {
	Name          string   `json:"name"`
	Backend       string   `json:"backend"`
	Local         string   `json:"local"`
	Remote        string   `json:"remote"`
	Prefer        string   `json:"prefer"`
	Ignore        []string `json:"ignore"`
	LocalHostname string   `json:"localHostname"`
	RemoteUnison  string   `json:"remoteUnison"`
	Times         bool     `json:"times"`
	Auto          bool     `json:"auto"`
	Batch         bool     `json:"batch"`
}

// Config is the versioned pairs.json document.
type Config struct {
	Version     int    `json:"version"`
	DefaultPair string `json:"defaultPair"`
	Pairs       []Pair `json:"pairs"`
}

// Store loads/saves {Dir}/pairs.json.
type Store struct {
	Dir string
}

// pairsPath returns the absolute path to pairs.json.
func (s *Store) pairsPath() string {
	return filepath.Join(s.Dir, "pairs.json")
}

// Load reads pairs.json. Missing file → empty Config version 1, nil error.
func (s *Store) Load() (*Config, error) {
	path := s.pairsPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{Version: 1, Pairs: []Pair{}}, nil
		}
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("load pairs.json: %w", err)
	}
	if cfg.Pairs == nil {
		cfg.Pairs = []Pair{}
	}
	if cfg.Version == 0 {
		cfg.Version = 1
	}
	return &cfg, nil
}

// Save writes pairs.json (creates Dir if needed).
func (s *Store) Save(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	if err := os.MkdirAll(s.Dir, 0o755); err != nil {
		return err
	}
	if cfg.Version == 0 {
		cfg.Version = 1
	}
	if cfg.Pairs == nil {
		cfg.Pairs = []Pair{}
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(s.pairsPath(), data, 0o644)
}

// GetPair returns a copy of the named pair or an error containing "unknown pair".
func GetPair(store *Store, name string) (*Pair, error) {
	cfg, err := store.Load()
	if err != nil {
		return nil, err
	}
	for i := range cfg.Pairs {
		if cfg.Pairs[i].Name == name {
			p := cfg.Pairs[i]
			return &p, nil
		}
	}
	return nil, fmt.Errorf("unknown pair: %s", name)
}

// ListPairs returns all pairs (empty slice if none).
func ListPairs(store *Store) ([]Pair, error) {
	cfg, err := store.Load()
	if err != nil {
		return nil, err
	}
	out := make([]Pair, len(cfg.Pairs))
	copy(out, cfg.Pairs)
	return out, nil
}

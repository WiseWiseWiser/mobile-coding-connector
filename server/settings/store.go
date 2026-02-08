package settings

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Store is a file-backed settings store.
// Each namespace gets its own JSON file in the base directory.
type Store struct {
	mu      sync.Mutex
	baseDir string
}

// NewStore creates a new settings store at the given directory.
// The directory is created if it does not exist.
func NewStore(baseDir string) (*Store, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("create settings directory: %w", err)
	}
	return &Store{baseDir: baseDir}, nil
}

// filePath returns the path to the JSON file for the given namespace.
func (s *Store) filePath(namespace string) string {
	return filepath.Join(s.baseDir, namespace+".json")
}

// Load reads settings for the given namespace into the target struct.
// If the file does not exist, target is left unchanged (zero value).
func (s *Store) Load(namespace string, target interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.filePath(namespace))
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No settings file yet, use defaults
		}
		return fmt.Errorf("read settings %s: %w", namespace, err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("parse settings %s: %w", namespace, err)
	}
	return nil
}

// Save writes the given value as JSON to the namespace file.
func (s *Store) Save(namespace string, value interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal settings %s: %w", namespace, err)
	}
	if err := os.WriteFile(s.filePath(namespace), data, 0644); err != nil {
		return fmt.Errorf("write settings %s: %w", namespace, err)
	}
	return nil
}

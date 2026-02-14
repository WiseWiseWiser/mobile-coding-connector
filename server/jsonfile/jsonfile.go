package jsonfile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type JSONFile[T any] struct {
	filePath string
	mu       sync.RWMutex
	data     *T
	loaded   bool
}

func New[T any](filePath string) *JSONFile[T] {
	return &JSONFile[T]{
		filePath: filePath,
	}
}

func (j *JSONFile[T]) GetPath() string {
	return j.filePath
}

func (j *JSONFile[T]) Load() error {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.loadLocked()
}

func (j *JSONFile[T]) loadLocked() error {
	data, err := os.ReadFile(j.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Create directory if needed
			dir := filepath.Dir(j.filePath)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
			// Use zero value if file doesn't exist
			var zero T
			j.data = &zero
			j.loaded = true
			return nil
		}
		return fmt.Errorf("failed to read file: %w", err)
	}

	var dataT T
	if err := json.Unmarshal(data, &dataT); err != nil {
		return fmt.Errorf("failed to unmarshal: %w", err)
	}

	j.data = &dataT
	j.loaded = true
	return nil
}

func (j *JSONFile[T]) Get() (T, error) {
	j.mu.RLock()
	defer j.mu.RUnlock()

	if !j.loaded {
		if err := j.loadLocked(); err != nil {
			var zero T
			return zero, err
		}
	}

	if j.data == nil {
		var zero T
		return zero, fmt.Errorf("data not loaded")
	}

	return *j.data, nil
}

func (j *JSONFile[T]) MustGet() T {
	val, err := j.Get()
	if err != nil {
		panic(fmt.Sprintf("jsonfile: MustGet failed: %v", err))
	}
	return val
}

func (j *JSONFile[T]) Update(fn func(*T) error) error {
	j.mu.Lock()
	defer j.mu.Unlock()

	if !j.loaded {
		if err := j.loadLocked(); err != nil {
			return err
		}
	}

	if err := fn(j.data); err != nil {
		return err
	}

	return j.saveLocked()
}

func (j *JSONFile[T]) Set(data T) error {
	j.mu.Lock()
	defer j.mu.Unlock()

	j.data = &data
	j.loaded = true

	return j.saveLocked()
}

func (j *JSONFile[T]) saveLocked() error {
	// Ensure directory exists
	dir := filepath.Dir(j.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := json.MarshalIndent(j.data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal: %w", err)
	}

	if err := os.WriteFile(j.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func (j *JSONFile[T]) Save() error {
	j.mu.Lock()
	defer j.mu.Unlock()

	if !j.loaded {
		return fmt.Errorf("data not loaded")
	}

	return j.saveLocked()
}

func (j *JSONFile[T]) Exists() bool {
	j.mu.RLock()
	_, err := os.Stat(j.filePath)
	j.mu.RUnlock()
	return err == nil
}

package authproxy

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func LoadOrCreateSecret(path string) ([]byte, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("secret path is required")
	}
	if data, err := os.ReadFile(path); err == nil {
		if secret, ok := decodeSecretBytes(data); ok {
			return secret, nil
		}
	}
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, fmt.Errorf("generate auth secret: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, []byte(hex.EncodeToString(secret)+"\n"), 0600); err != nil {
		return nil, err
	}
	return secret, nil
}

func decodeSecretBytes(data []byte) ([]byte, bool) {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return nil, false
	}
	if decoded, err := hex.DecodeString(trimmed); err == nil && len(decoded) >= 32 {
		return decoded[:32], true
	}
	if len(data) >= 32 {
		return data[:32], true
	}
	return nil, false
}

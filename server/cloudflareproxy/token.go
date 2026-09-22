package cloudflareproxy

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

func randomHex(nBytes int) (string, error) {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func newMappingID() (string, error) {
	s, err := randomHex(4)
	if err != nil {
		return "", err
	}
	return "map-" + s, nil
}

func newSecret() (string, error) {
	return randomHex(16)
}

func newGeneratedToken() (string, error) {
	s, err := newSecret()
	if err != nil {
		return "", err
	}
	return s, nil
}

// prepareAuth mutates cfg for a start: rotate generatedToken when token is
// empty; clear generatedToken when token is set. rotated is true when a new
// generatedToken was written.
func prepareAuth(cfg *Config) (rotated bool, err error) {
	if trim(cfg.Token) != "" {
		if cfg.GeneratedToken != "" {
			cfg.GeneratedToken = ""
		}
		return false, nil
	}
	tok, err := newGeneratedToken()
	if err != nil {
		return false, fmt.Errorf("generate token: %w", err)
	}
	cfg.GeneratedToken = tok
	return true, nil
}

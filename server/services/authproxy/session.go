package authproxy

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type session struct {
	User    string `json:"user"`
	TokenFP string `json:"fp"`
	Expires int64  `json:"exp"`
}

func tokenFingerprint(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:16])
}

func encodeSession(secret []byte, s session) (string, error) {
	raw, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write(raw)
	return base64.RawURLEncoding.EncodeToString(raw) + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

func decodeSession(secret []byte, value string) (*session, error) {
	parts := strings.Split(value, ".")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid session")
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid session")
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid session")
	}
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write(raw)
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return nil, fmt.Errorf("invalid session")
	}
	var s session
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, fmt.Errorf("invalid session")
	}
	return &s, nil
}

func (s *session) expiredAt(now time.Time) bool {
	if s == nil || s.Expires <= 0 {
		return true
	}
	return now.Unix() >= s.Expires
}

package authproxy

import (
	"context"
	"crypto/subtle"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/xhd2015/ai-critic/server/auth"
)

const (
	TokenModeShared = "shared"
	TokenModeCustom = "custom"

	defaultCookieName = "svc-auth-token"
	defaultTTL        = 7 * 24 * time.Hour
)

type Config struct {
	BackendPort  int
	BackendURL   string
	AuthUser     string
	TokenMode    string
	CustomToken  string
	Secret       []byte
	SecretPath   string
	CookieName   string
	TTL          time.Duration
	Now          func() time.Time
	LookupShared func() ([]string, error)
}

type Server struct {
	cfg    Config
	ln     net.Listener
	srv    *http.Server
	port   int
	closed atomic.Bool
}

func Start(cfg Config) (*Server, error) {
	cfg = normalizeConfig(cfg)
	if len(cfg.Secret) < 32 {
		if strings.TrimSpace(cfg.SecretPath) == "" {
			return nil, fmt.Errorf("auth proxy secret is required")
		}
		secret, err := LoadOrCreateSecret(cfg.SecretPath)
		if err != nil {
			return nil, err
		}
		cfg.Secret = secret
	} else {
		cfg.Secret = cfg.Secret[:32]
	}

	backend, err := cfg.backendURL()
	if err != nil {
		return nil, err
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen auth proxy: %w", err)
	}

	handler, err := newHandler(cfg, backend)
	if err != nil {
		ln.Close()
		return nil, err
	}

	s := &Server{
		cfg:  cfg,
		ln:   ln,
		port: ln.Addr().(*net.TCPAddr).Port,
	}
	s.srv = &http.Server{Handler: handler}
	go func() {
		_ = s.srv.Serve(ln)
		s.closed.Store(true)
	}()
	return s, nil
}

func (s *Server) Port() int {
	if s == nil {
		return 0
	}
	return s.port
}

func (s *Server) Alive() bool {
	return s != nil && !s.closed.Load()
}

func (s *Server) Matches(backendPort int, authUser, tokenMode, customToken string) bool {
	if s == nil {
		return false
	}
	return s.cfg.BackendPort == backendPort &&
		s.cfg.AuthUser == strings.TrimSpace(authUser) &&
		s.cfg.TokenMode == normalizeTokenMode(tokenMode, customToken) &&
		s.cfg.CustomToken == strings.TrimSpace(customToken)
}

func (s *Server) Close() error {
	if s == nil {
		return nil
	}
	s.closed.Store(true)
	if s.srv == nil {
		if s.ln != nil {
			return s.ln.Close()
		}
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return s.srv.Shutdown(ctx)
}

func normalizeConfig(cfg Config) Config {
	cfg.AuthUser = strings.TrimSpace(cfg.AuthUser)
	cfg.CustomToken = strings.TrimSpace(cfg.CustomToken)
	cfg.TokenMode = normalizeTokenMode(cfg.TokenMode, cfg.CustomToken)
	if cfg.TokenMode == TokenModeShared {
		cfg.CustomToken = ""
	}
	if cfg.CookieName == "" {
		cfg.CookieName = defaultCookieName
	}
	if cfg.TTL <= 0 {
		cfg.TTL = defaultTTL
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.LookupShared == nil {
		cfg.LookupShared = auth.ExportCredentials
	}
	return cfg
}

func normalizeTokenMode(mode, customToken string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case TokenModeCustom:
		return TokenModeCustom
	case TokenModeShared:
		return TokenModeShared
	default:
		if strings.TrimSpace(customToken) != "" {
			return TokenModeCustom
		}
		return TokenModeShared
	}
}

func (c Config) backendURL() (*url.URL, error) {
	if strings.TrimSpace(c.BackendURL) != "" {
		parsed, err := url.Parse(c.BackendURL)
		if err != nil {
			return nil, fmt.Errorf("backend url: %w", err)
		}
		if parsed.Scheme == "" || parsed.Host == "" {
			return nil, fmt.Errorf("backend url is invalid")
		}
		return parsed, nil
	}
	if c.BackendPort <= 0 || c.BackendPort > 65535 {
		return nil, fmt.Errorf("backend port must be between 1 and 65535")
	}
	return url.Parse(fmt.Sprintf("http://127.0.0.1:%d", c.BackendPort))
}

func (c Config) validTokens() ([]string, error) {
	if c.TokenMode == TokenModeCustom {
		if c.CustomToken == "" {
			return nil, nil
		}
		return []string{c.CustomToken}, nil
	}
	if c.LookupShared == nil {
		return nil, nil
	}
	tokens, err := c.LookupShared()
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(tokens))
	for _, token := range tokens {
		token = strings.TrimSpace(token)
		if token != "" {
			out = append(out, token)
		}
	}
	return out, nil
}

func (c Config) tokenOK(token string) bool {
	token = strings.TrimSpace(token)
	if token == "" {
		return false
	}
	tokens, err := c.validTokens()
	if err != nil {
		return false
	}
	for _, candidate := range tokens {
		if subtle.ConstantTimeCompare([]byte(candidate), []byte(token)) == 1 {
			return true
		}
	}
	return false
}

func (c Config) userOK(user string) bool {
	user = strings.TrimSpace(user)
	if user == "" {
		return false
	}
	if c.AuthUser == "" {
		return true
	}
	return user == c.AuthUser
}

func (c Config) fingerprintOK(fp string) bool {
	if fp == "" {
		return false
	}
	tokens, err := c.validTokens()
	if err != nil {
		return false
	}
	for _, token := range tokens {
		if tokenFingerprint(token) == fp {
			return true
		}
	}
	return false
}

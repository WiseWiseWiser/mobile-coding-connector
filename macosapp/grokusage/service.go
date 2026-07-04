package grokusage

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	defaultShowUsageBin = "/Users/xhd2015/go/bin/debug-grok-show-usage"
	envShowUsageBin     = "GROK_SHOW_USAGE_BIN"
	refreshInterval     = 60 * time.Second
)

// GrokUsageStatus is the fetch/cache state exposed to API clients.
type GrokUsageStatus string

const (
	StatusLoading GrokUsageStatus = "loading"
	StatusReady   GrokUsageStatus = "ready"
	StatusError   GrokUsageStatus = "error"
)

// GrokUsageResponse is the JSON shape for GET /api/grok/usage.
type GrokUsageResponse struct {
	Status      GrokUsageStatus `json:"status"`
	WeeklyLimit string          `json:"weekly_limit,omitempty"`
	NextReset   string          `json:"next_reset,omitempty"`
	Error       string          `json:"error,omitempty"`
	UpdatedAt   string          `json:"updated_at,omitempty"`
}

// Service fetches and caches grok usage on a background refresh loop.
type Service struct {
	bin      string
	extraEnv map[string]string

	mu       sync.Mutex
	fetching bool
	cached   GrokUsageResponse

	stopCh   chan struct{}
	stopOnce sync.Once
}

// NewService creates a grok usage service with the default or env-configured binary.
func NewService() *Service {
	bin := os.Getenv(envShowUsageBin)
	if bin == "" {
		bin = defaultShowUsageBin
	}
	return newService(bin)
}

func newService(bin string) *Service {
	return &Service{
		bin:      bin,
		extraEnv: make(map[string]string),
		cached: GrokUsageResponse{
			Status: StatusLoading,
		},
		stopCh: make(chan struct{}),
	}
}

// Start begins the 60s background refresh loop.
func (s *Service) Start() {
	go s.refreshLoop()
}

// Stop ends the background refresh loop.
func (s *Service) Stop() {
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
}

// Get returns the current cached response.
func (s *Service) Get() GrokUsageResponse {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cached
}

// EnsureFetch triggers a fetch when no successful refresh has completed yet.
func (s *Service) EnsureFetch() {
	s.mu.Lock()
	needsFetch := s.cached.UpdatedAt == ""
	s.mu.Unlock()
	if needsFetch {
		s.tryFetch()
	}
}

func (s *Service) refreshLoop() {
	ticker := time.NewTicker(refreshInterval)
	defer ticker.Stop()
	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.tryFetch()
		}
	}
}

func (s *Service) tryFetch() {
	s.mu.Lock()
	if s.fetching {
		s.mu.Unlock()
		return
	}
	s.fetching = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.fetching = false
		s.mu.Unlock()
	}()

	s.fetchOnce()
}

func (s *Service) fetchOnce() {
	cmd := exec.Command(s.bin)
	cmd.Env = s.buildEnv()

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	now := time.Now().UTC().Format(time.RFC3339)

	s.mu.Lock()
	defer s.mu.Unlock()

	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		s.cached = GrokUsageResponse{
			Status:    StatusError,
			Error:     msg,
			UpdatedAt: now,
		}
		return
	}

	info, parseErr := ParseShowUsageOutput(stdout.String())
	if parseErr != nil {
		s.cached = GrokUsageResponse{
			Status:    StatusError,
			Error:     parseErr.Error(),
			UpdatedAt: now,
		}
		return
	}

	s.cached = GrokUsageResponse{
		Status:      StatusReady,
		WeeklyLimit: info.WeeklyLimit,
		NextReset:   info.NextReset,
		UpdatedAt:   now,
	}
}

func (s *Service) buildEnv() []string {
	env := os.Environ()
	for key, val := range s.extraEnv {
		env = append(env, fmt.Sprintf("%s=%s", key, val))
	}
	return env
}

// TestExported_NewService creates a service that execs the given binary path.
func TestExported_NewService(bin string) *Service {
	return newService(bin)
}

// TestExported_FetchOnce performs a single synchronous fetch for doctest harness.
func (s *Service) TestExported_FetchOnce(t *testing.T) GrokUsageResponse {
	t.Helper()
	s.fetchOnce()
	return s.Get()
}

// TestExported_SetEnv sets an extra environment variable for the show-usage exec.
func (s *Service) TestExported_SetEnv(key, val string) {
	s.extraEnv[key] = val
}

// TestExported_TriggerRefresh starts an asynchronous refresh (skips if one is in flight).
func (s *Service) TestExported_TriggerRefresh() {
	go s.tryFetch()
}
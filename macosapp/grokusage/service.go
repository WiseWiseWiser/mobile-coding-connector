package grokusage

import (
	"context"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xhd2015/agent-pro/agent/grok/tty"
	"github.com/xhd2015/ai-critic/macosapp/menubar"
)

const (
	refreshInterval        = 60 * time.Second
	defaultFetchTimeoutSec = 60
	envShowUsageTimeout    = "GROK_SHOW_USAGE_TIMEOUT"
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
	Status       GrokUsageStatus `json:"status"`
	WeeklyLimit  string          `json:"weekly_limit,omitempty"`
	NextReset    string          `json:"next_reset,omitempty"`
	ResetAt      string          `json:"reset_at,omitempty"`
	ResetDisplay string          `json:"reset_display,omitempty"`
	TimeLeft     string          `json:"time_left,omitempty"`
	Error        string          `json:"error,omitempty"`
	UpdatedAt    string          `json:"updated_at,omitempty"`
}

// Service fetches and caches grok usage on a background refresh loop.
type Service struct {
	extraEnv map[string]string
	nowFunc  func() time.Time

	mu       sync.Mutex
	fetching bool
	cached   GrokUsageResponse

	stopCh   chan struct{}
	stopOnce sync.Once
}

// NewService creates a grok usage service backed by agent-pro tty fetch.
func NewService() *Service {
	return newService()
}

func newService() *Service {
	return &Service{
		extraEnv: make(map[string]string),
		cached: GrokUsageResponse{
			Status: StatusLoading,
		},
		stopCh: make(chan struct{}),
	}
}

func (s *Service) now() time.Time {
	if s.nowFunc != nil {
		return s.nowFunc()
	}
	return time.Now()
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

// Get returns the current cached response, recomputing time_left from reset_at + now.
func (s *Service) Get() GrokUsageResponse {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.cached
	if out.Status == StatusReady && out.ResetAt != "" {
		if resetAt, err := time.Parse(time.RFC3339, out.ResetAt); err == nil {
			out.TimeLeft = menubar.FormatTimeLeftFromInstant(resetAt, s.now())
		}
	}
	return out
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
	restore := s.applyExtraEnv()
	defer restore()

	timeout := fetchTimeoutFromEnv()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	info, err := tty.FetchUsageWithOptions(ctx, tty.Options{
		MaxAttempts: 1,
	})
	now := s.now()
	nowStr := now.UTC().Format(time.RFC3339)

	s.mu.Lock()
	defer s.mu.Unlock()

	if err != nil {
		s.cached = GrokUsageResponse{
			Status:    StatusError,
			Error:     strings.TrimSpace(err.Error()),
			UpdatedAt: nowStr,
		}
		return
	}

	resetAt, resetDisplay, timeLeft := menubar.ResolveStructuredReset(info.NextReset, now)
	s.cached = GrokUsageResponse{
		Status:       StatusReady,
		WeeklyLimit:  info.WeeklyLimit,
		NextReset:    info.NextReset,
		ResetAt:      resetAt,
		ResetDisplay: resetDisplay,
		TimeLeft:     timeLeft,
		UpdatedAt:    nowStr,
	}
}

func fetchTimeoutFromEnv() time.Duration {
	timeoutSec := defaultFetchTimeoutSec
	if v := strings.TrimSpace(os.Getenv(envShowUsageTimeout)); v != "" {
		if sec, err := strconv.Atoi(v); err == nil && sec > 0 {
			timeoutSec = sec
		}
	}
	return time.Duration(timeoutSec) * time.Second
}

func (s *Service) applyExtraEnv() func() {
	type saved struct {
		key string
		val string
		set bool
	}
	var savedVars []saved
	for key, val := range s.extraEnv {
		prev, had := os.LookupEnv(key)
		savedVars = append(savedVars, saved{key: key, val: prev, set: had})
		_ = os.Setenv(key, val)
	}
	return func() {
		for _, item := range savedVars {
			if item.set {
				_ = os.Setenv(item.key, item.val)
			} else {
				_ = os.Unsetenv(item.key)
			}
		}
	}
}

// TestExported_NewService creates a service for doctest harness.
func TestExported_NewService() *Service {
	return newService()
}

// TestExported_FetchOnce performs a single synchronous fetch for doctest harness.
func (s *Service) TestExported_FetchOnce(t *testing.T) GrokUsageResponse {
	t.Helper()
	s.fetchOnce()
	return s.Get()
}

// TestExported_SetEnv sets an extra environment variable for the tty fetch.
func (s *Service) TestExported_SetEnv(key, val string) {
	s.extraEnv[key] = val
}

// TestExported_TriggerRefresh starts an asynchronous refresh (skips if one is in flight).
func (s *Service) TestExported_TriggerRefresh() {
	go s.tryFetch()
}

// TestExported_SeedReady seeds a ready cache with fixed structured reset fields.
func (s *Service) TestExported_SeedReady(resetAt, resetDisplay, nextReset, weekly string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cached = GrokUsageResponse{
		Status:       StatusReady,
		WeeklyLimit:  weekly,
		NextReset:    nextReset,
		ResetAt:      resetAt,
		ResetDisplay: resetDisplay,
		UpdatedAt:    s.now().UTC().Format(time.RFC3339),
	}
}

// TestExported_SetNow injects a fixed wall clock for Get() time_left recompute.
func (s *Service) TestExported_SetNow(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	fixed := now
	s.nowFunc = func() time.Time { return fixed }
}

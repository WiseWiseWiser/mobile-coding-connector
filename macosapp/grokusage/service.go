package grokusage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xhd2015/ai-critic/macosapp/menubar"
	dotgrokusage "github.com/xhd2015/dot-pkgs/go-pkgs/shell/grok/usage"
)

const (
	refreshInterval = 10 * time.Minute
	// envUsageFixture, when set to a JSON file path, short-circuits the HTTP
	// fetch for in-process/API doctests (replaces GROK_SHOW_USAGE_COMMAND).
	envUsageFixture = "AI_CRITIC_GROK_USAGE_FIXTURE"
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
	Period       string          `json:"period,omitempty"` // "weekly" | "monthly"
	NextReset    string          `json:"next_reset,omitempty"`
	ResetAt      string          `json:"reset_at,omitempty"`
	ResetDisplay string          `json:"reset_display,omitempty"`
	TimeLeft     string          `json:"time_left,omitempty"`
	Error        string          `json:"error,omitempty"`
	UpdatedAt    string          `json:"updated_at,omitempty"`
}

// FetchResult is the normalized usage payload returned by a fetcher.
type FetchResult struct {
	WeeklyLimit string
	Period      string // "weekly" | "monthly" | ""
	NextReset   string
}

type fetchFunc func(context.Context) (*FetchResult, error)

// Service fetches and caches grok usage on a background refresh loop.
type Service struct {
	fetcher  fetchFunc
	nowFunc  func() time.Time
	extraEnv map[string]string

	mu       sync.Mutex
	fetching bool
	cached   GrokUsageResponse

	stopCh   chan struct{}
	stopOnce sync.Once
}

// extraEnvMu serializes process env apply around in-process fetch for parallel doctests.
var extraEnvMu sync.Mutex

// NewService creates a grok usage service backed by HTTP billing fetch.
func NewService() *Service {
	return newService(defaultFetcher)
}

func newService(fetcher fetchFunc) *Service {
	if fetcher == nil {
		fetcher = defaultFetcher
	}
	return &Service{
		fetcher:  fetcher,
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

func defaultFetcher(ctx context.Context) (*FetchResult, error) {
	if path := strings.TrimSpace(os.Getenv(envUsageFixture)); path != "" {
		return loadUsageFixture(path)
	}
	snap, err := dotgrokusage.Fetch(ctx, dotgrokusage.FetchOpts{})
	if err != nil {
		return nil, err
	}
	return mapBillingSnapshot(snap), nil
}

func mapBillingSnapshot(snap dotgrokusage.Snapshot) *FetchResult {
	out := &FetchResult{
		Period: strings.TrimSpace(snap.PeriodType),
	}
	switch {
	case snap.UsedPercent >= 0:
		out.WeeklyLimit = fmt.Sprintf("%d%%", snap.UsedPercent)
	default:
		// No numeric cap — surface absolute used without inventing %.
		out.WeeklyLimit = fmt.Sprintf("%d", snap.Used)
	}
	if !snap.ResetAt.IsZero() {
		// Bare local wall clock (no invented TZ); matches ResolveStructuredReset.
		out.NextReset = snap.ResetAt.In(time.Local).Format("January 2, 15:04")
	}
	return out
}

type usageFixtureFile struct {
	WeeklyLimit string `json:"weekly_limit"`
	Period      string `json:"period"`
	NextReset   string `json:"next_reset"`
	Error       string `json:"error"`
}

func loadUsageFixture(path string) (*FetchResult, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("grok usage fixture: %w", err)
	}
	var fx usageFixtureFile
	if err := json.Unmarshal(raw, &fx); err != nil {
		return nil, fmt.Errorf("grok usage fixture decode: %w", err)
	}
	if msg := strings.TrimSpace(fx.Error); msg != "" {
		return nil, fmt.Errorf("%s", msg)
	}
	return &FetchResult{
		WeeklyLimit: strings.TrimSpace(fx.WeeklyLimit),
		Period:      strings.TrimSpace(fx.Period),
		NextReset:   strings.TrimSpace(fx.NextReset),
	}, nil
}

// Start begins the 10-minute background refresh loop.
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

// FetchNow performs a synchronous fetch, skipping when one is already in flight.
func (s *Service) FetchNow() {
	s.tryFetch()
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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	info, err := s.fetcher(ctx)
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
		Period:       info.Period,
		NextReset:    info.NextReset,
		ResetAt:      resetAt,
		ResetDisplay: resetDisplay,
		TimeLeft:     timeLeft,
		UpdatedAt:    nowStr,
	}
}

func (s *Service) applyExtraEnv() func() {
	if len(s.extraEnv) == 0 {
		return func() {}
	}
	type saved struct {
		key string
		val string
		set bool
	}
	extraEnvMu.Lock()
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
		extraEnvMu.Unlock()
	}
}

// TestExported_NewService creates a service for doctest harness.
func TestExported_NewService() *Service {
	return newService(defaultFetcher)
}

// TestExported_SetFetcher replaces the default HTTP fetch for doctest harness.
func TestExported_SetFetcher(s *Service, fn fetchFunc) {
	s.fetcher = fn
}

// TestExported_FetchOnce performs a single synchronous fetch for doctest harness.
func (s *Service) TestExported_FetchOnce(t *testing.T) GrokUsageResponse {
	t.Helper()
	s.fetchOnce()
	return s.Get()
}

// TestExported_SetEnv sets an extra environment variable applied around fetch
// (e.g. AI_CRITIC_GROK_USAGE_FIXTURE for API subprocess tests).
func (s *Service) TestExported_SetEnv(key, val string) {
	if s.extraEnv == nil {
		s.extraEnv = make(map[string]string)
	}
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

package codexusage

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	agentusage "github.com/xhd2015/agent-pro/agent/usage"
	"github.com/xhd2015/ai-critic/macosapp/debuglog"
	"github.com/xhd2015/ai-critic/macosapp/menubar"
)

const refreshInterval = 60 * time.Second

// CodexUsageStatus is the fetch/cache state exposed to API clients.
type CodexUsageStatus string

const (
	StatusLoading CodexUsageStatus = "loading"
	StatusReady   CodexUsageStatus = "ready"
	StatusError   CodexUsageStatus = "error"
)

// CodexUsageResponse is the JSON shape for GET /api/codex/usage.
type CodexUsageResponse struct {
	Status       CodexUsageStatus `json:"status"`
	MonthlyUsage string           `json:"monthly_usage,omitempty"`
	CreditsUsed  string           `json:"credits_used,omitempty"`
	CreditsTotal string           `json:"credits_total,omitempty"`
	NextReset    string           `json:"next_reset,omitempty"`
	ResetAt      string           `json:"reset_at,omitempty"`
	ResetDisplay string           `json:"reset_display,omitempty"`
	TimeLeft     string           `json:"time_left,omitempty"`
	Error        string           `json:"error,omitempty"`
	UpdatedAt    string           `json:"updated_at,omitempty"`
}

type fetchFunc func(context.Context) (*agentusage.Snapshot, error)

// Service fetches and caches codex usage on a background refresh loop.
type Service struct {
	fetcher fetchFunc
	nowFunc func() time.Time

	mu       sync.Mutex
	fetching bool
	cached   CodexUsageResponse

	stopCh   chan struct{}
	stopOnce sync.Once
}

// NewService creates a codex usage service with in-process fetch.
func NewService() *Service {
	return newService(defaultFetcher)
}

func newService(fetcher fetchFunc) *Service {
	return &Service{
		fetcher: fetcher,
		cached: CodexUsageResponse{
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

func defaultFetcher(ctx context.Context) (*agentusage.Snapshot, error) {
	return agentusage.Fetch(ctx, agentusage.Codex)
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
func (s *Service) Get() CodexUsageResponse {
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
		debuglog.Write(debuglog.Entry{
			Event: "fetch_skip_overlap",
			Labels: map[string]string{
				"component": "codexusage",
				"provider":  "codex",
				"phase":     "service",
			},
		})
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
	debuglog.Write(debuglog.Entry{
		Event: "fetch_begin",
		Labels: map[string]string{
			"component": "codexusage",
			"provider":  "codex",
			"phase":     "service",
		},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	snap, err := s.fetcher(ctx)
	now := s.now()
	nowStr := now.UTC().Format(time.RFC3339)

	s.mu.Lock()
	defer s.mu.Unlock()

	if err != nil {
		s.cached = CodexUsageResponse{
			Status:    StatusError,
			Error:     err.Error(),
			UpdatedAt: nowStr,
		}
		debuglog.Write(debuglog.Entry{
			Event: "cache_update",
			Labels: map[string]string{
				"component": "codexusage",
				"provider":  "codex",
				"phase":     "service",
			},
			Fields: map[string]any{
				"status": string(StatusError),
				"error":  err.Error(),
			},
		})
		return
	}

	resetAt, resetDisplay, timeLeft := menubar.ResolveStructuredReset(snap.Reset, now)
	s.cached = CodexUsageResponse{
		Status:       StatusReady,
		MonthlyUsage: snap.UsagePercent,
		CreditsUsed:  formatCreditAmount(snap.CreditsUsed),
		CreditsTotal: formatCreditAmount(snap.CreditsTotal),
		NextReset:    snap.Reset,
		ResetAt:      resetAt,
		ResetDisplay: resetDisplay,
		TimeLeft:     timeLeft,
		UpdatedAt:    nowStr,
	}
	debuglog.Write(debuglog.Entry{
		Event: "cache_update",
		Labels: map[string]string{
			"component": "codexusage",
			"provider":  "codex",
			"phase":     "service",
		},
		Fields: map[string]any{
			"status":         string(StatusReady),
			"monthly_usage":  s.cached.MonthlyUsage,
			"credits_used":   s.cached.CreditsUsed,
			"credits_total":  s.cached.CreditsTotal,
			"next_reset":     s.cached.NextReset,
			"reset_at":       s.cached.ResetAt,
			"reset_display":  s.cached.ResetDisplay,
			"time_left":      s.cached.TimeLeft,
		},
	})
}

func formatCreditAmount(raw string) string {
	raw = strings.ReplaceAll(strings.TrimSpace(raw), ",", "")
	if raw == "" {
		return ""
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return raw
	}
	return formatWithCommas(n)
}

// TestExported_NewService creates a service with the default in-process fetcher.
func TestExported_NewService() *Service {
	return newService(defaultFetcher)
}

// TestExported_SetFetcher replaces the default in-process fetch for doctest harness.
func TestExported_SetFetcher(s *Service, fn fetchFunc) {
	s.fetcher = fn
}

// TestExported_FetchOnce performs a single synchronous fetch for doctest harness.
func (s *Service) TestExported_FetchOnce(t *testing.T) CodexUsageResponse {
	t.Helper()
	s.fetchOnce()
	return s.Get()
}

// TestExported_TriggerRefresh starts an asynchronous refresh (skips if one is in flight).
func (s *Service) TestExported_TriggerRefresh() {
	go s.tryFetch()
}

// TestExported_SeedReady seeds a ready cache with fixed structured reset fields.
func (s *Service) TestExported_SeedReady(resetAt, resetDisplay, nextReset, monthly string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cached = CodexUsageResponse{
		Status:       StatusReady,
		MonthlyUsage: monthly,
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

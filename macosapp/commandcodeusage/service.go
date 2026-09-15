// Package commandcodeusage fetches and caches Command Code account usage for one
// config home (one account), for the menu-bar usage items.
package commandcodeusage

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xhd2015/agent-pro/agent/commandcode"
	"github.com/xhd2015/ai-critic/macosapp/menubar"
)

const (
	refreshInterval = 10 * time.Minute
	fetchTimeout    = 60 * time.Second
	// detailWidth is the fixed panel width for the stored detail text: the menu
	// app renders it verbatim, so it must not depend on a terminal.
	detailWidth = 80
	// unknownPercent marks a window the account does not expose.
	unknownPercent = -1
)

// Status is the fetch/cache state exposed to usage items.
type Status string

const (
	StatusLoading Status = "loading"
	StatusReady   Status = "ready"
	StatusError   Status = "error"
)

// Response is the cached usage state for one Command Code home.
type Response struct {
	Status          Status
	PlanName        string
	PlanStatus      string
	CyclePercent    int
	CycleRemaining  string
	CycleRequests   string
	DaysToRenewal   int
	FiveHourPercent int
	WeeklyPercent   int
	FiveHourResetAt string
	WeeklyResetAt   string
	UsageURL        string
	Detail          string
	Error           string
	UpdatedAt       string

	// FiveHourLeft and WeeklyLeft are recomputed from the reset instants on
	// every Get, so the countdown stays fresh between fetches.
	FiveHourLeft string
	WeeklyLeft   string
}

// Snapshot is the provider payload a fetcher returns; the service turns it into
// a Response with reset countdowns derived from reset instants.
type Snapshot struct {
	PlanName        string
	PlanStatus      string
	CyclePercent    int
	CycleRemaining  string
	CycleRequests   string
	DaysToRenewal   int
	FiveHourPercent int
	WeeklyPercent   int
	FiveHourResetAt time.Time
	WeeklyResetAt   time.Time
	UsageURL        string
	Detail          string
}

type fetchFunc func(ctx context.Context) (*Snapshot, error)

// Service fetches and caches usage for one home on a background refresh loop.
type Service struct {
	home    string
	apiURL  string
	fetcher fetchFunc
	nowFunc func() time.Time

	mu       sync.Mutex
	fetching bool
	cached   Response

	stopCh   chan struct{}
	stopOnce sync.Once
}

// NewService creates a service for the given Command Code config dir.
func NewService(home, apiURL string) *Service {
	return newService(home, apiURL, nil)
}

func newService(home, apiURL string, fetcher fetchFunc) *Service {
	if fetcher == nil {
		fetcher = func(ctx context.Context) (*Snapshot, error) {
			return fetchSnapshot(ctx, home, apiURL)
		}
	}
	return &Service{
		home:    home,
		apiURL:  apiURL,
		fetcher: fetcher,
		cached: Response{Status: StatusLoading, CyclePercent: unknownPercent, DaysToRenewal: unknownPercent,
			FiveHourPercent: unknownPercent, WeeklyPercent: unknownPercent},
		stopCh: make(chan struct{}),
	}
}

// Home returns the config dir this service reads credentials from.
func (s *Service) Home() string { return s.home }

func (s *Service) now() time.Time {
	if s.nowFunc != nil {
		return s.nowFunc()
	}
	return time.Now()
}

// Start begins the background refresh loop.
func (s *Service) Start() {
	go s.refreshLoop()
}

// Stop ends the background refresh loop.
func (s *Service) Stop() {
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
}

// Get returns the cached response with countdowns recomputed for the current time.
func (s *Service) Get() Response {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.cached
	now := s.now()
	out.FiveHourLeft = leftText(out.FiveHourResetAt, now)
	out.WeeklyLeft = leftText(out.WeeklyResetAt, now)
	return out
}

// EnsureFetch triggers a fetch when no fetch has completed yet.
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
	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	defer cancel()

	snap, err := s.fetcher(ctx)
	nowStr := s.now().UTC().Format(time.RFC3339)

	s.mu.Lock()
	defer s.mu.Unlock()

	if err != nil {
		s.cached = Response{
			Status:          StatusError,
			Error:           strings.TrimSpace(err.Error()),
			UpdatedAt:       nowStr,
			CyclePercent:    unknownPercent,
			DaysToRenewal:   unknownPercent,
			FiveHourPercent: unknownPercent,
			WeeklyPercent:   unknownPercent,
		}
		return
	}

	s.cached = Response{
		Status:          StatusReady,
		PlanName:        snap.PlanName,
		PlanStatus:      snap.PlanStatus,
		CyclePercent:    snap.CyclePercent,
		CycleRemaining:  snap.CycleRemaining,
		CycleRequests:   snap.CycleRequests,
		DaysToRenewal:   snap.DaysToRenewal,
		FiveHourPercent: snap.FiveHourPercent,
		WeeklyPercent:   snap.WeeklyPercent,
		FiveHourResetAt: formatInstant(snap.FiveHourResetAt),
		WeeklyResetAt:   formatInstant(snap.WeeklyResetAt),
		UsageURL:        snap.UsageURL,
		Detail:          snap.Detail,
		UpdatedAt:       nowStr,
	}
}

// TitleSuffix is the menu-bar percent for this account, e.g. "5%".
func TitleSuffix(r Response) string {
	if r.Status != StatusReady || r.CyclePercent < 0 {
		return ""
	}
	return fmt.Sprintf("%d%%", r.CyclePercent)
}

// FormatBody renders the dropdown body (no provider prefix) for one account.
func FormatBody(r Response) string {
	var parts []string
	if r.CyclePercent >= 0 {
		parts = append(parts, fmt.Sprintf("%d%% used", r.CyclePercent))
	}
	if r.CycleRemaining != "" {
		parts = append(parts, r.CycleRemaining+" left")
	}
	if r.CycleRequests != "" {
		parts = append(parts, r.CycleRequests+" requests")
	}
	if r.FiveHourPercent >= 0 {
		parts = append(parts, fmt.Sprintf("5h %d%%", r.FiveHourPercent))
	}
	if r.WeeklyPercent >= 0 {
		parts = append(parts, fmt.Sprintf("Weekly %d%%", r.WeeklyPercent))
	}
	switch {
	case r.DaysToRenewal == 0:
		parts = append(parts, "renews today")
	case r.DaysToRenewal > 0:
		parts = append(parts, fmt.Sprintf("renews in %dd", r.DaysToRenewal))
	}
	return strings.Join(parts, ", ")
}

func fetchSnapshot(ctx context.Context, home, apiURL string) (*Snapshot, error) {
	client, err := commandcode.NewClient(home, apiURL)
	if err != nil {
		return nil, err
	}
	data, err := client.FetchUsageWithOptions(ctx, commandcode.UsageOptions{})
	if err != nil {
		return nil, err
	}
	if data.Whoami == nil {
		if len(data.Errors) > 0 {
			return nil, errors.New(strings.Join(data.Errors, "; "))
		}
		return nil, errors.New("no account data returned")
	}

	now := time.Now()
	view := commandcode.ProjectUsageView(data, now)
	snap := &Snapshot{
		PlanName:        planName(view),
		PlanStatus:      subscriptionStatus(view),
		CyclePercent:    unknownPercent,
		DaysToRenewal:   unknownPercent,
		FiveHourPercent: unknownPercent,
		WeeklyPercent:   unknownPercent,
		UsageURL:        view.UsageURL,
		Detail:          strings.TrimRight(commandcode.FormatUsage(view, commandcode.FormatOptions{Width: detailWidth}), "\n"),
	}
	if view.Credits.HasCreditsInfo {
		snap.CyclePercent = int(math.Round(view.Credits.UsagePercent))
		snap.CycleRemaining = commandcode.FormatCredits(view.Credits.TotalRemaining)
	}
	if view.Summary != nil {
		snap.CycleRequests = groupDigits(int64(view.Summary.TotalCount))
	}
	if view.DaysLeft != nil {
		snap.DaysToRenewal = *view.DaysLeft
	}
	if view.WindowLimits != nil {
		if w := view.WindowLimits.FiveHour; w != nil {
			snap.FiveHourPercent = windowPercent(w)
			snap.FiveHourResetAt = windowResetTime(w)
		}
		if w := view.WindowLimits.Weekly; w != nil {
			snap.WeeklyPercent = windowPercent(w)
			snap.WeeklyResetAt = windowResetTime(w)
		}
	}
	return snap, nil
}

func planName(view *commandcode.View) string {
	if view.Plan == nil {
		return ""
	}
	return strings.TrimSpace(view.Plan.Name)
}

func subscriptionStatus(view *commandcode.View) string {
	if view.Subscription == nil {
		return ""
	}
	return strings.TrimSpace(view.Subscription.Status)
}

func windowPercent(w *commandcode.WindowSpan) int {
	if w.Cap <= 0 {
		return unknownPercent
	}
	percent := int(math.Round(w.Used / w.Cap * 100))
	if percent < 0 {
		return 0
	}
	if percent > 100 {
		return 100
	}
	return percent
}

func windowResetTime(w *commandcode.WindowSpan) time.Time {
	if w.ResetAt <= 0 {
		return time.Time{}
	}
	return time.UnixMilli(int64(w.ResetAt))
}

func formatInstant(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

func leftText(resetAt string, now time.Time) string {
	if resetAt == "" {
		return ""
	}
	parsed, err := time.Parse(time.RFC3339, resetAt)
	if err != nil {
		return ""
	}
	return menubar.FormatTimeLeftFromInstant(parsed, now)
}

func groupDigits(n int64) string {
	if n < 0 {
		return fmt.Sprintf("%d", n)
	}
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var parts []string
	for len(s) > 3 {
		parts = append([]string{s[len(s)-3:]}, parts...)
		s = s[:len(s)-3]
	}
	return s + "," + strings.Join(parts, ",")
}

// TestExported_NewService creates a service with an injectable fetcher for tests.
func TestExported_NewService(home, apiURL string) *Service {
	return newService(home, apiURL, func(context.Context) (*Snapshot, error) {
		return nil, errors.New("no fetcher installed")
	})
}

// TestExported_NewLiveService builds a service that performs real provider
// fetches, so tests can point it at a fixture HTTP API.
func TestExported_NewLiveService(home, apiURL string) *Service {
	return newService(home, apiURL, nil)
}

// TestExported_SetFetcher replaces the fetch implementation.
func TestExported_SetFetcher(s *Service, fn fetchFunc) {
	s.fetcher = fn
}

// TestExported_SetNow injects a fixed wall clock for Get() countdown recompute.
func TestExported_SetNow(s *Service, now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	fixed := now
	s.nowFunc = func() time.Time { return fixed }
}

// TestExported_FetchOnce performs a single synchronous fetch.
func TestExported_FetchOnce(t *testing.T, s *Service) Response {
	t.Helper()
	s.fetchOnce()
	return s.Get()
}

// TestExported_SeedReady seeds a ready cache with fixed snapshot fields.
func TestExported_SeedReady(s *Service, snap Snapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cached = Response{
		Status:          StatusReady,
		PlanName:        snap.PlanName,
		PlanStatus:      snap.PlanStatus,
		CyclePercent:    snap.CyclePercent,
		CycleRemaining:  snap.CycleRemaining,
		CycleRequests:   snap.CycleRequests,
		DaysToRenewal:   snap.DaysToRenewal,
		FiveHourPercent: snap.FiveHourPercent,
		WeeklyPercent:   snap.WeeklyPercent,
		FiveHourResetAt: formatInstant(snap.FiveHourResetAt),
		WeeklyResetAt:   formatInstant(snap.WeeklyResetAt),
		UsageURL:        snap.UsageURL,
		Detail:          snap.Detail,
		UpdatedAt:       s.now().UTC().Format(time.RFC3339),
	}
}

package cloudflare

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/xhd2015/ai-critic/server/cloudflareproxy"
)

// proxyCountsTTL bounds how often the edge is asked for its mapping list. The
// GUI polls service status every few seconds, and several callers read domain
// status per request, so the answer is shared briefly rather than refetched.
const proxyCountsTTL = 5 * time.Second

var (
	proxyCountsMu    sync.Mutex
	proxyCountsAt    time.Time
	proxyCounts      map[string]int
	proxyCountsErr   error
	proxyCountsValid bool
)

// ProxyModeEnabled reports whether this server publishes through the edge
// reverse proxy rather than running its own cloudflared tunnel.
func ProxyModeEnabled() bool {
	cfg, err := LoadConfig()
	if err != nil {
		return false
	}
	return proxyModeEnabled(cfg)
}

// ProxySessionDomains lists the domains with a live local publish session, in
// sorted order. A session being registered does not prove the edge still has a
// dial pool; use ProxyDialCounts for that.
func ProxySessionDomains() []string {
	proxySessMu.Lock()
	defer proxySessMu.Unlock()
	domains := make([]string, 0, len(proxySess))
	for domain := range proxySess {
		domains = append(domains, domain)
	}
	sort.Strings(domains)
	return domains
}

// ProxyDialCounts returns the edge's live dial count per published hostname.
//
// The local session registry cannot tell a healthy publication from one whose
// pool died: the session stays registered either way. The edge knows, so this
// is the authoritative liveness source. An error means the edge was
// unreachable and the state is unknown, not that nothing is published.
func ProxyDialCounts() (map[string]int, error) {
	proxyCountsMu.Lock()
	defer proxyCountsMu.Unlock()

	if proxyCountsValid && time.Since(proxyCountsAt) < proxyCountsTTL {
		return proxyCounts, proxyCountsErr
	}

	cfg, err := LoadConfig()
	if err != nil {
		proxyCounts, proxyCountsErr, proxyCountsAt, proxyCountsValid = nil, err, time.Now(), true
		return nil, err
	}
	if !proxyModeEnabled(cfg) || strings.TrimSpace(cfg.ProxyURL) == "" || strings.TrimSpace(cfg.Token) == "" {
		err := &ProxyDisabledError{}
		proxyCounts, proxyCountsErr, proxyCountsAt, proxyCountsValid = nil, err, time.Now(), true
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	mappings, err := (&cloudflareproxy.APIClient{
		BaseURL: strings.TrimSpace(cfg.ProxyURL),
		Token:   strings.TrimSpace(cfg.Token),
	}).ListMappingsContext(ctx)
	if err != nil {
		proxyCounts, proxyCountsErr, proxyCountsAt, proxyCountsValid = nil, err, time.Now(), true
		return nil, err
	}

	counts := make(map[string]int, len(mappings))
	for _, mapping := range mappings {
		counts[mapping.Hostname] = mapping.WS
	}
	proxyCounts, proxyCountsErr, proxyCountsAt, proxyCountsValid = counts, nil, time.Now(), true
	return counts, nil
}

// ProxyDisabledError reports that this server does not publish through the edge.
type ProxyDisabledError struct{}

func (*ProxyDisabledError) Error() string {
	return "cloudflare proxy mode is not enabled"
}

// SetTestProxyDialCounts injects a canned edge view and returns a restore func,
// so callers outside this package can test their handling of live, dead and
// unreachable edges without a real proxy.
// SetTestProxySession registers or clears a local publish session for tests.
func SetTestProxySession(domain string, active bool) {
	if active {
		proxySessMu.Lock()
		proxySess[domain] = &proxySession{cancel: func() {}}
		proxySessMu.Unlock()
		return
	}
	stopProxySession(domain)
}

func SetTestProxyDialCounts(counts map[string]int, err error) (restore func()) {
	proxyCountsMu.Lock()
	prevCounts, prevErr, prevAt, prevValid := proxyCounts, proxyCountsErr, proxyCountsAt, proxyCountsValid
	proxyCounts, proxyCountsErr, proxyCountsAt, proxyCountsValid = counts, err, time.Now(), true
	proxyCountsMu.Unlock()

	return func() {
		proxyCountsMu.Lock()
		proxyCounts, proxyCountsErr, proxyCountsAt, proxyCountsValid = prevCounts, prevErr, prevAt, prevValid
		proxyCountsMu.Unlock()
	}
}

package systemservices

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/xhd2015/ai-critic/server/cloudflare"
	"github.com/xhd2015/ai-critic/server/domains"
	"github.com/xhd2015/ai-critic/server/services"
)

// proxyDeadGrace is how long a hostname may show no dials before it counts as
// dead. A publish session needs a moment to establish its pool, so a restart
// would otherwise flash a false "dead" for hosts that are merely warming up.
const proxyDeadGrace = 60 * time.Second

// expectedHost is one hostname this server intends to publish through the edge.
type expectedHost struct {
	Host      string
	ServiceID string // empty for a domain-tunnel host
}

// hostState is one hostname joined with the edge's view of it.
type hostState struct {
	Host      string
	ServiceID string
	Dials     int
	State     string
}

// cloudFlareProxyController reports the edge's live dial pools for every
// hostname this server publishes. It is status-only except for Restart, which
// republishes hosts whose pool is dead or missing.
//
// The local session registry cannot tell a healthy publication from one whose
// pool died, so the edge is the source of truth. SystemStatus runs while the
// services manager holds its lock, so the manager pushes the hostnames it owns
// (SetSystemHosts) and the controller never calls back into it.
type cloudFlareProxyController struct {
	mu    sync.Mutex
	owned []services.SystemHost
	// pending records when a hostname was first seen without a pool, so a
	// warm-up is not mistaken for a dead publication.
	pending map[string]time.Time

	// Test seams. Production leaves them nil.
	expectedHosts func() []expectedHost
	dialCounts    func() (map[string]int, error)
	republish     func(expectedHost) error
	proxyURL      func() string
	deadGrace     time.Duration
}

// SetSystemHosts records the hostnames the services manager owns.
func (c *cloudFlareProxyController) SetSystemHosts(hosts []services.SystemHost) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.owned = append([]services.SystemHost(nil), hosts...)
}

func (c *cloudFlareProxyController) ownedHosts() []services.SystemHost {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]services.SystemHost(nil), c.owned...)
}

func (c *cloudFlareProxyController) expected() []expectedHost {
	if c.expectedHosts != nil {
		return c.expectedHosts()
	}

	out := make([]expectedHost, 0)
	seen := map[string]bool{}
	if cfg, err := domains.LoadDomains(); err == nil {
		for _, d := range cfg.Domains {
			if d.Provider != domains.ProviderCloudflare || d.Domain == "" || seen[d.Domain] {
				continue
			}
			seen[d.Domain] = true
			out = append(out, expectedHost{Host: d.Domain})
		}
	}
	for _, host := range c.ownedHosts() {
		if host.Host == "" || seen[host.Host] {
			continue
		}
		seen[host.Host] = true
		out = append(out, expectedHost{Host: host.Host, ServiceID: host.ServiceID})
	}
	return out
}

func (c *cloudFlareProxyController) counts() (map[string]int, error) {
	if c.dialCounts != nil {
		return c.dialCounts()
	}
	return cloudflare.ProxyDialCounts()
}

func (c *cloudFlareProxyController) edgeURL() string {
	if c.proxyURL != nil {
		return c.proxyURL()
	}
	cfg, err := cloudflare.LoadConfig()
	if err != nil || cfg == nil {
		return ""
	}
	return strings.TrimSpace(cfg.ProxyURL)
}

func (c *cloudFlareProxyController) grace() time.Duration {
	if c.deadGrace > 0 {
		return c.deadGrace
	}
	return proxyDeadGrace
}

// classify joins the expected hostnames with the edge's dial counts, applying
// the warm-up grace so a connecting pool is not reported as dead.
func (c *cloudFlareProxyController) classify(now time.Time) ([]hostState, error) {
	counts, err := c.counts()
	if err != nil {
		return nil, err
	}

	seen := map[string]bool{}
	states := make([]hostState, 0)
	for _, host := range c.expected() {
		if seen[host.Host] {
			continue
		}
		seen[host.Host] = true

		count, ok := counts[host.Host]
		state := hostState{Host: host.Host, ServiceID: host.ServiceID, Dials: count}
		switch {
		case ok && count > 0:
			c.clearPending(host.Host)
			state.State = services.SystemHostLive
		case !c.markPending(host.Host, now):
			// No pool yet, but the grace window has not elapsed: the session is
			// still connecting, which is not a failure.
			state.State = services.SystemHostStarting
		case !ok:
			state.State = services.SystemHostMissing
		default:
			state.State = services.SystemHostDead
		}
		states = append(states, state)
	}
	sort.Slice(states, func(i, j int) bool { return states[i].Host < states[j].Host })
	return states, nil
}

// markPending records a hostname seen without a pool and reports whether it has
// been that way long enough to be considered dead rather than warming up.
func (c *cloudFlareProxyController) markPending(host string, now time.Time) (dead bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.pending == nil {
		c.pending = map[string]time.Time{}
	}
	first, ok := c.pending[host]
	if !ok {
		c.pending[host] = now
		return false
	}
	return now.Sub(first) >= c.grace()
}

func (c *cloudFlareProxyController) clearPending(host string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.pending, host)
}

func (c *cloudFlareProxyController) SystemStatus() services.SystemRuntime {
	runtime := services.SystemRuntime{Edge: c.edgeURL()}
	if len(c.expected()) == 0 {
		runtime.Detail = "not configured"
		return runtime
	}

	states, err := c.classify(time.Now())
	if err != nil {
		runtime.Status = services.StatusUnknown
		runtime.Detail = "edge unreachable: " + err.Error()
		hosts := make([]services.SystemHostStatus, 0, len(c.expected()))
		for _, host := range c.expected() {
			hosts = append(hosts, services.SystemHostStatus{
				Host:  host.Host,
				State: services.SystemHostUnknown,
			})
		}
		sort.Slice(hosts, func(i, j int) bool { return hosts[i].Host < hosts[j].Host })
		runtime.Hosts = hosts
		return runtime
	}

	hosts := make([]services.SystemHostStatus, 0, len(states))
	live, dead, missing, starting, dials := 0, 0, 0, 0, 0
	for _, state := range states {
		hosts = append(hosts, services.SystemHostStatus{
			Host:  state.Host,
			Dials: state.Dials,
			State: state.State,
		})
		switch state.State {
		case services.SystemHostLive:
			live++
			dials += state.Dials
		case services.SystemHostStarting:
			starting++
		case services.SystemHostMissing:
			missing++
		default:
			dead++
		}
	}
	runtime.Hosts = hosts

	runtime.Detail = fmt.Sprintf("%d published · %d dials · %d dead", live, dials, dead)
	if starting > 0 {
		runtime.Detail += fmt.Sprintf(" · %d starting", starting)
	}
	if missing > 0 {
		runtime.Detail += fmt.Sprintf(" · %d missing", missing)
	}

	switch {
	case dead > 0 || missing > 0:
		runtime.Status = services.StatusError
		runtime.Running = false
	case starting > 0:
		runtime.Status = services.StatusStarting
		runtime.Running = true
	default:
		runtime.Status = services.StatusRunning
		runtime.Running = true
	}
	return runtime
}

// RestartSystem republishes every hostname that is not currently serving,
// including one still inside the warm-up grace: an operator asking for a repair
// has already decided the host should be live, so the grace that keeps status
// from crying wolf must not make the repair a no-op. Healthy pools are left
// alone. It runs without the manager lock held.
func (c *cloudFlareProxyController) RestartSystem() error {
	states, err := c.classify(time.Now())
	if err != nil {
		return fmt.Errorf("cannot republish: edge unreachable: %w", err)
	}
	republish := c.republish
	if republish == nil {
		republish = defaultRepublishProxyHost
	}

	var failed []string
	recovered := 0
	for _, state := range states {
		if state.State == services.SystemHostLive {
			continue
		}
		if err := republish(expectedHost{Host: state.Host, ServiceID: state.ServiceID}); err != nil {
			failed = append(failed, state.Host+": "+err.Error())
			continue
		}
		recovered++
	}
	if len(failed) > 0 {
		return fmt.Errorf("republished %d hostnames, %d failed: %s",
			recovered, len(failed), strings.Join(failed, "; "))
	}
	return nil
}

func defaultRepublishProxyHost(host expectedHost) error {
	if host.ServiceID != "" {
		return services.GetDefaultManager().RepublishForward(host.ServiceID)
	}
	port := domains.GetServerPort()
	if port == 0 {
		return fmt.Errorf("server port is not set")
	}
	_, err := domains.StartHostDomainTunnel(host.Host, port, func(string) {})
	return err
}

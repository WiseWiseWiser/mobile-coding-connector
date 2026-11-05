package unified_tunnel

import (
	"fmt"
	"sync"
	"time"
)

// An aged cloudflared connector starts holding requests at the edge: the same
// hostname answers in milliseconds when probed locally, but through the tunnel a
// large share of requests stall for several seconds. Reachability checks cannot
// see it — every request still succeeds, only late — and a fresh connector is
// fast again, so the only signal is latency. These bounds turn that signal into
// one restart.
const (
	// slowProbeThreshold is where a successful probe counts as stalled. Local
	// origin responses are ~10ms and a healthy tunnel ~100-300ms on this host,
	// while a degrading connector lands at 4-5s. A slow origin would have to be
	// slower than this on every mapping for half a minute to trip it.
	slowProbeThreshold = 3 * time.Second
	// slowProbeStrikes is how many consecutive stalled probes confirm it. Health
	// checks run every 10s per mapping, so 3 strikes is about half a minute of
	// sustained stalls rather than one unlucky request.
	slowProbeStrikes = 3
	// slowRestartCooldown keeps the cure from being worse than the disease: a
	// restart drops every live browser connection (SSE, websockets), so it must
	// stay rare.
	slowRestartCooldown = 20 * time.Minute
)

// slowConnectorDetector counts consecutive successful-but-stalled probes and
// decides when they justify restarting the connector.
type slowConnectorDetector struct {
	mu        sync.Mutex
	strikes   int
	lastFired time.Time
	now       func() time.Time
	// threshold overrides slowProbeThreshold; tests shrink it.
	threshold time.Duration
}

func (d *slowConnectorDetector) stallLimit() time.Duration {
	if d.threshold > 0 {
		return d.threshold
	}
	return slowProbeThreshold
}

func (d *slowConnectorDetector) clock() time.Time {
	if d.now != nil {
		return d.now()
	}
	return time.Now()
}

// observe records one probe result and reports whether the connector should be
// restarted. Only a healthy response can be stalled: a failure has its own
// restart path, and Cloudflare 530 is the connector still registering.
func (d *slowConnectorDetector) observe(elapsed time.Duration, healthy bool) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if !healthy || elapsed < d.stallLimit() {
		d.strikes = 0
		return false
	}
	d.strikes++
	if d.strikes < slowProbeStrikes {
		return false
	}
	d.strikes = 0
	now := d.clock()
	if !d.lastFired.IsZero() && now.Sub(d.lastFired) < slowRestartCooldown {
		return false
	}
	d.lastFired = now
	return true
}

// restartTunnelForSlowness is a seam so tests can observe the restart decision
// without spawning cloudflared.
var restartTunnelForSlowness = (*UnifiedTunnelManager).RestartForSlowness

// RestartForSlowness restarts the tunnel process after the health checks saw
// repeated stalled responses on hostname. The tunnel is one process shared by
// every mapping, so the mapping the stall was seen on only picks a label.
func (utm *UnifiedTunnelManager) RestartForSlowness(hostname string) error {
	utm.mu.RLock()
	mappingID := ""
	for _, m := range utm.mappings {
		if m.Hostname == hostname {
			mappingID = m.ID
			break
		}
	}
	utm.mu.RUnlock()
	if mappingID == "" {
		return fmt.Errorf("no mapping for hostname %s", hostname)
	}
	fmt.Printf("[unified-tunnel] connector slow: %s answered later than %v on %d probes, restarting tunnel\n",
		hostname, slowProbeThreshold, slowProbeStrikes)
	return utm.RestartMapping(mappingID)
}

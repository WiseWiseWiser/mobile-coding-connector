package unified_tunnel

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// newTestDetector keeps detector logging out of the real health log.
func newTestDetector(now func() time.Time) *slowConnectorDetector {
	d := &slowConnectorDetector{now: now, logf: func(string, ...any) {}}
	return d
}

func TestSlowConnectorDetectorIgnoresFastProbes(t *testing.T) {
	d := newTestDetector(nil)
	for i := 0; i < slowProbeStrikes*2; i++ {
		if d.observe("slow.example", 120*time.Millisecond, true) {
			t.Fatalf("fast probe %d asked for a restart", i)
		}
	}
	if d.strikes != 0 {
		t.Fatalf("strikes = %d after fast probes, want 0", d.strikes)
	}
}

func TestSlowConnectorDetectorRestartsWithThreeConsecutiveStalls(t *testing.T) {
	d := newTestDetector(nil)
	for i := 0; i < slowProbeStrikes-1; i++ {
		if d.observe("slow.example", 5*time.Second, true) {
			t.Fatalf("stall %d restarted too early", i+1)
		}
	}
	if !d.observe("slow.example", 5*time.Second, true) {
		t.Fatalf("stall %d did not ask for a restart", slowProbeStrikes)
	}
	if d.strikes != 0 {
		t.Fatalf("strikes = %d after restart, want 0", d.strikes)
	}
}

func TestSlowConnectorDetectorFastProbeResetsStrikes(t *testing.T) {
	d := newTestDetector(nil)
	slow := func() bool { return d.observe("slow.example", 5*time.Second, true) }
	slow()
	slow()
	if d.observe("slow.example", 90*time.Millisecond, true) {
		t.Fatal("fast probe restarted the tunnel")
	}
	slow()
	if slow() {
		t.Fatal("restarted after a gap that should have reset the strike count")
	}
}

func TestSlowConnectorDetectorUnhealthyResetsStrikes(t *testing.T) {
	d := newTestDetector(nil)
	d.observe("slow.example", 5*time.Second, true)
	d.observe("slow.example", 5*time.Second, true)
	// A failed probe has its own restart path and must not count as a stall.
	if d.observe("slow.example", 20*time.Second, false) {
		t.Fatal("unhealthy probe restarted the tunnel")
	}
	if d.observe("slow.example", 5*time.Second, true) {
		t.Fatal("unhealthy probe should have cleared the earlier strikes")
	}
}

func TestSlowConnectorDetectorCooldownSuppressesRepeatedRestarts(t *testing.T) {
	now := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	d := newTestDetector(func() time.Time { return now })
	stallThrice := func() bool {
		d.observe("slow.example", 5*time.Second, true)
		d.observe("slow.example", 5*time.Second, true)
		return d.observe("slow.example", 5*time.Second, true)
	}
	if !stallThrice() {
		t.Fatal("first degradation did not restart the tunnel")
	}
	now = now.Add(slowRestartCooldown / 2)
	if stallThrice() {
		t.Fatal("restart repeated inside the cooldown")
	}
	now = now.Add(slowRestartCooldown)
	if !stallThrice() {
		t.Fatal("degradation after the cooldown did not restart the tunnel")
	}
}

func TestSlowConnectorDetectorThreshold(t *testing.T) {
	d := newTestDetector(nil)
	if d.observe("slow.example", slowProbeThreshold-time.Millisecond, true) {
		t.Fatal("probe under the threshold counted as stalled")
	}
	if d.strikes != 0 {
		t.Fatalf("strikes = %d, want 0", d.strikes)
	}
}

// Stalled-but-successful probes must restart exactly once, through the seam.
func TestCheckMappingHealthRestartsStalledConnector(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(60 * time.Millisecond)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()
	defer SetHealthCheckBaseURLForTest(srv.URL)()

	var mu sync.Mutex
	var restarts []string
	restore := setRestartTunnelForSlownessForTest(func(_ *UnifiedTunnelManager, hostname string) error {
		mu.Lock()
		defer mu.Unlock()
		restarts = append(restarts, hostname)
		return nil
	})
	defer restore()

	utm := NewUnifiedTunnelManager("test")
	utm.mappings["m1"] = &IngressMapping{ID: "m1", Hostname: "slow.example"}
	utm.slowProbes.threshold = 20 * time.Millisecond
	utm.slowProbes.logf = func(string, ...any) {}

	for i := 0; i < slowProbeStrikes; i++ {
		if !utm.checkMappingHealth("slow.example") {
			t.Fatalf("probe %d reported unhealthy for a 401 response", i+1)
		}
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		mu.Lock()
		n := len(restarts)
		got := append([]string(nil), restarts...)
		mu.Unlock()
		if n == 1 {
			if got[0] != "slow.example" {
				t.Fatalf("restart hostname = %q", got[0])
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("restarts = %v, want exactly one", got)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestRestartForSlownessUnknownHostname(t *testing.T) {
	utm := NewUnifiedTunnelManager("test")
	if err := utm.RestartForSlowness("missing.example"); err == nil {
		t.Fatal("expected an error for a hostname with no mapping")
	}
}

// setRestartTunnelForSlownessForTest swaps the restart seam. Returns a restore func.
func setRestartTunnelForSlownessForTest(fn func(*UnifiedTunnelManager, string) error) func() {
	prev := restartTunnelForSlowness
	restartTunnelForSlowness = fn
	return func() { restartTunnelForSlowness = prev }
}

// The restart must leave a breadcrumb: the app's stdout is not captured, so the
// health log is the only place answering "why did the page reconnect?".
func TestSlowConnectorDetectorLogsRestart(t *testing.T) {
	var lines []string
	d := &slowConnectorDetector{logf: func(format string, args ...any) {
		lines = append(lines, fmt.Sprintf(format, args...))
	}}
	for i := 0; i < slowProbeStrikes; i++ {
		d.observe("eluc-on-mac.xhd2015.xyz", 5*time.Second, true)
	}
	if len(lines) != slowProbeStrikes {
		t.Fatalf("log lines = %v, want %d", lines, slowProbeStrikes)
	}
	last := lines[len(lines)-1]
	if !strings.Contains(last, "restarting connector") || !strings.Contains(last, "eluc-on-mac.xhd2015.xyz") {
		t.Fatalf("restart breadcrumb = %q", last)
	}
}

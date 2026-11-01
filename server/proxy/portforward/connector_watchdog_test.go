package portforward

import (
	"errors"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	pkgcf "github.com/xhd2015/dot-pkgs/go-pkgs/cloudflare"
)

type watchdogHarness struct {
	dir        string
	probed     []string
	prober     func(host string) (time.Duration, error)
	stallPids  []int
	stallBytes int
	stopped    []int
	attached   []string
	now        time.Time
	recycledAt []time.Time
}

func newWatchdogHarness(t *testing.T) *watchdogHarness {
	t.Helper()
	h := &watchdogHarness{
		dir: t.TempDir(),
		now: time.Now(),
	}
	root := filepath.Join(h.dir, "managed-tunnels")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}

	prevConfigDir := watchdogConfigDir
	prevProbe := watchdogProbeHost
	prevStop := watchdogStopPID
	prevAttach := watchdogAttach
	prevNow := watchdogNow
	prevStall := watchdogEdgeStallBytes
	prevAlive := watchdogAlivePID
	t.Cleanup(func() {
		watchdogConfigDir = prevConfigDir
		watchdogProbeHost = prevProbe
		watchdogStopPID = prevStop
		watchdogAttach = prevAttach
		watchdogNow = prevNow
		watchdogEdgeStallBytes = prevStall
		watchdogAlivePID = prevAlive
		resetWatchdogForTest()
	})

	watchdogConfigDir = func() (string, error) { return h.dir, nil }
	watchdogProbeHost = func(_ *http.Client, host string) (time.Duration, error) {
		return h.probe(host)
	}
	// Default: no stall signal (tests opt in via h.stallBytes).
	watchdogEdgeStallBytes = func(pid int) int {
		h.stallPids = append(h.stallPids, pid)
		return h.stallBytes
	}
	// Recorded connector pids are treated as alive so the stall probe runs.
	watchdogAlivePID = func(int) bool { return true }
	watchdogNow = func() time.Time { return h.now }
	resetWatchdogForTest()
	return h
}

func (h *watchdogHarness) probe(host string) (time.Duration, error) {
	h.probed = append(h.probed, host)
	if h.prober == nil {
		return 10 * time.Millisecond, nil
	}
	return h.prober(host)
}

func (h *watchdogHarness) installTunnel(t *testing.T, name string, st *pkgcf.TunnelState) {
	t.Helper()
	dir := filepath.Join(h.dir, "managed-tunnels", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := pkgcf.SaveTunnelState(dir, st); err != nil {
		t.Fatal(err)
	}
}

func TestConnectorWatchdogRecyclesAfterThreeStrikes(t *testing.T) {
	h := newWatchdogHarness(t)
	h.installTunnel(t, "tunnel-a", &pkgcf.TunnelState{
		TunnelName:   "tunnel-a",
		ConnectorPID: 0,
		Hosts: map[string]*pkgcf.HostEntry{
			"eluc-on-mac.example.xyz": {Service: "http://127.0.0.1:8422"},
		},
	})
	h.prober = func(string) (time.Duration, error) { return 4 * time.Second, nil }

	var attachCalls int
	prevAttach := watchdogAttach
	prevStop := watchdogStopPID
	watchdogAttach = func(st *pkgcf.TunnelState, configDir, host, service string) error {
		attachCalls++
		h.attached = append(h.attached, host)
		return nil
	}
	watchdogStopPID = func(pid int) { h.stopped = append(h.stopped, pid) }
	t.Cleanup(func() { watchdogAttach = prevAttach; watchdogStopPID = prevStop })

	for i := 0; i < 2; i++ {
		connectorWatchdogSweep()
	}
	if attachCalls != 0 {
		t.Fatalf("recycled after %d strikes, want none until %d", attachCalls, connectorSlowStrikes)
	}
	connectorWatchdogSweep()
	if attachCalls != 1 || len(h.attached) != 1 || h.attached[0] != "eluc-on-mac.example.xyz" {
		t.Fatalf("after 3 strikes attach calls = %d attached = %v, want 1× eluc-on-mac.example.xyz", attachCalls, h.attached)
	}
	if len(h.stopped) != 1 {
		t.Fatalf("connector stop calls = %v, want 1", h.stopped)
	}

	// Cooldown: more slow probes must not recycle again.
	h.now = h.now.Add(connectorRecycleCooldown / 2)
	for i := 0; i < connectorSlowStrikes; i++ {
		connectorWatchdogSweep()
	}
	if attachCalls != 1 {
		t.Fatalf("recycled during cooldown: attach calls = %d", attachCalls)
	}

	// After the cooldown a fresh 3-strike run recycles again.
	h.now = h.now.Add(connectorRecycleCooldown)
	for i := 0; i < connectorSlowStrikes; i++ {
		connectorWatchdogSweep()
	}
	if attachCalls != 2 {
		t.Fatalf("attach calls after cooldown recycle = %d, want 2", attachCalls)
	}
}

func TestConnectorWatchdogErrorsAndFastProbesResetStrikes(t *testing.T) {
	h := newWatchdogHarness(t)
	h.installTunnel(t, "tunnel-a", &pkgcf.TunnelState{
		TunnelName: "tunnel-a",
		Hosts: map[string]*pkgcf.HostEntry{
			"eluc-on-mac.example.xyz": {Service: "http://127.0.0.1:8422"},
		},
	})

	sequence := []struct {
		d   time.Duration
		err error
	}{
		{4 * time.Second, nil},
		{4 * time.Second, nil},
		{0, errors.New("dial timeout")}, // network down: reset, never recycle
		{4 * time.Second, nil},
		{4 * time.Second, nil},
	}
	idx := 0
	h.prober = func(string) (time.Duration, error) {
		if idx >= len(sequence) {
			t.Fatal("unexpected extra probe")
		}
		s := sequence[idx]
		idx++
		return s.d, s.err
	}

	var attachCalls int
	prevAttach := watchdogAttach
	watchdogAttach = func(*pkgcf.TunnelState, string, string, string) error { attachCalls++; return nil }
	t.Cleanup(func() { watchdogAttach = prevAttach })

	for i := 0; i < len(sequence); i++ {
		connectorWatchdogSweep()
	}
	if attachCalls != 0 {
		t.Fatalf("recycled despite error reset: attach calls = %d", attachCalls)
	}

	// A fast probe between slow probes also resets the streak.
	h.prober = nil
	for i := 0; i < connectorSlowStrikes-1; i++ {
		connectorWatchdogSweep()
	}
	h.prober = func(string) (time.Duration, error) { return 10 * time.Millisecond, nil }
	connectorWatchdogSweep()
	h.prober = func(string) (time.Duration, error) { return 4 * time.Second, nil }
	connectorWatchdogSweep()
	if attachCalls != 0 {
		t.Fatalf("recycled despite fast-probe reset: attach calls = %d", attachCalls)
	}
}

func TestConnectorWatchdogSkipsHostlessAndBrokenTunnels(t *testing.T) {
	h := newWatchdogHarness(t)
	h.installTunnel(t, "tunnel-empty", &pkgcf.TunnelState{TunnelName: "tunnel-empty"})
	h.installTunnel(t, "tunnel-nilhost", &pkgcf.TunnelState{
		TunnelName: "tunnel-nilhost",
		Hosts:      map[string]*pkgcf.HostEntry{"x.example": nil},
	})
	if err := os.MkdirAll(filepath.Join(h.dir, "managed-tunnels", "tunnel-nostate"), 0o755); err != nil {
		t.Fatal(err)
	}

	connectorWatchdogSweep()
	if len(h.probed) != 0 {
		t.Fatalf("probed %v, want no probes", h.probed)
	}
}

func TestPreferredWatchdogHostSortedDeterministic(t *testing.T) {
	st := &pkgcf.TunnelState{Hosts: map[string]*pkgcf.HostEntry{
		"beta.example":  {Service: "http://127.0.0.1:2"},
		"alpha.example": {Service: "http://127.0.0.1:1"},
		"gamma.example": {Service: ""},
	}}
	host, service, ok := preferredWatchdogHost(st)
	if !ok || host != "alpha.example" || service != "http://127.0.0.1:1" {
		t.Fatalf("got host=%q service=%q ok=%v", host, service, ok)
	}
}

func TestConnectorWatchdogRecyclesOnPinnedBlackhole(t *testing.T) {
	h := newWatchdogHarness(t)
	h.installTunnel(t, "tunnel-a", &pkgcf.TunnelState{
		TunnelName:   "tunnel-a",
		ConnectorPID: 4242,
		Hosts: map[string]*pkgcf.HostEntry{
			"eluc-on-mac.example.xyz": {Service: "http://127.0.0.1:8422"},
		},
	})
	// Probes stay fast — the blackhole hides from sparse TTFB sampling — but
	// the connector's edge socket sits pinned at exactly the base send-buffer
	// cap, the kernel-side signature of a peer that never drains.
	h.prober = func(string) (time.Duration, error) { return 80 * time.Millisecond, nil }
	h.stallBytes = connectorBlackholeQueueBytes

	var attachCalls int
	prevAttach := watchdogAttach
	prevStop := watchdogStopPID
	watchdogAttach = func(*pkgcf.TunnelState, string, string, string) error { attachCalls++; return nil }
	watchdogStopPID = func(pid int) { h.stopped = append(h.stopped, pid) }
	t.Cleanup(func() { watchdogAttach = prevAttach; watchdogStopPID = prevStop })

	connectorWatchdogSweep()
	connectorWatchdogSweep()
	if attachCalls != 0 {
		t.Fatalf("recycled after %d sweeps, want none until %d", attachCalls, connectorBlackholeStrikes)
	}
	connectorWatchdogSweep()
	if attachCalls != 1 {
		t.Fatalf("attach calls after %d sweeps = %d, want 1", connectorBlackholeStrikes, attachCalls)
	}
	if len(h.stopped) != 1 || h.stopped[0] != 4242 {
		t.Fatalf("stopped = %v, want [4242]", h.stopped)
	}
	// The stall probe targeted the connector pid from state.json.
	if len(h.stallPids) == 0 || h.stallPids[0] != 4242 {
		t.Fatalf("stall probes = %v, want [4242 ...]", h.stallPids)
	}

	// A second pinned blackhole recycles again despite the cooldown.
	h.now = h.now.Add(connectorRecycleCooldown / 2)
	h.installTunnel(t, "tunnel-a", &pkgcf.TunnelState{
		TunnelName:   "tunnel-a",
		ConnectorPID: 5151,
		Hosts: map[string]*pkgcf.HostEntry{
			"eluc-on-mac.example.xyz": {Service: "http://127.0.0.1:8422"},
		},
	})
	for i := 0; i < connectorBlackholeStrikes; i++ {
		connectorWatchdogSweep()
	}
	if attachCalls != 2 {
		t.Fatalf("attach calls after second blackhole = %d, want 2", attachCalls)
	}
	if len(h.stopped) < 2 || h.stopped[len(h.stopped)-1] != 5151 {
		t.Fatalf("stopped = %v, want last stop 5151", h.stopped)
	}
}

func TestConnectorWatchdogIgnoresDrainingQueues(t *testing.T) {
	h := newWatchdogHarness(t)
	h.installTunnel(t, "tunnel-a", &pkgcf.TunnelState{
		TunnelName:   "tunnel-a",
		ConnectorPID: 4242,
		Hosts: map[string]*pkgcf.HostEntry{
			"eluc-on-mac.example.xyz": {Service: "http://127.0.0.1:8422"},
		},
	})
	h.prober = func(string) (time.Duration, error) { return 80 * time.Millisecond, nil }

	var attachCalls int
	prevAttach := watchdogAttach
	prevStop := watchdogStopPID
	watchdogAttach = func(*pkgcf.TunnelState, string, string, string) error { attachCalls++; return nil }
	watchdogStopPID = func(pid int) { h.stopped = append(h.stopped, pid) }
	t.Cleanup(func() { watchdogAttach = prevAttach; watchdogStopPID = prevStop })

	// A busy tunnel (streaming responses, client retry floods) pushes queues
	// far past the base cap — measured 130KB-1.6MB — that drain within a tick
	// or two. None of those readings is the exact-cap pin, so none may recycle,
	// no matter how long the busy pattern lasts.
	for _, size := range []int{
		686 * 1024, 131072, 1617 * 1024, 1072507, 130072, 64 * 1024, 131072, 177565,
	} {
		h.stallBytes = size
		connectorWatchdogSweep()
	}
	if attachCalls != 0 {
		t.Fatalf("recycled on draining/live queues: attach calls = %d", attachCalls)
	}
	if len(h.stopped) != 0 {
		t.Fatalf("stopped = %v, want none", h.stopped)
	}
}

func TestConnectorEdgeStallBytesParsesNetstat(t *testing.T) {
	if _, err := exec.LookPath("lsof"); err != nil {
		t.Skip("lsof unavailable")
	}
	if _, err := exec.LookPath("netstat"); err != nil {
		t.Skip("netstat unavailable")
	}
	// This test process has no :7844 connections; the plumbing must read as 0
	// without error. A live connector would return its real queue depth.
	if got := connectorEdgeStallBytes(os.Getpid()); got != 0 {
		t.Fatalf("stall bytes for test pid = %d, want 0 (no :7844 sockets)", got)
	}
}

func TestStopConnectorPIDTerminatesProcessGroup(t *testing.T) {
	cmd := exec.Command("sleep", "30")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Pgid: 0}
	if err := cmd.Start(); err != nil {
		t.Skipf("cannot start sleep: %v", err)
	}
	pid := cmd.Process.Pid
	// Mirror the supervisor: without a concurrent Wait the killed child
	// becomes a zombie and pidAlive would stay true until reap.
	reaped := make(chan struct{})
	go func() { _ = cmd.Wait(); close(reaped) }()
	if !pidAlive(pid) {
		t.Fatal("dummy process not alive after start")
	}
	stopConnectorPID(pid)
	select {
	case <-reaped:
	case <-time.After(time.Second):
		t.Fatal("killed process was not reaped")
	}
	if pidAlive(pid) {
		t.Fatal("connector pid still alive after stopConnectorPID")
	}
}

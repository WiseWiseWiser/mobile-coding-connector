package systemservices

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/xhd2015/ai-critic/server/cloudflare"
	"github.com/xhd2015/ai-critic/server/services"
)

func stubEdge(t *testing.T, counts map[string]int, err error) {
	t.Helper()
	t.Cleanup(cloudflare.SetTestProxyDialCounts(counts, err))
}

func TestCloudflareProxyAggregatesExpectedHosts(t *testing.T) {
	ctrl := &cloudFlareProxyController{
		expectedHosts: func() []expectedHost {
			return []expectedHost{
				{Host: "nest.example.com"},
				{Host: "hello.example.com", ServiceID: "svc-hello"},
			}
		},
		dialCounts: func() (map[string]int, error) {
			return map[string]int{
				"nest.example.com":  30,
				"hello.example.com": 32,
				"other.example.com": 8,
			}, nil
		},
		proxyURL: func() string { return "https://cfproxy.example.com" },
	}

	runtime := ctrl.SystemStatus()
	if !runtime.Running || runtime.Status != services.StatusRunning {
		t.Fatalf("runtime = %#v, want running", runtime)
	}
	if runtime.Edge != "https://cfproxy.example.com" {
		t.Fatalf("Edge = %q", runtime.Edge)
	}
	if runtime.Detail != "2 published · 62 dials · 0 dead" {
		t.Fatalf("Detail = %q", runtime.Detail)
	}
	if len(runtime.Hosts) != 2 {
		t.Fatalf("hosts = %#v, extra edge mappings must be ignored", runtime.Hosts)
	}
	if runtime.Hosts[0].Host != "hello.example.com" || runtime.Hosts[0].Dials != 32 || runtime.Hosts[0].State != services.SystemHostLive {
		t.Fatalf("first host = %#v", runtime.Hosts[0])
	}
}

func TestCloudflareProxyFlagsDeadAndMissingHosts(t *testing.T) {
	ctrl := &cloudFlareProxyController{
		expectedHosts: func() []expectedHost {
			return []expectedHost{
				{Host: "dead.example.com"},
				{Host: "missing.example.com"},
				{Host: "live.example.com"},
			}
		},
		dialCounts: func() (map[string]int, error) {
			return map[string]int{"dead.example.com": 0, "live.example.com": 12}, nil
		},
		deadGrace: time.Millisecond,
	}

	// The first observation only opens the grace window; the second reports the
	// hostnames that stayed without a pool.
	ctrl.SystemStatus()
	time.Sleep(5 * time.Millisecond)

	runtime := ctrl.SystemStatus()
	if runtime.Running || runtime.Status != services.StatusError {
		t.Fatalf("runtime = %#v, want error", runtime)
	}
	if !strings.Contains(runtime.Detail, "1 dead") || !strings.Contains(runtime.Detail, "1 missing") {
		t.Fatalf("Detail = %q", runtime.Detail)
	}

	byHost := map[string]services.SystemHostStatus{}
	for _, host := range runtime.Hosts {
		byHost[host.Host] = host
	}
	if byHost["dead.example.com"].State != services.SystemHostDead {
		t.Fatalf("dead host = %#v", byHost["dead.example.com"])
	}
	if byHost["missing.example.com"].State != services.SystemHostMissing {
		t.Fatalf("missing host = %#v", byHost["missing.example.com"])
	}
	if byHost["live.example.com"].State != services.SystemHostLive || byHost["live.example.com"].Dials != 12 {
		t.Fatalf("live host = %#v", byHost["live.example.com"])
	}
}

func TestCloudflareProxyNeverClaimsRunningWhenEdgeUnreachable(t *testing.T) {
	ctrl := &cloudFlareProxyController{
		expectedHosts: func() []expectedHost {
			return []expectedHost{{Host: "nest.example.com"}}
		},
		dialCounts: func() (map[string]int, error) {
			return nil, errors.New("dial tcp: i/o timeout")
		},
	}

	runtime := ctrl.SystemStatus()
	if runtime.Running || runtime.Status != services.StatusUnknown {
		t.Fatalf("runtime = %#v, want unknown, not running", runtime)
	}
	if !strings.Contains(runtime.Detail, "edge unreachable") {
		t.Fatalf("Detail = %q", runtime.Detail)
	}
	if len(runtime.Hosts) != 1 || runtime.Hosts[0].State != services.SystemHostUnknown {
		t.Fatalf("hosts = %#v", runtime.Hosts)
	}
}

func TestCloudflareProxyRestartRepublishesDeadHosts(t *testing.T) {
	var got []string
	ctrl := &cloudFlareProxyController{
		expectedHosts: func() []expectedHost {
			return []expectedHost{
				{Host: "live.example.com", ServiceID: "svc-live"},
				{Host: "dead.example.com", ServiceID: "svc-dead"},
				{Host: "missing.example.com"},
			}
		},
		dialCounts: func() (map[string]int, error) {
			return map[string]int{"live.example.com": 8, "dead.example.com": 0}, nil
		},
		deadGrace: time.Millisecond,
		republish: func(host expectedHost) error {
			got = append(got, host.Host)
			return nil
		},
	}

	// Open the grace window first, so the dead host counts as dead rather than
	// still connecting.
	ctrl.SystemStatus()
	time.Sleep(5 * time.Millisecond)

	if err := ctrl.RestartSystem(); err != nil {
		t.Fatalf("RestartSystem() error = %v", err)
	}
	if fmt.Sprint(got) != "[dead.example.com missing.example.com]" {
		t.Fatalf("republished = %v, want only dead and missing hosts", got)
	}
}

func TestCloudflareProxyIsStatusOnlyExceptRestart(t *testing.T) {
	m := services.NewManagerFromDefinitions(nil)
	if err := m.RegisterSystemService(services.SystemService{
		ID:         "sys-cf-test",
		Name:       "Cloudflare Proxy",
		Controller: &cloudFlareProxyController{expectedHosts: func() []expectedHost { return nil }},
	}); err != nil {
		t.Fatalf("RegisterSystemService() error = %v", err)
	}

	status := m.List()[0]
	want := []string{services.SystemActionRestart}
	if strings.Join(status.Actions, ",") != strings.Join(want, ",") {
		t.Fatalf("Actions = %v, want %v", status.Actions, want)
	}
	if _, err := m.Start("sys-cf-test"); err == nil || !strings.Contains(err.Error(), "does not support start") {
		t.Fatalf("Start() error = %v, want a status-only error", err)
	}
	if err := m.Stop("sys-cf-test"); err == nil || !strings.Contains(err.Error(), "does not support stop") {
		t.Fatalf("Stop() error = %v, want a status-only error", err)
	}
}

// TestListDoesNotDeadlockWithCloudflareProxyStatus guards the bug that made
// /api/services hang: List holds the manager lock while calling SystemStatus,
// so the controller must not call back into the manager.
func TestListDoesNotDeadlockWithCloudflareProxyStatus(t *testing.T) {
	stubEdge(t, map[string]int{"svc.example.com": 4}, nil)
	m := services.NewManagerFromDefinitions([]services.ServiceDefinition{
		{
			ID:          "svc-hello",
			Name:        "hello",
			Command:     "run",
			PortForward: &services.ServicePortForward{Port: 8080, Provider: "cloudflare_owned", Label: "svc.example.com"},
		},
	})
	if err := Register(m); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	done := make(chan []services.ServiceStatus, 1)
	go func() { done <- m.List() }()

	select {
	case statuses := <-done:
		var found *services.ServiceStatus
		for i := range statuses {
			if statuses[i].ID == CFProxyID {
				found = &statuses[i]
			}
		}
		if found == nil {
			t.Fatal("Cloudflare Proxy missing from List()")
		}
		if len(found.Hosts) != 1 || found.Hosts[0].Host != "svc.example.com" {
			t.Fatalf("hosts = %#v, want the pushed service host", found.Hosts)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("List() did not return: the controller deadlocked on the manager lock")
	}
}

// TestCloudflareProxyTreatsWarmUpAsStarting guards the false alarm seen live:
// pools take a moment to connect after a restart, and reporting that as "dead"
// would cry wolf on every boot.
func TestCloudflareProxyTreatsWarmUpAsStarting(t *testing.T) {
	ctrl := &cloudFlareProxyController{
		expectedHosts: func() []expectedHost {
			return []expectedHost{{Host: "warm.example.com", ServiceID: "svc-warm"}}
		},
		dialCounts: func() (map[string]int, error) {
			return map[string]int{"warm.example.com": 0}, nil
		},
		deadGrace: time.Hour,
	}

	runtime := ctrl.SystemStatus()
	if runtime.Status != services.StatusStarting || !runtime.Running {
		t.Fatalf("runtime = %#v, want starting while the pool warms up", runtime)
	}
	if runtime.Hosts[0].State != services.SystemHostStarting {
		t.Fatalf("host state = %q, want starting", runtime.Hosts[0].State)
	}
	if !strings.Contains(runtime.Detail, "1 starting") {
		t.Fatalf("Detail = %q, want it to mention the warm-up", runtime.Detail)
	}

	// The grace window is about status: it keeps a warming pool from being
	// reported as dead. An explicit restart ignores it and repairs the host
	// anyway, which TestCloudflareProxyRestartActsInsideTheGraceWindow covers.
	republished := 0
	ctrl.republish = func(expectedHost) error { republished++; return nil }
	if err := ctrl.RestartSystem(); err != nil {
		t.Fatalf("RestartSystem() error = %v", err)
	}
	if republished != 1 {
		t.Fatalf("republished %d hosts, want the warming host repaired on request", republished)
	}
}

// TestCloudflareProxyReportsDeadAfterGrace covers the other half: the grace
// window must not hide a publication that is really gone.
func TestCloudflareProxyReportsDeadAfterGrace(t *testing.T) {
	ctrl := &cloudFlareProxyController{
		expectedHosts: func() []expectedHost {
			return []expectedHost{{Host: "gone.example.com"}}
		},
		dialCounts: func() (map[string]int, error) {
			return map[string]int{"gone.example.com": 0}, nil
		},
		deadGrace: time.Millisecond,
	}

	// First observation starts the window.
	if runtime := ctrl.SystemStatus(); runtime.Status != services.StatusStarting {
		t.Fatalf("first status = %q, want starting", runtime.Status)
	}
	time.Sleep(5 * time.Millisecond)

	runtime := ctrl.SystemStatus()
	if runtime.Status != services.StatusError {
		t.Fatalf("runtime = %#v, want error once the grace window elapsed", runtime)
	}
	if runtime.Hosts[0].State != services.SystemHostDead {
		t.Fatalf("host state = %q, want dead", runtime.Hosts[0].State)
	}

	// A pool that comes back clears the window, so the next outage gets its own.
	ctrl.dialCounts = func() (map[string]int, error) {
		return map[string]int{"gone.example.com": 7}, nil
	}
	if runtime := ctrl.SystemStatus(); runtime.Status != services.StatusRunning {
		t.Fatalf("recovered status = %q, want running", runtime.Status)
	}
	ctrl.dialCounts = func() (map[string]int, error) {
		return map[string]int{"gone.example.com": 0}, nil
	}
	if runtime := ctrl.SystemStatus(); runtime.Status != services.StatusStarting {
		t.Fatalf("status after a new outage = %q, want a fresh grace window", runtime.Status)
	}
}

// TestCloudflareProxyRestartActsInsideTheGraceWindow guards the other
// half of the grace design: the window must not turn an explicit repair into a
// silent no-op. An operator clicking Restart right after an outage hits hosts
// that were only just observed without a pool, and those must still be
// republished.
func TestCloudflareProxyRestartActsInsideTheGraceWindow(t *testing.T) {
	var got []string
	ctrl := &cloudFlareProxyController{
		expectedHosts: func() []expectedHost {
			return []expectedHost{
				{Host: "live.example.com", ServiceID: "svc-live"},
				{Host: "warm.example.com", ServiceID: "svc-warm"},
			}
		},
		dialCounts: func() (map[string]int, error) {
			return map[string]int{"live.example.com": 9, "warm.example.com": 0}, nil
		},
		// A long grace keeps the dead host in the warm-up state.
		deadGrace: time.Hour,
		republish: func(host expectedHost) error {
			got = append(got, host.Host)
			return nil
		},
	}

	// First observation opens the grace window, so the host reads "starting".
	if runtime := ctrl.SystemStatus(); runtime.Status != services.StatusStarting {
		t.Fatalf("status = %q, want starting before the restart", runtime.Status)
	}

	if err := ctrl.RestartSystem(); err != nil {
		t.Fatalf("RestartSystem() error = %v", err)
	}
	if fmt.Sprint(got) != "[warm.example.com]" {
		t.Fatalf("republished = %v, want the warming host repaired and the live one left alone", got)
	}
}

func TestCloudflareProxyListedActionsIncludeRestart(t *testing.T) {
	stubEdge(t, map[string]int{}, nil)
	m := services.NewManagerFromDefinitions(nil)
	if err := Register(m); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	for _, status := range m.List() {
		if status.ID != CFProxyID {
			continue
		}
		if strings.Join(status.Actions, ",") != services.SystemActionRestart {
			t.Fatalf("Cloudflare Proxy actions = %v, want [restart]", status.Actions)
		}
		return
	}
	t.Fatal("Cloudflare Proxy was not registered")
}

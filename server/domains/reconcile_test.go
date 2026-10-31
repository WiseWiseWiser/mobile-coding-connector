package domains

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	cloudflareSettings "github.com/xhd2015/ai-critic/server/cloudflare"
)

func fakeReconciler(t *testing.T) *domainReconciler {
	t.Helper()
	r := newDomainReconciler()
	r.retryMin, r.retryMax, r.probeInterval = 10*time.Millisecond, 40*time.Millisecond, 15*time.Millisecond
	r.configured = func(string) (bool, error) { return true, nil }
	r.start = func(domain string, port int, log cloudflareSettings.LogFunc) (*cloudflareSettings.DomainTunnelStatus, error) {
		return &cloudflareSettings.DomainTunnelStatus{Status: "active", TunnelURL: "https://" + domain}, nil
	}
	r.retry = nil
	r.stop = func(string, cloudflareSettings.LogFunc) error { return nil }
	r.probe = func(string) (int, error) { return 200, nil }
	t.Cleanup(r.shutdown)
	return r
}

func waitDomainCondition(t *testing.T, fn func() bool) {
	t.Helper()
	deadline := time.NewTimer(3 * time.Second)
	defer deadline.Stop()
	tick := time.NewTicker(5 * time.Millisecond)
	defer tick.Stop()
	for {
		if fn() {
			return
		}
		select {
		case <-deadline.C:
			t.Fatal("condition was not reached")
		case <-tick.C:
		}
	}
}

func TestBootstrapConfigReadFailureIsRetried(t *testing.T) {
	r := fakeReconciler(t)
	var attempts atomic.Int32
	r.loadConfig = func() (*DomainsConfig, error) {
		if attempts.Add(1) < 3 {
			return nil, errors.New("temporarily unavailable")
		}
		return &DomainsConfig{Domains: []DomainEntry{{Domain: "config.example", Provider: ProviderCloudflare}}}, nil
	}
	r.startConfigured(23712)
	waitDomainCondition(t, func() bool { s, ok := r.status("config.example"); return ok && s.Status == "active" })
	if attempts.Load() != 3 {
		t.Fatalf("config attempts=%d", attempts.Load())
	}
}

func TestBootstrapStopBeforeConfigReadDoesNotRestart(t *testing.T) {
	r := fakeReconciler(t)
	ready := make(chan struct{})
	entered := make(chan struct{})
	r.loadConfig = func() (*DomainsConfig, error) {
		close(entered)
		<-ready
		return &DomainsConfig{Domains: []DomainEntry{{Domain: "stopped.example", Provider: ProviderCloudflare}}}, nil
	}
	var starts atomic.Int32
	r.start = func(string, int, cloudflareSettings.LogFunc) (*cloudflareSettings.DomainTunnelStatus, error) {
		starts.Add(1)
		return &cloudflareSettings.DomainTunnelStatus{Status: "active"}, nil
	}
	r.startConfigured(23712)
	<-entered
	if err := r.stopDomain("stopped.example", nil); err != nil {
		t.Fatal(err)
	}
	close(ready)
	select {
	case <-time.After(80 * time.Millisecond):
	}
	if starts.Load() != 0 {
		t.Fatal("loader restarted manually stopped domain")
	}
}

func TestBootstrapRetriesNetworkFailureThenBecomesReachable(t *testing.T) {
	r := fakeReconciler(t)
	var attempts atomic.Int32
	r.start = func(domain string, port int, log cloudflareSettings.LogFunc) (*cloudflareSettings.DomainTunnelStatus, error) {
		if attempts.Add(1) < 3 {
			return nil, fmt.Errorf("lookup api.cloudflare.com: no such host")
		}
		return &cloudflareSettings.DomainTunnelStatus{Status: "active"}, nil
	}
	r.startAutomatic("fresh.example", 23712)
	waitDomainCondition(t, func() bool { s, ok := r.status("fresh.example"); return ok && s.Status == "active" })
	if attempts.Load() != 3 {
		t.Fatalf("attempts=%d", attempts.Load())
	}
	logs := getHealthCheckLogs("fresh.example")
	if len(logs) == 0 {
		t.Fatal("startup failures were not logged for /service")
	}
}

func TestBootstrapStopWaitsForInflightStartAndCancelsRetries(t *testing.T) {
	r := fakeReconciler(t)
	started, release, stopped := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var calls atomic.Int32
	r.start = func(string, int, cloudflareSettings.LogFunc) (*cloudflareSettings.DomainTunnelStatus, error) {
		calls.Add(1)
		close(started)
		<-release
		return &cloudflareSettings.DomainTunnelStatus{Status: "active"}, nil
	}
	r.stop = func(string, cloudflareSettings.LogFunc) error { close(stopped); return nil }
	r.startAutomatic("stop.example", 23712)
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("start did not run")
	}
	done := make(chan error, 1)
	go func() { done <- r.stopDomain("stop.example", nil) }()
	select {
	case <-stopped:
		t.Fatal("stop raced ahead of in-flight start")
	case <-time.After(40 * time.Millisecond):
	}
	close(release)
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("stop blocked")
	}
	if calls.Load() != 1 {
		t.Fatalf("calls=%d", calls.Load())
	}
	select {
	case <-stopped:
	default:
		t.Fatal("backend was not stopped")
	}
	if s, _ := r.status("stop.example"); s.Status != "stopped" {
		t.Fatalf("status=%+v", s)
	}
	select {
	case <-time.After(100 * time.Millisecond):
	}
	if calls.Load() != 1 {
		t.Fatal("canceled worker restarted stopped domain")
	}
}

func TestBootstrapStopFailureReportsErrorWithoutRetry(t *testing.T) {
	r := fakeReconciler(t)
	var starts atomic.Int32
	r.start = func(string, int, cloudflareSettings.LogFunc) (*cloudflareSettings.DomainTunnelStatus, error) {
		starts.Add(1)
		return &cloudflareSettings.DomainTunnelStatus{Status: "active"}, nil
	}
	r.stop = func(string, cloudflareSettings.LogFunc) error { return errors.New("detach failed") }
	r.startAutomatic("stop-error.example", 23712)
	waitDomainCondition(t, func() bool { s, _ := r.status("stop-error.example"); return s.Status == "active" })
	if err := r.stopDomain("stop-error.example", nil); err == nil {
		t.Fatal("stop error was lost")
	}
	status, _ := r.status("stop-error.example")
	if status.Status != "error" || !strings.Contains(status.Error, "detach failed") {
		t.Fatalf("status=%+v", status)
	}
	select {
	case <-time.After(80 * time.Millisecond):
	}
	if starts.Load() != 1 {
		t.Fatal("failed stop unexpectedly restarted")
	}
}

func TestBootstrapDoesNotDuplicateConcurrentAutoStart(t *testing.T) {
	r := fakeReconciler(t)
	entered, release := make(chan struct{}), make(chan struct{})
	var calls atomic.Int32
	r.start = func(string, int, cloudflareSettings.LogFunc) (*cloudflareSettings.DomainTunnelStatus, error) {
		if calls.Add(1) == 1 {
			close(entered)
		}
		<-release
		return &cloudflareSettings.DomainTunnelStatus{Status: "active"}, nil
	}
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); r.startAutomatic("once.example", 23712) }()
	}
	wg.Wait()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("start did not run")
	}
	if calls.Load() != 1 {
		t.Fatalf("concurrent start calls=%d", calls.Load())
	}
	close(release)
	waitDomainCondition(t, func() bool { s, _ := r.status("once.example"); return s.Status == "active" })
}

func TestBootstrapRecoveryBypassesStaleBackendStatus(t *testing.T) {
	r := fakeReconciler(t)
	var healthy atomic.Bool
	var retried atomic.Int32
	r.probe = func(string) (int, error) {
		if healthy.Load() {
			return 200, nil
		}
		return 502, nil
	}
	r.retry = func(string, int, cloudflareSettings.LogFunc) (*cloudflareSettings.DomainTunnelStatus, error) {
		retried.Add(1)
		healthy.Store(true)
		return &cloudflareSettings.DomainTunnelStatus{Status: "active"}, nil
	}
	r.startAutomatic("stale.example", 23712)
	waitDomainCondition(t, func() bool { s, _ := r.status("stale.example"); return s.Status == "active" && retried.Load() == 1 })
}

func TestBootstrapConnectingDoesNotKillConnector(t *testing.T) {
	r := fakeReconciler(t)
	r.probeInterval = 20 * time.Millisecond
	r.connectingBeforeRetry = 10
	var starts, stops atomic.Int32
	r.start = func(string, int, cloudflareSettings.LogFunc) (*cloudflareSettings.DomainTunnelStatus, error) {
		starts.Add(1)
		return &cloudflareSettings.DomainTunnelStatus{Status: "active"}, nil
	}
	r.stop = func(string, cloudflareSettings.LogFunc) error { stops.Add(1); return nil }
	r.probe = func(string) (int, error) { return 530, nil }
	r.startAutomatic("connecting.example", 23712)
	waitDomainCondition(t, func() bool {
		s, ok := r.status("connecting.example")
		return ok && s.Status == "connecting" && starts.Load() == 1
	})
	select {
	case <-time.After(80 * time.Millisecond):
	}
	if starts.Load() != 1 || stops.Load() != 0 {
		t.Fatalf("connecting caused start=%d stop=%d", starts.Load(), stops.Load())
	}
}

func TestBootstrapManualFailureRetriesAndRemovedConfigStops(t *testing.T) {
	r := fakeReconciler(t)
	var exists atomic.Bool
	exists.Store(true)
	r.configured = func(string) (bool, error) { return exists.Load(), nil }
	var starts atomic.Int32
	r.start = func(string, int, cloudflareSettings.LogFunc) (*cloudflareSettings.DomainTunnelStatus, error) {
		if starts.Add(1) == 1 {
			return nil, errors.New("offline")
		}
		return &cloudflareSettings.DomainTunnelStatus{Status: "active"}, nil
	}
	if _, err := r.startManual("manual.example", 23712, nil); err == nil {
		t.Fatal("expected first manual failure")
	}
	waitDomainCondition(t, func() bool { s, _ := r.status("manual.example"); return s.Status == "active" })
	exists.Store(false)
	waitDomainCondition(t, func() bool { s, _ := r.status("manual.example"); return s.Status == "stopped" })
}

func TestBootstrapBackendRecoveryPreservesSiblingRoutes(t *testing.T) {
	r := fakeReconciler(t)
	var mu sync.Mutex
	routes := map[string]string{"sibling.example": "http://127.0.0.1:3080"}
	var calls atomic.Int32
	r.start = func(domain string, port int, log cloudflareSettings.LogFunc) (*cloudflareSettings.DomainTunnelStatus, error) {
		if calls.Add(1) == 1 {
			return nil, errors.New("offline")
		}
		mu.Lock()
		routes[domain] = fmt.Sprintf("http://127.0.0.1:%d", port)
		mu.Unlock()
		return &cloudflareSettings.DomainTunnelStatus{Status: "active"}, nil
	}
	r.startAutomatic("bootstrap.example", 23712)
	waitDomainCondition(t, func() bool { s, _ := r.status("bootstrap.example"); return s.Status == "active" })
	mu.Lock()
	defer mu.Unlock()
	if routes["sibling.example"] != "http://127.0.0.1:3080" || routes["bootstrap.example"] != "http://127.0.0.1:23712" {
		t.Fatalf("routes=%v", routes)
	}
}

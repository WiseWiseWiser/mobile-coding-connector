package gomodrelay

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func dialerWithTimeout(d time.Duration) func(ctx context.Context, network, addr string) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: d}
	return dialer.DialContext
}

// slow upstream that accepts but never responds (pre-header hang)
func hangingUpstream() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {} // never respond; test closes the server
	}))
}

func TestRelayForwardsWhenUp(t *testing.T) {
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"Version":"v1.6.0"}`))
	}))
	defer up.Close()

	r := NewRelay(Config{Port: 0, Upstream: up.URL, DialTimeoutMS: 1000, DownRecheckMS: 100}, nil)
	defer r.Close()
	if ok, _, err := r.ProbeUp(); !ok {
		t.Fatalf("probe: %v", err)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/github.com/google/uuid/@v/v1.6.0.info", nil))
	if w.Code != 200 || w.Body.String() != `{"Version":"v1.6.0"}` {
		t.Fatalf("forward: %d %q", w.Code, w.Body.String())
	}
}

func TestRelayFast404WhenUpstreamDown(t *testing.T) {
	// reserve a port, then close the server -> connection refused
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := up.URL
	up.Close()

	var warns atomic.Int32
	r := NewRelay(Config{Port: 0, Upstream: url, DialTimeoutMS: 500, DownRecheckMS: 10_000}, func(f string, a ...any) {
		warns.Add(1)
	})
	defer r.Close()
	if ok, _, err := r.ProbeUp(); ok {
		t.Fatalf("probe should fail on closed upstream: %v", err)
	}

	start := time.Now()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/github.com/google/uuid/@v/v1.6.0.info", nil))
	elapsed := time.Since(start)
	if w.Code != 404 {
		t.Fatalf("down: got %d want 404", w.Code)
	}
	if elapsed > 100*time.Millisecond {
		t.Fatalf("down 404 took %s, want instant", elapsed)
	}
	if warns.Load() == 0 {
		t.Fatal("expected a warning log line on 404 fallback")
	}
	// second request also instant-404 (health cache)
	w2 := httptest.NewRecorder()
	start = time.Now()
	r.ServeHTTP(w2, httptest.NewRequest("GET", "/x/@v/v1.info", nil))
	if w2.Code != 404 || time.Since(start) > 100*time.Millisecond {
		t.Fatalf("second down request: %d in %s", w2.Code, time.Since(start))
	}
}

func TestRelayPreHeaderHang404(t *testing.T) {
	up := hangingUpstream()
	// Close in a goroutine: httptest.Server.Close waits for active handlers,
	// and hangingUpstream's handler never returns.
	t.Cleanup(func() { go up.Close() })

	r := NewRelay(Config{Port: 0, Upstream: up.URL, DialTimeoutMS: 1000, DownRecheckMS: 10_000}, nil)
	// shrink response-header timeout via a wrapped transport for test speed
	r.client = &http.Client{
		Transport: &http.Transport{
			DialContext:           dialerWithTimeout(500 * time.Millisecond),
			ResponseHeaderTimeout: 150 * time.Millisecond,
		},
	}
	defer r.Close()
	if ok, _, err := r.ProbeUp(); !ok {
		t.Fatalf("probe: %v", err)
	}
	start := time.Now()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/some/@v/v1.info", nil))
	elapsed := time.Since(start)
	if w.Code != 404 {
		t.Fatalf("pre-header hang: got %d want 404", w.Code)
	}
	if elapsed > 2*time.Second {
		t.Fatalf("pre-header hang took %s, want <=~150ms+slack", elapsed)
	}
}

func TestRelayRecoversAfterUpstreamReturns(t *testing.T) {
	closed := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	closedURL := closed.URL
	closed.Close()

	relay := NewRelay(Config{Port: 0, Upstream: closedURL, DialTimeoutMS: 200, DownRecheckMS: 30}, nil)
	defer relay.Close()
	if ok, _, err := relay.ProbeUp(); ok {
		t.Fatalf("probe should fail: %v", err)
	}

	// upstream comes back
	srvUp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok-body"))
	}))
	defer srvUp.Close()
	relay.cfg.Upstream = srvUp.URL

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		relay.mu.Lock()
		upNow := relay.up
		relay.mu.Unlock()
		if upNow {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	w := httptest.NewRecorder()
	relay.ServeHTTP(w, httptest.NewRequest("GET", "/mod/@v/v1.info", nil))
	if w.Code != 200 || w.Body.String() != "ok-body" {
		t.Fatalf("recovery: %d %q", w.Code, w.Body.String())
	}
}

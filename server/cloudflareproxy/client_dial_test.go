package cloudflareproxy

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// dialRejector wraps a real edge server so a test can make /dial fail on
// demand, leaving every other route intact.
type dialRejector struct {
	edge   *Server
	status atomic.Int32 // 0 = serve normally
	dials  atomic.Int64
}

func (d *dialRejector) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/dial/") {
		d.dials.Add(1)
		if code := d.status.Load(); code != 0 {
			http.Error(w, http.StatusText(int(code)), int(code))
			return
		}
	}
	d.edge.ServeHTTP(w, r)
}

// newDialRejector starts a real edge behind a switch that can reject dials.
func newDialRejector(t *testing.T, token string) (*dialRejector, *httptest.Server) {
	t.Helper()
	edge, _ := testServer(t, token)
	rej := &dialRejector{edge: edge}
	ts := httptest.NewServer(rej)
	t.Cleanup(ts.Close)
	return rej, ts
}

// consumePooledDial uses the single pooled socket so the worker re-dials and
// meets whatever the rejector currently answers.
func consumePooledDial(t *testing.T, ts *httptest.Server, host string) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, ts.URL+"/ping", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = host
	resp, err := http.DefaultClient.Do(req)
	if err == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

// publishUntilConnected starts a pool and waits for its first dial.
func publishUntilConnected(t *testing.T, ctx context.Context, ts *httptest.Server, host string) <-chan error {
	t.Helper()
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "ok")
	}))
	t.Cleanup(origin.Close)

	cli := &APIClient{BaseURL: ts.URL, Token: "secret"}
	var out syncBuffer
	done := make(chan error, 1)
	go func() {
		done <- cli.Publish(ctx, PublishOpts{
			Hostname: host,
			Forward:  origin.URL,
			Pool:     1,
			Stdout:   &out,
			Stderr:   io.Discard,
			OnReady:  func() {},
		})
	}()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(out.String(), "connected") {
			return done
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("publish never connected:\n%s", out.String())
	return done
}

// TestGatewayErrorDoesNotKillThePool is the regression test for the outage:
// a non-101 response from an intermediary (a Cloudflare 502/503) used to be
// classified as a closed mapping, which cancelled the hostname's whole pool and
// silently ended its publish session. The pool must ride it out and retry.
func TestGatewayErrorDoesNotKillThePool(t *testing.T) {
	rej, ts := newDialRejector(t, "secret")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := publishUntilConnected(t, ctx, ts, "gw.example.com")

	// Now every further dial meets a gateway error.
	rej.status.Store(http.StatusBadGateway)
	consumePooledDial(t, ts, "gw.example.com")

	select {
	case err := <-done:
		t.Fatalf("a gateway error ended the publish session: %v", err)
	case <-time.After(2 * time.Second):
	}
	if rej.dials.Load() < 2 {
		t.Fatalf("worker did not retry the dial (dials=%d)", rej.dials.Load())
	}
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("publish did not stop after cancellation")
	}
}

// TestUnauthorizedDialEndsTheSession keeps the other half honest: 401 means the
// mapping id is gone or its token is wrong, so the dial URL can never serve
// again and the session must end instead of retrying forever.
func TestUnauthorizedDialEndsTheSession(t *testing.T) {
	rej, ts := newDialRejector(t, "secret")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := publishUntilConnected(t, ctx, ts, "gone.example.com")

	rej.status.Store(http.StatusUnauthorized)
	consumePooledDial(t, ts, "gone.example.com")

	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "mapping closed") {
			t.Fatalf("publish ended with %v, want a mapping-closed error", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("a 401 dial did not end the session")
	}
}

func TestDialFailureClassification(t *testing.T) {
	cases := []struct {
		name string
		code int
		want bool
	}{
		{"unauthorized is terminal", http.StatusUnauthorized, true},
		{"bad gateway retries", http.StatusBadGateway, false},
		{"service unavailable retries", http.StatusServiceUnavailable, false},
		{"gateway timeout retries", http.StatusGatewayTimeout, false},
		{"internal error retries", http.StatusInternalServerError, false},
		{"not found retries", http.StatusNotFound, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := &http.Response{StatusCode: tc.code}
			err := dialFailure(resp, io.ErrUnexpectedEOF)
			if got := isMappingClosed(err); got != tc.want {
				t.Fatalf("isMappingClosed(%d) = %v, want %v", tc.code, got, tc.want)
			}
		})
	}
	if got := isMappingClosed(dialFailure(nil, io.ErrUnexpectedEOF)); got {
		t.Fatal("a transport failure with no response must stay retryable")
	}
}

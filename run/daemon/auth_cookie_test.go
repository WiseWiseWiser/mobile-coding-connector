package daemon

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xhd2015/ai-critic/server/config"
)

type cookieAuthRoundTripFunc func(*http.Request) (*http.Response, error)

func (fn cookieAuthRoundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return fn(r) }

func TestManagementBearerHelper(t *testing.T) {
	if os.Getenv("AI_CRITIC_MANAGEMENT_BEARER_HELPER") != "1" {
		return
	}
	home, err := filepath.EvalSymlinks(os.Getenv("HOME"))
	if err != nil {
		t.Fatal(err)
	}
	file, err := filepath.Abs(config.CredentialsFile)
	if err != nil || !strings.HasPrefix(file, home+string(filepath.Separator)) {
		t.Fatalf("fixture would write outside isolated HOME: %s", file)
	}
	for _, fixed := range []string{"/root/.ai-critic/server-credentials", "/root/.config/ai-critic/server-credentials"} {
		if _, err := os.Stat(fixed); err == nil {
			t.Skip("isolated fixture must not read an existing root credential file")
		}
	}
	if err := os.MkdirAll(filepath.Dir(file), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte("fixture-management-token\n"), 0600); err != nil {
		t.Fatal(err)
	}
	fallback := filepath.Join(home, ".ai-critic", "server-credentials")
	if fallback != file {
		if err := os.MkdirAll(filepath.Dir(fallback), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fallback, []byte("fixture-management-token\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	seen := make(map[string]int)
	// This override is confined to the helper process: no real HTTP requests,
	// live management ports, global stdio/env swapping, or actual service actions.
	http.DefaultTransport = cookieAuthRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("Authorization") != "Bearer fixture-management-token" || r.Header.Get("Cookie") != "" {
			return nil, fmt.Errorf("management request must authenticate with Bearer, not a cookie")
		}
		seen[r.URL.Path]++
		body := "ok"
		if r.URL.Path == "/api/domains" {
			body = `{"domains":[]}`
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})
	if !CallShutdownEndpoint() {
		t.Fatal("shutdown authentication failed")
	}
	if !callExecRestartEndpoint() {
		t.Fatal("exec-restart authentication failed")
	}
	server := &HTTPServer{}
	if result := server.fixStaleTunnels(23712); result["status"] != "ok" {
		t.Fatalf("tunnel list authentication failed: %v", result)
	}
	if !server.fixDomainTunnel("fixture.example", 23712, "fixture-management-token") {
		t.Fatal("tunnel stop/start authentication failed")
	}
	for _, path := range []string{"/api/shutdown", "/api/server/exec-restart", "/api/domains", "/api/domains/tunnel/stop", "/api/domains/tunnel/start"} {
		if seen[path] != 1 {
			t.Fatalf("missing management call %s", path)
		}
	}
}

func TestManagementHTTPUsesBearerIndependentOfCookieName(t *testing.T) {
	home := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestManagementBearerHelper$", "-test.v")
	cmd.Dir = home
	cmd.Env = append(os.Environ(), "AI_CRITIC_MANAGEMENT_BEARER_HELPER=1", "HOME="+home)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("isolated management calls: %v\n%s", err, out)
	}
	if strings.Contains(string(out), "--- SKIP") {
		t.Skip("root credential store prevents an isolated fixture")
	}
}

func TestManagedServerArgsPreserveCookieName(t *testing.T) {
	got := buildManagedServerArgs(49192, []string{"--port", "12345", "--auth-cookie-name", "pi-agent-token"})
	if strings.Join(got, " ") != "--port 49192 --auth-cookie-name pi-agent-token" {
		t.Fatalf("child/restart args lost cookie configuration: %v", got)
	}
}

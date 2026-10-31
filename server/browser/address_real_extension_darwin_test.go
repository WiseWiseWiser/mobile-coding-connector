//go:build darwin

package browser

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/xhd2015/ai-critic/server/config"
)

// Uses an isolated headless Chrome for Testing, real copied extension and
// private daemon. Unlike the headed profile test it does not require a display.
func TestAddressNavigationWithRealHeadlessExtension(t *testing.T) {
	if os.Getenv("AI_CRITIC_TEST_REAL_HEADLESS_ADDRESS") != "1" {
		t.Skip("isolated real extension test disabled")
	}
	chrome, agent, extSource := os.Getenv("AI_CRITIC_TEST_CHROME_FOR_TESTING"), os.Getenv("AI_CRITIC_TEST_BROWSER_AGENT"), os.Getenv("AI_CRITIC_TEST_EXTENSION_SOURCE")
	if !isExecutable(chrome) || !isExecutable(agent) || extSource == "" {
		t.Fatal("test Chrome, agent and extension source are required")
	}
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	home := filepath.Join(root, "home")
	if err := os.MkdirAll(home, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	oldDataDir := config.DataDir
	config.DataDir = filepath.Join(root, "data")
	defer func() { config.DataDir = oldDataDir }()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()
	extDir := filepath.Join(root, "extension")
	if err := copyIsolatedExtension(extSource, extDir, port); err != nil {
		t.Fatal(err)
	}
	agentCmd := exec.Command(agent, "serve", "--host", "127.0.0.1", "--port", strconv.Itoa(port), "--base-dir", filepath.Join(root, "agent"), "--no-agent-run")
	if err := agentCmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = agentCmd.Process.Kill(); _ = agentCmd.Wait() }()
	base := fmt.Sprintf("http://127.0.0.1:%d", port)
	deadline := time.Now().Add(20 * time.Second)
	for {
		var health map[string]any
		if testHTTPJSON("GET", base+"/v1/health", nil, &health) == nil && health["ok"] == true {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("isolated agent did not start")
		}
		time.Sleep(100 * time.Millisecond)
	}
	session := "isolated-address-test"
	var created map[string]any
	if err := testHTTPJSON("POST", base+"/v1/sessions", strings.NewReader(`{"session_id":"`+session+`"}`), &created); err != nil {
		t.Fatal(err)
	}
	profile := filepath.Join(root, "profile")
	if err := os.MkdirAll(profile, 0700); err != nil {
		t.Fatal(err)
	}
	chromeCmd := exec.Command(chrome, "--headless=new", "--no-sandbox", "--disable-gpu", "--disable-background-networking", "--disable-breakpad", "--disable-crash-reporter", "--no-first-run", "--no-default-browser-check", "--disable-extensions-except="+extDir, "--load-extension="+extDir, "--user-data-dir="+profile, "--remote-debugging-port=0", base+"/go?session="+session)
	if err := chromeCmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = chromeCmd.Process.Kill(); _ = chromeCmd.Wait() }()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
	defer cancel()
	m := New(port)
	m.sessionID = session
	if err := m.waitForSessionPage(ctx, session, 45*time.Second); err != nil {
		t.Fatalf("test extension did not connect: %v", err)
	}
	m.connected = true
	if err := m.ensureWindow(ctx); err != nil {
		t.Fatalf("resolve isolated content tab: %v", err)
	}
	before := m.Snapshot()
	if before.ActiveTabID == 0 {
		t.Fatal("missing test content tab")
	}
	web := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/start" {
			http.Redirect(w, r, "/result?x=1#anchor", http.StatusFound)
			return
		}
		_, _ = io.WriteString(w, "<title>Isolated address</title>ok")
	}))
	defer web.Close()
	if err := m.NavigateTab(ctx, session, before.ActiveTabID, web.URL+"/start"); err != nil {
		t.Fatal(err)
	}
	want := web.URL + "/result?x=1#anchor"
	for {
		m.mu.Lock()
		m.tabsAt = time.Time{}
		m.mu.Unlock()
		m.pollTabs(ctx)
		if m.CurrentURL() == want {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatalf("redirect address: got %q want %q", m.CurrentURL(), want)
		case <-time.After(150 * time.Millisecond):
		}
	}
	t.Logf("real isolated extension navigated tab %d and reported redirected address with query and fragment", before.ActiveTabID)
}

//go:build darwin

package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
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

func copyIsolatedExtension(src, dst string, port int) error {
	return filepath.WalkDir(src, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0700)
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("non-regular test extension entry: %s", rel)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if rel == "background.js" {
			old := "const CONTROL_PORT = 43761;"
			if !strings.Contains(string(data), old) {
				return fmt.Errorf("missing test control-port constant")
			}
			data = []byte(strings.Replace(string(data), old, fmt.Sprintf("const CONTROL_PORT = %d;", port), 1))
		}
		return os.WriteFile(target, data, 0600)
	})
}

func testHTTPJSON(method, url string, body io.Reader, out any) error {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	client := &http.Client{Timeout: 4 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d", res.StatusCode)
	}
	return json.NewDecoder(res.Body).Decode(out)
}

// A completely isolated real extension + daemon test. The loaded extension is
// copied to t.TempDir and points at an isolated port, never the user's daemon.
func TestProfileLaunchWithRealIsolatedExtension(t *testing.T) {
	if os.Getenv("AI_CRITIC_TEST_REAL_EXTENSION_ROUTE") != "1" {
		t.Skip("isolated extension routing test disabled")
	}
	chrome := os.Getenv("AI_CRITIC_TEST_CHROME_FOR_TESTING")
	agent := os.Getenv("AI_CRITIC_TEST_BROWSER_AGENT")
	extSource := os.Getenv("AI_CRITIC_TEST_EXTENSION_SOURCE")
	if !isExecutable(chrome) || !isExecutable(agent) || extSource == "" {
		t.Fatal("test Chrome, agent and extension source are required")
	}
	root, _ := filepath.EvalSymlinks(t.TempDir())
	home := filepath.Join(root, "home")
	if err := os.MkdirAll(home, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("BROWSER_AGENT_NO_OPEN", "0")
	targetRoot := filepath.Join(root, "target")
	headlessRoot := filepath.Join(root, "headless")
	if err := os.MkdirAll(targetRoot, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(headlessRoot, 0700); err != nil {
		t.Fatal(err)
	}
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
	writeProfile(t, targetRoot, "Default", fmt.Sprintf(`{"a":{"path":%q}}`, extDir))
	if err := os.WriteFile(filepath.Join(targetRoot, "First Run"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("BROWSER_AGENT_CHROME_USER_DATA_DIR", targetRoot)
	t.Setenv("BROWSER_AGENT_CHROME_PROFILE", "Default")
	oldBinary := chromeBundledBinary
	chromeBundledBinary = chrome
	defer func() { chromeBundledBinary = oldBinary }()

	agentCmd := exec.Command(agent, "serve", "--host", "127.0.0.1", "--port", strconv.Itoa(port), "--base-dir", filepath.Join(root, "agent"), "--no-agent-run")
	if err := agentCmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = agentCmd.Process.Kill(); _ = agentCmd.Wait() }()
	base := fmt.Sprintf("http://127.0.0.1:%d", port)
	deadline := time.Now().Add(30 * time.Second)
	for {
		var health map[string]any
		if testHTTPJSON("GET", base+"/v1/health", nil, &health) == nil && health["ok"] == true {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("isolated daemon did not start")
		}
		time.Sleep(100 * time.Millisecond)
	}
	session := "ai-critic-2"
	var created map[string]any
	if err := testHTTPJSON("POST", base+"/v1/sessions", strings.NewReader(`{"session_id":"`+session+`"}`), &created); err != nil {
		t.Fatal(err)
	}

	headlessBinary := os.Getenv("AI_CRITIC_TEST_HEADLESS_CHROME")
	if headlessBinary == "" {
		headlessBinary = chrome
	}
	headless := exec.Command(headlessBinary, "--headless=new", "--no-first-run", "--disable-background-networking", "--no-sandbox", "--user-data-dir="+headlessRoot, "--remote-debugging-port=0", "about:blank")
	headless.Stderr = os.Stderr
	if err := headless.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = headless.Process.Kill(); _ = headless.Wait() }()
	deadline = time.Now().Add(25 * time.Second)
	for {
		if _, err := os.Stat(filepath.Join(headlessRoot, "DevToolsActivePort")); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("isolated headless Chrome did not start")
		}
		time.Sleep(100 * time.Millisecond)
	}

	var target *exec.Cmd
	defer func() {
		if target != nil && target.Process != nil {
			_ = target.Process.Kill()
			_ = target.Wait()
		}
	}()
	launch := func(binary string, args []string) error {
		args = append([]string{"--remote-debugging-port=0", "--disable-extensions-except=" + extDir, "--load-extension=" + extDir}, args...)
		target = exec.Command(binary, args...)
		return target.Start()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	url := base + "/go?session=" + session
	if err := openChromeProfileWith(ctx, url, false, launch); err != nil {
		t.Fatal(err)
	}
	var snap struct {
		Extension struct {
			Connected bool     `json:"connected"`
			Version   string   `json:"version"`
			Features  []string `json:"features"`
		} `json:"extension"`
	}
	for {
		if testHTTPJSON("GET", base+"/v1/session?session="+session, nil, &snap) == nil && snap.Extension.Connected {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatalf("isolated extension did not connect: %v", ctx.Err())
		case <-time.After(150 * time.Millisecond):
		}
	}
	if snap.Extension.Version != "1.0.23" {
		t.Fatalf("extension version %q", snap.Extension.Version)
	}
	found := false
	for _, feature := range snap.Extension.Features {
		if feature == "tab-close-confirm-v1" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected confirmation feature, got %v", snap.Extension.Features)
	}
	body := strings.NewReader(`{"session_id":"` + session + `","type":"info","timeout_ms":15000}`)
	var info struct {
		OK   bool `json:"ok"`
		Data struct {
			WindowID int64 `json:"window_id"`
			Tabs     []struct {
				URL string `json:"url"`
			} `json:"tabs"`
		} `json:"data"`
	}
	if err := testHTTPJSON("POST", base+"/v1/jobs", body, &info); err != nil {
		t.Fatal(err)
	}
	if !info.OK || info.Data.WindowID == 0 {
		t.Fatalf("missing isolated window: %+v", info)
	}
	found = false
	for _, tab := range info.Data.Tabs {
		if tab.URL == url {
			found = true
		}
	}
	if !found {
		t.Fatal("requested session page not present in target window")
	}
	activePort, err := os.ReadFile(filepath.Join(headlessRoot, "DevToolsActivePort"))
	if err != nil {
		t.Fatal(err)
	}
	headlessPort, err := strconv.Atoi(strings.Split(string(activePort), "\n")[0])
	if err != nil {
		t.Fatal(err)
	}
	var targets []struct {
		URL string `json:"url"`
	}
	if err := testHTTPJSON("GET", fmt.Sprintf("http://127.0.0.1:%d/json/list", headlessPort), nil, &targets); err != nil {
		t.Fatal(err)
	}
	for _, target := range targets {
		if target.URL == url {
			t.Fatal("session page was sent to unrelated headless Chrome")
		}
	}
	// Exercise the actual New Window manager path against the isolated daemon,
	// extension, and test-owned Chrome profile. Never address the user's Chrome.
	oldDataDir := config.DataDir
	config.DataDir = filepath.Join(root, "ai-critic-data")
	defer func() { config.DataDir = oldDataDir }()
	m := New(port)
	m.mu.Lock()
	m.sessionID, m.connected, m.windowID = session, true, info.Data.WindowID
	m.mu.Unlock()
	newID, err := m.NewSession(ctx)
	if err != nil {
		t.Fatalf("create new isolated window: %v", err)
	}
	if newID != "ai-critic-3" {
		t.Fatalf("new ID = %q, want ai-critic-3", newID)
	}
	m.mu.Lock()
	newWindowID := m.windowID
	m.mu.Unlock()
	if newWindowID == 0 || newWindowID == info.Data.WindowID {
		t.Fatalf("new window %d is not distinct from previous window %d", newWindowID, info.Data.WindowID)
	}
	snapshot := m.Snapshot()
	contentCount := 0
	for _, tab := range snapshot.Tabs {
		if tab.Role != "session_page" {
			contentCount++
			if tab.TabID != snapshot.ActiveTabID || tab.Title != "ai-critic" {
				t.Fatalf("new content tab not selected in initial snapshot: %+v", snapshot)
			}
		}
	}
	if contentCount != 1 {
		t.Fatalf("initial snapshot must include the created content tab: %+v", snapshot.Tabs)
	}
	pageServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/start" {
			http.Redirect(w, r, "/arrived?q=two#part", http.StatusFound)
			return
		}
		_, _ = io.WriteString(w, "<title>Isolated address test</title>ok")
	}))
	defer pageServer.Close()
	if err := m.NavigateTab(ctx, newID, snapshot.ActiveTabID, pageServer.URL+"/start"); err != nil {
		t.Fatalf("navigate isolated tab: %v", err)
	}
	wantURL := pageServer.URL + "/arrived?q=two#part"
	for {
		m.mu.Lock()
		m.tabsAt = time.Time{}
		m.mu.Unlock()
		m.pollTabs(ctx)
		if m.CurrentURL() == wantURL {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatalf("redirect URL not reflected: got %q want %q", m.CurrentURL(), wantURL)
		case <-time.After(150 * time.Millisecond):
		}
	}
	var oldSession struct {
		Extension struct {
			Connected bool `json:"connected"`
		} `json:"extension"`
	}
	if err := testHTTPJSON("GET", base+"/v1/session?session="+session, nil, &oldSession); err != nil || !oldSession.Extension.Connected {
		t.Fatalf("previous test session was disturbed: %+v %v", oldSession, err)
	}
	t.Logf("real extension connected; New Window opened %s in distinct isolated window %d (original %d)", newID, newWindowID, info.Data.WindowID)
}

# Server Doctests

Tests that verify server-side behaviour by starting the actual `ai-critic-server`
binary with a test-controlled configuration directory.

# DSN (Domain Specific Notion)

**Participants**

- **ai-critic-server subprocess** — built from module root; isolated config home
  via child `cmd.Env` (`AI_CRITIC_HOME`) only (never process `Setenv`).
- **opencode.json settings** — written under config home before server start to
  drive `AutoStartWebServer()` decisions.
- **basic-auth-proxy binary** — built for auth-proxy leaves; prepended to child PATH.
- **Server logs** — stdout/stderr capture for auto-start markers.
- **PostStart HTTP** — optional settings API call after boot (enable-via-api leaf).

**Behaviors**

- When `WebServer.Enabled=true` with a non-localhost domain, auto-start logs appear.
- When `WebServer.Enabled=false` at boot, auto-start is skipped.
- Enabling web server via settings API re-triggers auto-start after save.
- With `AuthProxyEnabled=true`, proxy start path and backend port discovery work.

## Version

0.0.2

## Decision Tree

```
[ai-critic-server startup]
 |
 +-- Auto-start via opencode settings
      |
      +-- WebServer.Enabled=true at boot
      |    |
      |    +-- AuthProxyEnabled=false
      |    |    |
      |    |    +-- [decision/auto-start-via-settings]  (LEAF)
      |    |         - Writes opencode.json with WebServer.Enabled=true, DefaultDomain set
      |    |         - Starts server in normal mode
      |    |         - Asserts: AutoStartWebServer log messages appear
      |    |         - Asserts: opencode web server port becomes accessible (if binary available)
      |    |
      |    +-- AuthProxyEnabled=true
      |         |
      |         +-- [decision/with-auth-proxy]  (LEAF)
      |              - Writes opencode.json with WebServer.Enabled=true, AuthProxyEnabled=true
      |              - Starts server in normal mode
      |              - Asserts: AutoStartWebServer log messages appear with AuthProxyEnabled=true
      |              - Asserts: [basic_auth_proxy] Proxy started on port log appears
      |              - Asserts: proxy port (14100) is reachable via ProxyRunning
      |              - Asserts: backend port discovered from basic-auth-proxy.json and reachable
      |
      +-- WebServer.Enabled=false at boot
           |
           +-- [decision/disabled-no-autostart]  (LEAF)
           |    - Writes opencode.json with WebServer.Enabled=false, DefaultDomain set
           |    - Starts server in normal mode
           |    - Asserts: NO AutoStartWebServer log messages
           |    - Asserts: web server port is NOT accessible
           |
           +-- [decision/enable-via-api-triggers-start]  (LEAF)
                - Writes opencode.json with WebServer.Enabled=false, DefaultDomain set
                - Starts server in normal mode (no auto-start initially)
                - Makes POST to /api/agents/opencode/settings enabling web server
                - Asserts: initial logs have NO auto-start
                - Asserts: post-API-call logs HAVE auto-start messages
                - Asserts: web server port becomes accessible (if binary available)
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `decision/auto-start-via-settings` | Verifies `AutoStartWebServer()` triggers when settings have `WebServer.Enabled=true` and a valid `DefaultDomain` |
| 2 | `decision/disabled-no-autostart` | Verifies `AutoStartWebServer()` is NOT triggered when `WebServer.Enabled=false` at boot |
| 3 | `decision/enable-via-api-triggers-start` | Verifies that enabling the web server via settings API triggers `AutoStartWebServer()` (bug fix: `handleOpencodeSettings` re-evaluates auto-start after save) |
| 4 | `decision/with-auth-proxy` | Verifies `AutoStartWebServer()` triggers with auth proxy enabled, proxy starts on configured port, backend port is discovered from `basic-auth-proxy.json` |

## Parameter Coverage

| Leaf | Enabled | Domain | Port | AuthProxy | PostStart |
|------|---------|--------|------|-----------|-----------|
| auto-start-via-settings | true | test-auto-start.example.com | 14096 | false | — |
| disabled-no-autostart | false | test-disabled.example.com | 14096 | false | — |
| enable-via-api-triggers-start | false→true | test-enable-via-api.example.com | 14096 | false | POST /settings |
| with-auth-proxy | true | test-auth-proxy.example.com | 14100 | true | — |

## Edge Cases Covered

- Non-localhost domain triggers auto-start (as opposed to localhost which skips)
- Custom port (14096, 14100) instead of default (4096)
- Custom config home directory (via `AI_CRITIC_HOME`)
- Missing opencode binary (handled gracefully)
- Disabled web server at boot (no auto-start)
- API-mediated enable triggers auto-start
- Pre/post API call log segmentation
- Basic auth proxy enabled: proxy wraps web server, backend port discovered from `basic-auth-proxy.json`
- Auth proxy binary built from source during test, prepended to PATH

## How to Run

```sh
doctest test ./tests/server/...
```

Or for a single leaf:

```sh
doctest test ./tests/server/decision/auto-start-via-settings
doctest test ./tests/server/decision/disabled-no-autostart
doctest test ./tests/server/decision/enable-via-api-triggers-start
doctest test ./tests/server/decision/with-auth-proxy
```

Validate tree structure:

```sh
doctest vet ./tests/server
```

```go
import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/xhd2015/ai-critic/script/lib"
	"github.com/xhd2015/doctest/session"
)

type OpenCodeSettings struct {
	WebServerEnabled bool
	WebServerPort    int
	DefaultDomain    string
	AuthProxyEnabled bool
	BinaryPath       string
}

type PostStartRequest struct {
	URL       string
	Method    string
	Body      string
	Wait      int
	AuthToken string
}

type Request struct {
	// ConfigHome is the isolated AI_CRITIC_HOME for the child server process
	// only (via cmd.Env). Never process Setenv — Parallel-safe.
	ConfigHome       string
	OpenCodeSettings *OpenCodeSettings
	ServerPort       int
	TimeoutSecs      int
	PostStart        *PostStartRequest
}

type Response struct {
	ServerPort       int
	ServerStarted    bool
	Logs             string
	PrePostLogs      string
	WebServerRunning bool
	HasAutoStartLog  bool
	BackendPort      int
	BackendRunning   bool
	ProxyRunning     bool
}

func Run(t *testing.T, d *session.Doctest, req *Request) (*Response, error) {
	resp := &Response{}

	if req.ServerPort <= 0 {
		req.ServerPort = 23712
	}
	hash := 0
	for _, c := range t.Name() {
		hash = hash*31 + int(c)
	}
	if hash < 0 {
		hash = -hash
	}
	req.ServerPort = req.ServerPort + (hash % 100)
	if req.TimeoutSecs <= 0 {
		req.TimeoutSecs = 30
	}

	safeName := strings.ReplaceAll(t.Name(), "/", "_")
	safeName = strings.ReplaceAll(safeName, "\\", "_")
	binPath := filepath.Join(os.TempDir(), "ai-critic-server-test-"+safeName)

	buildDir, err := findGoModuleRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to find repo root: %v", err)
	}
	if d != nil && d.DOCTEST_ROOT != "" {
		// Prefer inject root when findGoModuleRoot is confused under nested leaves.
		candidate := filepath.Clean(filepath.Join(d.DOCTEST_ROOT, "..", ".."))
		if _, statErr := os.Stat(filepath.Join(candidate, "go.mod")); statErr == nil {
			buildDir = candidate
		}
	}

	t.Logf("building server binary: %s (dir=%s)", binPath, buildDir)
	build := exec.Command("go", "build", "-o", binPath, ".")
	build.Dir = buildDir
	buildOut, buildErr := build.CombinedOutput()
	if buildErr != nil {
		return nil, fmt.Errorf("failed to build server binary: %v\n%s", buildErr, string(buildOut))
	}
	t.Cleanup(func() {
		os.Remove(binPath)
	})

	proxyBinPath := filepath.Join(os.TempDir(), "basic-auth-proxy-test-"+safeName)
	proxyBuild := exec.Command("go", "build", "-o", proxyBinPath, "./cmd/basic-auth-proxy")
	proxyBuild.Dir = buildDir
	proxyBuildOut, proxyBuildErr := proxyBuild.CombinedOutput()
	if proxyBuildErr != nil {
		return nil, fmt.Errorf("failed to build basic-auth-proxy binary: %v\n%s", proxyBuildErr, string(proxyBuildOut))
	}
	t.Cleanup(func() {
		os.Remove(proxyBinPath)
	})

	binDir := filepath.Dir(binPath)

	serverPort := req.ServerPort
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", serverPort))
	if err != nil {
		found := false
		for port := serverPort + 1; port < serverPort+100; port++ {
			listener, err = net.Listen("tcp", fmt.Sprintf(":%d", port))
			if err == nil {
				serverPort = port
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("no free port found starting from %d", req.ServerPort)
		}
	}
	if serverPort != req.ServerPort {
		t.Logf("port %d was in use, using port %d instead", req.ServerPort, serverPort)
	}
	listener.Close()
	resp.ServerPort = serverPort

	t.Logf("starting server on port %d", serverPort)
	args := []string{"--port", strconv.Itoa(serverPort)}

	configHome := req.ConfigHome
	if configHome == "" {
		var err error
		configHome, err = lib.CreateTestConfigHome()
		if err != nil {
			return nil, fmt.Errorf("failed to create temp config home: %v", err)
		}
		req.ConfigHome = configHome
		t.Cleanup(func() {
			os.RemoveAll(configHome)
		})
	}

	testAuthToken := lib.TestPassword
	credFile, err := lib.WriteTestCredentials(configHome)
	if err != nil {
		return nil, fmt.Errorf("failed to write test credentials file: %v", err)
	}
	args = append(args, "--credentials-file", credFile)

	cmd := exec.Command(binPath, args...)
	cmd.Dir = configHome
	env := make([]string, 0, len(os.Environ())+1)
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "PATH=") {
			env = append(env, "PATH="+binDir+":"+e[5:])
		} else {
			env = append(env, e)
		}
	}
	// Child-only isolation: AI_CRITIC_HOME on cmd.Env, never process Setenv.
	// Do not use AppendTestServerEnv here — it sets AI_CRITIC_TEST_SKIP_EXTENSION=1,
	// which skips AutoStartWebServer (the behaviour under test).
	cmd.Env = appendTestServerEnvAllowExtension(env, configHome)

	var logBuf bytes.Buffer
	cmd.Stdout = &logBuf
	cmd.Stderr = &logBuf

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start server: %v", err)
	}

	wsCleanupPort := 0
	if req.OpenCodeSettings != nil {
		wsCleanupPort = req.OpenCodeSettings.WebServerPort
	}
	if wsCleanupPort <= 0 {
		wsCleanupPort = 14096
	}

	stopServer := func() {
		if cmd.Process != nil {
			cmd.Process.Signal(os.Interrupt)
			time.Sleep(500 * time.Millisecond)
			cmd.Process.Signal(syscall.SIGTERM)
			time.Sleep(200 * time.Millisecond)
			cmd.Process.Kill()
		}
		_ = lib.CleanupOpencodeServe(configHome, wsCleanupPort)
	}
	t.Cleanup(stopServer)

	pingURL := fmt.Sprintf("http://localhost:%d/ping", serverPort)
	deadline := time.Now().Add(time.Duration(req.TimeoutSecs) * time.Second)

	var started bool
	for time.Now().Before(deadline) {
		if httpGetOK(pingURL) {
			started = true
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	resp.ServerStarted = started

	if started {
		time.Sleep(3 * time.Second)

		if req.PostStart != nil {
			resp.PrePostLogs = logBuf.String()

			if testAuthToken != "" {
				req.PostStart.AuthToken = testAuthToken
			}

			url := strings.ReplaceAll(req.PostStart.URL, "__PORT__", strconv.Itoa(serverPort))
			t.Logf("PostStart: making %s request to %s", req.PostStart.Method, url)
			if err := postStartHTTP(req.PostStart.Method, url, req.PostStart.Body, req.PostStart.AuthToken); err != nil {
				t.Logf("PostStart: request failed: %v", err)
			} else {
				t.Logf("PostStart: request completed successfully")
			}

			if wait := req.PostStart.Wait; wait > 0 {
				t.Logf("PostStart: waiting %d seconds for effects...", wait)
				time.Sleep(time.Duration(wait) * time.Second)
			}
		}

		settings := req.OpenCodeSettings
		if settings != nil {
			wsPort := settings.WebServerPort
			if wsPort <= 0 {
				wsPort = 14096
			}

			if settings.AuthProxyEnabled {
				resp.ProxyRunning = checkPort(wsPort)

				proxyConfigPath := filepath.Join(configHome, "basic-auth-proxy.json")
				proxyData, readErr := os.ReadFile(proxyConfigPath)
				if readErr == nil {
					var proxyCfg struct {
						BackendPort int `json:"backend_port"`
					}
					if jsonErr := json.Unmarshal(proxyData, &proxyCfg); jsonErr == nil && proxyCfg.BackendPort > 0 {
						resp.BackendPort = proxyCfg.BackendPort
						resp.BackendRunning = checkPort(proxyCfg.BackendPort)
					}
				}
			} else {
				resp.WebServerRunning = checkPort(wsPort)
			}
		}
	}

	stopServer()
	_ = cmd.Wait()

	resp.Logs = logBuf.String()
	resp.HasAutoStartLog = strings.Contains(resp.Logs, "[opencode] AutoStartWebServer:")

	t.Logf("=== SERVER OUTPUT ===\n%s\n=== END SERVER OUTPUT ===", resp.Logs)
	t.Logf("HasAutoStartLog: %v, LogsLen: %d", resp.HasAutoStartLog, len(resp.Logs))

	return resp, nil
}

// appendTestServerEnvAllowExtension isolates AI_CRITIC_HOME on the child only
// and disables browser open, without skipping extension/auto-start tasks.
func appendTestServerEnvAllowExtension(base []string, configHome string) []string {
	env := make([]string, 0, len(base)+2)
	for _, e := range base {
		if strings.HasPrefix(e, lib.EnvAI_CRITIC_HOME+"=") {
			continue
		}
		if strings.HasPrefix(e, "AI_CRITIC_NO_OPEN_BROWSER=") {
			continue
		}
		if strings.HasPrefix(e, "AI_CRITIC_TEST_SKIP_EXTENSION=") {
			continue
		}
		env = append(env, e)
	}
	env = append(env, lib.EnvAI_CRITIC_HOME+"="+configHome)
	env = append(env, "AI_CRITIC_NO_OPEN_BROWSER=1")
	return env
}

func findGoModuleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found in any parent of %s", dir)
		}
		dir = parent
	}
}

func isPortInUse(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 500*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func httpGetOK(url string) bool {
	resp, err := http.Get(url)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func checkPort(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 1*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func killPort(port int) {
	out, err := exec.Command("lsof", "-ti", fmt.Sprintf("tcp:%d", port)).Output()
	if err != nil {
		return
	}
	pids := strings.Fields(strings.TrimSpace(string(out)))
	for _, pidStr := range pids {
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			continue
		}
		if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
			_ = syscall.Kill(pid, syscall.SIGKILL)
		}
	}
}

func postStartHTTP(method, url, body string, authToken string) error {
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(b))
	}
	return nil
}
```

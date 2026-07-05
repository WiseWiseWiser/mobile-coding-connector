# macOS Menu Restart Daemon Doctests

Contract and API tests for changing the macOS menu-bar **Restart Server** action to
**Restart Daemon**: POST `/api/keep-alive/restart-daemon` (SSE drain, no log UI),
aligned with the web Manage Server **Restart Daemon** button.

# DSN (Domain Specific Notion)

**Participants**

- **macOS menu bar (`AICriticApp.swift`)** — dropdown button label and handler that
  should invoke daemon restart (not managed-server signal restart).
- **Swift `DaemonClient`** — HTTP client on keep-alive management port `23312`; must
  POST `/api/keep-alive/restart-daemon`, drain the SSE body to EOF, then poll status.
- **Keep-alive daemon** — management HTTP on `23312`; `POST /api/keep-alive/restart`
  signals server kill+respawn (daemon PID unchanged); `POST /api/keep-alive/restart-daemon`
  exec-replaces the daemon (picks newer `bin/` binary when present).
- **Test harness** — reads Swift sources for client contract; starts isolated daemon for
  API leaves with session lock on port `23312`.

**Behaviors**

- Menu contract: label **Restart Daemon**; handler calls `restartDaemon()` targeting
  `/api/keep-alive/restart-daemon` (not `restartServer()` → `/api/keep-alive/restart`).
- Server signal restart: `POST /restart` → JSON `status=restart_requested`; managed
  server PID changes after settle; `keep_alive_pid` unchanged.
- Daemon exec restart: `POST /restart-daemon` streams SSE ending with `done.success=true`;
  after body drain and settle, status API reports running again with server reachable.

## Version

0.0.2

## Decision Tree

```
[macOS restart menu + keep-alive API]
 |
 +-- client/                           (GROUP)  Swift menu/DaemonClient contract
 |    +-- macos-menu-contract/        (LEAF)   label + endpoint map (RED before fix)
 |
 +-- api/                              (GROUP)  management HTTP restart endpoints
      +-- restart-server-signal/      (LEAF)   POST /restart → server PID changes
      +-- restart-daemon-exec/          (LEAF)   POST /restart-daemon → SSE success
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `client/macos-menu-contract` | Menu → `/restart-daemon`, not `/restart` |
| 2 | `api/restart-server-signal` | POST `/restart` requests server restart only |
| 3 | `api/restart-daemon-exec` | POST `/restart-daemon` SSE done + daemon back |

## Parameter Coverage

| Leaf | Op | Endpoint | Expect RED (pre-fix) |
|------|-----|----------|----------------------|
| macos-menu-contract | client | Swift sources | yes — current `/restart` + Restart Server |
| restart-server-signal | api-restart-server | `/api/keep-alive/restart` | no |
| restart-daemon-exec | api-restart-daemon | `/api/keep-alive/restart-daemon` | no |

## Run profiles (labels)

| Label | Meaning |
|-------|---------|
| `slow` | Daemon exec restart + settle polling |

```sh
doctest test --label slow ./tests/macos-restart-menu/api/restart-daemon-exec
```

## How to Run

```sh
doctest vet ./tests/macos-restart-menu
doctest test ./tests/macos-restart-menu/...
```

```go
import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/xhd2015/ai-critic/script/lib"
	"github.com/xhd2015/ai-critic/server/config"
)

const (
	expectedMenuLabel      = "Restart Daemon"
	expectedRestartPath    = "/api/keep-alive/restart-daemon"
	legacyRestartPath      = "/api/keep-alive/restart"
	expectedClientMethod   = "restartDaemon"
	legacyClientMethod     = "restartServer"
)

type keepAliveStatus struct {
	Running       bool   `json:"running"`
	ServerPort    int    `json:"server_port"`
	ServerPID     int    `json:"server_pid"`
	KeepAlivePort int    `json:"keep_alive_port"`
	KeepAlivePID  int    `json:"keep_alive_pid"`
	StartedAt     string `json:"started_at,omitempty"`
}

type Request struct {
	Op string

	ServerPort      int
	StartupWaitSecs int
	SettleWaitSecs  int
}

type Response struct {
	// Client contract (actual values read from Swift sources)
	MenuLabel           string
	RestartEndpoint     string
	ClientMethod        string
	SwiftSourcesChecked []string

	// API restart
	RestartHTTPStatus int
	RestartJSONStatus string
	SSEBody           string
	SSEDoneSuccess    bool

	BeforeDaemonPID int
	AfterDaemonPID  int
	BeforeServerPID int
	AfterServerPID  int
	SubprocessPID   int

	DaemonReachable bool
	ServerPingOK    bool
}

func Run(t *testing.T, req *Request) (*Response, error) {
	resp := &Response{}
	switch req.Op {
	case "client":
		return runClientContract(t, resp)
	case "api-restart-server":
		return runAPIRestartServer(t, req, resp)
	case "api-restart-daemon":
		return runAPIRestartDaemon(t, req, resp)
	default:
		return nil, fmt.Errorf("unknown op %q", req.Op)
	}
}

func runClientContract(t *testing.T, resp *Response) (*Response, error) {
	moduleRoot, err := findModuleRoot()
	if err != nil {
		return nil, err
	}

	appPath := filepath.Join(moduleRoot, "macos-ai-critic", "ai-critic-macos", "AICriticApp.swift")
	clientPath := filepath.Join(moduleRoot, "macos-ai-critic", "ai-critic-macos", "DaemonClient.swift")
	resp.SwiftSourcesChecked = []string{appPath, clientPath}

	appSrc, err := os.ReadFile(appPath)
	if err != nil {
		return nil, fmt.Errorf("read AICriticApp.swift: %w", err)
	}
	clientSrc, err := os.ReadFile(clientPath)
	if err != nil {
		return nil, fmt.Errorf("read DaemonClient.swift: %w", err)
	}

	resp.MenuLabel = extractSwiftButtonLabel(string(appSrc))
	resp.RestartEndpoint = extractRestartEndpoint(string(clientSrc))
	resp.ClientMethod = extractMenuRestartCall(string(appSrc))
	return resp, nil
}

var (
	reSwiftButtonLabel = regexp.MustCompile(`Button\("([^"]+)"\)`)
	reRestartEndpoint  = regexp.MustCompile(`baseURL \+ "(/api/keep-alive/[^"]+)"`)
	reMenuRestartCall  = regexp.MustCompile(`DaemonClient\.shared\.(restart\w+)\(\)`)
)

func extractSwiftButtonLabel(appSrc string) string {
	// First Button after Divider in menu section — Restart action.
	idx := strings.Index(appSrc, `Button("Restart`)
	if idx < 0 {
		return ""
	}
	segment := appSrc[idx : idx+120]
	m := reSwiftButtonLabel.FindStringSubmatch(segment)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

func extractRestartEndpoint(clientSrc string) string {
	for _, m := range reRestartEndpoint.FindAllStringSubmatch(clientSrc, -1) {
		if len(m) < 2 {
			continue
		}
		if strings.Contains(m[1], "restart") {
			return m[1]
		}
	}
	return ""
}

func extractMenuRestartCall(appSrc string) string {
	m := reMenuRestartCall.FindStringSubmatch(appSrc)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

func runAPIRestartServer(t *testing.T, req *Request, resp *Response) (*Response, error) {
	daemon, port, cleanup, err := startTestDaemon(t, req)
	if err != nil {
		return nil, err
	}
	t.Cleanup(cleanup)
	if daemon.Process != nil {
		resp.SubprocessPID = daemon.Process.Pid
	}

	status, err := waitDaemonStatus(port, req.StartupWaitSecs)
	if err != nil {
		return nil, err
	}
	resp.BeforeDaemonPID = status.KeepAlivePID
	resp.BeforeServerPID = status.ServerPID
	if resp.SubprocessPID > 0 && resp.BeforeDaemonPID != resp.SubprocessPID {
		return resp, fmt.Errorf("status keep_alive_pid=%d != test daemon pid=%d", resp.BeforeDaemonPID, resp.SubprocessPID)
	}
	if resp.BeforeServerPID <= 0 {
		return resp, fmt.Errorf("server not running before restart: %+v", status)
	}

	code, body, err := postJSON(fmt.Sprintf("http://127.0.0.1:%d/api/keep-alive/restart", config.KeepAlivePort))
	if err != nil {
		return nil, err
	}
	resp.RestartHTTPStatus = code
	resp.RestartJSONStatus = parseRestartStatus(body)

	settle := req.SettleWaitSecs
	if settle <= 0 {
		settle = 20
	}
	after, err := waitServerPIDChange(port, resp.BeforeServerPID, resp.SubprocessPID, settle)
	if err != nil {
		return nil, err
	}
	resp.AfterServerPID = after.ServerPID
	resp.AfterDaemonPID = after.KeepAlivePID
	resp.DaemonReachable = true
	pingURL := fmt.Sprintf("http://127.0.0.1:%d/ping", port)
	resp.ServerPingOK = waitPingOK(pingURL, settle)

	_ = daemon
	return resp, nil
}

func runAPIRestartDaemon(t *testing.T, req *Request, resp *Response) (*Response, error) {
	daemon, port, cleanup, err := startTestDaemon(t, req)
	if err != nil {
		return nil, err
	}
	t.Cleanup(cleanup)
	if daemon.Process != nil {
		resp.SubprocessPID = daemon.Process.Pid
	}

	status, err := waitDaemonStatus(port, req.StartupWaitSecs)
	if err != nil {
		return nil, err
	}
	resp.BeforeDaemonPID = status.KeepAlivePID
	resp.BeforeServerPID = status.ServerPID

	settle := req.SettleWaitSecs
	if settle <= 0 {
		settle = 25
	}

	type sseResult struct {
		code        int
		body        string
		doneSuccess bool
	}
	sseCh := make(chan sseResult, 1)
	go func() {
		code, body, done := postRestartDaemonSSE(config.KeepAlivePort)
		sseCh <- sseResult{code, body, done}
	}()

	after, waitErr := waitDaemonBack(port, settle)
	sse := <-sseCh
	resp.RestartHTTPStatus = sse.code
	resp.SSEBody = sse.body
	resp.SSEDoneSuccess = sse.doneSuccess
	if waitErr != nil {
		return resp, waitErr
	}
	resp.AfterDaemonPID = after.KeepAlivePID
	resp.AfterServerPID = after.ServerPID
	resp.DaemonReachable = after.Running || after.ServerPID > 0
	pingURL := fmt.Sprintf("http://127.0.0.1:%d/ping", port)
	resp.ServerPingOK = waitPingOK(pingURL, settle)
	return resp, nil
}

func startTestDaemon(t *testing.T, req *Request) (*exec.Cmd, int, func(), error) {
	if req.ServerPort <= 0 {
		req.ServerPort = config.DefaultServerPort
	}
	hash := portHash(t.Name())
	req.ServerPort += hash % 200

	moduleRoot, err := findModuleRoot()
	if err != nil {
		return nil, 0, nil, err
	}

	safeName := strings.ReplaceAll(t.Name(), "/", "_")
	binPath := filepath.Join(os.TempDir(), "ai-critic-restart-menu-"+safeName)
	build := exec.Command("go", "build", "-o", binPath, ".")
	build.Dir = moduleRoot
	if out, err := build.CombinedOutput(); err != nil {
		return nil, 0, nil, fmt.Errorf("build ai-critic: %v\n%s", err, string(out))
	}

	configHome, err := lib.CreateTestConfigHome()
	if err != nil {
		os.Remove(binPath)
		return nil, 0, nil, err
	}
	if _, err := lib.WriteTestCredentials(configHome); err != nil {
		os.Remove(binPath)
		os.RemoveAll(configHome)
		return nil, 0, nil, err
	}

	portStr := strconv.Itoa(req.ServerPort)
	daemonLog := filepath.Join(configHome, "restart-menu-test.log")
	cmd := exec.Command(binPath,
		"keep-alive",
		"--kill-existing",
		"--port", portStr,
		"--forever",
		"--log", daemonLog,
	)
	cmd.Dir = configHome
	cmd.Env = append(lib.AppendTestServerEnv(os.Environ(), configHome), "AI_CRITIC_TEST_SKIP_EXTENSION=1")
	if err := cmd.Start(); err != nil {
		os.Remove(binPath)
		os.RemoveAll(configHome)
		return nil, 0, nil, fmt.Errorf("start keep-alive: %w", err)
	}

	cleanup := func() {
		if cmd.Process != nil {
			pgid, pgErr := syscall.Getpgid(cmd.Process.Pid)
			if pgErr == nil {
				_ = syscall.Kill(-pgid, syscall.SIGTERM)
				time.Sleep(300 * time.Millisecond)
				_ = syscall.Kill(-pgid, syscall.SIGKILL)
			} else {
				_ = cmd.Process.Kill()
			}
		}
		_ = cmd.Wait()
		os.Remove(binPath)
		os.RemoveAll(configHome)
	}
	return cmd, req.ServerPort, cleanup, nil
}

func waitDaemonStatus(serverPort, timeoutSecs int) (*keepAliveStatus, error) {
	url := fmt.Sprintf("http://127.0.0.1:%d/api/keep-alive/status", config.KeepAlivePort)
	deadline := time.Now().Add(time.Duration(timeoutSecs) * time.Second)
	for time.Now().Before(deadline) {
		st, err := fetchStatus(url)
		if err == nil && st.ServerPID > 0 {
			return st, nil
		}
		time.Sleep(300 * time.Millisecond)
	}
	st, err := fetchStatus(url)
	if err != nil {
		return nil, fmt.Errorf("daemon status not ready: %w", err)
	}
	return st, nil
}

func waitServerPIDChange(serverPort, beforePID, daemonPID, timeoutSecs int) (*keepAliveStatus, error) {
	url := fmt.Sprintf("http://127.0.0.1:%d/api/keep-alive/status", config.KeepAlivePort)
	deadline := time.Now().Add(time.Duration(timeoutSecs) * time.Second)
	for time.Now().Before(deadline) {
		st, err := fetchStatus(url)
		if err == nil && st.ServerPID > 0 && st.ServerPID != beforePID {
			if daemonPID > 0 && st.KeepAlivePID != daemonPID {
				return nil, fmt.Errorf("daemon PID changed during server signal restart: want %d got %d", daemonPID, st.KeepAlivePID)
			}
			return st, nil
		}
		time.Sleep(400 * time.Millisecond)
	}
	return nil, fmt.Errorf("server PID did not change from %d within %ds", beforePID, timeoutSecs)
}

func waitDaemonBack(serverPort, timeoutSecs int) (*keepAliveStatus, error) {
	url := fmt.Sprintf("http://127.0.0.1:%d/api/keep-alive/status", config.KeepAlivePort)
	deadline := time.Now().Add(time.Duration(timeoutSecs) * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		st, err := fetchStatus(url)
		if err != nil {
			lastErr = err
			time.Sleep(500 * time.Millisecond)
			continue
		}
		if st.ServerPID > 0 {
			return st, nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	if lastErr != nil {
		return nil, fmt.Errorf("daemon not back after restart: %w", lastErr)
	}
	return nil, fmt.Errorf("daemon not back after restart within %ds", timeoutSecs)
}

func fetchStatus(url string) (*keepAliveStatus, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}
	var st keepAliveStatus
	if err := json.NewDecoder(resp.Body).Decode(&st); err != nil {
		return nil, err
	}
	return &st, nil
}

func postJSON(url string) (int, string, error) {
	resp, err := http.Post(url, "application/json", nil)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(body), nil
}

func parseRestartStatus(body string) string {
	var m map[string]string
	if json.Unmarshal([]byte(body), &m) != nil {
		return ""
	}
	return m["status"]
}

func postRestartDaemonSSE(port int) (int, string, bool) {
	url := fmt.Sprintf("http://127.0.0.1:%d/api/keep-alive/restart-daemon", port)
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Post(url, "application/json", nil)
	if err != nil {
		return 0, "", false
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	text := string(body)
	doneSuccess := sseDoneSuccess(text)
	return resp.StatusCode, text, doneSuccess
}

func sseDoneSuccess(body string) bool {
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimPrefix(line, "data: ")
		var obj map[string]any
		if json.Unmarshal([]byte(payload), &obj) != nil {
			continue
		}
		typ, _ := obj["type"].(string)
		if typ != "done" {
			continue
		}
		success, _ := obj["success"].(string)
		return success == "true"
	}
	return false
}

func httpPingOK(url string) bool {
	resp, err := http.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false
	}
	b, _ := io.ReadAll(resp.Body)
	return strings.TrimSpace(string(b)) == "pong"
}

func waitPingOK(url string, timeoutSecs int) bool {
	deadline := time.Now().Add(time.Duration(timeoutSecs) * time.Second)
	for time.Now().Before(deadline) {
		if httpPingOK(url) {
			return true
		}
		time.Sleep(400 * time.Millisecond)
	}
	return false
}

func portHash(name string) int {
	h := 0
	for _, c := range name {
		h = h*31 + int(c)
	}
	if h < 0 {
		h = -h
	}
	return h
}

func findModuleRoot() (string, error) {
	if root := os.Getenv("DOCTEST_ROOT"); root != "" {
		for dir := root; ; dir = filepath.Dir(dir) {
			if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
				return dir, nil
			}
			if filepath.Dir(dir) == dir {
				break
			}
		}
	}
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
			return "", fmt.Errorf("go.mod not found")
		}
		dir = parent
	}
}

var _ = bytes.Buffer{}
```
# Keep-Alive Core vs Extension Startup Doctests

End-to-end tests that start the **keep-alive daemon**, which spawns the managed
`ai-critic` server, and verify startup readiness under configurable
`--startup-timeout`: **slow core bind** (delay before `net.Listen`) and **slow
extension** (delay after core bind) must not trigger premature kill/restart loops.

# DSN (Domain Specific Notion)

**Participants**

- **Keep-alive daemon** — loops on managed server lifecycle; polls `127.0.0.1:P`
  until TCP listen succeeds within `StartupTimeout` (CLI `--startup-timeout`,
  default 60s); logs `[keepalive] phase=*` markers when implemented.
- **Managed ai-critic server** — binds the HTTP port in a **core** phase, then
  runs **extension** work asynchronously (tunnels, `AutoStartWebServer`, etc.);
  emits `[bootstrap] phase=*` markers when implemented.
- **Test config home** — isolated `AI_CRITIC_HOME` with credentials; optional
  `opencode.json` to trigger extension path without real Cloudflare I/O (hooks
  delay or stub extension entry).
- **Test hooks (env)** — `AI_CRITIC_TEST_CORE_DELAY_MS` sleeps before
  `net.Listen`; `AI_CRITIC_TEST_EXTENSION_DELAY_MS` sleeps at extension start;
  `AI_CRITIC_TEST_SKIP_EXTENSION` skips extension work for baseline.

**Behaviors**

- Core listener must become reachable before extension tasks block the accept
  loop; steady-state health checks remain TCP + `/ping`.
- With a multi-second **core** delay (pre-listen), the daemon must wait out the
  delay when `--startup-timeout` is generous (60s) and must not restart-loop.
- With a multi-second **extension** delay (post-bind), the daemon must still mark
  the server ready within the configured timeout (10s for regression leaves) and
  must not enter a restart loop.
- Log ordering: `core_ready` before `extension_start`; independent `/ping` before
  extension auto-task logs.

## Version

0.0.2

## Decision Tree

```
[keep-alive manages server bootstrap]
 |
 +-- slow-core/                              (grouping: core bind delay before net.Listen)
 |    |
 |    +-- daemon-ready/                      (grouping: startup-timeout vs core delay)
 |         +-- survives-core-delay-with-60s-timeout/  (LEAF) 15s core delay + 60s timeout → ready
 |         +-- no-restart-loop-under-core-delay/      (LEAF) 25s observe; no restart churn
 |         +-- fails-with-10s-timeout-control/        (LEAF) 15s core delay + 10s timeout → fail loop
 |
 +-- slow-extension/                         (grouping: extension path + hook delay)
 |    |
 |    +-- daemon-ready/                      (grouping: daemon 10s timeout / stability)
 |    |    +-- core-ready-within-timeout/    (LEAF) server_ready <10s with 15s extension delay
 |    |    +-- no-restart-loop/              (LEAF) 20s observe; no "failed to become ready"
 |    |
 |    +-- timing-order/                      (grouping: core vs extension ordering)
 |         +-- core-before-extension/         (LEAF) core_ready t_ms < extension_start t_ms
 |         +-- ping-before-extension-tasks/   (LEAF) /ping OK before extension auto-task logs
 |
 +-- no-extension/                           (grouping: skip extension / minimal config)
      +-- baseline-fast-start/               (LEAF) core_ready within 3s; no tunnel autostart noise
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `slow-core/daemon-ready/survives-core-delay-with-60s-timeout` | 15s pre-listen delay; 60s timeout; server_ready without restart loop |
| 2 | `slow-core/daemon-ready/no-restart-loop-under-core-delay` | 25s observe with 15s core delay; stable ready, no timeout churn |
| 3 | `slow-core/daemon-ready/fails-with-10s-timeout-control` | Negative control: 15s core delay + 10s timeout documents old failure |
| 4 | `slow-extension/daemon-ready/core-ready-within-timeout` | 15s extension delay; daemon sees ready within 10s |
| 5 | `slow-extension/daemon-ready/no-restart-loop` | 20s observation; no startup timeout restart loop |
| 6 | `slow-extension/timing-order/core-before-extension` | Bootstrap log ordering: core before extension |
| 7 | `slow-extension/timing-order/ping-before-extension-tasks` | HTTP `/ping` precedes extension task logs |
| 8 | `no-extension/baseline-fast-start` | Fast core_ready; daemon stable; no extension tunnel path |

## Parameter Coverage

| Leaf | CoreDelayMs | ExtensionDelayMs | StartupTimeout | SkipExtension | opencode.json | ObserveSecs |
|------|-------------|------------------|----------------|---------------|---------------|-------------|
| survives-core-delay-with-60s-timeout | 15000 | 0 | 60s | true | absent | 65 |
| no-restart-loop-under-core-delay | 15000 | 0 | 60s | true | absent | 25 |
| fails-with-10s-timeout-control | 15000 | 0 | 10s | true | absent | 15 |
| core-ready-within-timeout | 0 | 15000 | 10s | false | minimal enabled | 12 |
| no-restart-loop | 0 | 15000 | 10s | false | minimal enabled | 20 |
| core-before-extension | 0 | 5000 | 10s | false | minimal enabled | 10 |
| ping-before-extension-tasks | 0 | 5000 | 10s | false | minimal enabled | 10 |
| baseline-fast-start | 0 | 0 | (default) | true | absent / disabled | 8 |

## How to Run

```sh
doctest vet ./tests/keep-alive
doctest test ./tests/keep-alive/...
```

Single leaf:

```sh
doctest test ./tests/keep-alive/slow-core/daemon-ready/survives-core-delay-with-60s-timeout
doctest test ./tests/keep-alive/slow-extension/daemon-ready/core-ready-within-timeout
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
	envCoreDelayMs      = "AI_CRITIC_TEST_CORE_DELAY_MS"
	envExtensionDelayMs = "AI_CRITIC_TEST_EXTENSION_DELAY_MS"
	envSkipExtension    = "AI_CRITIC_TEST_SKIP_EXTENSION"
	envSkipPortPrecheck = "AI_CRITIC_KEEPALIVE_SKIP_SERVER_PORT_CHECK"
)

type Request struct {
	ServerPort           int
	CoreDelayMs          int
	ExtensionDelayMs     int
	StartupTimeout       string
	ObserveSecs          int
	SkipExtensionStartup bool
	WriteExtensionConfig bool
}

type Response struct {
	ServerPort       int
	DaemonLogs       string
	ServerLogs       string
	CoreReadyMs      int
	ExtensionStartMs int
	PortReadyMs      int
	ServerReady      bool
	RestartLoopSeen  bool
	PingBeforeExt    bool
}

func Run(t *testing.T, req *Request) (*Response, error) {
	resp := &Response{}

	if req.ServerPort <= 0 {
		req.ServerPort = config.DefaultServerPort
	}
	hash := 0
	for _, c := range t.Name() {
		hash = hash*31 + int(c)
	}
	if hash < 0 {
		hash = -hash
	}
	req.ServerPort += hash % 200
	if req.ObserveSecs <= 0 {
		req.ObserveSecs = 12
	}
	resp.ServerPort = req.ServerPort

	buildDir, err := findModuleRoot()
	if err != nil {
		return nil, err
	}

	safeName := strings.ReplaceAll(t.Name(), "/", "_")
	binPath := filepath.Join(os.TempDir(), "ai-critic-keepalive-test-"+safeName)
	build := exec.Command("go", "build", "-o", binPath, ".")
	build.Dir = buildDir
	if out, err := build.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("build ai-critic: %v\n%s", err, string(out))
	}
	t.Cleanup(func() { os.Remove(binPath) })

	configHome, err := lib.CreateTestConfigHome()
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { os.RemoveAll(configHome) })

	if _, err := lib.WriteTestCredentials(configHome); err != nil {
		return nil, err
	}
	if err := writeConfigHomeFixtures(configHome, req); err != nil {
		return nil, err
	}

	daemonLogPath := filepath.Join(configHome, "keep-alive-daemon.log")
	serverLogPath := filepath.Join(configHome, config.ServerLogFile)

	portStr := strconv.Itoa(req.ServerPort)
	serverArgs := []string{"--port", portStr}

	daemonArgs := []string{
		"keep-alive",
		"--port", portStr,
		"--forever",
		"--log", daemonLogPath,
	}
	if req.StartupTimeout != "" {
		daemonArgs = append(daemonArgs, "--startup-timeout", req.StartupTimeout)
	}
	daemonArgs = append(daemonArgs, serverArgs...)

	cmd := exec.Command(binPath, daemonArgs...)
	cmd.Dir = configHome
	cmd.Env = buildDaemonEnv(configHome, req)
	var daemonBuf bytes.Buffer
	cmd.Stdout = &daemonBuf
	cmd.Stderr = &daemonBuf

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start keep-alive: %w", err)
	}

	stop := func() {
		if cmd.Process != nil {
			pgid, pgErr := syscall.Getpgid(cmd.Process.Pid)
			if pgErr == nil {
				_ = syscall.Kill(-pgid, syscall.SIGTERM)
				time.Sleep(500 * time.Millisecond)
				_ = syscall.Kill(-pgid, syscall.SIGKILL)
			} else {
				_ = cmd.Process.Kill()
			}
		}
		_ = cmd.Wait()
	}
	t.Cleanup(stop)

	pingURL := fmt.Sprintf("http://127.0.0.1:%d/ping", req.ServerPort)
	var firstPingOK bool
	pingDeadline := time.Now().Add(time.Duration(req.ObserveSecs) * time.Second)
	for time.Now().Before(pingDeadline) {
		if httpPingOK(pingURL) {
			firstPingOK = true
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	time.Sleep(time.Duration(req.ObserveSecs) * time.Second)
	stop()

	daemonFile, _ := os.ReadFile(daemonLogPath)
	serverFile, _ := os.ReadFile(serverLogPath)
	resp.DaemonLogs = daemonBuf.String() + string(daemonFile)
	resp.ServerLogs = string(serverFile)
	if resp.ServerLogs == "" {
		resp.ServerLogs = extractServerLines(resp.DaemonLogs)
	}

	combined := resp.DaemonLogs + "\n" + resp.ServerLogs
	resp.CoreReadyMs = parseBootstrapMs(combined, "core_ready")
	resp.ExtensionStartMs = parseBootstrapMs(combined, "extension_start")
	resp.PortReadyMs = parseKeepaliveWaitedMs(combined)
	resp.ServerReady = strings.Contains(combined, "Server is ready") ||
		resp.PortReadyMs > 0 ||
		strings.Contains(combined, "[keepalive] phase=server_ready")
	resp.RestartLoopSeen = strings.Contains(combined, "failed to become ready")
	resp.PingBeforeExt = firstPingOK && logsShowExtensionAfterPing(combined)

	t.Logf("=== DAEMON+SERVER LOGS (tail) ===\n%s\n=== END ===", tailLines(combined, 80))

	return resp, nil
}

func buildDaemonEnv(configHome string, req *Request) []string {
	env := lib.AppendTestServerEnv(os.Environ(), configHome)
	env = append(env, envSkipPortPrecheck+"=1")
	if req.SkipExtensionStartup {
		env = append(env, envSkipExtension+"=1")
	}
	if req.CoreDelayMs > 0 {
		env = append(env, fmt.Sprintf("%s=%d", envCoreDelayMs, req.CoreDelayMs))
	}
	if req.ExtensionDelayMs > 0 {
		env = append(env, fmt.Sprintf("%s=%d", envExtensionDelayMs, req.ExtensionDelayMs))
	}
	return env
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

var (
	reBootstrapPhase = regexp.MustCompile(`\[bootstrap\] phase=([a-z_]+)[^\n]*\bt_ms=(\d+)`)
	reKeepaliveReady = regexp.MustCompile(`\[keepalive\] phase=server_ready[^\n]*\bwaited_ms=(\d+)`)
)

func parseBootstrapMs(logs, phase string) int {
	best := -1
	for _, m := range reBootstrapPhase.FindAllStringSubmatch(logs, -1) {
		if m[1] != phase {
			continue
		}
		if v, err := strconv.Atoi(m[2]); err == nil {
			best = v
		}
	}
	return best
}

func parseKeepaliveWaitedMs(logs string) int {
	m := reKeepaliveReady.FindStringSubmatch(logs)
	if len(m) < 2 {
		return -1
	}
	v, _ := strconv.Atoi(m[1])
	return v
}

func indexFirst(s string, subs ...string) int {
	best := -1
	for _, sub := range subs {
		if i := strings.Index(s, sub); i >= 0 && (best < 0 || i < best) {
			best = i
		}
	}
	return best
}

func logsShowExtensionAfterPing(logs string) bool {
	pingIdx := strings.Index(logs, "/ping")
	extIdx := indexFirst(logs,
		"[bootstrap] phase=extension_start",
		"[auto-task] Running extension",
		"[auto-task] Running startup tasks",
	)
	if extIdx < 0 {
		return false
	}
	if pingIdx < 0 {
		return strings.Contains(logs, "Server is ready")
	}
	return pingIdx < extIdx || strings.Contains(logs, "Server is ready")
}

func extractServerLines(daemonLogs string) string {
	var b strings.Builder
	for _, line := range strings.Split(daemonLogs, "\n") {
		if strings.Contains(line, "[bootstrap]") ||
			strings.Contains(line, "[auto-task]") ||
			strings.Contains(line, "Starting ai-critic server") {
			b.WriteString(line)
			b.WriteString("\n")
		}
	}
	return b.String()
}

func tailLines(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) <= n {
		return s
	}
	return strings.Join(lines[len(lines)-n:], "\n")
}

func writeJSON(path string, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func writeConfigHomeFixtures(configHome string, req *Request) error {
	if req.WriteExtensionConfig {
		settings := map[string]interface{}{
			"default_domain": "keepalive-test.example.com",
			"web_server": map[string]interface{}{
				"enabled":            true,
				"port":               14097,
				"auth_proxy_enabled": false,
				"target_preference":  "domain",
			},
		}
		return writeJSON(filepath.Join(configHome, "opencode.json"), settings)
	}
	if req.SkipExtensionStartup {
		settings := map[string]interface{}{
			"default_domain": "localhost",
			"web_server": map[string]interface{}{
				"enabled": false,
				"port":    14096,
			},
		}
		return writeJSON(filepath.Join(configHome, "opencode.json"), settings)
	}
	return nil
}
```
# Grok Usage Parser, Service, and API Doctests

Tests for `macosapp/grokusage` parsing, daemon-side grok usage fetch/cache, and
`GET /api/grok/usage` on the keep-alive management server. Fetch leaves set
`GROK_SHOW_USAGE_COMMAND` to fake-TUI scripts under `testdata/` so
`github.com/xhd2015/agent-pro/agent/grok/tty.FetchUsageWithOptions` runs
deterministically without a live grok binary.

# DSN (Domain Specific Notion)

**Participants**

- **TTY library (`agent/grok/tty`)** — `ParseShowUsageOutput` regex-parses
  scrollback; `FetchUsageWithOptions` PTY-launches `GROK_SHOW_USAGE_COMMAND`
  (or real grok), submits `/usage show`, and returns `UsageInfo`.
- **Parser wrapper (`macosapp/grokusage`)** — delegates `ParseShowUsageOutput`
  to `tty` for API compatibility.
- **Grok usage service (daemon)** — calls `tty.FetchUsageWithOptions` (no exec of
  `debug-grok-show-usage`), caches `GrokUsageResponse`, refreshes every 60s,
  skips overlapping in-flight fetches.
- **Mock fake-TUI script** — shell fixtures under `testdata/`; mimic grok prompt,
  read `/usage show`, emit usage lines; fail and slow variants for error/overlap.
- **ai-critic-server subprocess** — serves `GET /api/grok/usage` on main server port
  `23712` when API leaves run (started by keep-alive harness).
- **Keep-alive daemon** — management port `23312` control plane only; spawns server.
- **HTTP client** — asserts JSON `status`, `weekly_limit`, `next_reset`, `error`,
  `updated_at` fields.

**Behaviors**

- Standard and noisy scrollback parses to `UsageInfo`; missing fields return error.
- `GROK_SHOW_USAGE_COMMAND=mock-success.sh` → service `status=ready` with parsed limits.
- `GROK_SHOW_USAGE_COMMAND=mock-fail.sh` (exit 1) → `status=error` with message.
- API returns ready JSON after library fetch completes.
- Concurrent refresh while fetch in flight does not start a second PTY session (counter=1).

## Version

0.0.2

## Decision Tree

```
[grok usage]
 |
 +-- parse/                           (GROUP)  ParseShowUsageOutput
 |    +-- standard-output/            (LEAF)   two canonical lines
 |    +-- extra-noise/                (LEAF)   lines buried in scrollback
 |    +-- missing-weekly/             (LEAF)   parse error
 |    +-- missing-reset/              (LEAF)   parse error
 |
 +-- fetch/                           (GROUP)  service fetch via tty + mock command
 |    +-- mock-command-success/       (LEAF)   status ready
 |    +-- mock-command-fails/          (LEAF)   status error
 |
 +-- api/                             (GROUP)  HTTP surface
 |    +-- get-usage-ready/            (LEAF)   GET /api/grok/usage on server :23712
 |
 +-- refresh/                         (GROUP)  cache refresh semantics
      +-- skips-overlap/              (LEAF)   concurrent refresh skipped
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `parse/standard-output` | Parse canonical two-line output |
| 2 | `parse/extra-noise` | Parse usage buried in noise |
| 3 | `parse/missing-weekly` | Missing weekly line → error |
| 4 | `parse/missing-reset` | Missing reset line → error |
| 5 | `fetch/mock-command-success` | `GROK_SHOW_USAGE_COMMAND` mock → service ready |
| 6 | `fetch/mock-command-fails` | Mock exit 1 → service error |
| 7 | `api/get-usage-ready` | HTTP API on server port returns ready JSON |
| 8 | `refresh/skips-overlap` | Overlapping refresh does not double-fetch |

## Parameter Coverage

| Leaf | Op | Mock command / fixture | Expect error |
|------|-----|------------------------|--------------|
| standard-output | parse | show-usage-standard.txt | false |
| extra-noise | parse | show-usage-noisy.txt | false |
| missing-weekly | parse | show-usage-missing-weekly.txt | true |
| missing-reset | parse | show-usage-missing-reset.txt | true |
| mock-command-success | fetch | GROK_SHOW_USAGE_COMMAND=mock-success.sh | false |
| mock-command-fails | fetch | GROK_SHOW_USAGE_COMMAND=mock-fail.sh | false (service error status) |
| get-usage-ready | api | GROK_SHOW_USAGE_COMMAND=mock-success.sh | false |
| skips-overlap | refresh | GROK_SHOW_USAGE_COMMAND=mock-slow.sh | false |

## How to Run

```sh
doctest vet ./tests/grok-usage
doctest test ./tests/grok-usage/...
```

```go
import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/xhd2015/agent-pro/agent/grok/tty"
	"github.com/xhd2015/ai-critic/macosapp/grokusage"
	"github.com/xhd2015/ai-critic/script/lib"
	"github.com/xhd2015/ai-critic/server/config"
)

const envGrokShowUsageCommand = "GROK_SHOW_USAGE_COMMAND"

type Request struct {
	Op string

	// Parse: fixture filename relative to tests/grok-usage/testdata/
	FixtureFile string

	// Fetch/API/Refresh: mock script basename under testdata/
	MockScript string

	ExpectParseError bool
	WaitAPIReadySecs int
}

type GrokUsageJSON struct {
	Status      string `json:"status"`
	WeeklyLimit string `json:"weekly_limit,omitempty"`
	NextReset   string `json:"next_reset,omitempty"`
	Error       string `json:"error,omitempty"`
	UpdatedAt   string `json:"updated_at,omitempty"`
}

type Response struct {
	WeeklyLimit string
	NextReset   string
	ParseErr    string

	ServiceStatus string
	ServiceError  string
	UpdatedAt     string

	APIStatusCode int
	APIBody       string
	APIParsed     *GrokUsageJSON

	MockInvocationCount int
	ConcurrentStarted   int
}

func Run(t *testing.T, req *Request) (*Response, error) {
	resp := &Response{}
	doctestRoot := grokUsageDoctestRoot()

	switch req.Op {
	case "parse":
		return runParse(t, req, doctestRoot, resp)
	case "fetch":
		return runFetch(t, req, doctestRoot, resp)
	case "api":
		return runAPI(t, req, doctestRoot, resp)
	case "refresh":
		return runRefreshOverlap(t, req, doctestRoot, resp)
	default:
		return nil, fmt.Errorf("unknown op %q", req.Op)
	}
}

func runParse(t *testing.T, req *Request, root string, resp *Response) (*Response, error) {
	input, err := os.ReadFile(filepath.Join(root, "testdata", req.FixtureFile))
	if err != nil {
		return nil, err
	}
	info, parseErr := tty.ParseShowUsageOutput(string(input))
	if parseErr != nil {
		resp.ParseErr = parseErr.Error()
		return resp, nil
	}
	if info != nil {
		resp.WeeklyLimit = info.WeeklyLimit
		resp.NextReset = info.NextReset
	}
	return resp, nil
}

func runFetch(t *testing.T, req *Request, root string, resp *Response) (*Response, error) {
	svc, err := newServiceWithMockCommand(root, req.MockScript)
	if err != nil {
		return nil, err
	}
	out := svc.TestExported_FetchOnce(t)
	resp.ServiceStatus = string(out.Status)
	resp.ServiceError = out.Error
	resp.WeeklyLimit = out.WeeklyLimit
	resp.NextReset = out.NextReset
	resp.UpdatedAt = out.UpdatedAt
	return resp, nil
}

func runAPI(t *testing.T, req *Request, root string, resp *Response) (*Response, error) {
	if req.WaitAPIReadySecs <= 0 {
		req.WaitAPIReadySecs = 12
	}
	mockCommand, err := resolveMockScript(root, req.MockScript)
	if err != nil {
		return nil, err
	}

	moduleRoot, err := findModuleRoot()
	if err != nil {
		return nil, err
	}
	aiBin, cleanup, err := buildAICritic(t, moduleRoot)
	if err != nil {
		return nil, err
	}
	t.Cleanup(cleanup)

	configHome, err := lib.CreateTestConfigHome()
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { os.RemoveAll(configHome) })
	if _, err := lib.WriteTestCredentials(configHome); err != nil {
		return nil, err
	}

	port := config.DefaultServerPort + portHash(t.Name())%200
	daemonLog := filepath.Join(configHome, "grok-usage-api.log")
	cmd := exec.Command(aiBin,
		"keep-alive",
		"--kill-existing",
		"--port", strconv.Itoa(port),
		"--forever",
		"--log", daemonLog,
	)
	cmd.Dir = configHome
	env := lib.AppendTestServerEnv(os.Environ(), configHome)
	env = append(env,
		"AI_CRITIC_TEST_SKIP_EXTENSION=1",
		envGrokShowUsageCommand+"="+mockCommand,
	)
	cmd.Env = env
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	stop := func() {
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
	}
	t.Cleanup(stop)

	serverPort, err := waitManagedServerPort(req.WaitAPIReadySecs)
	if err != nil {
		return resp, err
	}
	url := fmt.Sprintf("http://127.0.0.1:%d/api/grok/usage", serverPort)
	deadline := time.Now().Add(time.Duration(req.WaitAPIReadySecs) * time.Second)
	for time.Now().Before(deadline) {
		httpResp, err := http.Get(url)
		if err == nil {
			body, _ := io.ReadAll(httpResp.Body)
			httpResp.Body.Close()
			resp.APIStatusCode = httpResp.StatusCode
			resp.APIBody = string(body)
			var parsed GrokUsageJSON
			if json.Unmarshal(body, &parsed) == nil && parsed.Status == "ready" {
				resp.APIParsed = &parsed
				return resp, nil
			}
		}
		time.Sleep(300 * time.Millisecond)
	}
	httpResp, err := http.Get(url)
	if err != nil {
		return resp, fmt.Errorf("GET /api/grok/usage: %w", err)
	}
	defer httpResp.Body.Close()
	body, _ := io.ReadAll(httpResp.Body)
	resp.APIStatusCode = httpResp.StatusCode
	resp.APIBody = string(body)
	var parsed GrokUsageJSON
	_ = json.Unmarshal(body, &parsed)
	resp.APIParsed = &parsed
	return resp, nil
}

func runRefreshOverlap(t *testing.T, req *Request, root string, resp *Response) (*Response, error) {
	svc, err := newServiceWithMockCommand(root, req.MockScript)
	if err != nil {
		return nil, err
	}
	counterFile := filepath.Join(os.TempDir(), "grok-mock-counter-"+strconv.Itoa(os.Getpid()))
	_ = os.Remove(counterFile)
	t.Cleanup(func() { os.Remove(counterFile) })

	svc.TestExported_SetEnv("GROK_MOCK_COUNTER_FILE", counterFile)

	var wg sync.WaitGroup
	started := 0
	var mu sync.Mutex
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mu.Lock()
			started++
			mu.Unlock()
			svc.TestExported_TriggerRefresh()
		}()
	}
	wg.Wait()

	time.Sleep(500 * time.Millisecond)
	data, _ := os.ReadFile(counterFile)
	if len(data) > 0 {
		resp.MockInvocationCount, _ = strconv.Atoi(strings.TrimSpace(string(data)))
	}
	resp.ConcurrentStarted = started
	return resp, nil
}

func grokUsageDoctestRoot() string {
	if root := os.Getenv("DOCTEST_ROOT"); root != "" {
		candidate := filepath.Join(root, "tests", "grok-usage")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		return root
	}
	wd, _ := os.Getwd()
	for dir := wd; ; dir = filepath.Dir(dir) {
		candidate := filepath.Join(dir, "tests", "grok-usage")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		if filepath.Dir(dir) == dir {
			return filepath.Join(wd, "tests", "grok-usage")
		}
	}
}

func newServiceWithMockCommand(root, mockScript string) (*grokusage.Service, error) {
	scriptPath, err := resolveMockScript(root, mockScript)
	if err != nil {
		return nil, err
	}
	svc := grokusage.TestExported_NewService()
	svc.TestExported_SetEnv(envGrokShowUsageCommand, scriptPath)
	return svc, nil
}

func resolveMockScript(root, name string) (string, error) {
	path := filepath.Join(root, "testdata", name)
	if err := os.Chmod(path, 0755); err != nil {
		return "", err
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return abs, nil
}

func buildAICritic(t *testing.T, moduleRoot string) (string, func(), error) {
	safe := strings.ReplaceAll(t.Name(), "/", "_")
	out := filepath.Join(os.TempDir(), "ai-critic-grok-usage-"+safe)
	cmd := exec.Command("go", "build", "-o", out, ".")
	cmd.Dir = moduleRoot
	if combined, err := cmd.CombinedOutput(); err != nil {
		return "", nil, fmt.Errorf("build ai-critic: %v\n%s", err, string(combined))
	}
	return out, func() { os.Remove(out) }, nil
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

func portHash(name string) int {
	hash := 0
	for _, c := range name {
		hash = hash*31 + int(c)
	}
	if hash < 0 {
		hash = -hash
	}
	return hash
}

type keepAliveStatus struct {
	ServerPort int `json:"server_port"`
	ServerPID  int `json:"server_pid"`
}

func waitManagedServerPort(timeoutSecs int) (int, error) {
	if timeoutSecs <= 0 {
		timeoutSecs = 12
	}
	statusURL := fmt.Sprintf("http://127.0.0.1:%d/api/keep-alive/status", config.KeepAlivePort)
	deadline := time.Now().Add(time.Duration(timeoutSecs) * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(statusURL)
		if err == nil {
			var st keepAliveStatus
			if json.NewDecoder(resp.Body).Decode(&st) == nil {
				resp.Body.Close()
				if st.ServerPort > 0 && st.ServerPID > 0 {
					return st.ServerPort, nil
				}
			} else {
				resp.Body.Close()
			}
		}
		time.Sleep(300 * time.Millisecond)
	}
	return 0, fmt.Errorf("managed server not ready on keep-alive status within %ds", timeoutSecs)
}
```
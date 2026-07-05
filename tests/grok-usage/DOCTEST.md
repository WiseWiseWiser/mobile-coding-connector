# Grok Usage Parser, Service, and API Doctests

Tests for `macosapp/grokusage` parsing, daemon-side in-process grok usage
fetch/cache, and `GET /api/grok/usage` on the keep-alive management server.
Service fetch leaves inject a mock `usage.Fetch` hook; API leaves drive the
daemon via `GROK_SHOW_USAGE_COMMAND` (no `GROK_SHOW_USAGE_BIN` shell wrapper).

# DSN (Domain Specific Notion)

**Participants**

- **Parser (`macosapp/grokusage`)** — `ParseShowUsageOutput` extracts
  `Weekly limit:` and `Next reset:` lines from command stdout.
- **Grok usage service (daemon)** — calls `agent/usage.Fetch(ctx, Grok)` in-process,
  caches `GrokUsageResponse`, refreshes every 60s, skips overlapping in-flight fetches.
- **Injectable fetch hook** — `TestExported_SetFetcher` replaces the default in-process
  fetch for deterministic service-layer tests (success, error, slow overlap).
- **Fake Grok TUI (`GROK_SHOW_USAGE_COMMAND`)** — env hook honored by
  `agent/grok/tty` for daemon API leaves.
- **Keep-alive daemon** — serves `GET /api/grok/usage` on management port `23312`
  when API leaves run.
- **HTTP client** — asserts JSON `status`, `weekly_limit`, `next_reset`, `error`,
  `updated_at` fields.

**Behaviors**

- Standard and noisy stdout parses to `UsageInfo`; missing fields return error.
- Injected fetch success → service `status=ready` with parsed limits.
- Injected fetch error → `status=error` with error message.
- API returns ready JSON after fake TUI fetch completes via env command hook.
- Concurrent refresh while fetch in flight does not start a second in-process fetch (count=1).

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
 +-- fetch/                           (GROUP)  service fetch via injectable hook
 |    +-- mock-command-success/       (LEAF)   status ready
 |    +-- mock-command-fails/          (LEAF)   status error
 |
 +-- api/                             (GROUP)  HTTP surface
 |    +-- get-usage-ready/            (LEAF)   GET /api/grok/usage JSON ready
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
| 5 | `fetch/mock-command-success` | Injected fetch → service ready |
| 6 | `fetch/mock-command-fails` | Injected fetch error → service error |
| 7 | `api/get-usage-ready` | HTTP API returns ready JSON |
| 8 | `refresh/skips-overlap` | Overlapping refresh does not double-fetch |

## Parameter Coverage

| Leaf | Op | Mock mechanism | Expect error |
|------|-----|--------------|--------------|
| standard-output | parse | fixture | false |
| extra-noise | parse | fixture | false |
| missing-weekly | parse | fixture | true |
| missing-reset | parse | fixture | true |
| mock-command-success | fetch | injectable success snapshot | false |
| mock-command-fails | fetch | injectable fetch error | false (service error status) |
| get-usage-ready | api | GROK_SHOW_USAGE_COMMAND fake TUI | false |
| skips-overlap | refresh | injectable slow fetcher | false |

## How to Run

```sh
doctest vet ./tests/grok-usage
doctest test ./tests/grok-usage/...
```

```go
import (
	"context"
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
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	agentusage "github.com/xhd2015/agent-pro/agent/usage"
	"github.com/xhd2015/ai-critic/macosapp/grokusage"
	"github.com/xhd2015/ai-critic/script/lib"
	"github.com/xhd2015/ai-critic/server/config"
)

type Request struct {
	Op string

	// Parse: fixture filename relative to tests/grok-usage/testdata/
	FixtureFile string

	// Fetch/Refresh: injectable fetch outcome
	FetchMode string // success | error | slow

	// API: fake TTY command hook for daemon child env
	ShowUsageCommand string

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

	FetchInvocationCount int
	ConcurrentStarted    int
}

func Run(t *testing.T, req *Request) (*Response, error) {
	resp := &Response{}
	doctestRoot := grokUsageDoctestRoot()

	switch req.Op {
	case "parse":
		return runParse(t, req, doctestRoot, resp)
	case "fetch":
		return runFetch(t, req, resp)
	case "api":
		return runAPI(t, req, doctestRoot, resp)
	case "refresh":
		return runRefreshOverlap(t, req, resp)
	default:
		return nil, fmt.Errorf("unknown op %q", req.Op)
	}
}

func runParse(t *testing.T, req *Request, root string, resp *Response) (*Response, error) {
	input, err := os.ReadFile(filepath.Join(root, "testdata", req.FixtureFile))
	if err != nil {
		return nil, err
	}
	info, parseErr := grokusage.ParseShowUsageOutput(string(input))
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

func runFetch(t *testing.T, req *Request, resp *Response) (*Response, error) {
	svc := grokusage.TestExported_NewService()
	grokusage.TestExported_SetFetcher(svc, fetcherForMode(req.FetchMode, nil))
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
	showCmd := req.ShowUsageCommand
	if showCmd == "" {
		showCmd = fakeGrokTUIDefault()
	}
	env = append(env,
		"AI_CRITIC_TEST_SKIP_EXTENSION=1",
		"GROK_SHOW_USAGE_COMMAND="+showCmd,
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

	url := fmt.Sprintf("http://127.0.0.1:%d/api/grok/usage", config.KeepAlivePort)
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

func runRefreshOverlap(t *testing.T, req *Request, resp *Response) (*Response, error) {
	var invocations atomic.Int32
	svc := grokusage.TestExported_NewService()
	grokusage.TestExported_SetFetcher(svc, fetcherForMode("slow", &invocations))

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
	resp.FetchInvocationCount = int(invocations.Load())
	resp.ConcurrentStarted = started
	return resp, nil
}

func fetcherForMode(mode string, counter *atomic.Int32) func(context.Context) (*agentusage.Snapshot, error) {
	switch mode {
	case "success", "slow", "":
		return func(ctx context.Context) (*agentusage.Snapshot, error) {
			if counter != nil {
				counter.Add(1)
				time.Sleep(2 * time.Second)
			}
			return grokSuccessSnapshot(), nil
		}
	case "error":
		return func(ctx context.Context) (*agentusage.Snapshot, error) {
			return nil, fmt.Errorf("mock grok usage fetch failed")
		}
	default:
		return func(ctx context.Context) (*agentusage.Snapshot, error) {
			return nil, fmt.Errorf("unknown fetch mode %q", mode)
		}
	}
}

func grokSuccessSnapshot() *agentusage.Snapshot {
	return &agentusage.Snapshot{
		Provider:     agentusage.Grok,
		UsagePercent: "6%",
		Reset:        "July 9, 16:55 PT",
	}
}

func fakeGrokTUIDefault() string {
	return `sh -c 'printf "Grok › "; read -r cmd; printf "Weekly limit: 6%%\nNext reset: July 9, 16:55 PT\n› "'`
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
```
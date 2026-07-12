# Codex Usage Parser, Service, and API Doctests

Tests for `macosapp/codexusage` parsing, daemon-side in-process codex usage
fetch/cache, and `GET /api/codex/usage` on the keep-alive management server.
Service fetch leaves inject a mock `usage.Fetch` hook; API leaves drive the
daemon via `CODEX_SHOW_STATUS_COMMAND` (no `CODEX_SHOW_STATUS_BIN` shell wrapper).

# DSN (Domain Specific Notion)

**Participants**

- **Parser (`macosapp/codexusage`)** — `ParseStatusOutput` extracts
  `Monthly usage:`, `Credits used:`, and `Next reset:` lines from command stdout.
- **Codex usage service (daemon)** — calls `agent/usage.Fetch(ctx, Codex)` in-process,
  caches `CodexUsageResponse`, refreshes every 60s, skips overlapping in-flight fetches.
- **Injectable fetch hook** — `TestExported_SetFetcher` replaces the default in-process
  fetch for deterministic service-layer tests (success, error, slow overlap).
- **Fake Codex TUI (`CODEX_SHOW_STATUS_COMMAND`)** — env hook honored by
  `agent/codex/tty` for daemon API leaves; isolated `TTY_WATCH_HOME` per run.
- **ai-critic-server subprocess** — serves `GET /api/codex/usage` on main server port
  `23712` when API leaves run (started by keep-alive harness).
- **Keep-alive daemon** — management port `23312` control plane only; spawns server.
- **HTTP client** — asserts JSON `status`, `monthly_usage`, `credits_used`,
  `credits_total`, `next_reset`, `error`, `updated_at` fields.

**Behaviors**

- Standard and noisy stdout parses to `UsageInfo`; missing monthly line returns error.
- Injected fetch success → service `status=ready` with parsed usage and formatted credits.
- Injected fetch error → `status=error` with error message.
- API returns ready JSON after fake TUI fetch completes via env command hook.
- Concurrent refresh while fetch in flight does not start a second in-process fetch (count=1).
- No binary resolution (`resolve.go` removed); no bundled `codex-show-status` indirection.

## Version

0.0.2

## Decision Tree

```
[codex usage]
 |
 +-- parse/                           (GROUP)  ParseStatusOutput
 |    +-- standard-output/            (LEAF)   three canonical lines
 |    +-- extra-noise/                (LEAF)   lines buried in scrollback
 |    +-- missing-monthly/            (LEAF)   parse error
 |
 +-- fetch/                           (GROUP)  service fetch via injectable hook
 |    +-- mock-command-success/       (LEAF)   status ready
 |    +-- mock-command-fails/          (LEAF)   status error
 |    +-- slow-boot-snapshot/         (LEAF)   synthetic slow TUI boot succeeds
 |    +-- timeout-slow-prompt/        (LEAF)   short timeout during 30s silent boot
 |    +-- timeout-no-status-response/ (LEAF)   short timeout when /status never renders
 |    +-- real-codex-inprocess/       (LEAF)   real codex CLI in-process fetch (slow)
 |
 +-- api/                             (GROUP)  HTTP surface
 |    +-- get-usage-ready/            (LEAF)   GET /api/codex/usage on server :23712
 |    +-- get-usage-timeout/           (LEAF)   GET :23712/api/codex/usage must not timeout-error
 |
 +-- tty-watch/                       (GROUP)  real tty-watch CLI timing
 |    +-- wait-idle-production-status/ (LEAF)  idle + /status\n\r ~16s
 |    +-- user-script-early-status/    (LEAF)   user 5+5 snapshot script fails
 |
 +-- refresh/                         (GROUP)  cache refresh semantics
 |    +-- skips-overlap/              (LEAF)   concurrent refresh skipped
 |
 +-- update-modal-skip/               (NESTED ROOT) auto-Skip Update available menu
      see tests/codex-usage/update-modal-skip/DOCTEST.md
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `parse/standard-output` | Parse canonical three-line output |
| 2 | `parse/extra-noise` | Parse usage buried in noise |
| 3 | `parse/missing-monthly` | Missing monthly line → error |
| 4 | `fetch/mock-command-success` | Injected fetch → service ready |
| 5 | `fetch/mock-command-fails` | Injected fetch error → service error |
| 6 | `fetch/slow-boot-snapshot` | Synthetic slow TUI → in-process fetch ready |
| 7 | `fetch/timeout-slow-prompt` | 30s silent boot → ready within service ctx (`slow`) |
| 8 | `fetch/timeout-no-status-response` | Never-respond fake → error (`slow && negative`) |
| 9 | `fetch/real-codex-inprocess` | Real codex CLI + daemon PATH → fetch ready |
| 10 | `api/get-usage-ready` | HTTP API on server port returns ready JSON |
| 11 | `api/get-usage-timeout` | HTTP API on server port returns timeout error (`slow && negative && requires-dist`) |
| 12 | `tty-watch/wait-idle-production-status` | Real CLI: idle then /status in ~16s (`slow && real-codex`) |
| 13 | `tty-watch/user-script-early-status` | Manual early script: no fields (`slow && real-codex && negative`) |
| 14 | `refresh/skips-overlap` | Overlapping refresh does not double-fetch |
| 15+ | `update-modal-skip/**` | Nested root: menu vs banner classify + auto-Skip fetch (see nested DOCTEST.md) |

## Parameter Coverage

| Leaf | Op | Mock mechanism | Expect parse error |
|------|-----|--------------|-------------------|
| standard-output | parse | show-status-standard.txt | false |
| extra-noise | parse | show-status-noisy.txt | false |
| missing-monthly | parse | show-status-missing-monthly.txt | true |
| mock-command-success | fetch | injectable success snapshot | false |
| mock-command-fails | fetch | injectable fetch error | false (service error status) |
| slow-boot-snapshot | fetch-inprocess | slow CODEX_SHOW_STATUS_COMMAND | false |
| timeout-slow-prompt | fetch-inprocess | 30s silent + 5s timeout | false |
| timeout-no-status-response | fetch-inprocess | never-respond → error | false |
| real-codex-inprocess | fetch-inprocess | real codex CLI (no command hook) | false |
| get-usage-ready | api | CODEX_SHOW_STATUS_COMMAND fake TUI | false |
| get-usage-timeout | api | never-respond → error JSON | false |
| wait-idle-production-status | ttywatch-real | wait idle + /status\n\r | false |
| user-script-early-status | ttywatch-real | early /status\r → no fields | false |
| skips-overlap | refresh | injectable slow fetcher | false |

## Run profiles (labels)

| Label | Meaning |
|-------|---------|
| `slow` | Long-running (fake TUI boot, ~90s timeout, or real codex) — skip in fast CI |
| `real-codex` | Requires real `codex` CLI on PATH |
| `negative` | Expects failure/error (anti-pattern or never-respond fake) |
| `requires-dist` | API leaf needs `ai-critic-react/dist` for daemon build |

```sh
# Fast default (no slow / real-codex / negative / requires-dist)
doctest test ./tests/codex-usage/...

# Slow integration including real codex
doctest test --label slow ./tests/codex-usage/...

# Real codex only
doctest test --label real-codex ./tests/codex-usage/...

# Negative contracts
doctest test --label negative ./tests/codex-usage/...
```

## How to Run

```sh
doctest vet ./tests/codex-usage
doctest test ./tests/codex-usage/...

# Nested update-modal-skip tree (auto-Skip + menu/banner classify):
doctest vet ./tests/codex-usage/update-modal-skip
doctest test ./tests/codex-usage/update-modal-skip/...
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

	codextty "github.com/xhd2015/agent-pro/agent/codex/tty"
	agentusage "github.com/xhd2015/agent-pro/agent/usage"
	"github.com/xhd2015/ai-critic/macosapp/codexusage"
	"github.com/xhd2015/ai-critic/script/lib"
	"github.com/xhd2015/ai-critic/server/config"
)

type Request struct {
	Op string

	// Parse: fixture filename relative to tests/codex-usage/testdata/
	FixtureFile string

	// Fetch/Refresh: injectable fetch outcome
	FetchMode string // success | error | slow

	// Fetch-inprocess: daemon-like env for real agent/usage.Fetch
	StripDaemonPATH bool
	UseRealCodex    bool
	RealCodexAttempts int

	// API / fetch-inprocess: fake TTY command hook
	ShowStatusCommand string
	TTYWatchHome      string
	SessionID         string
	FetchTimeoutSecs  int

	ExpectParseError bool
	WaitAPIReadySecs int
	WaitAPIError     bool // api: return when cached status=error

	// ttywatch-real: live CLI timing experiment
	TTYWatchMode   string // user-script | wait-idle-production
	BootPollCount  int
	StatusPollCount int
	MaxWaitSecs    int
}

type CodexUsageJSON struct {
	Status       string `json:"status"`
	MonthlyUsage string `json:"monthly_usage,omitempty"`
	CreditsUsed  string `json:"credits_used,omitempty"`
	CreditsTotal string `json:"credits_total,omitempty"`
	NextReset    string `json:"next_reset,omitempty"`
	Error        string `json:"error,omitempty"`
	UpdatedAt    string `json:"updated_at,omitempty"`
}

type Response struct {
	MonthlyUsage string
	CreditsUsed  string
	CreditsTotal string
	NextReset    string
	ParseErr     string

	ServiceStatus string
	ServiceError  string
	UpdatedAt     string

	APIStatusCode int
	APIBody       string
	APIParsed     *CodexUsageJSON

	FetchInvocationCount int
	ConcurrentStarted    int

	FetchAttemptCount  int
	FetchFailureCount  int
	FetchErrors        []string
	ResolvedCodexPath  string

	TTYWatchTranscript string
	PromptReadySecs    int
	StatusReadySecs    int
	TotalElapsedSecs   int
	StatusFieldsSeen   bool
	LastSnapshot       string
}

func Run(t *testing.T, req *Request) (*Response, error) {
	resp := &Response{}
	doctestRoot := codexUsageDoctestRoot()

	switch req.Op {
	case "parse":
		return runParse(t, req, doctestRoot, resp)
	case "fetch":
		return runFetch(t, req, resp)
	case "fetch-inprocess":
		return runFetchInProcess(t, req, resp)
	case "api":
		return runAPI(t, req, doctestRoot, resp)
	case "refresh":
		return runRefreshOverlap(t, req, resp)
	case "ttywatch-real":
		return runTTYWatchReal(t, req, resp)
	default:
		return nil, fmt.Errorf("unknown op %q", req.Op)
	}
}

func runParse(t *testing.T, req *Request, root string, resp *Response) (*Response, error) {
	input, err := os.ReadFile(filepath.Join(root, "testdata", req.FixtureFile))
	if err != nil {
		return nil, err
	}
	info, parseErr := codexusage.ParseStatusOutput(string(input))
	if parseErr != nil {
		resp.ParseErr = parseErr.Error()
		return resp, nil
	}
	if info != nil {
		resp.MonthlyUsage = info.MonthlyUsage
		resp.CreditsUsed = info.CreditsUsed
		resp.CreditsTotal = info.CreditsTotal
		resp.NextReset = info.NextReset
	}
	return resp, nil
}

func runFetch(t *testing.T, req *Request, resp *Response) (*Response, error) {
	svc := codexusage.TestExported_NewService()
	codexusage.TestExported_SetFetcher(svc, fetcherForMode(req.FetchMode, nil))
	out := svc.TestExported_FetchOnce(t)
	resp.ServiceStatus = string(out.Status)
	resp.ServiceError = out.Error
	resp.MonthlyUsage = out.MonthlyUsage
	resp.CreditsUsed = out.CreditsUsed
	resp.CreditsTotal = out.CreditsTotal
	resp.NextReset = out.NextReset
	resp.UpdatedAt = out.UpdatedAt
	return resp, nil
}

func runFetchInProcess(t *testing.T, req *Request, resp *Response) (*Response, error) {
	t.Helper()
	attempts := req.RealCodexAttempts
	if attempts <= 0 {
		attempts = 1
	}
	resp.FetchAttemptCount = attempts

	var last codexusage.CodexUsageResponse
	for i := 0; i < attempts; i++ {
		attemptReq := *req
		if attempts > 1 || req.UseRealCodex {
			attemptReq.TTYWatchHome = filepath.Join(t.TempDir(), fmt.Sprintf("tty-watch-%d", i))
		}
		sessionBase := strings.TrimSpace(req.SessionID)
		if sessionBase == "" {
			sessionBase = "codex-status-usage"
		}
		attemptReq.SessionID = fmt.Sprintf("%s-%d", sessionBase, i+1)

		restore := applyInProcessFetchEnv(t, &attemptReq)
		if attemptReq.UseRealCodex && resp.ResolvedCodexPath == "" {
			resp.ResolvedCodexPath = strings.TrimSpace(os.Getenv("AGENT_RUNNER_CODEX_PATH"))
		}

		svc := codexusage.TestExported_NewService()
		out := svc.TestExported_FetchOnce(t)
		restore()

		last = out
		if out.Status != codexusage.StatusReady {
			resp.FetchFailureCount++
			resp.FetchErrors = append(resp.FetchErrors, fmt.Sprintf("attempt %d: %s", i+1, out.Error))
			continue
		}
	}

	resp.ServiceStatus = string(last.Status)
	resp.ServiceError = last.Error
	resp.MonthlyUsage = last.MonthlyUsage
	resp.CreditsUsed = last.CreditsUsed
	resp.CreditsTotal = last.CreditsTotal
	resp.NextReset = last.NextReset
	resp.UpdatedAt = last.UpdatedAt
	return resp, nil
}

func applyInProcessFetchEnv(t *testing.T, req *Request) func() {
	t.Helper()
	keys := []string{
		"PATH",
		"TTY_WATCH_HOME",
		"CODEX_SHOW_STATUS_COMMAND",
		"CODEX_SHOW_STATUS_SESSION_ID",
		"CODEX_SHOW_STATUS_TIMEOUT",
		"AGENT_RUNNER_CODEX_PATH",
	}
	prev := make(map[string]string, len(keys))
	for _, k := range keys {
		prev[k] = os.Getenv(k)
	}

	daemonPATH := "/usr/bin:/bin:/usr/sbin:/sbin"
	if req.UseRealCodex {
		codexPath, err := resolveRealCodexCLI(t)
		if err != nil {
			t.Skipf("real codex CLI not available: %v", err)
		}
		req.ShowStatusCommand = codexPath
		_ = os.Unsetenv("CODEX_SHOW_STATUS_COMMAND")
		_ = os.Setenv("AGENT_RUNNER_CODEX_PATH", codexPath)
		binDir := filepath.Dir(codexPath)
		if req.StripDaemonPATH {
			_ = os.Setenv("PATH", daemonPATH+":"+binDir)
		}
	} else if req.StripDaemonPATH {
		_ = os.Setenv("PATH", daemonPATH)
	}

	ttyHome := req.TTYWatchHome
	if ttyHome == "" {
		ttyHome = filepath.Join(t.TempDir(), ".tty-watch")
	}
	_ = os.Setenv("TTY_WATCH_HOME", ttyHome)
	if !req.UseRealCodex {
		showCmd := req.ShowStatusCommand
		if showCmd == "" {
			showCmd = slowBootFakeCodexTUI()
		}
		_ = os.Setenv("CODEX_SHOW_STATUS_COMMAND", showCmd)
	}
	sid := strings.TrimSpace(req.SessionID)
	if sid == "" {
		sid = "codex-status-usage"
	}
	_ = os.Setenv("CODEX_SHOW_STATUS_SESSION_ID", sid)
	timeout := req.FetchTimeoutSecs
	if timeout <= 0 {
		timeout = 60
	}
	_ = os.Setenv("CODEX_SHOW_STATUS_TIMEOUT", strconv.Itoa(timeout))
	return func() {
		for _, k := range keys {
			if v, ok := prev[k]; ok {
				_ = os.Setenv(k, v)
			} else {
				_ = os.Unsetenv(k)
			}
		}
	}
}

func resolveRealCodexCLI(t *testing.T) (string, error) {
	t.Helper()
	if v := strings.TrimSpace(os.Getenv("CODEX_USAGE_TEST_CODEX_PATH")); v != "" {
		if _, err := os.Stat(v); err == nil {
			return v, nil
		}
	}
	for _, shell := range []struct {
		bin  string
		args []string
	}{
		{"bash", []string{"-lic", "command -v codex"}},
		{"zsh", []string{"-lic", "command -v codex"}},
	} {
		out, err := exec.Command(shell.bin, shell.args...).Output()
		if err != nil {
			continue
		}
		path := strings.TrimSpace(string(out))
		if path == "" {
			continue
		}
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("codex not found via login shell")
}

func runTTYWatchReal(t *testing.T, req *Request, resp *Response) (*Response, error) {
	t.Helper()
	codexPath, err := resolveRealCodexCLI(t)
	if err != nil {
		t.Skipf("real codex CLI not available: %v", err)
	}
	ttyWatch, err := resolveTTYWatchCLI(t)
	if err != nil {
		t.Skipf("tty-watch CLI not available: %v", err)
	}
	resp.ResolvedCodexPath = codexPath

	ttyHome := req.TTYWatchHome
	if ttyHome == "" {
		ttyHome = filepath.Join(t.TempDir(), ".tty-watch")
	}
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		sessionID = "codex-ttywatch-" + strings.ReplaceAll(t.Name(), "/", "-")
	}
	maxWait := req.MaxWaitSecs
	if maxWait <= 0 {
		maxWait = 30
	}

	env := append(os.Environ(),
		"TTY_WATCH_HOME="+ttyHome,
		"PATH=/usr/bin:/bin:/usr/sbin:/sbin:"+filepath.Dir(codexPath)+":"+filepath.Dir(ttyWatch),
	)
	var transcript strings.Builder
	logf := func(format string, args ...any) {
		line := fmt.Sprintf(format, args...)
		transcript.WriteString(line)
		transcript.WriteString("\n")
	}

	start := time.Now()
	elapsed := func() int { return int(time.Since(start).Seconds()) }

	run := func(name string, args ...string) (string, error) {
		cmd := exec.Command(name, args...)
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		text := strings.TrimSpace(string(out))
		logf("[%ds] %s %s", elapsed(), name, strings.Join(args, " "))
		if text != "" {
			logf("%s", text)
		}
		return text, err
	}
	snapshot := func(label string) string {
		out, err := run(ttyWatch, "snapshot", sessionID)
		if err != nil {
			logf("snapshot error: %v", err)
		}
		logf("[%ds] %s", elapsed(), label)
		if out != "" {
			logf("%s", out)
		}
		resp.LastSnapshot = out
		return out
	}
	parseSnapshot := func(snap string) bool {
		info, err := codextty.ParseStatusSnapshot(snap)
		if err != nil {
			return false
		}
		resp.StatusFieldsSeen = true
		resp.MonthlyUsage = info.MonthlyUsage
		resp.CreditsUsed = info.CreditsUsed
		resp.CreditsTotal = info.CreditsTotal
		resp.NextReset = info.NextReset
		return true
	}
	promptIdle := func(snap string) bool {
		if !strings.Contains(snap, "›") && !strings.Contains(snap, "\u203a") {
			return false
		}
		lower := strings.ToLower(snap)
		return !strings.Contains(lower, "model:") || !strings.Contains(lower, "loading")
	}

	codexArgs := []string{
		"run", "--session-id", sessionID, "--detach", codexPath,
		"--dangerously-bypass-approvals-and-sandbox", "-c", "mcp_servers={}",
	}
	if _, err := run(ttyWatch, codexArgs...); err != nil {
		return resp, fmt.Errorf("tty-watch run: %w", err)
	}
	_, _ = run(ttyWatch, "list")

	mode := strings.TrimSpace(req.TTYWatchMode)
	switch mode {
	case "user-script":
		bootPolls := req.BootPollCount
		if bootPolls <= 0 {
			bootPolls = 5
		}
		statusPolls := req.StatusPollCount
		if statusPolls <= 0 {
			statusPolls = 5
		}
		for i := 0; i < bootPolls; i++ {
			snapshot(fmt.Sprintf("boot snapshot %d", i))
			time.Sleep(time.Second)
		}
		logf("[%ds] send /status\\r (user-script)", elapsed())
		if _, err := run(ttyWatch, "send", sessionID, "/status\r"); err != nil {
			return resp, fmt.Errorf("tty-watch send: %w", err)
		}
		for i := 0; i < statusPolls; i++ {
			snap := snapshot(fmt.Sprintf("post-status snapshot %d", i))
			if parseSnapshot(snap) {
				resp.StatusReadySecs = elapsed()
				break
			}
			time.Sleep(time.Second)
		}

	case "wait-idle-production", "":
		deadline := start.Add(time.Duration(maxWait) * time.Second)
		for i := 0; resp.PromptReadySecs == 0 && time.Now().Before(deadline); i++ {
			snap := snapshot(fmt.Sprintf("wait-prompt poll %d", i))
			if promptIdle(snap) {
				resp.PromptReadySecs = elapsed()
				logf("[%ds] prompt idle", resp.PromptReadySecs)
				break
			}
			time.Sleep(time.Second)
		}
		logf("[%ds] send /status\\n\\r (production)", elapsed())
		if _, err := run(ttyWatch, "send", sessionID, "/status\n\r"); err != nil {
			return resp, fmt.Errorf("tty-watch send: %w", err)
		}
		time.Sleep(time.Second)
		for i := 0; !resp.StatusFieldsSeen && time.Now().Before(deadline); i++ {
			snap := snapshot(fmt.Sprintf("wait-status poll %d", i))
			if parseSnapshot(snap) {
				resp.StatusReadySecs = elapsed()
				logf("[%ds] status fields visible", resp.StatusReadySecs)
				break
			}
			time.Sleep(time.Second)
		}

	default:
		return resp, fmt.Errorf("unknown TTYWatchMode %q", mode)
	}

	resp.TotalElapsedSecs = elapsed()
	resp.TTYWatchTranscript = transcript.String()
	_, _ = run(ttyWatch, "kill", sessionID)
	logf("SUMMARY prompt_ready=%ds status_ready=%ds total=%ds status_seen=%v",
		resp.PromptReadySecs, resp.StatusReadySecs, resp.TotalElapsedSecs, resp.StatusFieldsSeen)
	resp.TTYWatchTranscript = transcript.String()
	return resp, nil
}

func resolveTTYWatchCLI(t *testing.T) (string, error) {
	t.Helper()
	if v := strings.TrimSpace(os.Getenv("CODEX_USAGE_TEST_TTY_WATCH_PATH")); v != "" {
		if _, err := os.Stat(v); err == nil {
			return v, nil
		}
	}
	for _, shell := range []struct {
		bin  string
		args []string
	}{
		{"bash", []string{"-lic", "command -v tty-watch"}},
		{"zsh", []string{"-lic", "command -v tty-watch"}},
	} {
		out, err := exec.Command(shell.bin, shell.args...).Output()
		if err != nil {
			continue
		}
		path := strings.TrimSpace(string(out))
		if path != "" {
			if _, err := os.Stat(path); err == nil {
				return path, nil
			}
		}
	}
	return "", fmt.Errorf("tty-watch not found via login shell")
}

func slowBootFakeCodexTUI() string {
	return slowBootFakeCodexTUIWithDelay(12)
}

func slowBootFakeCodexTUIWithDelay(silentSecs int) string {
	if silentSecs <= 0 {
		silentSecs = 12
	}
	// Silent PTY (Codex cloud-config stall), then prompt + /status fields.
	return fmt.Sprintf(`sh -c 'sleep %d; printf "Codex › "; read -r cmd; printf "Monthly credit limit: 42%%%% left (resets 08:00 on 1 Aug)\n6,519 of 11,250 credits used\n› "'`, silentSecs)
}

func neverRespondFakeCodexTUI() string {
	// Prompt appears quickly; /status input is read but no parseable output is rendered.
	return `sh -c 'printf "Codex › "; read -r cmd; sleep 120'`
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
	daemonLog := filepath.Join(configHome, "codex-usage-api.log")
	cmd := exec.Command(aiBin,
		"keep-alive",
		"--kill-existing",
		"--port", strconv.Itoa(port),
		"--forever",
		"--log", daemonLog,
	)
	cmd.Dir = configHome
	env := lib.AppendTestServerEnv(os.Environ(), configHome)
	ttyHome := req.TTYWatchHome
	if ttyHome == "" {
		ttyHome = filepath.Join(t.TempDir(), ".tty-watch")
	}
	showCmd := req.ShowStatusCommand
	if showCmd == "" {
		showCmd = fakeCodexTUIDefault()
	}
	sid := strings.TrimSpace(req.SessionID)
	if sid == "" {
		sid = "codex-status-usage"
	}
	timeout := req.FetchTimeoutSecs
	if timeout <= 0 {
		timeout = 60
	}
	env = append(env,
		"AI_CRITIC_TEST_SKIP_EXTENSION=1",
		"TTY_WATCH_HOME="+ttyHome,
		"CODEX_SHOW_STATUS_COMMAND="+showCmd,
		"CODEX_SHOW_STATUS_SESSION_ID="+sid,
		"CODEX_SHOW_STATUS_TIMEOUT="+strconv.Itoa(timeout),
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
	url := fmt.Sprintf("http://127.0.0.1:%d/api/codex/usage", serverPort)
	deadline := time.Now().Add(time.Duration(req.WaitAPIReadySecs) * time.Second)
	for time.Now().Before(deadline) {
		httpResp, err := http.Get(url)
		if err == nil {
			body, _ := io.ReadAll(httpResp.Body)
			httpResp.Body.Close()
			resp.APIStatusCode = httpResp.StatusCode
			resp.APIBody = string(body)
			var parsed CodexUsageJSON
			if json.Unmarshal(body, &parsed) == nil {
				if parsed.Status == "ready" {
					resp.APIParsed = &parsed
					return resp, nil
				}
				if req.WaitAPIError && parsed.Status == "error" && parsed.UpdatedAt != "" {
					resp.APIParsed = &parsed
					return resp, nil
				}
			}
		}
		time.Sleep(300 * time.Millisecond)
	}
	httpResp, err := http.Get(url)
	if err != nil {
		return resp, fmt.Errorf("GET /api/codex/usage: %w", err)
	}
	defer httpResp.Body.Close()
	body, _ := io.ReadAll(httpResp.Body)
	resp.APIStatusCode = httpResp.StatusCode
	resp.APIBody = string(body)
	var parsed CodexUsageJSON
	_ = json.Unmarshal(body, &parsed)
	resp.APIParsed = &parsed
	return resp, nil
}

func runRefreshOverlap(t *testing.T, req *Request, resp *Response) (*Response, error) {
	var invocations atomic.Int32
	svc := codexusage.TestExported_NewService()
	codexusage.TestExported_SetFetcher(svc, fetcherForMode("slow", &invocations))

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
			return codexSuccessSnapshot(), nil
		}
	case "error":
		return func(ctx context.Context) (*agentusage.Snapshot, error) {
			return nil, fmt.Errorf("mock codex usage fetch failed")
		}
	default:
		return func(ctx context.Context) (*agentusage.Snapshot, error) {
			return nil, fmt.Errorf("unknown fetch mode %q", mode)
		}
	}
}

func codexSuccessSnapshot() *agentusage.Snapshot {
	return &agentusage.Snapshot{
		Provider:     agentusage.Codex,
		UsagePercent: "58%",
		CreditsUsed:  "6519",
		CreditsTotal: "11250",
		Reset:        "08:00 on 1 Aug",
	}
}

func fakeCodexTUIDefault() string {
	return `sh -c 'printf "Codex › "; read -r cmd; printf "Monthly credit limit: 42%% left (resets 08:00 on 1 Aug)\n6,519 of 11,250 credits used\n› "'`
}

func codexUsageDoctestRoot() string {
	if root := os.Getenv("DOCTEST_ROOT"); root != "" {
		candidate := filepath.Join(root, "tests", "codex-usage")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		return root
	}
	wd, _ := os.Getwd()
	for dir := wd; ; dir = filepath.Dir(dir) {
		candidate := filepath.Join(dir, "tests", "codex-usage")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		if filepath.Dir(dir) == dir {
			return filepath.Join(wd, "tests", "codex-usage")
		}
	}
}

func buildAICritic(t *testing.T, moduleRoot string) (string, func(), error) {
	safe := strings.ReplaceAll(t.Name(), "/", "_")
	out := filepath.Join(os.TempDir(), "ai-critic-codex-usage-"+safe)
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
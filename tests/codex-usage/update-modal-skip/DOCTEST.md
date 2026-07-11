# Codex Update Modal Auto-Skip Doctests

Signed-protocol tests for auto-Skip on Codex's blocking **Update available**
menu and for distinguishing that menu from a residual non-blocking **banner**.

Fixtures: `tests/codex-usage/testdata/update-modal-skip/` (see `PROTOCOL.md`).

# DSN (Domain Specific Notion)

**Participants**

- **Codex TUI (fake or live)** — interactive process in a tty-watch PTY. May show a
  full-screen **Update available** menu (options under `›`, `Press enter to continue`)
  or a residual top **banner** that still says “Update available” without menu options.
- **Signed snapshot fixtures** — live captures from codex-cli 0.143.0 under
  `../testdata/update-modal-skip/*.snapshot.txt` with SHA-256 in `PROTOCOL.md`.
- **Menu classifiers (`agenttty`)** — production helpers under test:
  - `IsBlockingUpdateMenu(text) bool` — true only for the blocking menu modal
  - `UpdateMenuSelection(text) string` — `UPDATE_NOW` \| `SKIP` \| `SKIP_UNTIL_NEXT` \| `""`
  - `checkCodexWritable` / `codex-tty` `CheckWritable` — must not treat residual banner
    as permanent “update available” loading once the menu is gone (still honor
    `model:loading`).
- **FetchStatus (`agent/codex/tty`)** — `waitForPrompt` must auto-Skip (CSI Down →
  verify `›` on Skip → Enter → poll until menu gone) before sending `/status`.
- **Fake Codex TUI scripts** — `fake-tui-auto-skip.py` / `fake-tui-stuck-update-now.py`
  driven via `CODEX_SHOW_STATUS_COMMAND` for in-process fetch leaves.
- **Codex usage service** — `TestExported_FetchOnce` → `agent/usage.Fetch` → ready
  usage fields (monthly / credits / reset).

**Behaviors**

- Fixture `01`: blocking menu, selection `UPDATE_NOW`; writable not idle (update).
- Fixture `02`: blocking menu, selection `SKIP`; writable not idle (update).
- Fixture `03b`/`04`: residual banner only (not blocking menu); writable must not
  report reason `codex update available` (banner alone is non-blocking). With
  `model: loading` present, state remains `loading` for model reason. With model
  not loading, prompt `›` → idle.
- Fixture `05`: `ParseStatusSnapshot` extracts monthly/credits/reset.
- Auto-Skip fake TUI: fetch-inprocess → `status=ready` with usage fields (not timeout).
- Stuck-on-Update-now fake: error (cannot select Skip / timeout); never `ready`;
  must not Enter while selection is Update now.

## Version

0.0.2

## Decision Tree

```
tests/codex-usage/update-modal-skip/          (NESTED ROOT)
 |
 +-- classify/                                (GROUP) fixture predicates
 |    +-- blocking-menu/                      (GROUP) IsBlocking=true
 |    |    +-- default-update-now/            (LEAF)  01 → UPDATE_NOW
 |    |    +-- skip-selected/                 (LEAF)  02 → SKIP
 |    +-- residual-banner/                    (GROUP) IsBlocking=false
 |    |    +-- menu-dismissed/                (LEAF)  03b → not menu; reason ≠ update available
 |    |    +-- banner-alone-idle/             (LEAF)  04 stripped of model:loading → idle
 |    +-- status-fields/                      (GROUP)
 |         +-- parse-succeeds/                (LEAF)  05 → monthly/credits/reset
 |
 +-- fetch/                                   (GROUP) in-process FetchStatus
      +-- auto-skip-ready/                    (LEAF)  fake menu → Skip → ready usage
      +-- stuck-on-update-now/                (LEAF)  refuse Skip → error (negative)
```

Parameter ranking (most → least significant):

1. **Op class** — fixture classify vs end-to-end fetch
2. **Screen kind** — blocking menu vs residual banner vs status screen vs fake mode
3. **Selection / outcome** — UPDATE_NOW vs SKIP; ready vs error

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `classify/blocking-menu/default-update-now` | `01` → blocking + `UPDATE_NOW` + writable loading(update) |
| 2 | `classify/blocking-menu/skip-selected` | `02` → blocking + `SKIP` + writable loading(update) |
| 3 | `classify/residual-banner/menu-dismissed` | `03b` → not blocking; writable reason not update-available |
| 4 | `classify/residual-banner/banner-alone-idle` | banner without model:loading → idle ready |
| 5 | `classify/status-fields/parse-succeeds` | `05` → ParseStatusSnapshot fields |
| 6 | `fetch/auto-skip-ready` | fake modal auto-Skip → service ready |
| 7 | `fetch/stuck-on-update-now` | never leave Update now → error (`slow && negative`) |

## Run profiles (labels)

| Label | Meaning |
|-------|---------|
| `slow` | May wait full service timeout (~90s) until implementer early-exits |
| `negative` | Expects error / non-ready |

## How to Run

```sh
doctest vet ./tests/codex-usage/update-modal-skip
doctest test ./tests/codex-usage/update-modal-skip/...

# Parent tree (existing leaves) stays independent:
doctest vet ./tests/codex-usage
doctest test ./tests/codex-usage/...
```

Expected **RED** until implementer:

1. Adds `agenttty.IsBlockingUpdateMenu` / `agenttty.UpdateMenuSelection` (compile RED
   until present; then assertion RED until logic is correct).
2. Narrows `checkCodexWritable` update gate to **menu** (not bare “update available”).
3. Implements Skip protocol in `waitForPrompt` (CSI Down → verify → Enter → poll).

```go
import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	codextty "github.com/xhd2015/agent-pro/agent/codex/tty"
	"github.com/xhd2015/agent-pro/pkgs/agenttty"
	"github.com/xhd2015/ai-critic/macosapp/codexusage"
)

// Request drives classify + fetch leaves for the update-modal-skip protocol.
type Request struct {
	Op string // classify | fetch-inprocess

	// classify
	FixtureFile string // basename under fixtures dir
	// When true, strip "model: ... loading" so residual banner idle can be asserted.
	StripModelLoading bool

	// fetch-inprocess
	ShowStatusCommand string
	TTYWatchHome      string
	SessionID         string
	FetchTimeoutSecs  int
	StripDaemonPATH   bool
	MarkerDir         string // FAKE_CODEX_MARKER_DIR for negative Enter detection
}

type Response struct {
	// classify
	SnapshotText   string
	IsBlockingMenu bool
	MenuSelection  string // UPDATE_NOW | SKIP | SKIP_UNTIL_NEXT | ""
	WritableReady  bool
	WritableState  string
	WritableReason string

	// status parse (classify status-fields or via ParseStatusSnapshot)
	MonthlyUsage string
	CreditsUsed  string
	CreditsTotal string
	NextReset    string
	ParseErr     string

	// fetch-inprocess
	ServiceStatus string
	ServiceError  string
	UpdatedAt     string
	MarkerFiles   []string
}

func Run(t *testing.T, req *Request) (*Response, error) {
	resp := &Response{}
	switch strings.TrimSpace(req.Op) {
	case "classify":
		return runClassify(t, req, resp)
	case "fetch-inprocess":
		return runFetchInProcess(t, req, resp)
	default:
		return nil, fmt.Errorf("unknown op %q", req.Op)
	}
}

func fixturesDir() string {
	// Nested DOCTEST root is tests/codex-usage/update-modal-skip.
	// Shared signed fixtures live at ../testdata/update-modal-skip.
	candidates := []string{
		filepath.Join(DOCTEST_ROOT, "..", "testdata", "update-modal-skip"),
		filepath.Join(DOCTEST_ROOT, "testdata", "update-modal-skip"),
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && st.IsDir() {
			return c
		}
	}
	// Walk from CWD for robustness outside generated packages.
	wd, _ := os.Getwd()
	for dir := wd; ; dir = filepath.Dir(dir) {
		c := filepath.Join(dir, "tests", "codex-usage", "testdata", "update-modal-skip")
		if st, err := os.Stat(c); err == nil && st.IsDir() {
			return c
		}
		c2 := filepath.Join(dir, "testdata", "update-modal-skip")
		if st, err := os.Stat(c2); err == nil && st.IsDir() {
			return c2
		}
		if filepath.Dir(dir) == dir {
			break
		}
	}
	return filepath.Join(DOCTEST_ROOT, "..", "testdata", "update-modal-skip")
}

func runClassify(t *testing.T, req *Request, resp *Response) (*Response, error) {
	t.Helper()
	if strings.TrimSpace(req.FixtureFile) == "" {
		return nil, fmt.Errorf("FixtureFile required for classify")
	}
	path := filepath.Join(fixturesDir(), req.FixtureFile)
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	text := string(raw)
	if req.StripModelLoading {
		// Turn model:loading into a settled model line so only the residual
		// update banner remains as the potential false-positive gate.
		var b strings.Builder
		for i, line := range strings.Split(text, "\n") {
			if i > 0 {
				b.WriteByte('\n')
			}
			lower := strings.ToLower(line)
			if strings.Contains(lower, "model:") && strings.Contains(lower, "loading") {
				line = strings.ReplaceAll(line, "loading", "gpt-5.5")
				line = strings.ReplaceAll(line, "Loading", "gpt-5.5")
			}
			b.WriteString(line)
		}
		text = b.String()
	}
	resp.SnapshotText = text

	// Production classifiers (implementer must export these on agenttty).
	// RED: missing symbols until implementer adds them; then wrong logic fails Assert.
	resp.IsBlockingMenu = agenttty.IsBlockingUpdateMenu(text)
	resp.MenuSelection = agenttty.UpdateMenuSelection(text)

	provider, ok := agenttty.Get("codex-tty")
	if !ok {
		return nil, fmt.Errorf("codex-tty provider not registered")
	}
	st := provider.CheckWritable([]byte(text))
	resp.WritableReady = st.Ready
	resp.WritableState = st.State
	resp.WritableReason = st.Reason

	// Always attempt status parse (leaf asserts only when expecting fields).
	if info, parseErr := codextty.ParseStatusSnapshot(text); parseErr != nil {
		resp.ParseErr = parseErr.Error()
	} else if info != nil {
		resp.MonthlyUsage = info.MonthlyUsage
		resp.CreditsUsed = info.CreditsUsed
		resp.CreditsTotal = info.CreditsTotal
		resp.NextReset = info.NextReset
	}
	return resp, nil
}

func runFetchInProcess(t *testing.T, req *Request, resp *Response) (*Response, error) {
	t.Helper()
	restore := applyInProcessFetchEnv(t, req)
	defer restore()

	svc := codexusage.TestExported_NewService()
	out := svc.TestExported_FetchOnce(t)
	resp.ServiceStatus = string(out.Status)
	resp.ServiceError = out.Error
	resp.MonthlyUsage = out.MonthlyUsage
	resp.CreditsUsed = out.CreditsUsed
	resp.CreditsTotal = out.CreditsTotal
	resp.NextReset = out.NextReset
	resp.UpdatedAt = out.UpdatedAt

	if req.MarkerDir != "" {
		entries, _ := os.ReadDir(req.MarkerDir)
		for _, e := range entries {
			if !e.IsDir() {
				resp.MarkerFiles = append(resp.MarkerFiles, e.Name())
			}
		}
	}
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
		"FAKE_CODEX_MARKER_DIR",
	}
	prev := make(map[string]string, len(keys))
	for _, k := range keys {
		prev[k] = os.Getenv(k)
	}

	if req.StripDaemonPATH {
		_ = os.Setenv("PATH", "/usr/bin:/bin:/usr/sbin:/sbin")
	}
	ttyHome := req.TTYWatchHome
	if ttyHome == "" {
		ttyHome = filepath.Join(t.TempDir(), ".tty-watch")
	}
	_ = os.Setenv("TTY_WATCH_HOME", ttyHome)

	showCmd := strings.TrimSpace(req.ShowStatusCommand)
	if showCmd == "" {
		showCmd = autoSkipFakeCommand()
	}
	_ = os.Setenv("CODEX_SHOW_STATUS_COMMAND", showCmd)

	sid := strings.TrimSpace(req.SessionID)
	if sid == "" {
		sid = "codex-update-modal-skip"
	}
	_ = os.Setenv("CODEX_SHOW_STATUS_SESSION_ID", sid)

	timeout := req.FetchTimeoutSecs
	if timeout <= 0 {
		timeout = 60
	}
	_ = os.Setenv("CODEX_SHOW_STATUS_TIMEOUT", strconv.Itoa(timeout))

	if req.MarkerDir != "" {
		_ = os.MkdirAll(req.MarkerDir, 0o755)
		_ = os.Setenv("FAKE_CODEX_MARKER_DIR", req.MarkerDir)
	}

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

func autoSkipFakeCommand() string {
	return fakePythonCommand("fake-tui-auto-skip.py")
}

func stuckUpdateNowFakeCommand() string {
	return fakePythonCommand("fake-tui-stuck-update-now.py")
}

func fakePythonCommand(scriptName string) string {
	script := filepath.Join(fixturesDir(), scriptName)
	// Prefer absolute path so daemon-stripped PATH still finds the script via
	// explicit python3 argv[0] from /usr/bin when StripDaemonPATH is set.
	if abs, err := filepath.Abs(script); err == nil {
		script = abs
	}
	python := "python3"
	if p, err := exec.LookPath("python3"); err == nil {
		python = p
	}
	return python + " " + script
}
```

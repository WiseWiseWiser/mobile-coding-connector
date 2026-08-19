# Local-agent native-terminals CLI Doctests (in-process)

Classic TDD for **`local-agent native-terminals`** (aliases: `native-terminal`,
`native-terms`, `native-term`) list/focus + inline split TUI — **in-process**, no daemon.

**L2:** `agentcli.RunWithWriters` + `testhooks` HOME + injected Capture/Layout/Focus;
library TUI via `itermswitcher.View` / `ApplyKey`.
No `httptest`, no `--server`/`--token`, no live iTerm, no `label: e2e`.

Canonical command name in help and unknown-subcommand errors: **`native-terminals`**.
Bare `terminals` is removed. PTY `terminal` is unchanged.

# DSN (Domain Specific Notion)

**Participants**

- **local-agent CLI (`native-terminals` + aliases)** — `native-terminals` | `native-terminal` | `native-terms` | `native-term` → same `list` / `focus` / TUI.
- **remote-agent CLI** — rejects all four names (`unknown command: <typed>`).
- **agentcli.RunWithWriters** — L2 entry under `agentcliMu` + testhooks HOME.
- **localiterm2 library** — file last-good (`~/.ai-critic/iterm-inventory-cache.json` via HOME override) + incremental layout-diff + Focus hook.
- **testhooks inject** — Capture / Layout / ITermRunning / Focus + call counters; `SetTerminalsKeys` / `SetTerminalsInput` for TUI (no os.Stdin hijack).
- **itermswitcher** — human batch formatters + pure TUI `UIState` / `View` / `ApplyKey` (inline split, no alt-screen).

**Behaviors**

- `local-agent native-terminals` (any alias) never talks to the local server/app/daemon (no resolve client, no ping, no token, no `/api/local/iterm2/*`).
- Distinct from PTY `local-agent terminal`. Bare `terminals` is unknown. `remote-agent` rejects all four native-* names.
- Help Usage is always `local-agent native-terminals …` and lists `Aliases: native-terminal, native-terms, native-term`.
- Unknown subcommand (even via alias): `Error: unknown native-terminals subcommand: foo`.
- **Mode (stdout, not stdin):** TTY → inline split TUI; pipe/file → batch print. `--tty` forces TUI; `--no-tty` forces batch; both → error. `--json` always batch (no ANSI, even with `--tty`). `--color` + `--no-color` → error.
- **Warm list (batch):** complete last-good file → load + Layout increment; same IDs → no deep Capture; human mentions `incremental` or `up to date`.
- **Cold list:** missing file → full Capture, print, write cache file.
- **TUI first paint:** file last-good → status `cached` before Layout/Capture; split sidebar + list; rewrite box in place (leave last frame in scrollback). Keys: ↑↓/jk list, ←→/Tab panes, [] space filter, Enter focus+exit, q/Esc quit.
- `--json`: last inventory JSON on stdout, no ANSI, trailing `\n`.
- `--refresh`: force full recapture (Capture even when file warm); may add `sess-b`.
- `focus <id>`: in-process Focus hook (not POST). Missing arg / unknown id → `Error:` on stderr, non-zero.
- iTerm down: warning `iTerm2 is not running` (`FormatEmptyITerm`), not a missing command.
- Help text must **not** mention `/api`, `GET`, `POST`, server, or stream URL.
- User-facing stdout ends with `\n` (batch). Errors use `Error:` prefix on stderr.

## Version

0.0.2

## Decision Tree

```
[local-iterm2-switcher-cli]
 |
 +-- help/                         (GROUP)  discoverability (no HTTP wording)
 |    +-- top-level/               (LEAF)  native-terminals -h → list, focus, aliases
 |    +-- list/                    (LEAF)  list -h → Usage native-terminals, --json
 |    +-- focus/                   (LEAF)  focus -h → session-id
 |
 +-- list/                         (GROUP)  in-process batch list
 |    +-- human-warm-uptodate/     (LEAF)  seed file; Layout only; human tokens
 |    +-- json-last/               (LEAF)  --json last inventory, no ANSI
 |    +-- refresh-full/            (LEAF)  --refresh Capture + sess-b
 |    +-- iterm-down/              (LEAF)  warning iTerm2 is not running
 |
 +-- focus/                        (GROUP)  in-process Focus hook
 |    +-- ok/                      (LEAF)  FocusCalled sess-a, "focused"
 |    +-- missing-session/         (LEAF)  unknown id → Error
 |    +-- missing-arg/             (LEAF)  no id → Error, FocusCalled=false
 |
 +-- dispatch/                     (GROUP)  command identity + aliases
 |    +-- remote-unknown/          (LEAF)  remote-agent rejects all 4 names
 |    +-- unknown-subcommand/      (LEAF)  native-term foo → canonical error
 |    +-- aliases-equivalent/      (LEAF)  each alias list -h → Usage native-terminals
 |    +-- bare-terminals-unknown/  (LEAF)  bare terminals → unknown command
 |
 +-- flags/                        (GROUP)  mode / color conflicts
 |    +-- tty-and-no-tty/          (LEAF)  --tty --no-tty → Error
 |    +-- color-and-no-color/      (LEAF)  --color --no-color → Error
 |    +-- json-is-batch/           (LEAF)  --json --tty → JSON, no box/ANSI
 |
 +-- tui/                          (GROUP)  inline split TUI
      +-- render/                  (GROUP)  library View
      |    +-- warm-split/         (LEAF)  Terminals + sidebar + list + cached
      |    +-- list-cursor/        (LEAF)  default › on first session row
      +-- keys/                    (GROUP)  library ApplyKey
      |    +-- down-moves/         (LEAF)  j moves › to next session
      |    +-- tab-sidebar/        (LEAF)  Tab focuses sidebar
      |    +-- bracket-space/      (LEAF)  ] next Desktop filter
      |    +-- enter-focus/        (LEAF)  Enter → focus action + id
      |    +-- q-quit/             (LEAF)  q → quit action
      +-- cli/                     (GROUP)  --tty + scripted keys
           +-- paint-then-q/       (LEAF)  seed; --tty; q; split + cached
           +-- enter-focuses/      (LEAF)  seed; --tty; Enter → Focus sess-a
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `help/top-level` | `native-terminals -h` documents list/focus + aliases; no `/api` |
| 2 | `help/list` | `list -h` Usage `native-terminals`, `--json`/`--refresh`; no GET/stream |
| 3 | `help/focus` | `focus -h` documents session-id; no POST |
| 4 | `list/human-warm-uptodate` | Seed file; CaptureCalls=0; Layout≥1; Desktop 1 + grok review; incremental\|up to date; no HTTP leak |
| 5 | `list/json-last` | `--json` last inventory with sess-a; no ANSI; trailing `\n` |
| 6 | `list/refresh-full` | Seed file; `--refresh` Capture≥1 and sess-b |
| 7 | `list/iterm-down` | Warning `iTerm2 is not running` |
| 8 | `focus/ok` | FocusCalled, sess-a, exit 0, "focused" |
| 9 | `focus/missing-session` | Unknown id → Error, non-zero |
| 10 | `focus/missing-arg` | No id → Error, FocusCalled=false |
| 11 | `dispatch/remote-unknown` | remote-agent rejects all 4 names (`unknown command: <typed>`) |
| 12 | `dispatch/unknown-subcommand` | `native-term foo` → `unknown native-terminals subcommand: foo` |
| 13 | `dispatch/aliases-equivalent` | each of 4 names + `list -h` → Usage native-terminals + `--json` |
| 14 | `dispatch/bare-terminals-unknown` | bare `terminals list` → `unknown command: terminals` |
| 15 | `flags/tty-and-no-tty` | `--tty --no-tty` → Error cannot be specified together |
| 16 | `flags/color-and-no-color` | `--color --no-color` → Error cannot be specified together |
| 17 | `flags/json-is-batch` | `--json --tty` → Inventory JSON, no `┌`, no ANSI |
| 18 | `tui/render/warm-split` | Library View: Terminals, All, Desktop 1, grok review, cached, box |
| 19 | `tui/render/list-cursor` | Default `›` on first session (list), not only All |
| 20 | `tui/keys/down-moves` | `j` moves `›` to next session (2 sessions) |
| 21 | `tui/keys/tab-sidebar` | Tab → sidebar focus; list still visible |
| 22 | `tui/keys/bracket-space` | `]` advances Desktop filter; cross-space hide |
| 23 | `tui/keys/enter-focus` | Enter → action focus + session id |
| 24 | `tui/keys/q-quit` | `q` → action quit |
| 25 | `tui/cli/paint-then-q` | `--tty` + `q`; split + grok review + cached; FocusCalled=false |
| 26 | `tui/cli/enter-focuses` | `--tty` + Enter; FocusCalled + FocusSession=sess-a |

## Expected implementer surface

1. Register `native-terminals` + aliases `native-terminal`, `native-terms`, `native-term` (same handler). Remove bare `terminals`.
2. Help Usage always `native-terminals`; list aliases; unknown subcommand uses canonical name.
3. `agentcli` local list/focus: **no** `resolve()` / `client.Client` / HTTP.
4. Use `localiterm2` with cache under `testhooks.UserHomeDir()` → `~/.ai-critic/iterm-inventory-cache.json`.
5. Wire Capture/Layout/ITermRunning/Focus from testhooks; TUI keys from `TerminalsInput`.
6. Mode: TTY/`--tty` → split TUI; `--no-tty`/pipe → batch; `--json` always batch.
7. Help: no `/api`, `GET`, `POST`, server, or stream URL wording.

## How to Run

```sh
doctest vet ./tests/local-iterm2-switcher-cli
doctest test ./tests/local-iterm2-switcher-cli/...
```

```go
import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/xhd2015/ai-critic/cmd/agentcli"
	"github.com/xhd2015/ai-critic/cmd/agentcli/testhooks"
	"github.com/xhd2015/ai-critic/macosapp/itermswitcher"
	"github.com/xhd2015/ai-critic/server/localiterm2"
	"github.com/xhd2015/doctest/session"
	shelliterm "github.com/xhd2015/dot-pkgs/go-pkgs/shell/iterm2"
	kooliterm "github.com/xhd2015/kool/tools/iterm2"
)

// agentcliMu serializes testhooks + RunWithWriters (process-global hooks).
var agentcliMu sync.Mutex

// Request drives one leaf. No Mode=source / client leaf (CLI is not HTTP).
type Request struct {
	// Op: "" or "cli" (default RunWithWriters) | "tui" (library View/ApplyKey).
	Op string

	// Args after global flags, e.g. ["native-terminals", "list", "--json"].
	Args []string

	// CommandNames: multi-name dispatch loops (remote-unknown, aliases-equivalent).
	// remote: each name alone under RemoteProfile.
	// local + Args suffix (e.g. list -h): each name + Args.
	CommandNames []string

	// Profile: "local" (default) | "remote"
	Profile string

	// SkipHooks: help / unknown-command leaves (no Capture/Layout/Focus install).
	SkipHooks bool

	// SeedCache: write complete last-good Inventory JSON under HOME before Run.
	SeedCache bool
	// SeedTwoSessions: seed sess-a (Desktop 1) + sess-b (Desktop 2) for key nav.
	SeedTwoSessions bool

	ITermDown   bool
	SecondSnapB bool // Capture returns sess-a+sess-b (for --refresh)
	WindowSpace int  // reserved; fixture uses Desktop 1 / space 0

	// TUI library (Op=tui)
	// ApplyKeys: sequence for ApplyKey (e.g. ["j"], ["tab"], ["enter"], ["q"], ["]"]).
	ApplyKeys []string
	// TUIStatus initial chrome status (default "cached").
	TUIStatus string

	// CLI TUI: scripted keys via testhooks.SetTerminalsKeys (e.g. "q", "\r").
	Keys string
}

type Response struct {
	ExitCode int
	Stdout   string
	Stderr   string
	Combined string
	ErrMsg   string

	// In-process hook counters (testhooks; stay 0 until CLI calls the hooks).
	CaptureCalls int
	LayoutCalls  int
	FocusCalled  bool
	FocusSession string

	// HitHTTP is true if argv/output still looks like daemon HTTP / resolve client.
	HitHTTP bool

	CacheFileExists      bool
	CacheFileHasSessionA bool

	// Library / TUI render
	ViewText          string // joined View lines (ANSI stripped for asserts if needed)
	ViewRaw           string // raw joined lines
	FocusPane         string // list | sidebar
	ListCursorSession string // session id under › on list, if any
	HasListCursor     bool   // › present on a session row
	HasSidebarCursor  bool
	TUIAction         string // focus | quit | ""
	TUIActionSession  string
	SidebarFilterID   string // selected sidebar id after keys

	// Multi-name dispatch (remote-unknown / aliases-equivalent)
	// MultiOK is true when every CommandNames entry met the leaf contract.
	MultiOK      bool
	MultiDetail  string // first failure detail
	MultiChecked int
}

// nativeTerminalCommandNames is the locked equivalent set (canonical first).
func nativeTerminalCommandNames() []string {
	return []string{"native-terminals", "native-terminal", "native-terms", "native-term"}
}

func Run(t *testing.T, d *session.Doctest, req *Request) (*Response, error) {
	t.Helper()
	resp := &Response{}
	if req.Op == "tui" {
		return runTUI(t, req, resp)
	}
	if len(req.CommandNames) > 0 {
		return runMultiNames(t, req, resp)
	}
	return runCLI(t, req, resp)
}

func runMultiNames(t *testing.T, req *Request, resp *Response) (*Response, error) {
	t.Helper()
	ok := true
	detail := ""
	checked := 0
	var outParts, errParts []string
	for _, name := range req.CommandNames {
		sub := *req
		sub.CommandNames = nil
		if req.Profile == "remote" {
			sub.Args = []string{name}
			sub.Profile = "remote"
			sub.SkipHooks = true
		} else {
			// name + Args suffix (e.g. list -h)
			sub.Args = append([]string{name}, req.Args...)
			sub.SkipHooks = true
		}
		one := &Response{}
		if _, err := runCLI(t, &sub, one); err != nil {
			resp.MultiOK = false
			resp.MultiDetail = name + ": harness " + err.Error()
			resp.MultiChecked = checked
			resp.ExitCode = 1
			return resp, nil
		}
		checked++
		outParts = append(outParts, one.Stdout)
		errParts = append(errParts, one.Stderr+"\n"+one.ErrMsg)
		if req.Profile == "remote" {
			text := one.Stderr + "\n" + one.ErrMsg
			want := "unknown command: " + name
			if one.ExitCode == 0 || !strings.Contains(text, want) {
				ok = false
				if detail == "" {
					detail = fmt.Sprintf("%s: want %q; exit=%d err=%q", name, want, one.ExitCode, text)
				}
			}
		} else {
			// aliases-equivalent: list -h
			blob := one.Stdout + "\n" + one.Stderr + "\n" + one.ErrMsg
			if one.ExitCode != 0 {
				ok = false
				if detail == "" {
					detail = fmt.Sprintf("%s: exit=%d err=%q", name, one.ExitCode, blob)
				}
				continue
			}
			if !strings.Contains(blob, "native-terminals") {
				ok = false
				if detail == "" {
					detail = fmt.Sprintf("%s: help missing Usage native-terminals; out=%q", name, blob)
				}
			}
			if !strings.Contains(blob, "--json") {
				ok = false
				if detail == "" {
					detail = fmt.Sprintf("%s: help missing --json; out=%q", name, blob)
				}
			}
		}
	}
	resp.MultiOK = ok
	resp.MultiDetail = detail
	resp.MultiChecked = checked
	resp.Stdout = strings.Join(outParts, "\n---\n")
	resp.Stderr = strings.Join(errParts, "\n---\n")
	resp.Combined = strings.TrimSpace(resp.Stdout + "\n" + resp.Stderr)
	if !ok {
		resp.ExitCode = 1
	} else if req.Profile == "remote" {
		resp.ExitCode = 1 // remote rejects are non-zero (expected by leaf)
	} else {
		resp.ExitCode = 0
	}
	return resp, nil
}

func runTUI(t *testing.T, req *Request, resp *Response) (*Response, error) {
	t.Helper()
	inv := seedInventory(req)
	status := req.TUIStatus
	if status == "" {
		status = itermswitcher.StatusCached
	}
	state := itermswitcher.NewUIState(inv, status)
	var last itermswitcher.UIAction
	for _, key := range req.ApplyKeys {
		state, last = itermswitcher.ApplyKey(state, key)
	}
	resp.TUIAction = last.Name
	resp.TUIActionSession = last.SessionID
	resp.FocusPane = state.FocusPane
	if state.SidebarIndex >= 0 && state.SidebarIndex < len(state.SidebarIDs) {
		resp.SidebarFilterID = state.SidebarIDs[state.SidebarIndex]
	}
	lines := itermswitcher.View(state)
	resp.ViewRaw = strings.Join(lines, "\n")
	resp.ViewText = stripANSI(resp.ViewRaw)
	fillCursorProbe(resp, resp.ViewText, state)
	return resp, nil
}

func fillCursorProbe(resp *Response, view string, state itermswitcher.UIState) {
	resp.HasListCursor = false
	resp.HasSidebarCursor = false
	resp.ListCursorSession = ""
	for _, line := range strings.Split(view, "\n") {
		if !strings.Contains(line, "›") {
			continue
		}
		low := strings.ToLower(line)
		// Sidebar rows: All / Bookmarks / Desktop N
		if strings.Contains(low, "all") || strings.Contains(low, "bookmark") || strings.Contains(low, "desktop ") {
			resp.HasSidebarCursor = true
		}
		// Session rows: primary name or path
		if strings.Contains(low, "grok review") {
			resp.HasListCursor = true
			resp.ListCursorSession = "sess-a"
		} else if strings.Contains(low, "second pane") {
			resp.HasListCursor = true
			resp.ListCursorSession = "sess-a2"
		} else if strings.Contains(low, "new tab") {
			resp.HasListCursor = true
			resp.ListCursorSession = "sess-b"
		} else if strings.Contains(line, "/") || strings.Contains(line, "sess-") {
			resp.HasListCursor = true
		}
	}
	_ = state // FocusPane already copied to resp
}

func runCLI(t *testing.T, req *Request, resp *Response) (*Response, error) {
	t.Helper()
	args := append([]string(nil), req.Args...)
	if len(args) == 0 {
		args = []string{"native-terminals", "-h"}
	}

	prof := agentcli.LocalProfile()
	if req.Profile == "remote" {
		prof = agentcli.RemoteProfile()
	}

	home := t.TempDir()
	if req.SeedCache || req.SeedTwoSessions {
		writeSeedCache(t, home, req.SeedTwoSessions)
	}

	var stdout, stderr bytes.Buffer

	agentcliMu.Lock()
	defer agentcliMu.Unlock()
	testhooks.SetHomeOverride(home)
	testhooks.SetReachabilityForTest("up")
	defer testhooks.ResetInProcessOverrides()

	if !req.SkipHooks {
		installTerminalsHooks(req)
	}
	if req.Keys != "" {
		testhooks.SetTerminalsKeys(req.Keys)
	}

	// Never pass --server / --token: in-process must not need the daemon.
	runErr := agentcli.RunWithWriters(prof, args, &stdout, &stderr)
	resp.Stdout = stdout.String()
	resp.Stderr = stderr.String()
	if runErr != nil {
		resp.ExitCode = 1
		resp.ErrMsg = runErr.Error()
		if !strings.Contains(resp.Stderr, "Error:") {
			resp.Stderr += fmt.Sprintf("Error: %v\n", runErr)
		}
	}
	resp.Combined = strings.TrimSpace(resp.Stdout + "\n" + resp.Stderr)

	resp.CaptureCalls = testhooks.TerminalsCaptureCalls()
	resp.LayoutCalls = testhooks.TerminalsLayoutCalls()
	resp.FocusCalled, resp.FocusSession = testhooks.TerminalsFocusSession()
	resp.HitHTTP = detectHTTPLeak(args, resp.Stdout, resp.Stderr, resp.ErrMsg)
	fillCacheFileStats(resp, home)
	resp.ViewText = stripANSI(resp.Stdout)
	return resp, nil
}

func installTerminalsHooks(req *Request) {
	testhooks.SetTerminalsCapture(func() (*kooliterm.Snapshot, error) {
		if req.ITermDown {
			return nil, fmt.Errorf("Error: iTerm2 is not running")
		}
		if req.SecondSnapB {
			return fixtureSnapB(), nil
		}
		return fixtureSnap(), nil
	})
	testhooks.SetTerminalsLayout(func() (*kooliterm.Snapshot, error) {
		if req.SecondSnapB || req.SeedTwoSessions {
			return layoutIDsOnly(fixtureSnapTwoSpaces()), nil
		}
		return layoutIDsOnly(fixtureSnap()), nil
	})
	testhooks.SetTerminalsITermRunning(func() bool {
		return !req.ITermDown
	})
	testhooks.SetTerminalsFocus(func(ref shelliterm.SessionRef) error {
		if strings.TrimSpace(ref.SessionID) == "" {
			return fmt.Errorf("session-id is required")
		}
		if ref.SessionID != "sess-a" && ref.SessionID != "sess-b" {
			return fmt.Errorf("session not found: %s", ref.SessionID)
		}
		return nil
	})
}

func detectHTTPLeak(args []string, stdout, stderr, errMsg string) bool {
	for _, a := range args {
		if a == "--server" || a == "--token" || strings.HasPrefix(a, "http://") || strings.HasPrefix(a, "https://") {
			return true
		}
	}
	blob := stdout + "\n" + stderr + "\n" + errMsg
	low := strings.ToLower(blob)
	if strings.Contains(low, "/api/local/iterm2") {
		return true
	}
	if strings.Contains(low, "localhost") && strings.Contains(low, "/api/") {
		return true
	}
	// resolve-client failures when CLI still dials the daemon
	if strings.Contains(low, "connection refused") || strings.Contains(low, "dial tcp") {
		return true
	}
	if strings.Contains(low, "failed to") && strings.Contains(low, "server") {
		return true
	}
	return false
}

func fillCacheFileStats(resp *Response, home string) {
	path := filepath.Join(home, ".ai-critic", "iterm-inventory-cache.json")
	b, err := os.ReadFile(path)
	if err != nil {
		return
	}
	resp.CacheFileExists = true
	if strings.Contains(string(b), "sess-a") {
		resp.CacheFileHasSessionA = true
	}
}

func writeSeedCache(t *testing.T, home string, twoSessions bool) {
	t.Helper()
	dir := filepath.Join(home, ".ai-critic")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "iterm-inventory-cache.json")
	body := fixtureLastGoodCacheJSON()
	if twoSessions {
		body = fixtureTwoSessionCacheJSON()
	}
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

func seedInventory(req *Request) localiterm2.Inventory {
	raw := fixtureLastGoodCacheJSON()
	if req != nil && req.SeedTwoSessions {
		raw = fixtureTwoSessionCacheJSON()
	}
	var inv localiterm2.Inventory
	_ = json.Unmarshal([]byte(raw), &inv)
	return inv
}

func stripANSI(s string) string {
	re := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	return re.ReplaceAllString(s, "")
}

// fixtureLastGoodCacheJSON matches P1 server last-good (sess-a, cwd ai-critic).
func fixtureLastGoodCacheJSON() string {
	return `{
  "iterm_running": true,
  "cached_at": "2026-08-14T12:00:00Z",
  "from_cache": false,
  "refreshing": false,
  "desktops": [
    {
      "space_index": 0,
      "desktop": 1,
      "sessions": [
        {
          "session_id": "sess-a",
          "session_name": "grok review",
          "window_id": "42",
          "window_name": "ai-critic",
          "tab_index": 2,
          "tab_name": "grok",
          "cwd": "/Users/xhd2015/proj/ai-critic",
          "idle": false,
          "bookmarked": false,
          "space_index": 0,
          "desktop": 1
        }
      ]
    },
    {
      "space_index": 1,
      "desktop": 2,
      "sessions": []
    }
  ],
  "saved_notes": []
}`
}

// fixtureTwoSessionCacheJSON: sess-a on Desktop 1, sess-b on Desktop 2.
func fixtureTwoSessionCacheJSON() string {
	return `{
  "iterm_running": true,
  "cached_at": "2026-08-14T12:00:00Z",
  "from_cache": false,
  "refreshing": false,
  "desktops": [
    {
      "space_index": 0,
      "desktop": 1,
      "sessions": [
        {
          "session_id": "sess-a",
          "session_name": "grok review",
          "window_id": "42",
          "window_name": "ai-critic",
          "tab_index": 2,
          "tab_name": "grok",
          "cwd": "/Users/xhd2015/proj/ai-critic",
          "idle": false,
          "bookmarked": false,
          "space_index": 0,
          "desktop": 1
        },
        {
          "session_id": "sess-a2",
          "session_name": "second pane",
          "window_id": "42",
          "window_name": "ai-critic",
          "tab_index": 3,
          "tab_name": "sh",
          "cwd": "/tmp/a2",
          "idle": true,
          "bookmarked": false,
          "space_index": 0,
          "desktop": 1
        }
      ]
    },
    {
      "space_index": 1,
      "desktop": 2,
      "sessions": [
        {
          "session_id": "sess-b",
          "session_name": "new tab",
          "window_id": "43",
          "window_name": "other",
          "tab_index": 1,
          "tab_name": "sh",
          "cwd": "/tmp/other",
          "idle": true,
          "bookmarked": false,
          "space_index": 1,
          "desktop": 2
        }
      ]
    }
  ],
  "saved_notes": []
}`
}

func fixtureSnap() *kooliterm.Snapshot {
	cwd := "/Users/xhd2015/proj/ai-critic"
	idle := false
	return &kooliterm.Snapshot{
		Windows: []kooliterm.SnapshotWindow{{
			Index:    1,
			Name:     "ai-critic",
			WindowID: 42,
			Tabs: []kooliterm.SnapshotTab{{
				Index: 2,
				Name:  "grok",
				Sessions: []kooliterm.SnapshotSession{{
					Index: 1,
					ID:    "sess-a",
					Name:  "grok review",
					Cwd:   &cwd,
					Idle:  &idle,
				}},
			}},
		}},
	}
}

func fixtureSnapB() *kooliterm.Snapshot {
	snap := fixtureSnap()
	cwd := "/tmp/other"
	idle := true
	snap.Windows[0].Tabs[0].Sessions = append(snap.Windows[0].Tabs[0].Sessions, kooliterm.SnapshotSession{
		Index: 2,
		ID:    "sess-b",
		Name:  "new tab",
		Cwd:   &cwd,
		Idle:  &idle,
	})
	return snap
}

func fixtureSnapTwoSpaces() *kooliterm.Snapshot {
	snap := fixtureSnap()
	cwd := "/tmp/other"
	idle := true
	snap.Windows = append(snap.Windows, kooliterm.SnapshotWindow{
		Index:    2,
		Name:     "other",
		WindowID: 43,
		Tabs: []kooliterm.SnapshotTab{{
			Index: 1,
			Name:  "sh",
			Sessions: []kooliterm.SnapshotSession{{
				Index: 1,
				ID:    "sess-b",
				Name:  "new tab",
				Cwd:   &cwd,
				Idle:  &idle,
			}},
		}},
	})
	return snap
}

func layoutIDsOnly(snap *kooliterm.Snapshot) *kooliterm.Snapshot {
	if snap == nil {
		return nil
	}
	out := *snap
	out.Windows = make([]kooliterm.SnapshotWindow, len(snap.Windows))
	for i, w := range snap.Windows {
		nw := w
		nw.Tabs = make([]kooliterm.SnapshotTab, len(w.Tabs))
		for j, tab := range w.Tabs {
			nt := tab
			nt.Sessions = make([]kooliterm.SnapshotSession, len(tab.Sessions))
			for k, s := range tab.Sessions {
				nt.Sessions[k] = kooliterm.SnapshotSession{
					Index: s.Index,
					ID:    s.ID,
					Name:  s.Name,
				}
			}
			nw.Tabs[j] = nt
		}
		out.Windows[i] = nw
	}
	return &out
}
```

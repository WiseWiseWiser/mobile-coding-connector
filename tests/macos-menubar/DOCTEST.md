# macOS Menu Bar Usage Formatting Doctests

Pure-function tests for `macosapp/menubar` formatters — the Go spec mirrored by the
Swift `ai-critic-macos` menu-bar client when rendering grok/codex usage state.

# DSN (Domain Specific Notion)

**Participants**

- **FormatGrokLabel (`macosapp/menubar`)** — maps daemon `GrokUsageResponse` fields
  (`status`, `weekly_limit`, `error`) to a compact menu-bar string prefixed with
  `Grok ` (backward-compatible helper).
- **FormatMenuBarLabel** — selects grok or codex title from display mode
  (`rotating`|`grok`|`codex`) and rotating index; handles loading/error truncation.
- **FormatGrokDropdownLine** — single-line grok dropdown text:
  `Grok: {pct}(Weekly), Reset {local}, {timeLeft}`.
- **FormatCodexDropdownLine** — single-line codex dropdown text:
  `Codex: {pct}(Monthly) {used}/{total}, Reset {local}, {timeLeft}`.
- **FormatResetDisplay** — parses provider reset string to an absolute instant,
  converts to `now.Location()` (local), formats as `{Month} {day}, {HH}:{mm}`
  (24h clock, no timezone suffix); unparseable → raw string unchanged.
- **FormatTimeLeft** — parses provider reset strings (grok `July 9, 16:55 PT` or codex
  `08:00 on 1 Aug`), infers year from `now`, and returns compound relative text
  (`left 3d4h`, `left 4h5m`, `left 5m`, `left 0m`, or empty when unparseable).
- **Test harness** — invokes formatters with leaf-provided inputs and fixed `now`;
  no UI or network.

**Behaviors**

- `FormatGrokLabel`: `ready` + `weekly_limit` → `Grok {limit}`; `loading` → `Grok ...`;
  `error` → fixed short `Grok err` (not full daemon message).
- `formatCodexLabel`: `error` → fixed short `Codex err`.
- `FormatMenuBarLabel`: fixed `grok`/`codex` modes; rotating index 0 → grok, 1 → codex.
- `FormatResetDisplay`: grok PT reset → local wall clock; codex `08:00 on 1 Aug` →
  `Aug 1, 08:00`; unparseable → pass through raw reset string.
- `FormatTimeLeft`: compound two-tier units (≥24h → `d`+`h`, ≥1h → `h`+`m`, <1h → `m`);
  omit zero tail units; minutes floor to at least 1 when 0 < duration < 1h;
  duration ≤ 0 → `left 0m`; unparseable/empty reset → empty string.
- `FormatGrokDropdownLine`: ready → `Grok: {pct}(Weekly), Reset {local}, {timeLeft}`;
  `error` → `Grok: Error: {msg}` (no reset suffix on error); unparseable reset omits
  `{timeLeft}` but still shows `Reset {raw}`.
- `FormatCodexDropdownLine`: ready → `Codex: {pct}(Monthly) {used}/{total}, Reset {local}, {timeLeft}`;
  `error` → `Codex: Error: {msg}` (no reset suffix on error).
- Compact menu-bar pill labels (`label/*`) remain unchanged — no reset or relative time.

## Version

0.0.4

## Decision Tree

```
[menubar formatting]
 |
 +-- label/                           (GROUP)  menu-bar title labels (unchanged)
 |    +-- ready/                      (LEAF)   FormatGrokLabel ready + weekly_limit
 |    +-- loading/                    (LEAF)   FormatGrokLabel loading placeholder
 |    +-- error/                      (LEAF)   FormatGrokLabel error → `Grok err`
 |    +-- error-truncate/             (LEAF)   FormatGrokLabel long error → `Grok err`
 |    +-- codex-error/                (LEAF)   formatCodexLabel error → `Codex err`
 |    +-- grok-fixed/                 (LEAF)   FormatMenuBarLabel mode=grok
 |    +-- codex-fixed/                (LEAF)   FormatMenuBarLabel mode=codex
 |    +-- rotating-grok-slot/         (LEAF)   FormatMenuBarLabel rotating index=0
 |    +-- rotating-codex-slot/       (LEAF)   FormatMenuBarLabel rotating index=1
 |
 +-- relative-time/                   (GROUP)  FormatTimeLeft compound countdown
 |    +-- three-days/                 (LEAF)   exact 72h → `left 3d`
 |    +-- three-days-four-hours/      (LEAF)   76h → `left 3d4h`
 |    +-- two-days-five-hours/        (LEAF)   53h → `left 2d5h`
 |    +-- three-hours-five-minutes/   (LEAF)   3h5m → `left 3h5m`
 |    +-- four-hours-five-minutes/    (LEAF)   4h5m → `left 4h5m`
 |    +-- five-minutes/               (LEAF)   <1h → `left 5m`
 |    +-- two-minutes/                (LEAF)   <1h → `left 2m`
 |    +-- codex-three-days/           (LEAF)   codex reset format → `left 3d`
 |    +-- one-minute-floor/           (LEAF)   90s remaining → `left 1m`
 |    +-- zero-minutes/               (LEAF)   duration ≤ 0 → `left 0m`
 |    +-- unparseable-fallback/       (LEAF)   unparseable reset → empty
 |
 +-- reset-display/                   (GROUP)  FormatResetDisplay local time
 |    +-- grok-same-tz/               (LEAF)   PT reset in PDT → same wall clock
 |    +-- grok-cross-tz/              (LEAF)   PT reset in EDT → +3h local
 |    +-- grok-date-rollover/         (LEAF)   PT reset in JST → next calendar day
 |    +-- codex-reformat/             (LEAF)   `08:00 on 1 Aug` → `Aug 1, 08:00`
 |    +-- unparseable-fallback/       (LEAF)   unparseable → raw string
 |
 +-- dropdown/                        (GROUP)  dropdown single lines
 |    +-- grok-line/                  (LEAF)   FormatGrokDropdownLine ready + relative
 |    +-- grok-unparseable-reset/     (LEAF)   ready, unparseable reset → no `left`
 |    +-- grok-error/                 (LEAF)   FormatGrokDropdownLine error msg
 |    +-- codex-line/                 (LEAF)   FormatCodexDropdownLine ready + relative
 |    +-- codex-error/                (LEAF)   FormatCodexDropdownLine error msg
 |    +-- codex-timeout-error/        (LEAF)   FormatCodexDropdownLine timeout error
 |
 +-- client/                          (GROUP)  Swift grok/codex server-port contract
      +-- swift-grok-codex-server-port/ (LEAF)  usage via ServerClient :23712
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `label/ready` | `FormatGrokLabel` ready + `6%` → `Grok 6%` |
| 2 | `label/loading` | `FormatGrokLabel` loading → `Grok ...` |
| 3 | `label/error` | `FormatGrokLabel` error → `Grok err` |
| 4 | `label/error-truncate` | `FormatGrokLabel` long error → `Grok err` |
| 5 | `label/codex-error` | `formatCodexLabel` error → `Codex err` |
| 6 | `label/grok-fixed` | `FormatMenuBarLabel` mode=grok → `Grok 6%` |
| 7 | `label/codex-fixed` | `FormatMenuBarLabel` mode=codex → `Codex 58%` |
| 8 | `label/rotating-grok-slot` | rotating index=0 → `Grok 6%` |
| 9 | `label/rotating-codex-slot` | rotating index=1 → `Codex 58%` |
| 10 | `relative-time/three-days` | `FormatTimeLeft` grok reset → `left 3d` |
| 11 | `relative-time/three-days-four-hours` | 76h remaining → `left 3d4h` |
| 12 | `relative-time/two-days-five-hours` | 53h remaining → `left 2d5h` |
| 13 | `relative-time/three-hours-five-minutes` | 3h5m remaining → `left 3h5m` |
| 14 | `relative-time/four-hours-five-minutes` | 4h5m remaining → `left 4h5m` |
| 15 | `relative-time/five-minutes` | 5m remaining → `left 5m` |
| 16 | `relative-time/two-minutes` | 2m remaining → `left 2m` |
| 17 | `relative-time/codex-three-days` | `FormatTimeLeft` codex reset → `left 3d` |
| 18 | `relative-time/one-minute-floor` | 90s remaining → `left 1m` |
| 19 | `relative-time/zero-minutes` | reset at or before `now` → `left 0m` |
| 20 | `relative-time/unparseable-fallback` | `soon` → empty |
| 21 | `reset-display/grok-same-tz` | `July 9, 17:55 PT` in PDT → `July 9, 17:55` |
| 22 | `reset-display/grok-cross-tz` | `July 9, 17:55 PT` in EDT → `July 9, 20:55` |
| 23 | `reset-display/grok-date-rollover` | `July 9, 17:55 PT` in JST → `July 10, 09:55` |
| 24 | `reset-display/codex-reformat` | `08:00 on 1 Aug` → `Aug 1, 08:00` |
| 25 | `reset-display/unparseable-fallback` | `soon` → `soon` (unchanged) |
| 26 | `dropdown/grok-line` | `Grok: 6%(Weekly), Reset July 9, 16:55, left 3d` |
| 27 | `dropdown/grok-unparseable-reset` | ready + `soon` → no `left` suffix |
| 28 | `dropdown/grok-error` | `Grok: Error: timeout waiting` |
| 29 | `dropdown/codex-line` | `Codex: 58%(Monthly) 6,519/11,250, Reset Aug 1, 08:00, left 26d` |
| 30 | `dropdown/codex-error` | `Codex: Error: fork/exec ...` full message |
| 31 | `dropdown/codex-timeout-error` | `Codex: Error: timeout waiting for status output` |
| 32 | `client/swift-grok-codex-server-port` | AppState refresh uses ServerClient for grok/codex |

## Parameter Coverage

| Leaf | Op | Mode | NowRFC3339 | Key inputs |
|------|-----|------|------------|------------|
| ready | grok-label | — | — | status=ready, weekly=6% |
| loading | grok-label | — | — | status=loading |
| error | grok-label | — | — | status=error → `Grok err` |
| error-truncate | grok-label | — | — | status=error, long msg → `Grok err` |
| codex-error | menu-label | codex | — | codex status=error → `Codex err` |
| grok-fixed | menu-label | grok | — | grok ready 6% |
| codex-fixed | menu-label | codex | — | codex ready 58% |
| rotating-grok-slot | menu-label | rotating | — | grok ready 6% |
| rotating-codex-slot | menu-label | rotating | — | codex ready 58% |
| three-days | time-left | — | 2026-07-06T16:55:00-07:00 | reset=July 9, 16:55 PT |
| three-days-four-hours | time-left | — | 2026-07-06T16:55:00-07:00 | reset=July 9, 20:55 PT |
| two-days-five-hours | time-left | — | 2026-07-06T16:55:00-07:00 | reset=July 8, 21:55 PT |
| three-hours-five-minutes | time-left | — | 2026-07-06T16:55:00-07:00 | reset=July 6, 20:00 PT |
| four-hours-five-minutes | time-left | — | 2026-07-06T16:55:00-07:00 | reset=July 6, 21:00 PT |
| five-minutes | time-left | — | 2026-07-06T16:55:00-07:00 | reset=July 6, 17:00 PT |
| two-minutes | time-left | — | 2026-07-06T16:55:00-07:00 | reset=July 6, 16:57 PT |
| codex-three-days | time-left | — | 2026-07-06T08:00:00-07:00 | reset=08:00 on 9 Jul |
| one-minute-floor | time-left | — | 2026-07-06T16:55:00-07:00 | reset=July 6, 16:56:30 PT |
| zero-minutes | time-left | — | 2026-07-06T16:55:00-07:00 | reset=July 6, 16:55 PT |
| unparseable-fallback | time-left | — | 2026-07-06T16:55:00-07:00 | reset=soon |
| grok-same-tz | reset-display | — | 2026-07-06T16:55:00-07:00 | reset=July 9, 17:55 PT |
| grok-cross-tz | reset-display | — | 2026-07-06T16:55:00-04:00 | reset=July 9, 17:55 PT |
| grok-date-rollover | reset-display | — | 2026-07-06T16:55:00+09:00 | reset=July 9, 17:55 PT |
| codex-reformat | reset-display | — | 2026-07-06T08:00:00-07:00 | reset=08:00 on 1 Aug |
| unparseable-fallback (reset) | reset-display | — | 2026-07-06T16:55:00-07:00 | reset=soon |
| grok-line | grok-dropdown | — | 2026-07-06T16:55:00-07:00 | weekly=6%, reset=July 9, 16:55 PT |
| grok-unparseable-reset | grok-dropdown | — | 2026-07-06T16:55:00-07:00 | weekly=6%, reset=soon |
| grok-error | grok-dropdown | — | — | status=error, error=timeout waiting |
| codex-line | codex-dropdown | — | 2026-07-06T08:00:00-07:00 | monthly=58%, credits 6,519/11,250 |
| codex-error | codex-dropdown | — | — | status=error, fork/exec message |
| codex-timeout-error | codex-dropdown | — | — | status=error, timeout waiting for status output |
| swift-grok-codex-server-port | client | — | — | Swift sources use ServerClient :23712 |

## How to Run

```sh
doctest vet ./tests/macos-menubar
doctest test ./tests/macos-menubar/...
```

```go
import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/xhd2015/ai-critic/macosapp/menubar"
	"github.com/xhd2015/ai-critic/server/config"
)

type Request struct {
	Op string

	// Fixed clock for relative-time, reset-display, and dropdown formatters.
	NowRFC3339 string

	// FormatGrokLabel (Op=grok-label or empty)
	Status      string
	WeeklyLimit string
	ErrorMsg    string

	// FormatMenuBarLabel (Op=menu-label)
	DisplayMode    string
	RotatingIndex  int
	GrokStatus     string
	GrokWeekly     string
	GrokError      string
	CodexStatus    string
	CodexMonthly   string
	CodexError     string

	// FormatTimeLeft / FormatResetDisplay (Op=time-left or reset-display)
	Reset string

	// FormatGrokDropdownLine (Op=grok-dropdown)
	GrokReset string

	// FormatCodexDropdownLine (Op=codex-dropdown)
	CodexCreditsUsed  string
	CodexCreditsTotal string
	CodexReset        string
}

type Response struct {
	Label        string
	DropdownLine string
	TimeLeft     string
	ResetDisplay string
	MaxLabelLen  int

	// client contract
	GrokViaServerClient  bool
	CodexViaServerClient bool
	GrokViaDaemonClient  bool
	CodexViaDaemonClient bool
	SwiftSourcesChecked  []string
}

func parseNow(req *Request) (time.Time, error) {
	if req.NowRFC3339 == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, req.NowRFC3339)
}

func Run(t *testing.T, req *Request) (*Response, error) {
	resp := &Response{
		MaxLabelLen: menubar.TestExported_MaxLabelLen(),
	}
	op := req.Op
	if op == "" {
		op = "grok-label"
	}

	switch op {
	case "client":
		return runClientContract(t, resp)
	case "grok-label":
		resp.Label = menubar.FormatGrokLabel(req.Status, req.WeeklyLimit, req.ErrorMsg)
	case "menu-label":
		resp.Label = menubar.FormatMenuBarLabel(
			req.DisplayMode,
			req.RotatingIndex,
			req.GrokStatus, req.GrokWeekly, req.GrokError,
			req.CodexStatus, req.CodexMonthly, req.CodexError,
		)
	case "time-left":
		now, err := parseNow(req)
		if err != nil {
			return nil, fmt.Errorf("parse NowRFC3339: %w", err)
		}
		resp.TimeLeft = menubar.FormatTimeLeft(req.Reset, now)
	case "reset-display":
		now, err := parseNow(req)
		if err != nil {
			return nil, fmt.Errorf("parse NowRFC3339: %w", err)
		}
		resp.ResetDisplay = menubar.FormatResetDisplay(req.Reset, now)
	case "grok-dropdown":
		now, err := parseNow(req)
		if err != nil {
			return nil, fmt.Errorf("parse NowRFC3339: %w", err)
		}
		resp.DropdownLine = menubar.FormatGrokDropdownLine(
			req.GrokStatus, req.WeeklyLimit, req.GrokReset, req.GrokError, now,
		)
	case "codex-dropdown":
		now, err := parseNow(req)
		if err != nil {
			return nil, fmt.Errorf("parse NowRFC3339: %w", err)
		}
		resp.DropdownLine = menubar.FormatCodexDropdownLine(
			req.CodexStatus,
			req.CodexMonthly,
			req.CodexCreditsUsed,
			req.CodexCreditsTotal,
			req.CodexReset,
			req.CodexError,
			now,
		)
	default:
		return nil, fmt.Errorf("unknown op %q", req.Op)
	}
	return resp, nil
}

func runClientContract(t *testing.T, resp *Response) (*Response, error) {
	moduleRoot, err := findModuleRoot()
	if err != nil {
		return nil, err
	}
	appPath := filepath.Join(moduleRoot, "macos-ai-critic", "ai-critic-macos", "AICriticApp.swift")
	serverPath := filepath.Join(moduleRoot, "macos-ai-critic", "ai-critic-macos", "ServerClient.swift")
	daemonPath := filepath.Join(moduleRoot, "macos-ai-critic", "ai-critic-macos", "DaemonClient.swift")
	resp.SwiftSourcesChecked = []string{appPath, serverPath, daemonPath}

	appSrc, err := os.ReadFile(appPath)
	if err != nil {
		return nil, fmt.Errorf("read AICriticApp.swift: %w", err)
	}
	serverSrc, err := os.ReadFile(serverPath)
	if err != nil {
		return nil, fmt.Errorf("read ServerClient.swift: %w", err)
	}
	daemonSrc, err := os.ReadFile(daemonPath)
	if err != nil {
		return nil, fmt.Errorf("read DaemonClient.swift: %w", err)
	}
	app := string(appSrc)
	server := string(serverSrc)
	daemon := string(daemonSrc)

	port := strconv.Itoa(config.DefaultServerPort)
	resp.GrokViaServerClient = strings.Contains(server, "/api/grok/usage") && strings.Contains(server, port)
	resp.CodexViaServerClient = strings.Contains(server, "/api/codex/usage") && strings.Contains(server, port)
	resp.GrokViaDaemonClient = strings.Contains(app, "DaemonClient.shared.grokUsage") ||
		(strings.Contains(daemon, "/api/grok/usage") && strings.Contains(app, "grokUsage"))
	resp.CodexViaDaemonClient = strings.Contains(app, "DaemonClient.shared.codexUsage") ||
		(strings.Contains(daemon, "/api/codex/usage") && strings.Contains(app, "codexUsage"))
	return resp, nil
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
```
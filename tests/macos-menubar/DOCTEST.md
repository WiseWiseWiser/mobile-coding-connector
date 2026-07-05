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
- **FormatGrokDropdownLine** — single-line grok dropdown text with weekly limit,
  absolute reset, and relative countdown suffix.
- **FormatCodexDropdownLine** — single-line codex dropdown text with monthly usage,
  credits fraction, absolute reset, and relative countdown suffix.
- **FormatTimeLeft** — parses provider reset strings (grok `July 9, 16:55 PT` or codex
  `08:00 on 1 Aug`), infers year from `now`, and returns compact relative text
  (`left 3d`, `left 3h`, `left 2min`, or empty when unparseable).
- **FormatResetSuffix** — comma-prefixed relative suffix for dropdown parentheses
  (`, left 3d` or empty when unparseable).
- **Test harness** — invokes formatters with leaf-provided inputs and fixed `now`;
  no UI or network.

**Behaviors**

- `FormatGrokLabel`: `ready` + `weekly_limit` → `Grok {limit}`; `loading` → `Grok ...`;
  `error` → fixed short `Grok err` (not full daemon message).
- `formatCodexLabel`: `error` → fixed short `Codex err`.
- `FormatMenuBarLabel`: fixed `grok`/`codex` modes; rotating index 0 → grok, 1 → codex.
- `FormatTimeLeft`: floor to largest unit (≥24h→days, ≥1h→hours, <1h→minutes);
  minutes floor to at least 1 when 0 < duration < 1h; duration ≤ 0 → `left 0min`;
  unparseable/empty reset → empty string.
- `FormatResetSuffix`: `, ` + `FormatTimeLeft` when parseable; empty otherwise.
- `FormatGrokDropdownLine`: ready → `Grok: Weekly Limit: … (Reset <abs>, left <rel>)`;
  `error` → `Grok: Error: {msg}` (no reset suffix on error).
- `FormatCodexDropdownLine`: ready → `Codex: Monthly Usage: … (Reset <abs>, left <rel>)`;
  `error` → `Codex: Error: {msg}` (no reset suffix on error).
- Compact menu-bar pill labels (`label/*`) remain unchanged — no reset or relative time.

## Version

0.0.3

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
 +-- relative-time/                   (GROUP)  FormatTimeLeft relative countdown
 |    +-- three-days/                 (LEAF)   grok reset, ≥24h → `left 3d`
 |    +-- three-hours/                (LEAF)   grok reset, ≥1h <24h → `left 3h`
 |    +-- two-minutes/                (LEAF)   grok reset, <1h → `left 2min`
 |    +-- codex-three-days/           (LEAF)   codex reset format → `left 3d`
 |    +-- one-minute-floor/           (LEAF)   90s remaining → `left 1min`
 |    +-- zero-minutes/               (LEAF)   duration ≤ 0 → `left 0min`
 |    +-- unparseable-fallback/       (LEAF)   unparseable reset → empty
 |
 +-- dropdown/                        (GROUP)  dropdown single lines
      +-- reset-suffix/               (GROUP)  FormatResetSuffix comma prefix
      |    +-- grok-three-days/       (LEAF)   parseable grok → `, left 3d`
      |    +-- codex-three-days/      (LEAF)   parseable codex → `, left 3d`
      |    +-- unparseable/           (LEAF)   unparseable → empty
      +-- grok-line/                  (LEAF)   FormatGrokDropdownLine ready + relative
      +-- grok-error/                 (LEAF)   FormatGrokDropdownLine error msg
      +-- codex-line/                 (LEAF)   FormatCodexDropdownLine ready + relative
      +-- codex-error/                (LEAF)   FormatCodexDropdownLine error msg
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
| 11 | `relative-time/three-hours` | `FormatTimeLeft` grok reset → `left 3h` |
| 12 | `relative-time/two-minutes` | `FormatTimeLeft` grok reset → `left 2min` |
| 13 | `relative-time/codex-three-days` | `FormatTimeLeft` codex reset → `left 3d` |
| 14 | `relative-time/one-minute-floor` | 90s remaining → `left 1min` |
| 15 | `relative-time/zero-minutes` | reset at or before `now` → `left 0min` |
| 16 | `relative-time/unparseable-fallback` | `soon` → empty |
| 17 | `dropdown/reset-suffix/grok-three-days` | `FormatResetSuffix` grok → `, left 3d` |
| 18 | `dropdown/reset-suffix/codex-three-days` | `FormatResetSuffix` codex → `, left 3d` |
| 19 | `dropdown/reset-suffix/unparseable` | `FormatResetSuffix` unparseable → empty |
| 20 | `dropdown/grok-line` | `Grok: Weekly Limit: 6% (Reset July 9, 16:55 PT, left 3d)` |
| 21 | `dropdown/grok-error` | `Grok: Error: timeout waiting` |
| 22 | `dropdown/codex-line` | `Codex: Monthly Usage: 58% — 6,519/11,250 (Reset 08:00 on 1 Aug, left 26d)` |
| 23 | `dropdown/codex-error` | `Codex: Error: fork/exec ...` full message |

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
| three-hours | time-left | — | 2026-07-06T16:55:00-07:00 | reset=July 6, 20:00 PT |
| two-minutes | time-left | — | 2026-07-06T16:55:00-07:00 | reset=July 6, 16:57 PT |
| codex-three-days | time-left | — | 2026-07-06T08:00:00-07:00 | reset=08:00 on 9 Jul |
| one-minute-floor | time-left | — | 2026-07-06T16:55:00-07:00 | reset=July 6, 16:56:30 PT |
| zero-minutes | time-left | — | 2026-07-06T16:55:00-07:00 | reset=July 6, 16:55 PT |
| unparseable-fallback | time-left | — | 2026-07-06T16:55:00-07:00 | reset=soon |
| grok-three-days | reset-suffix | — | 2026-07-06T16:55:00-07:00 | reset=July 9, 16:55 PT |
| codex-three-days | reset-suffix | — | 2026-07-06T08:00:00-07:00 | reset=08:00 on 9 Jul |
| unparseable | reset-suffix | — | 2026-07-06T16:55:00-07:00 | reset=soon |
| grok-line | grok-dropdown | — | 2026-07-06T16:55:00-07:00 | weekly=6%, reset=July 9, 16:55 PT |
| grok-error | grok-dropdown | — | — | status=error, error=timeout waiting |
| codex-line | codex-dropdown | — | 2026-07-06T08:00:00-07:00 | monthly=58%, credits 6,519/11,250 |
| codex-error | codex-dropdown | — | — | status=error, fork/exec message |

## How to Run

```sh
doctest vet ./tests/macos-menubar
doctest test ./tests/macos-menubar/...
```

```go
import (
	"fmt"
	"testing"
	"time"

	"github.com/xhd2015/ai-critic/macosapp/menubar"
)

type Request struct {
	Op string

	// Fixed clock for relative-time and dropdown formatters (RFC3339, America/Los_Angeles in tests).
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

	// FormatTimeLeft / FormatResetSuffix (Op=time-left or reset-suffix)
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
	ResetSuffix  string
	MaxLabelLen  int
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
	case "reset-suffix":
		now, err := parseNow(req)
		if err != nil {
			return nil, fmt.Errorf("parse NowRFC3339: %w", err)
		}
		resp.ResetSuffix = menubar.FormatResetSuffix(req.Reset, now)
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
```
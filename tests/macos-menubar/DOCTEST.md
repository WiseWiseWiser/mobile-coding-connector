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
- **FormatGrokDropdownLine** — single-line grok dropdown text with weekly limit and reset.
- **FormatCodexDropdownLine** — single-line codex dropdown text with monthly usage,
  credits fraction, and reset.
- **Test harness** — invokes formatters with leaf-provided inputs; no UI or network.

**Behaviors**

- `FormatGrokLabel`: `ready` + `weekly_limit` → `Grok {limit}`; `loading` → `Grok ...`;
  `error` → fixed short `Grok err` (not full daemon message).
- `formatCodexLabel`: `error` → fixed short `Codex err`.
- `FormatMenuBarLabel`: fixed `grok`/`codex` modes; rotating index 0 → grok, 1 → codex.
- `FormatGrokDropdownLine`: ready → `Grok: Weekly Limit: …`; `error` → `Grok: Error: {msg}`.
- `FormatCodexDropdownLine`: ready → `Codex: Monthly Usage: …`; `error` → `Codex: Error: {msg}`.

## Version

0.0.2

## Decision Tree

```
[menubar formatting]
 |
 +-- label/                           (GROUP)  menu-bar title labels
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
 +-- dropdown/                        (GROUP)  dropdown single lines
      +-- grok-line/                  (LEAF)   FormatGrokDropdownLine ready
      +-- grok-error/                 (LEAF)   FormatGrokDropdownLine error msg
      +-- codex-line/                 (LEAF)   FormatCodexDropdownLine ready
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
| 10 | `dropdown/grok-line` | `Grok: Weekly Limit: 6% (Reset July 9, 16:55 PT)` |
| 11 | `dropdown/grok-error` | `Grok: Error: timeout waiting` |
| 12 | `dropdown/codex-line` | `Codex: Monthly Usage: 58% — 6,519/11,250 (Reset 08:00 on 1 Aug)` |
| 13 | `dropdown/codex-error` | `Codex: Error: fork/exec ...` full message |

## Parameter Coverage

| Leaf | Op | Mode | RotatingIndex | Key inputs |
|------|-----|------|---------------|------------|
| ready | grok-label | — | — | status=ready, weekly=6% |
| loading | grok-label | — | — | status=loading |
| error | grok-label | — | — | status=error → `Grok err` |
| error-truncate | grok-label | — | — | status=error, long msg → `Grok err` |
| codex-error | menu-label | codex | — | codex status=error → `Codex err` |
| grok-fixed | menu-label | grok | — | grok ready 6% |
| codex-fixed | menu-label | codex | — | codex ready 58% |
| rotating-grok-slot | menu-label | rotating | 0 | grok ready 6% |
| rotating-codex-slot | menu-label | rotating | 1 | codex ready 58% |
| grok-line | grok-dropdown | — | — | weekly=6%, reset=July 9, 16:55 PT |
| grok-error | grok-dropdown | — | — | status=error, error=timeout waiting |
| codex-line | codex-dropdown | — | — | monthly=58%, credits 6,519/11,250 |
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

	"github.com/xhd2015/ai-critic/macosapp/menubar"
)

type Request struct {
	Op string

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

	// FormatGrokDropdownLine (Op=grok-dropdown)
	GrokReset string

	// FormatCodexDropdownLine (Op=codex-dropdown)
	CodexCreditsUsed  string
	CodexCreditsTotal string
	CodexReset        string
}

type Response struct {
	Label       string
	DropdownLine string
	MaxLabelLen int
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
	case "grok-dropdown":
		resp.DropdownLine = menubar.FormatGrokDropdownLine(
			req.GrokStatus, req.WeeklyLimit, req.GrokReset, req.GrokError,
		)
	case "codex-dropdown":
		resp.DropdownLine = menubar.FormatCodexDropdownLine(
			req.CodexStatus,
			req.CodexMonthly,
			req.CodexCreditsUsed,
			req.CodexCreditsTotal,
			req.CodexReset,
			req.CodexError,
		)
	default:
		return nil, fmt.Errorf("unknown op %q", req.Op)
	}
	return resp, nil
}
```
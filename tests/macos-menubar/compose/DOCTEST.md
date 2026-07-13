# Menu Bar Compose-Only Dropdown Doctests

Pure-function tests for structured-field dropdown composition in
`macosapp/menubar` — the Go contract mirrored by Swift menubar UI when
rendering ready usage rows from API `reset_display` + `time_left` (no re-parse
of raw `next_reset`).

Nested under `tests/macos-menubar` so existing parse/format leaves stay
compile-green while this root stays RED until `ComposeGrokDropdownLine` /
`ComposeCodexDropdownLine` exist.

# DSN (Domain Specific Notion)

**Participants**

- **ComposeGrokDropdownLine (`macosapp/menubar`)** — concatenates ready grok
  structured fields into one dropdown line:
  `"Grok: " + weekly + "(Weekly), Reset " + reset_display + optional ", " + time_left`.
  Does **not** parse `next_reset`, does not call `FormatResetDisplay` /
  `FormatTimeLeft`, does not perform `Date`/`time` math.
- **ComposeCodexDropdownLine (`macosapp/menubar`)** — same compose-only contract
  for codex monthly/credits:
  `"Codex: " + monthly + "(Monthly) " + used + "/" + total + ", Reset " + reset_display`
  + optional `", " + time_left`.
- **Test harness** — invokes compose helpers with leaf-provided structured
  strings; no network, no clock.

**Behaviors**

- Ready + non-empty `time_left` → full line including `, left …`.
- Ready + empty `time_left` → line ends at `Reset {reset_display}` (no `, left`).
- Error/loading paths are out of scope here (covered by existing `dropdown/*`
  raw-parse helpers used as backend producers of `reset_display`/`time_left`).

## Version

0.0.1

## Decision Tree

```
[compose-only dropdown]
 |
 +-- grok-ready/                 (LEAF)   weekly + reset_display + time_left
 +-- grok-empty-time-left/       (LEAF)   time_left empty → no ", left …"
 +-- codex-ready/                (LEAF)   monthly/credits + reset_display + time_left
 +-- codex-empty-time-left/      (LEAF)   time_left empty → no ", left …"
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `grok-ready` | `Grok: 61%(Weekly), Reset July 17, 08:55, left 4d` |
| 2 | `grok-empty-time-left` | `Grok: 61%(Weekly), Reset July 17, 08:55` (no left) |
| 3 | `codex-ready` | `Codex: 58%(Monthly) 6,519/11,250, Reset Aug 1, 08:00, left 26d` |
| 4 | `codex-empty-time-left` | `Codex: 58%(Monthly) 6,519/11,250, Reset Aug 1, 08:00` |

## Parameter Coverage

| Leaf | Op | weekly/monthly | reset_display | time_left |
|------|-----|----------------|---------------|-----------|
| grok-ready | grok-compose | 61% | July 17, 08:55 | left 4d |
| grok-empty-time-left | grok-compose | 61% | July 17, 08:55 | (empty) |
| codex-ready | codex-compose | 58% + credits | Aug 1, 08:00 | left 26d |
| codex-empty-time-left | codex-compose | 58% + credits | Aug 1, 08:00 | (empty) |

## How to Run

```sh
doctest vet ./tests/macos-menubar/compose
doctest test ./tests/macos-menubar/compose/...
```

```go
import (
	"fmt"
	"testing"

	"github.com/xhd2015/ai-critic/macosapp/menubar"
)

type Request struct {
	Op string

	// ComposeGrokDropdownLine
	Status       string
	WeeklyLimit  string
	ResetDisplay string
	TimeLeft     string
	ErrorMsg     string

	// ComposeCodexDropdownLine
	MonthlyUsage string
	CreditsUsed  string
	CreditsTotal string
}

type Response struct {
	DropdownLine string
}

func Run(t *testing.T, req *Request) (*Response, error) {
	resp := &Response{}
	switch req.Op {
	case "grok-compose":
		// Production API (implementer) — compose only; no parse of next_reset.
		resp.DropdownLine = menubar.ComposeGrokDropdownLine(
			req.Status,
			req.WeeklyLimit,
			req.ResetDisplay,
			req.TimeLeft,
			req.ErrorMsg,
		)
	case "codex-compose":
		resp.DropdownLine = menubar.ComposeCodexDropdownLine(
			req.Status,
			req.MonthlyUsage,
			req.CreditsUsed,
			req.CreditsTotal,
			req.ResetDisplay,
			req.TimeLeft,
			req.ErrorMsg,
		)
	default:
		return nil, fmt.Errorf("unknown op %q", req.Op)
	}
	return resp, nil
}
```

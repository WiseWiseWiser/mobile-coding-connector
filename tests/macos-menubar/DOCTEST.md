# macOS Menu Bar Grok Label Formatting Doctests

Pure-function tests for `macosapp/menubar.FormatGrokLabel` — the Go spec mirrored
by the Swift `ai-critic-macos` menu-bar client when rendering grok usage state.

# DSN (Domain Specific Notion)

**Participants**

- **FormatGrokLabel (`macosapp/menubar`)** — maps daemon `GrokUsageResponse` fields
  (`status`, `weekly_limit`, `error`) to a compact menu-bar string prefixed with
  `Grok `.
- **Test harness** — invokes `FormatGrokLabel` with leaf-provided inputs; no UI
  or network.

**Behaviors**

- `ready` + `weekly_limit` → `Grok {limit}` (e.g. `Grok 6%`).
- `loading` → `Grok ...`.
- `error` + message → `Grok {message}` (full message when short).
- Long error messages truncate to menu-bar-safe length (default max 40 runes).

## Version

0.0.2

## Decision Tree

```
[FormatGrokLabel]
 |
 +-- label/                           (GROUP)  status-driven label text
      +-- ready/                      (LEAF)   ready + weekly_limit
      +-- loading/                    (LEAF)   loading placeholder
      +-- error/                      (LEAF)   error message shown
      +-- error-truncate/             (LEAF)   long error truncated
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `label/ready` | `ready` + `6%` → `Grok 6%` |
| 2 | `label/loading` | `loading` → `Grok ...` |
| 3 | `label/error` | `error` + `timeout waiting` → full message |
| 4 | `label/error-truncate` | long error truncated for menu bar |

## Parameter Coverage

| Leaf | Status | WeeklyLimit | ErrorMsg |
|------|--------|-------------|----------|
| ready | ready | 6% | (empty) |
| loading | loading | (empty) | (empty) |
| error | error | (empty) | timeout waiting |
| error-truncate | error | (empty) | 80+ char string |

## How to Run

```sh
doctest vet ./tests/macos-menubar
doctest test ./tests/macos-menubar/...
```

```go
import (
	"testing"

	"github.com/xhd2015/ai-critic/macosapp/menubar"
)

type Request struct {
	Status      string
	WeeklyLimit string
	ErrorMsg    string
}

type Response struct {
	Label       string
	MaxLabelLen int
}

func Run(t *testing.T, req *Request) (*Response, error) {
	label := menubar.FormatGrokLabel(req.Status, req.WeeklyLimit, req.ErrorMsg)
	return &Response{
		Label:       label,
		MaxLabelLen: menubar.TestExported_MaxLabelLen(),
	}, nil
}
```
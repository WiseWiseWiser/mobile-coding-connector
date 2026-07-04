# macOS Open in Browser Label Formatting Doctests

Pure-function tests for `macosapp/browser.FormatOpenInBrowserLabel` — the Go spec
mirrored by the Swift `OpenInBrowserLabelFormatter` when rendering the menu-bar
Open in Browser action.

# DSN (Domain Specific Notion)

**Participants**

- **FormatOpenInBrowserLabel (`macosapp/browser`)** — maps stored browser
  preference (`default`, `chrome`, `firefox`, `opera`) to the menu label.
- **Test harness** — invokes `FormatOpenInBrowserLabel` with leaf-provided inputs;
  no UI or network.

**Behaviors**

- `default` or empty → `Open in Browser`.
- `chrome` → `Open in Browser(Chrome)`.
- `firefox` → `Open in Browser(Firefox)`.
- `opera` → `Open in Browser(Opera)`.
- Unknown values fall back to `Open in Browser`.

## Version

0.0.1

## Decision Tree

```
[FormatOpenInBrowserLabel]
 |
 +-- label/                           (GROUP)  browser-driven menu label
      +-- default/                    (LEAF)   default preference
      +-- chrome/                     (LEAF)   Chrome suffix
      +-- firefox/                    (LEAF)   Firefox suffix
      +-- opera/                      (LEAF)   Opera suffix
      +-- unknown/                    (LEAF)   unknown value fallback
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `label/default` | `default` → `Open in Browser` |
| 2 | `label/chrome` | `chrome` → `Open in Browser(Chrome)` |
| 3 | `label/firefox` | `firefox` → `Open in Browser(Firefox)` |
| 4 | `label/opera` | `opera` → `Open in Browser(Opera)` |
| 5 | `label/unknown` | `safari` → `Open in Browser` |

## How to Run

```sh
doctest vet ./tests/macos-browser
doctest test ./tests/macos-browser/...
```

```go
import (
	"testing"

	"github.com/xhd2015/ai-critic/macosapp/browser"
)

type Request struct {
	Browser string
}

type Response struct {
	Label string
}

func Run(t *testing.T, req *Request) (*Response, error) {
	return &Response{
		Label: browser.FormatOpenInBrowserLabel(req.Browser),
	}, nil
}
```
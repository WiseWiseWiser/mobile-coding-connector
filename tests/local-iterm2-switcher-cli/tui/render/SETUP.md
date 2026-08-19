# Scenario

**Feature**: library View paints locked split chrome from last-good

```
NewUIState(seed inventory, status=cached) -> View
  -> Terminals title, sidebar All / Desktop 1, session grok review, status cached
  -> split box (┌ or │) — not chip-row-only
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "tui"
	req.TUIStatus = "cached"
	return nil
}
```

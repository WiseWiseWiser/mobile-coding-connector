# Scenario

**Feature**: warm split first paint from last-good (status cached)

```
seed inventory sess-a / Desktop 1
NewUIState(..., "cached") -> View
  -> Terminals, All, Desktop 1, grok review, cached, box drawing
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "tui"
	req.TUIStatus = "cached"
	// single-session fixture via default seedInventory
	return nil
}
```

# Scenario

**Feature**: default list focus puts › on the first session row (right pane)

```
NewUIState(seed) -> View
  -> › on first session (grok review), not only on All
  -> FocusPane defaults to list
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

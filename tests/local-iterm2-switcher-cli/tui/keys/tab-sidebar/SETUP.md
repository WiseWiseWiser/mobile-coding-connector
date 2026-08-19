# Scenario

**Feature**: Tab moves focus to the sidebar pane

```
ApplyKey("tab") -> FocusPane=sidebar; › on a left row; list still visible
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "tui"
	req.ApplyKeys = []string{"tab"}
	return nil
}
```

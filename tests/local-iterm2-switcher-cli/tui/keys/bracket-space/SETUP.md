# Scenario

**Feature**: ] advances Desktop filter without leaving the list

```
seed sess-a (Desktop 1) + sess-b (Desktop 2)
ApplyKey("]") -> sidebar Desktop advances; list refilters
  -> right pane must not show both spaces' sessions as unfiltered All
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "tui"
	req.SeedTwoSessions = true
	req.ApplyKeys = []string{"]"}
	return nil
}
```

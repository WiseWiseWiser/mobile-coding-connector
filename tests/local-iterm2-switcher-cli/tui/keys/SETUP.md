# Scenario

**Feature**: library ApplyKey two-pane navigation

```
ApplyKey(j) -> › moves down list
ApplyKey(tab) -> focus sidebar
ApplyKey(]) -> next Desktop filter
ApplyKey(enter) -> action focus + session id
ApplyKey(q) -> action quit
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

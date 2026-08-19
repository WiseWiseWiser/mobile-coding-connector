# Scenario

**Feature**: j / down moves › to the next session on the list

```
seed two sessions on Desktop 1 (grok review, second pane)
ApplyKey("j") -> View › on second session (not grok review)
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "tui"
	req.SeedTwoSessions = true
	req.ApplyKeys = []string{"j"}
	return nil
}
```

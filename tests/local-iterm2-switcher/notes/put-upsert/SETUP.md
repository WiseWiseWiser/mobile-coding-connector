# Scenario

**Feature**: PUT note upserts and is readable from the store

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "notes"
	req.ITermRunning = true
	req.SessionID = "sess-a"
	req.Note = "fix auth on staging"
	return nil
}
```

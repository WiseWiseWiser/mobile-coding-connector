# Scenario

**Feature**: PUT bookmarked true with no note persists a starred record

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "notes"
	req.ITermRunning = true
	req.SessionID = "sess-a"
	req.OmitNote = true
	req.SetBookmarked = true
	req.Bookmarked = true
	return nil
}
```

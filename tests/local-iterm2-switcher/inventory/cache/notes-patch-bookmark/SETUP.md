# Scenario

**Feature**: PUT bookmark patches the in-memory inventory cache

```
GET inventory (fills cache) -> PUT bookmarked -> GET cache still shows star
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "inventory"
	req.ITermRunning = true
	req.WindowSpace = 1
	req.SessionID = "sess-a"
	req.DoNotesAfterGET = true
	req.AfterNoteBookmarked = true
	return nil
}
```

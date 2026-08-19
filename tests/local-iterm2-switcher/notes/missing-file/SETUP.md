# Scenario

**Feature**: missing notes file is an empty document

```go
import (
	"path/filepath"
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "notes_store"
	req.NotesPath = filepath.Join(t.TempDir(), "does-not-exist", "iterm-bookmarks.json")
	req.SessionID = "sess-a"
	return nil
}
```

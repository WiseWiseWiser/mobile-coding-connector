## Expected

1. Live session has no note.
2. One orphan in saved_notes with the dead note.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.NoteOnFirst != "" {
		t.Fatalf("live note=%q want empty", resp.NoteOnFirst)
	}
	if !resp.HasOrphan || resp.SavedCount != 1 {
		t.Fatalf("SavedCount=%d HasOrphan=%v", resp.SavedCount, resp.HasOrphan)
	}
	if resp.OrphanNote != "machine backup plan" {
		t.Fatalf("orphan note=%q", resp.OrphanNote)
	}
}
```

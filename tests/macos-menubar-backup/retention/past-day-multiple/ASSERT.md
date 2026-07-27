## Expected

1. Keep: `y-late.tar.xz` only.
2. Delete: `y-early.tar.xz`, `y-mid.tar.xz`.

## Errors

- Keeping all three yesterday files or the oldest of the day.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	wantKeep := []string{"y-late.tar.xz"}
	wantDel := []string{"y-early.tar.xz", "y-mid.tar.xz"}
	if !pathSetEqual(resp.KeepPaths, wantKeep) {
		t.Fatalf("KeepPaths = %v, want %v", resp.KeepPaths, wantKeep)
	}
	if !pathSetEqual(resp.DeletePaths, wantDel) {
		t.Fatalf("DeletePaths = %v, want %v", resp.DeletePaths, wantDel)
	}
}
```

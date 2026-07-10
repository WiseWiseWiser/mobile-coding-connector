## Expected

1. Keep: `t1`, `t2`, `t3` (all today <10), `y2` (newest yesterday), `d3` (only on that day).
2. Delete: `y1` (extra yesterday), `old` (8+ days).

## Errors

- Dropping valid today files, keeping both yesterday files, or retaining `old`.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	wantKeep := []string{"t1.tar.xz", "t2.tar.xz", "t3.tar.xz", "y2.tar.xz", "d3.tar.xz"}
	wantDel := []string{"y1.tar.xz", "old.tar.xz"}
	if !pathSetEqual(resp.KeepPaths, wantKeep) {
		t.Fatalf("KeepPaths = %v, want set %v", resp.KeepPaths, wantKeep)
	}
	if !pathSetEqual(resp.DeletePaths, wantDel) {
		t.Fatalf("DeletePaths = %v, want set %v", resp.DeletePaths, wantDel)
	}
}
```

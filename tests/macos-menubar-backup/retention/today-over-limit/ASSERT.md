## Expected

1. Keep paths: `today-02` … `today-11` (10 newest by modTime).
2. Delete paths: `today-00`, `today-01` (2 oldest today).

## Errors

- Keeping all 12, or deleting newest instead of oldest.

```go
import "fmt"
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	var wantKeep, wantDel []string
	for h := 2; h < 12; h++ {
		wantKeep = append(wantKeep, fmt.Sprintf("today-%02d.tar.xz", h))
	}
	wantDel = []string{"today-00.tar.xz", "today-01.tar.xz"}
	if !pathSetEqual(resp.KeepPaths, wantKeep) {
		t.Fatalf("KeepPaths = %v, want set %v", resp.KeepPaths, wantKeep)
	}
	if !pathSetEqual(resp.DeletePaths, wantDel) {
		t.Fatalf("DeletePaths = %v, want set %v", resp.DeletePaths, wantDel)
	}
}
```

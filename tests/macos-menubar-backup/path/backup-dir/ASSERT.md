## Expected

1. `BackupDir` is exactly `/Users/me/.backup/ai-critic/foo.example.com`.

## Errors

- Wrong root (e.g. `~/Library/...`), missing `ai-critic`, or using URL instead of host.

```go
import "path/filepath"
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join("/Users/me", ".backup", "ai-critic", "foo.example.com")
	if resp.BackupDir != want {
		t.Fatalf("BackupDir = %q, want %q", resp.BackupDir, want)
	}
}
```

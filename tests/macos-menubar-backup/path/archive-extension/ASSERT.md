## Expected

1. `Filename` is exactly `machine-backup-20260710-120000.tar.xz`.

## Errors

- Wrong extension (`.tgz`, `.tar.gz`), local timezone in stamp, or missing `machine-backup-` prefix.

```go
import "strings"
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "machine-backup-20260710-120000.tar.xz"
	if resp.Filename != want {
		t.Fatalf("Filename = %q, want %q", resp.Filename, want)
	}
	if !strings.HasSuffix(resp.Filename, ".tar.xz") {
		t.Fatalf("extension must be .tar.xz, got %q", resp.Filename)
	}
}
```

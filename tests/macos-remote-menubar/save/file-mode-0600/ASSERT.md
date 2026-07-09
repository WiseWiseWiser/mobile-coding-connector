## Expected

1. `SavedOK` is true.
2. `FileMode` permission bits equal `0600`.

## Errors

- Writing with default `0644` or other overly permissive modes.

```go
import (
	"os"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.SavedOK {
		t.Fatal("expected SavedOK")
	}
	want := os.FileMode(0o600)
	if resp.FileMode != want {
		t.Fatalf("file mode = %04o, want %04o", resp.FileMode, want)
	}
}
```

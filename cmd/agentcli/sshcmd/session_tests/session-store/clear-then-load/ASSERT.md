## Expected

1. `SaveErr` and `ClearErr` are empty.
2. `LoadErr` is empty.
3. `Loaded` is nil after Clear.

## Side Effects

- Session file for the profile is removed (or absent).

## Errors

None expected.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, d *session.Doctest, req *Request, resp *Response, err error) {
	t.Helper()
	_ = d
	_ = req
	if err != nil {
		t.Fatalf("harness Run returned unexpected error: %v", err)
	}
	if resp.SaveErr != "" {
		t.Fatalf("Save error: %s", resp.SaveErr)
	}
	if resp.ClearErr != "" {
		t.Fatalf("Clear error: %s", resp.ClearErr)
	}
	if resp.LoadErr != "" {
		t.Fatalf("Load after Clear must not error; got %q", resp.LoadErr)
	}
	if resp.Loaded != nil {
		t.Fatalf("Loaded must be nil after Clear; got %+v", resp.Loaded)
	}
}
```

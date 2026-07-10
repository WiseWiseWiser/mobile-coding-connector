## Expected

1. `Loading` is `false`.
2. `Projects` is exactly `["dot-pkgs", "other"]` (replaced, not appended to old).
3. `Error` is empty (cleared on success).

## Errors

- Keeping stale `old` without replacement.
- Leaving prior error set after success.

```go
import (
	"reflect"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.Loading {
		t.Fatal("Loading = true, want false after success")
	}
	want := []string{"dot-pkgs", "other"}
	if !reflect.DeepEqual(resp.Projects, want) {
		t.Fatalf("Projects = %#v, want %#v", resp.Projects, want)
	}
	if resp.Error != "" {
		t.Fatalf("Error = %q, want empty after success", resp.Error)
	}
}
```

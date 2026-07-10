## Expected

1. `Loading` is `false`.
2. `Projects` remains `["dot-pkgs"]`.
3. `Error` is `timeout`.

## Errors

- Wiping the list on failure.
- Leaving Loading stuck true.

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
		t.Fatal("Loading = true, want false after refresh failure")
	}
	want := []string{"dot-pkgs"}
	if !reflect.DeepEqual(resp.Projects, want) {
		t.Fatalf("Projects = %#v, want %#v (must not clear on fail)", resp.Projects, want)
	}
	if resp.Error != "timeout" {
		t.Fatalf("Error = %q, want %q", resp.Error, "timeout")
	}
}
```

## Expected

1. `Loading` is `true`.
2. `Projects` is still exactly `["dot-pkgs"]` (unchanged order and content).
3. `Error` remains empty (start does not invent a failure).

## Side Effects

- None beyond the returned state snapshot (pure reducer).

## Errors

- Clearing projects to `[]` when refresh begins.

```go
import (
	"reflect"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Loading {
		t.Fatal("Loading = false, want true after refresh start")
	}
	want := []string{"dot-pkgs"}
	if !reflect.DeepEqual(resp.Projects, want) {
		t.Fatalf("Projects = %#v, want %#v (must not clear on start)", resp.Projects, want)
	}
	if resp.Error != "" {
		t.Fatalf("Error = %q, want empty on start", resp.Error)
	}
}
```

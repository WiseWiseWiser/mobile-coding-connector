## Expected

1. `HasTopLevelRefresh` is `true` for both local and remote apps.

## Side Effects

- None (read-only source inspection).

## Errors

- Refresh removed or buried only under a submenu.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.HasTopLevelRefresh {
		t.Fatalf("missing top-level Refresh on local and/or remote (sources: %v)", resp.SwiftSourcesChecked)
	}
}
```

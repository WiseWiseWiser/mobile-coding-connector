## Expected

1. `HasPeriodicRefresh` is `true` — timer/sleep loop refreshes terminals (with services).

## Side Effects

- None (read-only source inspection).

## Errors

- Only manual Refresh; no background poll that includes terminals.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.HasPeriodicRefresh {
		t.Fatalf("missing periodic services+terminals refresh (sources: %v)", resp.SwiftSourcesChecked)
	}
}
```

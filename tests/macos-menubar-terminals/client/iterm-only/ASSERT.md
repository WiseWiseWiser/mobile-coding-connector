## Expected

1. `UsesITermOnly` is `true` — sources reference iTerm and do not fall back to Terminal.app.
2. `HasTerminalAppFallback` is `false`.

## Side Effects

- None (read-only source inspection).

## Errors

- Fallback open of `/Applications/Terminal.app` when iTerm missing.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.HasTerminalAppFallback {
		t.Fatalf("Terminal.app fallback present; want iTerm-only (sources: %v)", resp.SwiftSourcesChecked)
	}
	if !resp.UsesITermOnly {
		t.Fatalf("open path must reference iTerm and exclude Terminal.app fallback (sources: %v)", resp.SwiftSourcesChecked)
	}
}
```

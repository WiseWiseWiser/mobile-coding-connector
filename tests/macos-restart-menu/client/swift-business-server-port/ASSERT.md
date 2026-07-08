## Expected

1. `BusinessOnServerPort` is `true` — grok and codex usage on `ServerClient` port `23712`.
2. `DebugOnServerPort` is `true` — debug log settings on `/api/debug/log` via server port.
3. `GrokOnDaemonPort` is `false` — `DaemonClient` must not expose grok/codex usage routes.

## Side Effects

- None (read-only source inspection).

## Errors

- Business APIs still registered on daemon port `23312`.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.BusinessOnServerPort {
		t.Fatalf("grok/codex not on ServerClient:23712 (sources: %v)", resp.SwiftSourcesChecked)
	}
	if !resp.DebugOnServerPort {
		t.Fatalf("debug log API not on ServerClient:23712 (sources: %v)", resp.SwiftSourcesChecked)
	}
	if resp.GrokOnDaemonPort {
		t.Fatal("DaemonClient still serves grok/codex usage — must move to server port")
	}
}
```
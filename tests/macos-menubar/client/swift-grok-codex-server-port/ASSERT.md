## Expected

1. `GrokViaServerClient` is `true`.
2. `CodexViaServerClient` is `true`.
3. `GrokViaDaemonClient` is `false` — `AppState.refresh` must not call `DaemonClient.shared.grokUsage()`.
4. `CodexViaDaemonClient` is `false` — must not call `DaemonClient.shared.codexUsage()`.

## Side Effects

- None (read-only source inspection).

## Errors

- Usage refresh still goes through daemon port `23312`.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.GrokViaServerClient {
		t.Fatalf("grok usage not on ServerClient (sources: %v)", resp.SwiftSourcesChecked)
	}
	if !resp.CodexViaServerClient {
		t.Fatalf("codex usage not on ServerClient (sources: %v)", resp.SwiftSourcesChecked)
	}
	if resp.GrokViaDaemonClient {
		t.Fatalf("AppState still refreshes grok via DaemonClient (sources: %v)", resp.SwiftSourcesChecked)
	}
	if resp.CodexViaDaemonClient {
		t.Fatalf("AppState still refreshes codex via DaemonClient (sources: %v)", resp.SwiftSourcesChecked)
	}
}
```
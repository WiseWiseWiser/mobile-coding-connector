## Expected

1. `UsesITermOnly` is `true` — sources reference iTerm and do not fall back to Terminal.app.
2. `HasTerminalAppFallback` is `false`.
3. `OpensViaLocalITerm2API` is `true` — local product references
   `/api/local/iterm2/open` (not osascript-only product path).
4. `HasDirectITermOpenerProductPath` is `false` on local terminal open paths
   (`ITermOpener.openCommandOrAlert` retired for product terminals).

## Side Effects

- None (read-only source inspection).

## Errors

- Fallback open of `/Applications/Terminal.app` when iTerm missing.
- Local terminals still open only via direct `ITermOpener` osascript.

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
	if !resp.OpensViaLocalITerm2API {
		t.Fatalf("local open must use /api/local/iterm2/open (sources: %v)", resp.SwiftSourcesChecked)
	}
	if resp.HasDirectITermOpenerProductPath {
		t.Fatalf("local product terminals must not call ITermOpener.openCommand* (sources: %v)", resp.SwiftSourcesChecked)
	}
}
```

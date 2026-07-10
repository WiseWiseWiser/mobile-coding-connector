## Expected

1. `HasOpenITerm2API` is true — sources define openITerm2 and
   `/api/local/iterm2/open`.
2. `OpenITerm2UsesBearer` is true — Authorization/Bearer/token plumbing present
   on ServerClient.

## Side Effects

- None (read-only source inspection).

## Errors

- Still using direct `ITermOpener` only, with no HTTP open API.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.HasOpenITerm2API {
		t.Fatalf("missing openITerm2 and/or /api/local/iterm2/open (sources: %v)", resp.SwiftSourcesChecked)
	}
	if !resp.OpenITerm2UsesBearer {
		t.Fatalf("open request path must apply Bearer/Authorization (sources: %v)", resp.SwiftSourcesChecked)
	}
}
```

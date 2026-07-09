## Expected

1. `HasRemoteServerSwitcher` is `true` — level-1 Server/domain switcher present.

## Side Effects

- None (read-only source inspection).

## Errors

- Switcher only nested under Terminals, or entirely missing.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.HasRemoteServerSwitcher {
		t.Fatalf("remote app missing level-1 Server/domain switcher (sources: %v)", resp.SwiftSourcesChecked)
	}
}
```

## Expected

1. `HasLocalTerminalsMenu` is `true`.

## Side Effects

- None (read-only source inspection).

## Errors

- Local app missing Terminals menu structure.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.HasLocalTerminalsMenu {
		t.Fatalf("local app missing Terminals menu (sources: %v)", resp.SwiftSourcesChecked)
	}
}
```

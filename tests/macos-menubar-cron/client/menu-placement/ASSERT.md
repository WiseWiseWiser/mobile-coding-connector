## Expected

1. `MenuPlacementOK` is true for both local and remote apps.

## Side Effects

- None (read-only source inspection).

## Errors

- Cron before Services or after Terminals; missing sibling menus.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.MenuPlacementOK {
		t.Fatalf("Cron not placed after Services and before Terminals in both apps (sources: %v)", resp.SwiftSourcesChecked)
	}
}
```

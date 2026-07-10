## Expected

1. `EditorSaveUpdates` is true (editor context + updateCronTask / update path).

## Side Effects

- None (read-only source inspection).

## Errors

- Editor without update wiring; only create path present.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.EditorSaveUpdates {
		t.Fatalf("editor Save not wired to update/PUT (sources: %v)", resp.SwiftSourcesChecked)
	}
}
```

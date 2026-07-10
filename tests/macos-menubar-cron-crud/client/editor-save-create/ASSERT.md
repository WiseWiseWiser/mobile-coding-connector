## Expected

1. `EditorSaveCreates` is true (editor context + createCronTask / create path).

## Side Effects

- None (read-only source inspection).

## Errors

- Editor without create wiring; only update path present.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.EditorSaveCreates {
		t.Fatalf("editor Save not wired to create/POST (sources: %v)", resp.SwiftSourcesChecked)
	}
}
```

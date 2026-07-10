## Expected

1. `HasDefinitionFields` is true (definition/editor type + key fields present).

## Side Effects

- None (read-only source inspection).

## Errors

- Only title fields (name/status); missing command/workingDir/timeout for form.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.HasDefinitionFields {
		t.Fatalf("missing Cron definition/editor fields (sources: %v)", resp.SwiftSourcesChecked)
	}
}
```

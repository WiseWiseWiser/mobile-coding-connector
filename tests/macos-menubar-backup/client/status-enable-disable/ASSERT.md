## Expected

1. `HasStatusNestedEnableDisable` is true.

## Side Effects

- None (read-only source inspection).

## Errors

- Flat Enable at Backup root without Status nest, or missing Disable.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.HasStatusNestedEnableDisable {
		t.Fatalf("missing nested Status Enable/Disable (sources: %v)", resp.SwiftSourcesChecked)
	}
}
```

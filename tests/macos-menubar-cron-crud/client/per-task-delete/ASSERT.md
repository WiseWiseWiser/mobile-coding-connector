## Expected

1. `HasPerTaskDelete` is true.

## Side Effects

- None (read-only source inspection).

## Errors

- Missing Delete… action in nested task menu.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.HasPerTaskDelete {
		t.Fatalf("missing per-task Delete… (sources: %v)", resp.SwiftSourcesChecked)
	}
}
```

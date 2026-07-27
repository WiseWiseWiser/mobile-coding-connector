## Expected

1. `HasHistoryDisabled` is true.

## Side Effects

- None (read-only source inspection).

## Errors

- History navigates or is enabled; missing History entry.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.HasHistoryDisabled {
		t.Fatalf("History not present as disabled placeholder (sources: %v)", resp.SwiftSourcesChecked)
	}
}
```

## Expected

1. `StreamProgressConsumed` is true (progress callback / onEvent / format helpers).

## Side Effects

- None (read-only source inspection).

## Errors

- Only reading `done.archive_token` with no intermediate frame handling for UI.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.StreamProgressConsumed {
		t.Fatalf("stream progress not consumed for display (sources: %v)", resp.SwiftSourcesChecked)
	}
}
```

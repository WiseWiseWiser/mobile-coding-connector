## Expected

1. `HasRemoteTerminalsMenu` is `true`.

## Side Effects

- None (read-only source inspection).

## Errors

- Remote app missing Terminals menu structure.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.HasRemoteTerminalsMenu {
		t.Fatalf("remote app missing Terminals menu (sources: %v)", resp.SwiftSourcesChecked)
	}
}
```

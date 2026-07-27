## Expected

1. `HasNestedServiceMenu` is `true` — per-service nested `Menu` with action buttons.

## Side Effects

- None (read-only source inspection).

## Errors

- Flat button list without nested submenu per service.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.HasNestedServiceMenu {
		t.Fatalf("missing nested per-service Menu (sources: %v)", resp.SwiftSourcesChecked)
	}
}
```
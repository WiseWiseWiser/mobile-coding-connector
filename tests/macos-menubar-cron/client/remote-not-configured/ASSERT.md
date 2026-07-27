## Expected

1. `RemoteShowsNotConfigured` is true.

## Side Effects

- None (read-only source inspection).

## Errors

- Empty Cron menu with no placeholder; uses long status-line-only copy.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.RemoteShowsNotConfigured {
		t.Fatalf("remote Cron missing Not configured path (sources: %v)", resp.SwiftSourcesChecked)
	}
}
```

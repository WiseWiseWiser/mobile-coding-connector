## Expected

1. `RemoteShowsNotConfigured` is true.

## Side Effects

- None (read-only source inspection).

## Errors

- Empty Cron menu with no placeholder; uses long status-line-only copy.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.RemoteShowsNotConfigured {
		t.Fatalf("remote Cron missing Not configured path (sources: %v)", resp.SwiftSourcesChecked)
	}
}
```

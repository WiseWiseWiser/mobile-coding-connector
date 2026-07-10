## Expected

1. `HasRemoteNewCronTask` is true.

## Side Effects

- None (read-only source inspection).

## Errors

- Missing New Cron Task… on remote Cron menu.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.HasRemoteNewCronTask {
		t.Fatalf("remote app missing New Cron Task… (sources: %v)", resp.SwiftSourcesChecked)
	}
}
```

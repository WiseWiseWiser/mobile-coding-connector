## Expected

1. `HasLocalNewCronTask` is true.

## Side Effects

- None (read-only source inspection).

## Errors

- Missing New Cron Task… on local Cron menu.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.HasLocalNewCronTask {
		t.Fatalf("local app missing New Cron Task… (sources: %v)", resp.SwiftSourcesChecked)
	}
}
```

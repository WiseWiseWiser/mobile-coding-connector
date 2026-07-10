## Expected

1. `EnableImmediateShowsWindow` is true.

## Side Effects

- None (read-only source inspection).

## Errors

- Enable-immediate calling `runBackupNow(triggeredBySchedule: true)` only (silent).

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.EnableImmediateShowsWindow {
		t.Fatalf("enable-immediate path does not open progress window (sources: %v)", resp.SwiftSourcesChecked)
	}
}
```

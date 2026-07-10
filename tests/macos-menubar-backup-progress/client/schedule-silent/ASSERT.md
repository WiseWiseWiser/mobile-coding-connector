## Expected

1. `ScheduleSilent` is true (schedule uses triggeredBySchedule true + no window branch).

## Side Effects

- None (read-only source inspection).

## Errors

- Always opening the progress window including hourly ticks.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.ScheduleSilent {
		t.Fatalf("schedule path not silent / missing triggeredBySchedule:true (sources: %v)", resp.SwiftSourcesChecked)
	}
}
```

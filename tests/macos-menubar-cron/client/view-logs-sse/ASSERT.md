## Expected

1. `HasLogStreamEndpoint` is true — sources reference `/api/logs/stream`.
2. `ViewLogsUsesStream` is true — View Logs path uses stream, not `/usr/bin/tail`.

## Side Effects

- None (read-only source inspection).

## Errors

- Local-only tail Process; missing stream endpoint for cron View Logs.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.HasLogStreamEndpoint {
		t.Fatalf("missing /api/logs/stream (sources: %v)", resp.SwiftSourcesChecked)
	}
	if !resp.ViewLogsUsesStream {
		t.Fatalf("View Logs does not use SSE stream path (sources: %v)", resp.SwiftSourcesChecked)
	}
}
```

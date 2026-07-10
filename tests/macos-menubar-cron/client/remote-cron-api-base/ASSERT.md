## Expected

1. `RemoteCronUsesBaseURL` is true — remote/Shared reference cron-tasks against base URL client.
2. `RemoteCronUsesAuth` is true — token/Authorization/Bearer present in client path.
3. `RemoteAvoidsKeepAlivePort` is true — cron-tasks not paired with keep-alive port 23312.

## Side Effects

- None (read-only source inspection).

## Errors

- Cron calls go to daemon keep-alive port; missing Bearer for remote.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.RemoteCronUsesBaseURL {
		t.Fatalf("remote cron APIs missing baseURL /api/cron-tasks path (sources: %v)", resp.SwiftSourcesChecked)
	}
	if !resp.RemoteCronUsesAuth {
		t.Fatalf("remote cron path missing auth/token handling (sources: %v)", resp.SwiftSourcesChecked)
	}
	if !resp.RemoteAvoidsKeepAlivePort {
		t.Fatalf("cron-tasks appears tied to keep-alive port 23312 (sources: %v)", resp.SwiftSourcesChecked)
	}
}
```

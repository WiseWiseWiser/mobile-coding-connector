## Expected

1. `HasLogStreamEndpoint` is `true` — `ServerClient.swift` calls `/api/logs/stream`.
2. `ServerClientNoLocalTail` is `true` — `ServerClient.swift` does not spawn `/usr/bin/tail` or `Process` for service logs.
3. `LogWindowUsesStream` is `true` — `LogTailWindow.swift` uses SSE / `URLSession` / stream (not `executableURL` tail).
4. `LogStreamLines1000` is `true` — stream request includes `lines=1000` or `1000` initial lines.
5. `ViewLogsInvokesStream` is `true` — `AICriticApp.swift` View Logs invokes the server stream path (not local tail).

## Side Effects

- None (read-only source inspection).

## Errors

- View Logs still spawns local `tail -fn 1000` or reads log files in the GUI process cwd.

```go
import (
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.HasLogStreamEndpoint {
		t.Fatalf("ServerClient missing /api/logs/stream (sources: %v)", resp.SwiftSourcesChecked)
	}
	if !resp.ServerClientNoLocalTail {
		t.Fatalf("ServerClient still spawns local tail/Process for service logs (sources: %v)", resp.SwiftSourcesChecked)
	}
	if !resp.LogWindowUsesStream {
		t.Fatalf("LogTailWindow does not use SSE/URLSession stream (sources: %v)", resp.SwiftSourcesChecked)
	}
	if !resp.LogStreamLines1000 {
		t.Fatalf("log stream request missing lines=1000 (sources: %v)", resp.SwiftSourcesChecked)
	}
	if !resp.ViewLogsInvokesStream {
		t.Fatalf("View Logs does not invoke server stream path (sources: %v)", resp.SwiftSourcesChecked)
	}
}
```
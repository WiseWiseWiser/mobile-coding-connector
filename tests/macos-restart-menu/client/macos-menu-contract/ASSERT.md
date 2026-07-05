## Expected

1. `MenuLabel` is exactly `Restart Daemon`.
2. `ClientMethod` is `restartDaemon` (not `restartServer`).
3. `RestartEndpoint` is `/api/keep-alive/restart-daemon` (not `/api/keep-alive/restart`).
4. `RestartEndpoint` must not equal the legacy signal path `/api/keep-alive/restart`.

## Side Effects

- None (read-only source inspection).

## Errors

- Menu still labeled Restart Server or still calls `restartServer()` / `/restart`.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.MenuLabel != expectedMenuLabel {
		t.Fatalf("menu label = %q, want %q (sources: %v)", resp.MenuLabel, expectedMenuLabel, resp.SwiftSourcesChecked)
	}
	if resp.ClientMethod != expectedClientMethod {
		t.Fatalf("menu handler = DaemonClient.shared.%s(), want %s()", resp.ClientMethod, expectedClientMethod)
	}
	if resp.ClientMethod == legacyClientMethod {
		t.Fatalf("menu still calls legacy %s()", legacyClientMethod)
	}
	if resp.RestartEndpoint != expectedRestartPath {
		t.Fatalf("restart endpoint = %q, want %q", resp.RestartEndpoint, expectedRestartPath)
	}
	if resp.RestartEndpoint == legacyRestartPath {
		t.Fatalf("DaemonClient still targets legacy %s", legacyRestartPath)
	}
}
```
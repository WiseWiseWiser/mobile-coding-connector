## Expected

- `Run` succeeds.
- `StartDetachedCalled` with `StartDetachedSudo = true`.
- `StartDetachedPID` is 4242.
- Stdout contains PID, config path, and log path under cache dir.

## Side Effects

- Config written to cache `run.json`; log path `sing-box.log`.

## Errors

- None.

## Exit Code

- Success (parent exits after printing status).

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.RunErr != nil {
		t.Fatalf("run-tun detach error: %v", resp.RunErr)
	}
	if !resp.StartDetachedCalled {
		t.Fatal("StartDetached must be called")
	}
	if !resp.StartDetachedSudo {
		t.Fatal("non-root detach must use sudo")
	}
	if resp.StartDetachedPID != 4242 {
		t.Fatalf("pid = %d, want 4242", resp.StartDetachedPID)
	}
	out := resp.Stdout
	if !strings.Contains(out, "4242") {
		t.Fatalf("stdout must contain PID; got %q", out)
	}
	if resp.CacheConfigPath == "" || !strings.Contains(resp.CacheConfigPath, "run.json") {
		t.Fatalf("config path = %q, want run.json under cache", resp.CacheConfigPath)
	}
	if resp.CacheLogPath == "" || !strings.Contains(resp.CacheLogPath, "sing-box.log") {
		t.Fatalf("log path = %q, want sing-box.log", resp.CacheLogPath)
	}
	if !strings.Contains(out, "run.json") || !strings.Contains(out, "sing-box.log") {
		t.Fatalf("stdout must print paths; got %q", out)
	}
	if resp.RunSingBoxCalled {
		t.Fatal("foreground RunSingBox must not run in detach mode")
	}
}
```
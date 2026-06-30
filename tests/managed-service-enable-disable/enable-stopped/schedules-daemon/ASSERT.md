## Expected

1. `Run` completes without error and `Response.ActionResult` is non-nil.
2. `ActionResult.Message` mentions the service will not start immediately until the daemon checks (contains `daemon` and `next`).
3. Immediately after enable, `Response.TargetRunningImmediate` is false (no synchronous start).
4. On-disk `services.json` has `enabled: true` for the target.
5. After the wait window, `Response.TargetRunningAfterWait` is true with `TargetPID > 0`.
6. Live process check succeeds for the reported PID.

## Side Effects

- `enabled=true` persisted.
- `desiredRunning` scheduled; daemon starts the service within one reconcile window.

## Errors

- Synchronous `start()` in the enable handler (running immediately with zero wait).
- Service never starts within 7s after enable.

## Exit Code

0 from `Run`.

```go
import (
	"strings"
	"syscall"
	"testing"

	"github.com/xhd2015/doctest/assert"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ActionError != "" {
		t.Fatalf("enable API failed: %s", resp.ActionError)
	}
	if resp.ActionResult == nil {
		t.Fatal("ActionResult is nil")
	}

	msg := strings.ToLower(resp.ActionResult.Message)
	assert.Output(t, msg, `<contains>
daemon
next
</contains>`)

	if resp.TargetRunningImmediate {
		t.Fatal("service started immediately on enable; want deferred daemon start")
	}

	enabled, present := enabledFieldOnDisk(resp.ServicesOnDisk, req.TargetID)
	if !present || enabled == nil || !*enabled {
		t.Fatalf("services.json enabled = %v present=%v, want true", enabled, present)
	}

	if !resp.TargetRunningAfterWait || resp.TargetPID <= 0 {
		t.Fatalf("TargetRunningAfterWait=%v TargetPID=%d, want running with pid > 0 after wait",
			resp.TargetRunningAfterWait, resp.TargetPID)
	}
	if err := syscall.Kill(resp.TargetPID, 0); err != nil {
		t.Fatalf("process %d not alive after daemon window: %v", resp.TargetPID, err)
	}
}
```
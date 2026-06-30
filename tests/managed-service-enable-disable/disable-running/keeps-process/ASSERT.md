## Expected

1. `Run` completes without error and `Response.ActionResult` is non-nil.
2. `ActionResult.Status` is `ok`.
3. `ActionResult.Message` contains `won't stop immediately` (case-insensitive).
4. On-disk `services.json` has `enabled: false` for the target service.
5. `Response.TargetPID` is greater than zero and `processAlive(pid)` is true.
6. `GET /api/services` still reports the target as running or starting with the same live PID.

## Side Effects

- `services.json` updated with `enabled=false`.
- Service process continues running until manual stop.

## Errors

- Disable stops the process synchronously.
- Missing or wrong contextual message.
- `enabled` not persisted as false.

## Exit Code

0 from `Run`; assertion failures fail the test.

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
		t.Fatalf("disable API failed: %s", resp.ActionError)
	}
	if resp.ActionResult == nil {
		t.Fatal("ActionResult is nil")
	}
	if resp.ActionResult.Status != "ok" {
		t.Fatalf("status = %q, want ok", resp.ActionResult.Status)
	}

	assert.Output(t, strings.ToLower(resp.ActionResult.Message), `<contains>
won't stop immediately
</contains>`)

	enabled, present := enabledFieldOnDisk(resp.ServicesOnDisk, req.TargetID)
	if !present || enabled == nil || *enabled {
		t.Fatalf("services.json enabled = %v present=%v, want false", enabled, present)
	}

	if resp.TargetPID <= 0 {
		t.Fatalf("TargetPID = %d, want > 0 after disable on running service", resp.TargetPID)
	}
	if err := syscall.Kill(resp.TargetPID, 0); err != nil {
		t.Fatalf("process %d not alive after disable: %v", resp.TargetPID, err)
	}

	svc, ok := findServiceByID(resp.ServicesAfterAction, req.TargetID)
	if !ok {
		t.Fatalf("target %q missing from GET /api/services", req.TargetID)
	}
	if svc.PID != resp.TargetPID {
		t.Fatalf("API pid = %d, want %d", svc.PID, resp.TargetPID)
	}
	if svc.Status != "running" && svc.Status != "starting" {
		t.Fatalf("status = %q, want running or starting", svc.Status)
	}
}
```
## Expected

1. `Run` completes without error and `Response.ActionResult` is non-nil.
2. `ActionResult.Message` contains `already running` (case-insensitive).
3. On-disk `services.json` has `enabled: true` for the target.
4. `Response.TargetPID` is greater than zero and remains alive.
5. API status remains running/starting with the same PID.

## Side Effects

- `enabled=true` persisted without stopping or restarting the process.

## Errors

- Process stopped or restarted during enable.
- Missing already-running prompt.

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

	assert.Output(t, strings.ToLower(resp.ActionResult.Message), `<contains>
already running
</contains>`)

	enabled, present := enabledFieldOnDisk(resp.ServicesOnDisk, req.TargetID)
	if !present || enabled == nil || !*enabled {
		t.Fatalf("services.json enabled = %v present=%v, want true", enabled, present)
	}

	if resp.TargetPID <= 0 {
		t.Fatal("TargetPID <= 0, want running process")
	}
	if err := syscall.Kill(resp.TargetPID, 0); err != nil {
		t.Fatalf("process %d not alive: %v", resp.TargetPID, err)
	}

	svc, ok := findServiceByID(resp.ServicesAfterAction, req.TargetID)
	if !ok {
		t.Fatalf("target %q missing from GET /api/services", req.TargetID)
	}
	if svc.PID != resp.TargetPID {
		t.Fatalf("API pid = %d, want %d", svc.PID, resp.TargetPID)
	}
}
```
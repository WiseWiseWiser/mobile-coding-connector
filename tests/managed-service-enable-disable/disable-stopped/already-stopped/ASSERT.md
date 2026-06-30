## Expected

1. `Run` completes without error and `Response.ActionResult` is non-nil.
2. `ActionResult.Status` is `ok`.
3. `ActionResult.Message` contains `already stopped` (case-insensitive).
4. On-disk `services.json` has `enabled: false` for the target.
5. Target service remains stopped (`pid` 0 or not alive).

## Side Effects

- `services.json` updated with `enabled=false`.

## Errors

- Wrong prompt for stopped service.
- Service unexpectedly started.

## Exit Code

0 from `Run`.

```go
import (
	"strings"
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

	assert.Output(t, strings.ToLower(resp.ActionResult.Message), `<contains>
already stopped
</contains>`)

	enabled, present := enabledFieldOnDisk(resp.ServicesOnDisk, req.TargetID)
	if !present || enabled == nil || *enabled {
		t.Fatalf("services.json enabled = %v present=%v, want false", enabled, present)
	}

	svc, ok := findServiceByID(resp.ServicesAfterAction, req.TargetID)
	if !ok {
		t.Fatalf("target %q missing from GET /api/services", req.TargetID)
	}
	if svc.PID > 0 && processAlive(svc.PID) {
		t.Fatalf("service unexpectedly running with pid %d", svc.PID)
	}
}
```
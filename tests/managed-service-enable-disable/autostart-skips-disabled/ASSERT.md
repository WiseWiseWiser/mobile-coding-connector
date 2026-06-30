## Expected

1. `Run` completes without error after boot wait.
2. Service `enabled-svc` is running with `pid > 0` and a live process.
3. Service `disabled-svc` is not running (`pid` 0 or not alive).
4. On-disk `services.json` retains `enabled: false` only on `disabled-svc`; `enabled-svc` omits the field or has `enabled: true`.

## Side Effects

- Only the enabled definition receives boot auto-start.

## Errors

- Disabled service auto-starts at boot.
- Enabled service (field absent) does not auto-start.

## Exit Code

0 from `Run`.

```go
import (
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}

	enabledSvc, ok := findServiceByID(resp.ServicesAfterAction, "enabled-svc")
	if !ok {
		t.Fatal("enabled-svc missing from GET /api/services")
	}
	if enabledSvc.PID <= 0 || !processAlive(enabledSvc.PID) {
		t.Fatalf("enabled-svc pid=%d status=%q, want live process after boot autostart",
			enabledSvc.PID, enabledSvc.Status)
	}

	disabledSvc, ok := findServiceByID(resp.ServicesAfterAction, "disabled-svc")
	if !ok {
		t.Fatal("disabled-svc missing from GET /api/services")
	}
	if disabledSvc.PID > 0 && processAlive(disabledSvc.PID) {
		t.Fatalf("disabled-svc unexpectedly running with pid %d", disabledSvc.PID)
	}

	disabledOnDisk, present := enabledFieldOnDisk(resp.ServicesOnDisk, "disabled-svc")
	if !present || disabledOnDisk == nil || *disabledOnDisk {
		t.Fatalf("disabled-svc on-disk enabled = %v present=%v, want false", disabledOnDisk, present)
	}

	// enabled-svc should default to enabled when field is absent.
	if v, present := enabledFieldOnDisk(resp.ServicesOnDisk, "enabled-svc"); present && v != nil && !*v {
		t.Fatalf("enabled-svc on-disk enabled = false, want absent or true")
	}
}
```
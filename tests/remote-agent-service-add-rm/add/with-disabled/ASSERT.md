---
explanation: "L2 service add --disabled writes enabled=false"
---

## Expected

1. Exit 0.
2. Disk row `demo-disabled` has `enabled: false`.
3. Not running (PID == 0 / stopped).

## Exit Code

0.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v\ncombined:\n%s", err, resp.Combined)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}
	enabled, present := diskEnabled(resp.ServicesOnDisk, "demo-disabled")
	if !present || enabled == nil || *enabled {
		t.Fatalf("services.json enabled = %v present=%v, want false; disk=%v",
			enabled, present, resp.ServicesOnDisk)
	}
	if resp.TargetRunning || resp.TargetPID > 0 {
		t.Fatalf("disabled add should not start process; pid=%d running=%v",
			resp.TargetPID, resp.TargetRunning)
	}
}
```

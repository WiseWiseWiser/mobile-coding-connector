## Expected

1. `Run` completes without error and `Response.StartResult` is non-nil.
2. `Response.ServiceLog` contains `starting service`.
3. `Response.ServiceLog` does **not** contain `fork/exec /bin/bash`.
4. `Response.TargetRunning` is true with `TargetPID > 0`.

## Side Effects

- `workingDir` created before bash launch.
- Service log records a normal start marker.

## Errors

- Log contains `fork/exec /bin/bash` (regression of openclaw bug).
- Missing `starting service` marker in log.
- Service failed to start.

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
	if resp.StartError != "" {
		t.Fatalf("start API failed: %s", resp.StartError)
	}
	if resp.StartResult == nil {
		t.Fatal("StartResult is nil")
	}

	if resp.ServiceLog == "" {
		t.Fatal("ServiceLog is empty; expected start marker in services log file")
	}

	logLower := strings.ToLower(resp.ServiceLog)
	assert.Output(t, logLower, `<contains>
starting service
</contains>`)

	if strings.Contains(resp.ServiceLog, "fork/exec /bin/bash") {
		t.Fatalf("service log contains misleading fork/exec /bin/bash error:\n%s", resp.ServiceLog)
	}

	if !resp.TargetRunning || resp.TargetPID <= 0 {
		t.Fatalf("TargetRunning=%v TargetPID=%d, want running with pid > 0",
			resp.TargetRunning, resp.TargetPID)
	}
	if err := syscall.Kill(resp.TargetPID, 0); err != nil {
		t.Fatalf("process %d not alive: %v", resp.TargetPID, err)
	}
}
```
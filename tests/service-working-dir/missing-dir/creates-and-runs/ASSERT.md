## Expected

1. `Run` completes without error and `Response.StartResult` is non-nil.
2. `StartResult.Status` is `running` or `starting`.
3. `Response.WorkingDirExists` is true and `WorkingDirIsDir` is true.
4. `Response.TargetRunning` is true with `TargetPID > 0`.
5. `GET /api/services` reports target `status` as `running` or `starting`.
6. Live process check succeeds for the reported PID.

## Side Effects

- `workingDir` directory created on disk via `os.MkdirAll`.
- Service process running with `cmd.Dir` set to the created path.

## Errors

- Start API failure.
- `workingDir` still missing after start.
- Service not running or `pid <= 0`.

## Exit Code

0 from `Run`; assertion failures fail the test.

```go
import (
	"syscall"
	"testing"
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
	if resp.StartResult.Status != "running" && resp.StartResult.Status != "starting" {
		t.Fatalf("start status = %q, want running or starting", resp.StartResult.Status)
	}

	if !resp.WorkingDirExists || !resp.WorkingDirIsDir {
		t.Fatalf("WorkingDirExists=%v WorkingDirIsDir=%v for %q, want existing directory",
			resp.WorkingDirExists, resp.WorkingDirIsDir, resp.WorkingDir)
	}

	if !resp.TargetRunning || resp.TargetPID <= 0 {
		t.Fatalf("TargetRunning=%v TargetPID=%d, want running with pid > 0",
			resp.TargetRunning, resp.TargetPID)
	}
	if err := syscall.Kill(resp.TargetPID, 0); err != nil {
		t.Fatalf("process %d not alive: %v", resp.TargetPID, err)
	}

	svc, ok := findServiceByID(resp.ServicesAfterStart, req.TargetID)
	if !ok {
		t.Fatalf("target %q missing from GET /api/services", req.TargetID)
	}
	if svc.Status != "running" && svc.Status != "starting" {
		t.Fatalf("status = %q, want running or starting", svc.Status)
	}
	if svc.PID != resp.TargetPID {
		t.Fatalf("API pid = %d, want %d", svc.PID, resp.TargetPID)
	}
}
```
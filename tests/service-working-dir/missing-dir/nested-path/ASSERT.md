## Expected

1. `Run` completes without error and `Response.StartResult` is non-nil.
2. `Response.WorkingDirExists` is true and `WorkingDirIsDir` is true for nested path.
3. Parent segments `a` and `a/b` under the temp base also exist as directories.
4. `Response.TargetRunning` is true with `TargetPID > 0`.

## Side Effects

- Full nested `workingDir` tree created via `os.MkdirAll`.
- Service process running.

## Errors

- Only partial path created.
- Nested directory missing after start.
- Service not running.

## Exit Code

0 from `Run`; assertion failures fail the test.

```go
import (
	"os"
	"path/filepath"
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

	if !resp.WorkingDirExists || !resp.WorkingDirIsDir {
		t.Fatalf("WorkingDirExists=%v WorkingDirIsDir=%v for %q, want existing nested directory",
			resp.WorkingDirExists, resp.WorkingDirIsDir, resp.WorkingDir)
	}

	for _, rel := range []string{"a", "a/b", "a/b/c"} {
		p := filepath.Join(req.TempBase, rel)
		info, err := os.Stat(p)
		if err != nil {
			t.Fatalf("Stat(%q) error = %v, want existing directory", p, err)
		}
		if !info.IsDir() {
			t.Fatalf("%q is not a directory", p)
		}
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
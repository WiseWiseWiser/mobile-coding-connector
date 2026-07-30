---
explanation: "L2 service add --start leaves process running"
---

## Expected

1. Exit 0.
2. Stdout contains Created (or start confirmation) and `demo-start`.
3. Target is running: `TargetRunning` true and/or `TargetPID > 0`.
4. Disk has name `demo-start`.

## Exit Code

0.

```go
import (
	"strings"
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
	if !strings.Contains(resp.Stdout, "demo-start") {
		t.Fatalf("stdout missing demo-start:\n%s", resp.Stdout)
	}
	if !diskHasName(resp.ServicesOnDisk, "demo-start") {
		t.Fatalf("services.json missing demo-start; disk=%v", resp.ServicesOnDisk)
	}
	if !resp.TargetRunning && resp.TargetPID <= 0 {
		// also accept Status running from ListAll
		running := false
		for _, s := range resp.ServicesAfter {
			if s.Name == "demo-start" && (s.PID > 0 || s.Status == "running" || s.Status == "starting") {
				running = true
				break
			}
		}
		if !running {
			t.Fatalf("expected demo-start running with PID>0; after=%v targetPID=%d",
				resp.ServicesAfter, resp.TargetPID)
		}
	}
}
```

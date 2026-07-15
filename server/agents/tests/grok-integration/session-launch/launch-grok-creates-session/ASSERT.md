---
label: slow
explanation: real opencode serve startup when binary is installed
---

## Expected

1. `LaunchErr` is nil.
2. `LaunchSession` is non-nil.
3. `LaunchSession.AgentID` is `"grok"`.
4. `LaunchSession.Status` is `"running"` or `"starting"` (becomes running after health).

## Errors

- Launch failure or wrong agent_id fails the test.

```go
import (
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.LaunchErr != nil {
		t.Fatalf("launch error: %v", resp.LaunchErr)
	}
	if resp.LaunchSession == nil {
		t.Fatal("expected session info")
	}
	if resp.LaunchSession.AgentID != "grok" {
		t.Fatalf("agent_id = %q", resp.LaunchSession.AgentID)
	}
	st := resp.LaunchSession.Status
	if st != "running" && st != "starting" {
		t.Fatalf("status = %q", st)
	}
	if resp.LaunchSession.ProjectDir == "" {
		t.Fatal("project_dir empty")
	}
}
```
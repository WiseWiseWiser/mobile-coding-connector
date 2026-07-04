## Expected

1. `Run` completes without transport error.
2. `Response.RunErr` is non-empty (process exit or port-in-use message).
3. Combined output or `RunErr` mentions port conflict (`already in use` or `port` + `in use`).
4. `DaemonStatus` is nil (management API never came up on behalf of new daemon).
5. Server port occupier remains alive (`OccupierServerKilled` is false).

## Side Effects

- No new keep-alive daemon replaces the occupier.

## Errors

- Daemon starts successfully despite occupied port (regression).

## Exit Code

0 from `Run` harness; keep-alive subprocess exits non-zero.

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.RunErr == "" {
		t.Fatal("expected startup error but RunErr is empty")
	}
	lower := strings.ToLower(resp.RunErr + "\n" + resp.CombinedOutput)
	if !strings.Contains(lower, "already in use") && !strings.Contains(lower, "port") {
		t.Fatalf("error does not mention port conflict: %q", resp.RunErr)
	}
	if resp.DaemonStatus != nil && resp.DaemonStatus.KeepAlivePID > 0 {
		t.Fatal("daemon started despite occupied server port without --kill-existing")
	}
	if resp.OccupierServerPID > 0 && resp.OccupierServerKilled {
		t.Fatal("server occupier was killed without --kill-existing")
	}
}
```
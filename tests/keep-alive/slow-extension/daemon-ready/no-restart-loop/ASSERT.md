## Expected

1. `Response.RestartLoopSeen` is false — no `failed to become ready` in merged logs.
2. `Response.ServerReady` is true at least once during observation.
3. No more than one `Killing process group` sequence attributable to startup timeout
   in the observation window (heuristic: at most one `failed to become ready`).

## Side Effects

- Longer (20s) daemon observation per test.

## Errors

- Daemon fails to start.

## Exit Code

- `0` when no startup-timeout restart loop appears over 20s.

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.RestartLoopSeen {
		t.Fatal("saw failed to become ready — restart loop")
	}
	if !resp.ServerReady {
		t.Fatal("server never became ready during observation")
	}
	count := strings.Count(resp.DaemonLogs, "failed to become ready")
	if count > 0 {
		t.Fatalf("restart timeout message appeared %d times", count)
	}
}
```
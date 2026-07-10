## Expected

1. Create succeeds; after wait the target is `running` with live `PID > 0`.
2. History has **exactly 1** run entry (no second start while first is live).
3. Optionally: unfinished first run (`FinishedAt` empty) is acceptable.

## Side Effects

- Only one process for the task while sleep is active.

## Errors

- Second history entry started while first still running.
- No process started at all.

## Exit Code

0 from `Run`.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ActionError != "" {
		t.Fatalf("create failed: %s", resp.ActionError)
	}
	if resp.Target == nil {
		t.Fatal("target missing")
	}
	if resp.Target.PID <= 0 || !resp.ProcessAlive {
		t.Fatalf("want live process after wait; pid=%d alive=%v status=%q",
			resp.Target.PID, resp.ProcessAlive, resp.Target.Status)
	}
	if len(resp.History) != 1 {
		t.Fatalf("want exactly 1 history run while first still live, got %d: %+v",
			len(resp.History), resp.History)
	}
	// Guard: if multiple runs finished somehow, fail (overlap allowed would create more starts)
	started := 0
	for _, r := range resp.History {
		if r.StartedAt != "" {
			started++
		}
	}
	if started != 1 {
		t.Fatalf("want 1 started run, got %d", started)
	}
}
```

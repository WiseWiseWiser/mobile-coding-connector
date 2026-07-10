## Expected

1. Pre-snapshot: `PreRuns == 0` (disabled task did not fire during PreWait).
2. After enable + poll: `RunCount >= 1` (or history length ≥ 1).
3. Target `Enabled == true` after enable.

## Side Effects

- Enable flips scheduling; first fire occurs within poll window.

## Errors

- Runs while disabled.
- No runs after enable.

## Exit Code

0 from `Run`.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ActionError != "" {
		t.Fatalf("enable failed: %s", resp.ActionError)
	}
	if resp.PreRuns != 0 {
		t.Fatalf("disabled task fired during pre-wait: PreRuns=%d PreHistory=%+v",
			resp.PreRuns, resp.PreHistory)
	}
	if resp.RunCount < 1 {
		t.Fatalf("want ≥1 run after enable, got %d history=%+v", resp.RunCount, resp.History)
	}
	if resp.Target == nil {
		t.Fatal("target missing after enable")
	}
	if !resp.Target.Enabled {
		t.Fatal("target Enabled=false after enable")
	}
}
```

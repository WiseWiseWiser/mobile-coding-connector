## Expected

1. At least one history run exists.
2. After wait, process is not alive (`PID` 0 or dead).
3. Target `LastError` and/or history `Error` contains "timeout" (case-insensitive).
4. Status is not stuck in `running` with a live PID.

## Side Effects

- Process group killed on timeout.
- Error recorded on last run + history.

## Errors

- Sleep process still alive after timeout window.
- No timeout error recorded.

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
	if resp.RunCount < 1 && len(resp.History) < 1 {
		t.Fatal("want at least one history run after timeout")
	}
	if resp.ProcessAlive {
		t.Fatalf("process still alive after timeout window; pid=%d", resp.TargetPID)
	}
	errText := ""
	if resp.Target != nil {
		errText = resp.Target.LastError
	}
	for _, r := range resp.History {
		if r.Error != "" {
			errText += " " + r.Error
		}
	}
	if !containsFold(errText, "timeout") {
		t.Fatalf("want timeout in lastError/history error, got %q (history=%+v target=%+v)",
			errText, resp.History, resp.Target)
	}
}
```

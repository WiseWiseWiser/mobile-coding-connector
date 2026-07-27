## Expected

1. After Sweep, session is not alive.
2. Tunnel Stop was invoked (QuickStopCount >= 1).

## Errors

- Session remains after idle with zero traffic.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.StartErr != "" {
		t.Fatalf("StartErr: %s", resp.StartErr)
	}
	if resp.SessionAliveAfterSweep {
		t.Fatal("session should expire after idle with zero traffic")
	}
	if len(resp.ListAfter) != 0 {
		t.Fatalf("ListAfter should be empty, got %+v", resp.ListAfter)
	}
	if resp.QuickStopCount < 1 && resp.OwnedStopCount < 1 {
		t.Fatal("expected tunnel Stop on idle expiry")
	}
}
```

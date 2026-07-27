## Expected

1. StopErr empty.
2. ListAfter empty.
3. Provider stop count >= 1.

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
	if resp.StopErr != "" {
		t.Fatalf("StopErr: %s", resp.StopErr)
	}
	if len(resp.ListAfter) != 0 {
		t.Fatalf("expected empty list after stop, got %+v", resp.ListAfter)
	}
	if resp.QuickStopCount < 1 && resp.OwnedStopCount < 1 {
		t.Fatal("expected tunnel handle Stop")
	}
}
```

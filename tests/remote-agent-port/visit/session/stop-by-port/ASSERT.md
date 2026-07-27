## Expected

1. Stop by port succeeds; ListAfter empty.

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
		t.Fatalf("expected empty after stop by port: %+v", resp.ListAfter)
	}
}
```

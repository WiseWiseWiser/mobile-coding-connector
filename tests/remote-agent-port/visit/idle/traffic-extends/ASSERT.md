## Expected

1. SessionAliveAfterSweep is true after traffic + advance < idle.

## Errors

- Session expired despite recent traffic.

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
	if !resp.SessionAliveAfterSweep {
		t.Fatal("session should remain alive after traffic reset + advance < idle")
	}
}
```

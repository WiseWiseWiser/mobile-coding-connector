## Expected

1. Start succeeds; proxy accepts HTTP (status > 0).
2. `ActivityAfter` is not before `ActivityBefore`.
3. Prefer `ActivityAfter.After(ActivityBefore)` when Now moves on request handling.

## Errors

- Proxy unreachable.
- LastActivity moves backwards.

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
	if resp.ProxyHTTPStatus <= 0 {
		t.Fatal("expected HTTP status from reverse-proxy hop")
	}
	if resp.ActivityAfter.Before(resp.ActivityBefore) {
		t.Fatalf("LastActivity went backwards: before=%v after=%v", resp.ActivityBefore, resp.ActivityAfter)
	}
	if !resp.ActivityAfter.After(resp.ActivityBefore) {
		t.Fatalf("traffic should advance LastActivity; before=%v after=%v", resp.ActivityBefore, resp.ActivityAfter)
	}
}
```

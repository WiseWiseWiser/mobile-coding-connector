## Expected

1. EffectiveBrowser is `default`.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("harness: %v", err)
	}
	if resp.EffectiveBrowser != "default" {
		t.Fatalf("got %q want default", resp.EffectiveBrowser)
	}
}
```

## Expected

1. EffectiveBrowser is exactly `firefox`.

## Errors

- Returns chrome or default.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("harness: %v", err)
	}
	if resp.EffectiveBrowser != "firefox" {
		t.Fatalf("got %q want firefox", resp.EffectiveBrowser)
	}
}
```

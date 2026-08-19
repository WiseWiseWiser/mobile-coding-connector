## Expected

1. Help mentions `session` (session-id argument).
2. Help does **not** mention `/api`, `GET`, `POST`, or `server`.

## Errors

- Unknown command; no session mention; HTTP wording.

```go
import (
	"regexp"
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("harness: %v", err)
	}
	low := resp.Stdout + "\n" + resp.Stderr + "\n" + resp.ErrMsg
	if !regexp.MustCompile(`(?i)session`).MatchString(low) {
		t.Fatalf("focus help missing session-id; out=%q err=%q", resp.Combined, resp.ErrMsg)
	}
	for _, bad := range []string{`/api`, `(?i)\bGET\b`, `(?i)\bPOST\b`, `(?i)\bserver\b`} {
		if regexp.MustCompile(bad).MatchString(low) {
			t.Fatalf("focus help must not mention HTTP/API (%s); out=%q", bad, low)
		}
	}
}
```

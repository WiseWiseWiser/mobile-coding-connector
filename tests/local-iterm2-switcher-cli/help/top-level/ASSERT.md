## Expected

1. Combined help mentions `native-terminals`, `list`, `focus`.
2. Help lists aliases (`native-terminal`, `native-terms`, `native-term`).
3. Help does **not** mention `/api`, `GET`, `POST`, `server`, or stream URL paths.

## Errors

- `unknown command: native-terminals`
- HTTP/API wording in help

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
	for _, want := range []string{`native-terminals`, `(?i)\blist\b`, `(?i)\bfocus\b`} {
		if !regexp.MustCompile(want).MatchString(low) {
			t.Fatalf("help missing %s; out=%q err=%q", want, resp.Combined, resp.ErrMsg)
		}
	}
	for _, alias := range []string{`native-terminal`, `native-terms`, `native-term`} {
		if !regexp.MustCompile(regexp.QuoteMeta(alias)).MatchString(low) {
			t.Fatalf("help missing alias %s; out=%q", alias, low)
		}
	}
	for _, bad := range []string{`/api`, `(?i)\bGET\b`, `(?i)\bPOST\b`, `(?i)\bserver\b`, `inventory/stream`} {
		if regexp.MustCompile(bad).MatchString(low) {
			t.Fatalf("help must not mention HTTP/API (%s); out=%q", bad, low)
		}
	}
}
```

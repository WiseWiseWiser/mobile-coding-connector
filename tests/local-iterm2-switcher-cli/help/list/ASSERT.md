## Expected

1. Help Usage mentions `native-terminals`.
2. Help mentions `--json` and `--refresh`.
3. Help does **not** mention `/api`, `GET`, `POST`, `server`, or `inventory/stream`.

## Errors

- Unknown command; missing flags; HTTP wording.

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
	if !regexp.MustCompile(`native-terminals`).MatchString(low) {
		t.Fatalf("list help Usage must say native-terminals; out=%q err=%q", resp.Combined, resp.ErrMsg)
	}
	for _, want := range []string{`--json`, `--refresh`} {
		if !regexp.MustCompile(regexp.QuoteMeta(want)).MatchString(low) {
			t.Fatalf("list help missing %s; out=%q err=%q", want, resp.Combined, resp.ErrMsg)
		}
	}
	for _, bad := range []string{`/api`, `(?i)\bGET\b`, `(?i)\bPOST\b`, `(?i)\bserver\b`, `inventory/stream`} {
		if regexp.MustCompile(bad).MatchString(low) {
			t.Fatalf("list help must not mention HTTP/API (%s); out=%q", bad, low)
		}
	}
}
```

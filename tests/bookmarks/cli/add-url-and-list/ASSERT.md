## Expected

1. Exit 0.
2. Doc contains bm_cli with name CLI Dash and url http://127.0.0.1:7070.
3. Stdout may show success table including name.

## Errors

- Command missing; node not persisted.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("harness: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit=%d out=%q err=%q", resp.ExitCode, resp.Combined, resp.ErrMsg)
	}
	n := FindNode(resp.Doc, "bm_cli")
	if n == nil {
		n = FindNodeByName(resp.Doc, "CLI Dash")
	}
	if n == nil {
		t.Fatalf("added node missing; doc=%+v out=%q", resp.Doc, resp.Combined)
	}
	if n.URL != "http://127.0.0.1:7070" {
		t.Fatalf("url=%q", n.URL)
	}
}
```

## Expected

1. Exit 0, stdout names the default server.
2. The default server received exactly one `GET /ping`; the alias server none.
3. That request carried `Bearer tok-default`.

## Exit Code

0.

```go
import (
	"strings"
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}
	if resp.DefaultHits != 1 || resp.AliasHits != 0 {
		t.Fatalf("hits: default=%d alias=%d, want 1/0", resp.DefaultHits, resp.AliasHits)
	}
	if !strings.Contains(resp.Stdout, resp.DefaultURL) {
		t.Fatalf("stdout should name the default server %s:\n%s", resp.DefaultURL, resp.Stdout)
	}
	if len(resp.DefaultAuth) != 1 || resp.DefaultAuth[0] != aliasDefaultToken {
		t.Fatalf("default server auth = %v, want [%s]", resp.DefaultAuth, aliasDefaultToken)
	}
}
```

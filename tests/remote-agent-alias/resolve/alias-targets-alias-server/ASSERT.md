## Expected

1. Exit 0, stdout names the alias server.
2. The alias server received exactly one `GET /ping`; the default server none.
3. That request carried `Bearer tok-xdev` (the token saved for the alias's server).

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
	if resp.AliasHits != 1 || resp.DefaultHits != 0 {
		t.Fatalf("hits: alias=%d default=%d, want 1/0", resp.AliasHits, resp.DefaultHits)
	}
	if !strings.Contains(resp.Stdout, resp.AliasURL) {
		t.Fatalf("stdout should name the alias server %s:\n%s", resp.AliasURL, resp.Stdout)
	}
	if len(resp.AliasAuth) != 1 || resp.AliasAuth[0] != aliasLeafToken {
		t.Fatalf("alias server auth = %v, want [%s]", resp.AliasAuth, aliasLeafToken)
	}
}
```

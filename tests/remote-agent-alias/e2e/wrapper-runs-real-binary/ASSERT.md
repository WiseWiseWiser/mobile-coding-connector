---
label: heavy, e2e
---

## Expected

1. The wrapper execs the built binary recorded in its exec line.
2. Exit 0; stdout names the xdev server and reports `pong`.
3. The xdev server received exactly one `GET /ping` with `Bearer tok-xdev`;
   the default server received none.

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
	if resp.ClientBin == "" {
		t.Fatalf("L3 leaf did not build the product binary")
	}
	if !strings.Contains(resp.WrapperAfter, "exec "+resp.ClientBin+" --alias "+aliasLeafName) {
		t.Fatalf("wrapper should exec the built binary %s:\n%s", resp.ClientBin, resp.WrapperAfter)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("wrapper exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}
	if !strings.Contains(resp.Stdout, resp.AliasURL) {
		t.Fatalf("wrapper stdout should name the alias server %s:\n%s", resp.AliasURL, resp.Stdout)
	}
	if !strings.Contains(resp.Stdout, "pong") {
		t.Fatalf("wrapper stdout should report pong:\n%s", resp.Stdout)
	}
	if resp.AliasHits != 1 || resp.DefaultHits != 0 {
		t.Fatalf("hits: alias=%d default=%d, want 1/0", resp.AliasHits, resp.DefaultHits)
	}
	if len(resp.AliasAuth) != 1 || resp.AliasAuth[0] != aliasLeafToken {
		t.Fatalf("alias server auth = %v, want [%s]", resp.AliasAuth, aliasLeafToken)
	}
}
```

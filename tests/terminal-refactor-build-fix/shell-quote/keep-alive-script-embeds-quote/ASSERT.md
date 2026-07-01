## Expected

- Generated script contains shell-quoted bin path and spaced server args.
- Full script passes `sh -n` syntax check.

```go
import (
	"strings"
	"testing"

	"github.com/xhd2015/agent-pro/pkgs/shell"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("keep-alive script: %v\n%s", err, resp.KeepAliveScript)
	}
	if !resp.KeepAliveShNOK {
		t.Fatal("expected sh -n syntax check to pass")
	}
	binQuoted := shell.ShellQuote(req.KeepAliveBinPath)
	if !strings.Contains(resp.KeepAliveScript, binQuoted) {
		t.Fatalf("script missing quoted bin path %q\n%s", binQuoted, resp.KeepAliveScript)
	}
	for _, arg := range req.KeepAliveServerArgs {
		if strings.ContainsAny(arg, " \t") || strings.Contains(arg, "'") {
			argQuoted := shell.ShellQuote(arg)
			if !strings.Contains(resp.KeepAliveScript, argQuoted) {
				t.Fatalf("script missing quoted arg %q (%q)\n%s", arg, argQuoted, resp.KeepAliveScript)
			}
		}
	}
}
```
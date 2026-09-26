## Expected

1. Exit 0.
2. Stdout: `updated alias xdev (server: <old xdev URL> → https://stage.example.com)`.
3. Stderr warns that the new server has no saved token, with the wrapper-form hint.
4. The wrapper file is byte-identical before and after.
5. The store now points `xdev` at `https://stage.example.com`.

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
	wantUpdate := "updated alias " + aliasLeafName + " (server: " + resp.AliasURL + " → " + aliasStageServer + ")"
	if !strings.Contains(resp.Stdout, wantUpdate) {
		t.Fatalf("stdout missing %q:\n%s", wantUpdate, resp.Stdout)
	}
	if !strings.Contains(resp.Stderr, "warning: no token saved for "+aliasStageServer) {
		t.Fatalf("stderr missing the missing-token warning:\n%s", resp.Stderr)
	}
	if !strings.Contains(resp.Stderr, aliasLeafBinary+" config set --token-stdin") {
		t.Fatalf("stderr missing the wrapper-form hint:\n%s", resp.Stderr)
	}
	if resp.WrapperBefore == "" {
		t.Fatalf("prelude did not install a wrapper")
	}
	if resp.WrapperBefore != resp.WrapperAfter {
		t.Fatalf("retargeting rewrote the wrapper:\nbefore:\n%s\nafter:\n%s", resp.WrapperBefore, resp.WrapperAfter)
	}
	if got := aliasStoreServers(t, resp.StoreJSON)[aliasLeafName]; got != aliasStageServer {
		t.Fatalf("store server = %q, want %s\n%s", got, aliasStageServer, resp.StoreJSON)
	}
}
```

## Expected

1. Ping succeeds (exit 0) using local config + explicit `--server`.
2. `RemoteConfigBefore` and `RemoteConfigAfter` are byte-identical.

## Side Effects

`local-agent-config.json` may be read; `remote-agent-config.json` unchanged.

## Errors

- Remote sentinel file altered or deleted.

## Exit Code

0.

```go
import (
	"bytes"
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("ping failed exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}
	if !strings.Contains(resp.Stdout, "Status: ok") {
		t.Fatalf("expected successful ping; stdout:\n%s", resp.Stdout)
	}
	if !bytes.Equal(resp.RemoteConfigBefore, resp.RemoteConfigAfter) {
		t.Fatalf("remote-agent-config.json changed:\nbefore=%q\nafter=%q",
			resp.RemoteConfigBefore, resp.RemoteConfigAfter)
	}
}
```
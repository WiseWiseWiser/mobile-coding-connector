## Expected Output

```
<contains>
local-agent-only
</contains>
```

## Expected

1. Exit code is non-zero.
2. Output clearly says the command is local-agent-only or unsupported for remote-agent.
3. The local credential token is not printed.
4. `remote-agent-config.json` is byte-identical before and after the command.

## Side Effects

No local credential import; no remote-agent config mutation.

## Errors

- Command succeeds under remote-agent.
- Raw local credential appears in output or config.
- Remote config is rewritten.

## Exit Code

Non-zero.

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
	if resp.ExitCode == 0 {
		t.Fatalf("expected remote-agent to reject auth import-local; combined:\n%s", resp.Combined)
	}
	lower := strings.ToLower(resp.Combined)
	if !strings.Contains(lower, "local-agent-only") && !strings.Contains(lower, "unsupported") {
		t.Fatalf("expected local-agent-only/unsupported error; combined:\n%s", resp.Combined)
	}
	if strings.Contains(resp.Combined, "remote-must-not-import-token") {
		t.Fatalf("remote-agent leaked local credential; combined:\n%s", resp.Combined)
	}
	if !bytes.Equal(resp.RemoteConfigBefore, resp.RemoteConfigAfter) {
		t.Fatalf("remote-agent-config.json changed:\nbefore=%s\nafter=%s", resp.RemoteConfigBefore, resp.RemoteConfigAfter)
	}
	if strings.Contains(string(resp.RemoteConfigAfter), "remote-must-not-import-token") {
		t.Fatalf("remote-agent imported local credential into config:\n%s", resp.RemoteConfigAfter)
	}
}
```

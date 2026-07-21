## Expected

1. Exit 0.
2. Stdout matches local seed (full token); not remote sentinel.
3. No Config UI.

## Side Effects

None.

## Errors

Wrong file content or redacted token.

## Exit Code

0.

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	assertExitZero(t, resp)
	assertNoConfigUI(t, resp)
	assertPrettyConfigMatches(t, resp.Stdout, sampleLocalConfig())
	if strings.Contains(resp.Stdout, "remote-only-token") {
		t.Fatalf("leaked remote config into local --show; stdout:\n%s", resp.Stdout)
	}
}
```

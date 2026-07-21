## Expected

1. Exit 0.
2. Pretty empty-ish JSON (not the remote sentinel domains/tokens).
3. No Config UI.

## Side Effects

Does not rewrite remote sentinel into stdout.

## Errors

Stdout contains remote-only token or non-empty domains from sentinel.

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
	assertPrettyEmptyishConfigJSON(t, resp.Stdout)
	if strings.Contains(resp.Stdout, "remote-only-token") ||
		strings.Contains(resp.Stdout, "should-not-be-read.example.com") {
		t.Fatalf("local --show must not read remote-agent-config.json; stdout:\n%s", resp.Stdout)
	}
}
```

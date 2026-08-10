## Expected

1. No error.
2. Output args start with BaseArgs.
3. Contains `--event-bus-publish-port` and `30001`.
4. Contains `--event-bus-publish-token` and `tok-abc`.
5. Does not contain `--no-event-bus-publish`.

## Errors

- Missing flags for non-default port/token.

```go
import (
	"strings"
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, d *session.Doctest, req *Request, resp *Response, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	joined := strings.Join(resp.ArgsOut, " ")
	if len(resp.ArgsOut) < len(req.BaseArgs) {
		t.Fatalf("args too short: %v", resp.ArgsOut)
	}
	for i, a := range req.BaseArgs {
		if resp.ArgsOut[i] != a {
			t.Fatalf("ArgsOut[%d]=%q, want base %q; full=%v", i, resp.ArgsOut[i], a, resp.ArgsOut)
		}
	}
	if !containsPair(resp.ArgsOut, "--event-bus-publish-port", "30001") {
		t.Fatalf("missing --event-bus-publish-port 30001 in %v", resp.ArgsOut)
	}
	if !containsPair(resp.ArgsOut, "--event-bus-publish-token", "tok-abc") {
		t.Fatalf("missing --event-bus-publish-token tok-abc in %v", resp.ArgsOut)
	}
	if strings.Contains(joined, "--no-event-bus-publish") {
		t.Fatalf("unexpected --no-event-bus-publish in %v", resp.ArgsOut)
	}
}

func containsPair(args []string, flag, val string) bool {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == flag && args[i+1] == val {
			return true
		}
	}
	// also accept flag=value form
	want := flag + "=" + val
	for _, a := range args {
		if a == want {
			return true
		}
	}
	return false
}
```

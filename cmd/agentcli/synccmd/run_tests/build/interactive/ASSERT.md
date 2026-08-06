## Expected

1. Harness err nil; `BuildErr` empty.
2. Argv contains profile `remote-agent-mad-max`.
3. Argv does **not** contain `-batch`.
4. Env includes `UNISONLOCALHOSTNAME=remote-agent-mad-max`.

## Side Effects

- None.

## Errors

- None.

## Exit Code

- Nil builder error.

```go
import (
	"strings"
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, d *session.Doctest, req *Request, resp *Response, err error) {
	t.Helper()
	_ = d
	_ = req
	if err != nil {
		t.Fatalf("harness Run returned unexpected error: %v", err)
	}
	if resp.BuildErr != "" {
		t.Fatalf("BuildUnisonCmd error: %s", resp.BuildErr)
	}
	joined := argvJoined(resp.Argv)
	if !argvHasToken(resp.Argv, "remote-agent-mad-max") && !strings.Contains(joined, "remote-agent-mad-max") {
		t.Fatalf("argv missing profile remote-agent-mad-max; got %v", resp.Argv)
	}
	if argvHasToken(resp.Argv, "-batch") {
		t.Fatalf("interactive build must omit -batch; got %v", resp.Argv)
	}
	if !envHas(resp.Env, "UNISONLOCALHOSTNAME", "remote-agent-mad-max") {
		t.Fatalf("env missing UNISONLOCALHOSTNAME=remote-agent-mad-max; env=%v", resp.Env)
	}
}
```

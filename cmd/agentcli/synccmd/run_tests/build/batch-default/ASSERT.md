## Expected

1. Harness err nil; `BuildErr` empty.
2. Argv non-empty; `argv[0]` is LocalUnisonPath (fake-unison-bin).
3. Argv contains profile token `remote-agent-mad-max`.
4. Argv contains `-batch` (non-interactive + pair.Batch).
5. Env includes `UNISONLOCALHOSTNAME=remote-agent-mad-max`.

## Side Effects

- No Exec; no state file required.

## Errors

- None.

## Exit Code

- Nil builder error.

```go
import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, d *session.Doctest, req *Request, resp *Response, err error) {
	t.Helper()
	_ = d
	if err != nil {
		t.Fatalf("harness Run returned unexpected error: %v", err)
	}
	if resp.BuildErr != "" {
		t.Fatalf("BuildUnisonCmd error: %s", resp.BuildErr)
	}
	if len(resp.Argv) == 0 {
		t.Fatal("expected non-empty argv from BuildUnisonCmd")
	}
	wantBin := req.LocalUnisonPath
	if wantBin == "" {
		wantBin = "unison"
	}
	// argv[0] is binary, or full argv scanned for binary path.
	joined := argvJoined(resp.Argv)
	if resp.Argv[0] != wantBin && !strings.Contains(joined, filepath.Base(wantBin)) {
		t.Fatalf("argv binary: got %v want first=%q", resp.Argv, wantBin)
	}
	if !argvHasToken(resp.Argv, "remote-agent-mad-max") && !strings.Contains(joined, "remote-agent-mad-max") {
		t.Fatalf("argv missing profile remote-agent-mad-max; got %v", resp.Argv)
	}
	if !argvHasToken(resp.Argv, "-batch") {
		t.Fatalf("non-interactive batch build should include -batch; got %v", resp.Argv)
	}
	if !envHas(resp.Env, "UNISONLOCALHOSTNAME", "remote-agent-mad-max") {
		t.Fatalf("env missing UNISONLOCALHOSTNAME=remote-agent-mad-max; env=%v", resp.Env)
	}
}
```

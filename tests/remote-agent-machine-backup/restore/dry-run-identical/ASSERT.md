## Expected Output

Stream phase prints `skip (identical):` lines. Summary phase may follow with
`dry-run: machine restore plan` and entry counts.

```
...stream lines...
skip (identical): .bashrc
...summary...
dry-run: machine restore plan
```

## Expected

1. Exit code 0.
2. Combined output contains `skip (identical):` for at least `.bashrc`.
3. Combined output contains `dry-run: machine restore plan`.
4. Server `.bashrc` content unchanged from seed (`export FAKE=1`).
5. No restore writes (identical content remains).

## Side Effects

None (dry-run).

## Errors

- Missing skip lines for identical paths.
- Missing restore plan summary.
- Server files modified during dry-run.

## Exit Code

0.

```go
import (
	"strings"
	"testing"

	"github.com/xhd2015/doctest/assert"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}

// Relaxed v2: require skip line; allow stream lines before and summary after.
assert.Output(t, resp.Combined, `---
version: 2
---
...0 lines omitted...
skip (identical): .bashrc
...7 lines omitted...`)

	if !strings.Contains(resp.Combined, "skip (identical):") {
		t.Fatalf("expected skip (identical) lines; got:\n%s", resp.Combined)
	}
	if !strings.Contains(resp.Combined, "dry-run: machine restore plan") {
		t.Fatalf("missing restore plan summary; got:\n%s", resp.Combined)
	}

	got := readServerFile(t, resp.ServerHome, ".bashrc")
	want := "export FAKE=1\n"
	if got != want {
		t.Fatalf(".bashrc changed during dry-run: got %q want %q", got, want)
	}
}
```
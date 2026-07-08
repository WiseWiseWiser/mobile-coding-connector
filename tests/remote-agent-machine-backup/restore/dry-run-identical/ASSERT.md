## Expected Output

All-identical dry-run: **CLASSIFYING** may emit one representative `skip (identical):`
line (shortcut). No APPLYING section. Summary is `dry-run: machine restore plan`.

```
CLASSIFYING:
skip (identical): .bashrc

dry-run: machine restore plan
  ...
  TOTAL: ... entries
```

## Expected

1. Exit code 0.
2. Combined output has `CLASSIFYING:` and no `APPLYING:`.
3. CLASSIFYING contains `skip (identical):` for at least `.bashrc`.
4. Combined output contains `dry-run: machine restore plan`.
5. Server `.bashrc` content unchanged from seed (`export FAKE=1`).
6. No restore writes (identical content remains).

## Side Effects

None (dry-run).

## Errors

- Missing CLASSIFYING section or skip lines.
- Unexpected APPLYING section.
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

	combined := resp.Combined
	assertRestoreStreamSections(t, combined, false)

	assert.Output(t, combined, `---
version: 2
---
CLASSIFYING:
...0 lines omitted...
skip (identical): .bashrc
...7 lines omitted...`)

	if !strings.Contains(combined, "dry-run: machine restore plan") {
		t.Fatalf("missing restore plan summary; got:\n%s", combined)
	}

	got := readServerFile(t, resp.ServerHome, ".bashrc")
	want := "export FAKE=1\n"
	if got != want {
		t.Fatalf(".bashrc changed during dry-run: got %q want %q", got, want)
	}
}
```
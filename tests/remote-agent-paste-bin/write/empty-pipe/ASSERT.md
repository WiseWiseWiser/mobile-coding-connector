## Expected Output

stderr:

```
saved 0 bytes
```

stdout:

```
(silent)
```

## Expected

1. Exit code 0.
2. Stderr is exactly `saved 0 bytes` (green count); no preview block.
3. Stdout silent.
4. Scratch API `content` is empty string.

## Side Effects

Prior stale scratch overwritten with empty content.

## Errors

- Preview block after zero-byte save.
- Stdout echo for zero-byte write.
- Stale content remains in API.

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

	assert.Output(t, resp.Stderr, `---
version: 2
---
<ansi-color green>saved 0 bytes</ansi-color>
`)
	assertStdoutEmpty(t, resp.Stdout)
	assertScratchContentEmpty(t, resp.ScratchAfter.Content)

	if strings.Contains(resp.Stderr, "preview:") {
		t.Fatalf("zero-byte write must not print preview; stderr:\n%s", resp.Stderr)
	}
}
```
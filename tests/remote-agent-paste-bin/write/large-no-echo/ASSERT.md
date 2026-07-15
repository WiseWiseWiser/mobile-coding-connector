## Expected Output

stderr (shape):

```
saved 5000 bytes
preview:
  xxxxx... (200 bytes max)
  … (+4800 more bytes)
```

stdout:

```
(silent)
```

## Expected

1. Exit code 0.
2. Stderr contains `saved 5000 bytes`, `preview:`, and truncation hint `… (+4800 more bytes)`.
3. Stdout is silent (payload > 4096).
4. Scratch API stores full 5000-byte UTF-8 payload as plain text.

## Side Effects

`scratch.json` content is 5000 `x` bytes.

## Errors

- Full 5000 bytes echoed on stdout.
- Missing truncation hint when preview shorter than payload.
- Partial write to scratch API.

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

	assertStdoutEmpty(t, resp.Stdout)

	assert.Output(t, resp.Stderr, `---
version: 2
---
<ansi-color green>saved 5000 bytes</ansi-color>
<ansi-color gray>preview:</ansi-color>
  xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
<ansi-color gray>… (+4800 more bytes)</ansi-color>
`)

	want := string(repeatByte('x', largePayloadSize))
	assertScratchContentExact(t, resp.ScratchAfter.Content, want)

	if strings.Count(resp.Stderr, "x") < 10 {
		t.Fatalf("stderr preview too short; stderr:\n%s", resp.Stderr)
	}
}
```
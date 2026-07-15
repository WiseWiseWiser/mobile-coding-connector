## Expected Output

stderr:

```
saved 2 bytes
preview:
  hi
```

stdout:

```
(silent)
```

## Expected

1. Exit code 0.
2. Stderr still shows `saved 2 bytes` and preview.
3. Stdout silent despite small payload (quiet suppresses echo).
4. Scratch API updated to `hi`.

## Side Effects

Scratch content is `hi`.

## Errors

- Stdout echoes `hi` when `-q` is set.
- Missing stderr save summary.

## Exit Code

0.

```go
import (
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
<ansi-color green>saved 2 bytes</ansi-color>
<ansi-color gray>preview:</ansi-color>
  hi
`)
	assertStdoutEmpty(t, resp.Stdout)
	assertScratchContentExact(t, resp.ScratchAfter.Content, smallEchoPayload)
}
```
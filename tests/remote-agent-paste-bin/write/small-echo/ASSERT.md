## Expected Output

stderr:

```
saved 2 bytes
preview:
  hi
```

stdout:

```
hi
```

## Expected

1. Exit code 0.
2. Stderr contains green `saved 2 bytes` and a preview of `hi`.
3. Stdout echoes `hi` byte-for-byte (N ≤ 4096 echo threshold).
4. Scratch API `content` is `hi`.

## Side Effects

`scratch.json` updated with UTF-8 content `hi`.

## Errors

- Missing `saved 2 bytes` on stderr.
- Missing stdout echo for small payload.
- Scratch API not updated.

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
	assertStdoutExactBytes(t, resp.Stdout, []byte(smallEchoPayload))
	assertScratchContentExact(t, resp.ScratchAfter.Content, smallEchoPayload)
}
```
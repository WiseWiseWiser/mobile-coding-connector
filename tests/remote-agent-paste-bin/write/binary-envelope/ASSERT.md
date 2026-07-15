## Expected

1. Exit code 0.
2. Stderr contains `saved 12 bytes` (length of `before\x00after`).
3. Scratch API `content` uses `paste-bin:b64:` prefix envelope.
4. Decoded envelope bytes equal piped stdin payload.
5. Stdout echoes raw bytes when N ≤ 4096 (includes NUL).

## Side Effects

Scratch stores base64 envelope instead of raw invalid UTF-8.

## Errors

- Raw NUL stored without envelope.
- Decoded bytes differ from stdin.
- Missing stderr save summary.

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

	want := []byte(binaryEnvelopePayload)
	assert.Output(t, resp.Stderr, `---
version: 2
---
<ansi-color green>saved 12 bytes</ansi-color>
<ansi-color gray>preview:</ansi-color>
  before\x00after
`)
	assertStdoutExactBytes(t, resp.Stdout, want)
	assertScratchB64Envelope(t, resp.ScratchAfter.Content, want)

	if strings.Contains(resp.ScratchAfter.Content, "\x00") {
		t.Fatalf("API content must use b64 envelope, not raw NUL; got %q", resp.ScratchAfter.Content)
	}
}
```
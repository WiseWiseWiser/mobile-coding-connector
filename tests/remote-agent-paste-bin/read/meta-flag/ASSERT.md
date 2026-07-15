## Expected Output

stderr:

```
updated at 2026-07-14T08:30:00Z
```

stdout:

```
line1
line2
emoji🎉
```

## Expected

1. Exit code 0.
2. Stderr contains gray `updated at 2026-07-14T08:30:00Z` before content emission.
3. Stdout equals seeded content bytes.

## Side Effects

Scratch API unchanged.

## Errors

- Missing `updated at` line on stderr.
- Timestamp on stdout instead of stderr.
- Content mismatch on stdout.

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
<ansi-color gray>updated at 2026-07-14T08:30:00Z</ansi-color>
`)
	assertStdoutExactBytes(t, resp.Stdout, []byte(seededUTF8Content))
	assertScratchContentExact(t, resp.ScratchAfter.Content, seededUTF8Content)

	if strings.Contains(resp.Stdout, "updated at") {
		t.Fatalf("--meta timestamp must be on stderr, not stdout; stdout:\n%s", resp.Stdout)
	}
}
```
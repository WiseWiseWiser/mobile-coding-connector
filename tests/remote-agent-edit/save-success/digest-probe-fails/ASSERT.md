## Expected Output

stdout:

```
Downloading notes.md -> __STAGED__ (4 B)
Opening __EDITOR__ __STAGED__
Saved __REMOTE__ (4 B, md5 __MD5__)
```

stderr:

```
warning: remote digest unavailable (500 Internal Server Error: probe unavailable); downloading the full copy
```

## Expected

1. Exit code 0 — the failed probe is not fatal.
2. The warning names the probe failure on stderr, and it is the **only**
   warning: the fallback check must not repeat the reason.
3. A `Downloading` line, exactly one `/api/files/download` request, and two
   `/api/files/check` requests (digest probe, then digest-free fallback).
4. Remote `notes.md` contains `new\n`.

## Side Effects

- Identical staged copy re-downloaded (digest probe failed).

## Errors

- Exit 1 because the digest probe failed.
- No fallback request, or a `Skipped download` claim without a digest.

## Exit Code

0.

```go
import (
	"strings"
	"testing"

	"github.com/xhd2015/doctest/session"

	"github.com/xhd2015/doctest/assert"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	assertExit(t, resp, 0)

	combinedHasAll(t, resp.Combined,
		"remote digest unavailable",
		"downloading the full copy",
		"Downloading",
		"Saved "+resp.RemotePath,
	)
	combinedHasNone(t, resp.Stdout, "Skipped download", "warning:", "Error")
	if got := strings.Count(resp.Stderr, "warning:"); got != 1 {
		t.Fatalf("stderr warnings = %d, want 1 (one reason per run);\nstderr:\n%s", got, resp.Stderr)
	}

	if !strings.Contains(resp.Stderr, "warning: remote digest unavailable") {
		t.Fatalf("warning should be on stderr;\nstderr:\n%s", resp.Stderr)
	}
	assertRequestCount(t, resp, "/api/files/check", 2)
	assertRequestCount(t, resp, "/api/files/download", 1)
	assertRemoteContent(t, resp, "notes.md", "new\n")

	assert.Output(t, resp.Stdout, `---
version: 2
__STAGED__: type=string
__EDITOR__: type=string
__REMOTE__: type=string
__MD5__: type=string
---
Downloading notes.md -> __STAGED__ (4 B)
Opening __EDITOR__ __STAGED__
Saved __REMOTE__ (4 B, md5 __MD5__)
`)
}
```

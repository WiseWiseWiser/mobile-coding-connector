## Expected

1. Exit code 0.
2. `[n/4]` spine with `would:` under pack/upload/apply.
3. Stdout product `would: upload …`.
4. Server tree unchanged.

## Exit Code

0.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}

	assertStdoutEndsWithNewline(t, resp.Stdout)
	combinedHasAll(t, resp.Combined,
		"[1/4] resolve",
		"[2/4] pack", "would: pack tar.xz",
		"[3/4] upload", "would: upload archive",
		"[4/4] apply", "would: apply extract/merge",
		"would: upload",
	)
	combinedHasNone(t, resp.Combined, "\nuploaded ")

	assertTreeSnapshotUnchanged(t, "serverHome", resp.ServerFilesBeforeCLI, resp.ServerFilesAfterCLI)
	assertServerPathMissing(t, resp.ServerHome, "uploads/mirror/a.txt")
	assertServerPathMissing(t, resp.ServerHome, "uploads/mirror/sub/b.txt")
}
```

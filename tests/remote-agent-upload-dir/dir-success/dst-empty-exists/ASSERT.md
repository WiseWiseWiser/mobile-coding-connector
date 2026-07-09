## Expected

1. Exit code 0.
2. Stdout reports `2 files` for the directory upload.
3. `uploads/mirror/a.txt` and `uploads/mirror/sub/b.txt` exist with correct content.

## Side Effects

- Empty `uploads/mirror` populated with mirrored files.

## Errors

- Guard rejects empty destination.
- Partial or missing mirror tree.

## Exit Code

0.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}

	assertStdoutEndsWithNewline(t, resp.Stdout)
	combinedHasAll(t, resp.Combined, "2 files", "Upload complete", "uploads/mirror")
	assertServerFileContent(t, resp.ServerHome, "uploads/mirror/a.txt", "alpha\n")
	assertServerFileContent(t, resp.ServerHome, "uploads/mirror/sub/b.txt", "bravo\n")
}
```
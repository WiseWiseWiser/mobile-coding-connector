## Expected

1. Exit code 0.
2. Stdout references resolved directory `parent/proj`.
3. `parent/proj/file.txt` exists with `proj payload\n`.

## Side Effects

- Files land under `parent/proj/`, not directly under `parent/`.

## Errors

- `parent/file.txt` created instead of `parent/proj/file.txt`.
- Basename not appended for trailing-slash destination.

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
	combinedHasAll(t, resp.Combined, "parent/proj", "1 files", "Upload complete")
	assertServerPathMissing(t, resp.ServerHome, "parent/file.txt")
	assertServerFileContent(t, resp.ServerHome, "parent/proj/file.txt", "proj payload\n")
}
```
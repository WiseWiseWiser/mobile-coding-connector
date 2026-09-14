## Expected

1. Non-zero exit.
2. Combined output mentions destination must be missing or empty (or not a directory).
3. `uploads/mirror` file content unchanged.
4. No directory tree created at that path.

## Side Effects

None.

## Errors

- Exit 0 with directory created beside/over the file.
- Seeded file content mutated.

## Exit Code

Non-zero.

```go
import (
	"os"
	"strings"
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode == 0 {
		t.Fatalf("expected failure; combined:\n%s", resp.Combined)
	}

	lower := strings.ToLower(resp.Combined)
	if !strings.Contains(lower, "is a file") && !strings.Contains(lower, "refusing") && !strings.Contains(lower, "not a directory") {
		t.Fatalf("expected file-destination error; combined:\n%s", resp.Combined)
	}

	full := serverFilePath(resp.ServerHome, "uploads/mirror")
	info, err := os.Stat(full)
	if err != nil {
		t.Fatalf("stat %s: %v", full, err)
	}
	if info.IsDir() {
		t.Fatalf("%s became a directory", full)
	}
	assertServerFileContent(t, resp.ServerHome, "uploads/mirror", seedRemoteFileContent)
	assertServerPathMissing(t, resp.ServerHome, "uploads/mirror/incoming.txt")
}
```
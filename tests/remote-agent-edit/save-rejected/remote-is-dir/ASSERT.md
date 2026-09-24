## Expected Output

stderr:

```
Error: remote path __REMOTE__ is a directory, not a file
```

## Expected

1. Exit code 1.
2. The error names the resolved remote path and says it is a directory.
3. No `Downloading` line, no staged copy, no editor run.

## Side Effects

- None: the remote directory is untouched.

## Errors

- Download attempted for a directory (server 400 instead of a clear CLI error).
- A staged file created for a directory target.

## Exit Code

1.

```go
import (
	"os"
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	assertExit(t, resp, 1)

	combinedHasAll(t, resp.Combined,
		"Error: remote path "+resp.RemotePath+" is a directory, not a file",
	)
	combinedHasNone(t, resp.Combined, "Downloading", "Opening ", "Saved ")

	if _, statErr := os.Stat(resp.StagedPath); !os.IsNotExist(statErr) {
		t.Fatalf("staged path %s should not exist, stat err = %v", resp.StagedPath, statErr)
	}
}
```

## Expected

1. Non-zero exit.
2. Combined output states the directory is not a git repository (matches server
   `handleRepoOp` message shape).

## Side Effects

No `On branch` status output.

## Errors

- Exit 0 with plausible `git status` output.

## Exit Code

Non-zero.

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode == 0 {
		t.Fatalf("expected failure; combined:\n%s", resp.Combined)
	}
	oneOf := []string{"not a git repository", "dir is not a git repository"}
	for _, want := range oneOf {
		if strings.Contains(resp.Combined, want) {
			oneOf = nil
			break
		}
	}
	if oneOf != nil {
		t.Fatalf("expected one of %v in combined:\n%s", oneOf, resp.Combined)
	}
}
```
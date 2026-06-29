## Expected

1. Non-zero exit.
2. Denial error in combined output.

## Side Effects

No `origin` remote configured locally.

## Errors

- Exit 0 with `git remote -v` showing origin.

## Exit Code

Non-zero.

```go
import (
	"os/exec"
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode == 0 {
		t.Fatalf("expected denial; combined:\n%s", resp.Combined)
	}
	oneOf := []string{"not allowed", "denied", "remote add"}
	for _, want := range oneOf {
		if strings.Contains(resp.Combined, want) {
			oneOf = nil
			break
		}
	}
	if oneOf != nil {
		t.Fatalf("expected one of %v in combined:\n%s", oneOf, resp.Combined)
	}

	cmd := exec.Command("git", "remote")
	cmd.Dir = resp.RepoDir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git remote: %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("origin remote must not exist; remotes:\n%s", out)
	}
}
```
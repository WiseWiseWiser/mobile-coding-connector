## Expected

1. Non-zero exit.
2. Denial error in combined output.

## Side Effects

Branch `extra` still exists in the local repo.

## Errors

- Exit 0 and branch deleted.

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
	oneOf := []string{"not allowed", "denied", "branch -D"}
	for _, want := range oneOf {
		if strings.Contains(resp.Combined, want) {
			oneOf = nil
			break
		}
	}
	if oneOf != nil {
		t.Fatalf("expected one of %v in combined:\n%s", oneOf, resp.Combined)
	}

	cmd := exec.Command("git", "branch", "--list", "extra")
	cmd.Dir = resp.RepoDir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git branch --list: %v", err)
	}
	if len(out) == 0 {
		t.Fatal("branch extra should still exist after denied -D")
	}
}
```
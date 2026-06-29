## Expected

1. Non-zero exit.
2. Error message indicates mutating `config` is not permitted.

## Side Effects

Local repo config unchanged (`core.commentChar` still default).

## Errors

- Exit 0.

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
	oneOf := []string{"not allowed", "denied", "config --set"}
	for _, want := range oneOf {
		if strings.Contains(resp.Combined, want) {
			oneOf = nil
			break
		}
	}
	if oneOf != nil {
		t.Fatalf("expected one of %v in combined:\n%s", oneOf, resp.Combined)
	}

	cmd := exec.Command("git", "config", "--get", "core.commentChar")
	cmd.Dir = resp.RepoDir
	out, _ := cmd.Output()
	if string(out) == "%\n" {
		t.Fatal("config --set must not have been applied")
	}
}
```
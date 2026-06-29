## Expected

1. Non-zero exit.
2. Combined output indicates the subcommand is not allowed (CLI or server).

## Side Effects

`new.txt` remains unstaged in the worktree on disk.

## Errors

- Exit 0 with empty diff after implied staging.

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
	oneOf := []string{"not allowed", "denied", "unknown git subcommand: add"}
	for _, want := range oneOf {
		if strings.Contains(resp.Combined, want) {
			oneOf = nil
			break
		}
	}
	if oneOf != nil {
		t.Fatalf("expected one of %v in combined:\n%s", oneOf, resp.Combined)
	}

	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = resp.RepoDir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("local git status: %v", err)
	}
	if len(out) == 0 || !containsLine(string(out), "?? new.txt") {
		t.Fatalf("expected unstaged new.txt after denied add; porcelain:\n%s", out)
	}
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func containsLine(blob, line string) bool {
	for _, l := range splitLines(blob) {
		if l == line {
			return true
		}
	}
	return false
}
```
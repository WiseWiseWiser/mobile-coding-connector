## Expected

1. Exit 0.
2. Stdout mentions `tracked.txt` as modified and `untracked.txt` as untracked.

## Side Effects

None.

## Errors

- Clean worktree message.

## Exit Code

0.

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit %d; stdout:\n%s", resp.ExitCode, resp.Stdout)
	}
	out := resp.Stdout
	for _, want := range []string{"tracked.txt", "untracked.txt"} {
		if !strings.Contains(out, want) {
			t.Fatalf("stdout missing %q;\n%s", want, out)
		}
	}
	modified := []string{"modified:", "Changes not staged"}
	for _, want := range modified {
		if strings.Contains(out, want) {
			modified = nil
			break
		}
	}
	if modified != nil {
		t.Fatalf("stdout missing modified indicator; want one of %v;\n%s", modified, out)
	}
	untracked := []string{"Untracked files", "untracked"}
	for _, want := range untracked {
		if strings.Contains(out, want) {
			untracked = nil
			break
		}
	}
	if untracked != nil {
		t.Fatalf("stdout missing untracked indicator; want one of %v;\n%s", untracked, out)
	}
}
```
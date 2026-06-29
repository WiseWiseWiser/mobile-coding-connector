## Expected

1. Non-zero exit code.
2. Stderr (or combined) explains that `-C <dir>` is required for repo-scoped git commands.

## Side Effects

Server may start but must not receive a successful `/api/remote-agent/git/run` for `status`.

## Errors

- Command succeeds or prints git status without `-C`.

## Exit Code

Non-zero (typically 1).

```go
import (
	"strings"
	"testing"

	"github.com/xhd2015/doctest/assert"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode == 0 {
		t.Fatalf("expected failure; combined:\n%s", resp.Combined)
	}
	assert.Output(t, resp.Combined, `<contains>
Error:
</contains>`)
	oneOf := []string{
		"requires '-C <dir>'",
		"requires '-C <dir>' between 'git' and",
		"unknown git subcommand: status",
	}
	for _, want := range oneOf {
		if strings.Contains(resp.Combined, want) {
			oneOf = nil
			break
		}
	}
	if oneOf != nil {
		t.Fatalf("expected one of %v in combined:\n%s", oneOf, resp.Combined)
	}
	lower := strings.ToLower(resp.Combined)
	if strings.Contains(lower, "on branch") {
		t.Fatalf("must not run git status without -C; combined:\n%s", resp.Combined)
	}
}
```
## Expected

1. Non-zero exit.
2. Combined output contains `unknown git subcommand` and `frobnicate`.

## Side Effects

No git output from a real `frobnicate` command.

## Errors

- HTTP/API errors only (subcommand should be rejected locally).

## Exit Code

Non-zero.

```go
import (
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
Error: unknown git subcommand: frobnicate
</contains>`)
}
```
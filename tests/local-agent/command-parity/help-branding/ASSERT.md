## Expected Output

```
<contains>
Usage: local-agent
--port
23712
</contains>
```

## Expected

1. Exit code 0.
2. Help text uses `local-agent` branding, documents `--port`, and mentions default port `23712`.

## Side Effects

None.

## Errors

- Missing local-only flags or wrong binary name in help.

## Exit Code

0.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/assert"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("help should exit 0, got %d; stderr:\n%s", resp.ExitCode, resp.Stderr)
	}
	assert.Output(t, resp.Stdout, `
<contains>
Usage: local-agent
<start-with>
  --port
</start-with>
23712
</contains>`)
}
```
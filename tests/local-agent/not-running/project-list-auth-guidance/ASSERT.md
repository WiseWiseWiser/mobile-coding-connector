## Expected Output

```
<contains>
unauthorized
local-agent config
~/.ai-critic
server-credentials
</contains>
```

## Expected

1. Exit code is non-zero.
2. Output indicates an unauthorized/auth failure from the server.
3. Output includes a CLI authorization hint mentioning `local-agent config`.
4. Output includes a local-only hint mentioning `~/.ai-critic` and the local server credentials file.
5. Output does not print any successful project list.

## Side Effects

Server rejects the request; no config file is expected to be written by this command.

## Errors

- Missing actionable auth guidance.
- Missing local credential-file guidance.
- Successful exit with an invalid token.

## Exit Code

Non-zero.

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
		t.Fatalf("expected project list auth failure; stdout:\n%s\nstderr:\n%s", resp.Stdout, resp.Stderr)
	}
	lower := strings.ToLower(resp.Combined)
	if !strings.Contains(lower, "unauthorized") && !strings.Contains(lower, "auth") {
		t.Fatalf("expected unauthorized/auth failure; combined:\n%s", resp.Combined)
	}
	assert.Output(t, resp.Combined, `
<contains>
local-agent config
~/.ai-critic
server-credentials
</contains>`)
	if strings.Contains(resp.Stdout, "Project:") {
		t.Fatalf("project list should not render projects after auth failure; stdout:\n%s", resp.Stdout)
	}
}
```

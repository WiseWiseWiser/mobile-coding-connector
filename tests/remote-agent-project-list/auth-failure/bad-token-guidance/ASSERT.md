## Expected Output

```
<contains>
unauthorized
remote-agent config
</contains>
```

## Expected

1. Exit code is non-zero.
2. Output indicates unauthorized/auth failure.
3. Output includes an actionable hint mentioning `remote-agent config`.
4. Output does not include local-only `~/.ai-critic` or `server-credentials` guidance.

## Side Effects

Server rejects the request; remote config is not updated by this command.

## Errors

- Missing profile-specific authorization hint.
- Local credential-file guidance appears for the remote profile.

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
		t.Fatalf("expected auth failure; stdout:\n%s\nstderr:\n%s", resp.Stdout, resp.Stderr)
	}
	lower := strings.ToLower(resp.Combined)
	if !strings.Contains(lower, "unauthorized") && !strings.Contains(lower, "auth") {
		t.Fatalf("expected unauthorized/auth failure; combined:\n%s", resp.Combined)
	}
	assert.Output(t, resp.Combined, `
<contains>
remote-agent config
</contains>`)
	if strings.Contains(resp.Combined, "~/.ai-critic") || strings.Contains(resp.Combined, "server-credentials") {
		t.Fatalf("remote-agent must not print local credential-file guidance; combined:\n%s", resp.Combined)
	}
}
```

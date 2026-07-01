# Scenario

**Feature**: agent-pro `ShellQuote` replaces deleted `server/terminal` quoting

```
# shell quoting
dot-pkgs ptywrap.ShellQuote(input) -> POSIX-safe token
sh -c round-trip -> original value preserved
```

## Preconditions

- `github.com/xhd2015/agent-pro/pkgs/shell` exports `ShellQuote`.
- `sh` is available in PATH.

## Steps

1. Leaf sets `req.Phase` and optional quote inputs.
2. Harness calls `shell.ShellQuote` and validates via `sh -c`.

```go
import (
	"os/exec"
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh not in PATH")
	}
	return nil
}
```
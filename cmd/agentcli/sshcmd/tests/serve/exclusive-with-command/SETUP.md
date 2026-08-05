# Scenario

**Feature**: `--serve` cannot combine with a remote command

```
# exclusive error
operator -> sshcmd ["--serve", "ls"]
  -> error; ServeStarter not called; Runner not called
```

## Preconditions

- Args: `["--serve", "ls"]`.
- Product must reject exclusive combination (Parse and/or Run error).

## Steps

1. Set Args to serve plus a command token.
2. Assert error text mentions exclusive/`--serve`/command; zero Start/Runner calls.

## Context

- OpenSSH-shaped CLI: serve is not mixed with remote argv.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Args = []string{"--serve", "ls"}
	return nil
}
```

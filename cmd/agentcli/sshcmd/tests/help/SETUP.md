# Scenario

**Feature**: `remote-agent ssh` help mode

```
# help flags short-circuit before session or serve
operator -> sshcmd.Parse/Run(["--help"] | ["-h"])
  -> ModeHelp; Usage on Stdout; nil error; no Store/Serve/Runner
```

## Preconditions

- Mode under test is **help** (not serve, login, or command).
- Args are a single help flag set by the leaf (`--help` or `-h`).
- Session store remains empty; help must not require an active tunnel.

## Steps

1. Grouping marks help mode; leaves set the help flag form.
2. Run parses and executes help; captures stdout.

## Context

- Help is exclusive of other modes for this branch.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	// Leaves set Args to --help or -h.
	req.Session = nil
	return nil
}
```

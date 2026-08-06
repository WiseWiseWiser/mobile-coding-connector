# Scenario

**Feature**: CLI doctor dispatches RunCLI with injectable probes

```
operator -> RunCLI([unison doctor <name>], CLIOpts{probes}) -> stdout + err
```

## Preconditions

- Mode `cli`.
- Leaves seed store and hooks.

## Steps

1. Force Mode cli.
2. Leaves set Args and failure injectables.

## Context

- CLI must return non-nil error when critical checks fail.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Mode = "cli"
	return nil
}
```

# Scenario

**Feature**: init without required positionals fails

```
operator -> RunCLI([unison init mad-max]) -> error (missing local/remote)
```

## Preconditions

- Args incomplete: only name, no local/remote.

## Steps

1. Set Args to `unison init mad-max` only.
2. Assert non-nil error.

## Context

- Validation error.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Args = []string{"unison", "init", "mad-max"}
	return nil
}
```

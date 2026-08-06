# Scenario

**Feature**: Help surfaces for sync unison run

```
operator -> RunCLI([unison --help]) -> usage lists run
```

## Preconditions

- Grouping for help leaves.
- No pairs required.

## Steps

1. Default Mode to `cli` when empty.
2. Leaves set Args for help.

## Context

- P3 usage extension of UnisonUsage.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	if req.Mode == "" {
		req.Mode = "cli"
	}
	return nil
}
```

# Scenario

**Feature**: unison help lists run verb

```
operator -> RunCLI([unison --help]) -> stdout contains run
```

## Preconditions

- No pairs required.
- Args: `unison --help`.

## Steps

1. Set Args to unison long-flag help.
2. Assert stdout mentions `run`.

## Context

- P3 usage extension of P1/P2 UnisonUsage.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Mode = "cli"
	req.Args = []string{"unison", "--help"}
	return nil
}
```

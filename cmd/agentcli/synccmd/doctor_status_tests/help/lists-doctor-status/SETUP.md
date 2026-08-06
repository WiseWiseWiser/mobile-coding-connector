# Scenario

**Feature**: unison help lists doctor and status verbs

```
operator -> RunCLI([unison --help]) -> stdout contains doctor and status
```

## Preconditions

- No pairs required.
- Args: `unison --help` (or bare `unison`).

## Steps

1. Set Args to unison long-flag help.
2. Assert stdout mentions doctor and status.

## Context

- P2 usage extension of P1 UnisonUsage.

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

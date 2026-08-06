# Scenario

**Feature**: unison alone prints unison help

```
operator -> RunCLI([unison]) -> unison Usage listing init/list/show/set/rm
```

## Preconditions

- Args: `["unison"]`.

## Steps

1. Set Args to unison only.
2. Assert CRUD verbs on stdout.

## Context

- Unison backend help.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Args = []string{"unison"}
	return nil
}
```

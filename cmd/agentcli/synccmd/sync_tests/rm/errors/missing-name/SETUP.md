# Scenario

**Feature**: rm without name errors

```
operator -> rm (no name) -> error
```

## Preconditions

- Args: `unison rm` only.

## Steps

1. Missing name.
2. Assert error.

## Context

- Missing name.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Args = []string{"unison", "rm"}
	return nil
}
```

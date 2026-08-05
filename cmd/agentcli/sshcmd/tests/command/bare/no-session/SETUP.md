# Scenario

**Feature**: bare command fails without active session

```
# gate on command mode
operator -> ["ls"], Store empty
  -> error no active SSH tunnel ... --serve first
  -> Runner not called
```

## Preconditions

- Args: `["ls"]`.
- Session nil.

## Steps

1. Keep bare command Args; Session nil.
2. Assert same tunnel error contract as login no-session.

## Context

- Session gate applies to all client modes (login and command).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Args = []string{"ls"}
	req.Session = nil
	return nil
}
```

# Scenario

**Feature**: Install local hook error surfaces

```
Install(Scope=local, LocalEnsure → error "brew failed")
  -> InstallErr non-empty; LocalOK false; RemoteEnsure not called
```

## Preconditions

- Scope `local`.
- LocalEnsureErr set to a distinctive message.

## Steps

1. Mode install, Scope local, LocalEnsureErr.
2. Assert non-nil Install error containing the hook message; LocalOK false.

## Context

- Scenario 5 from P4 design.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Mode = "install"
	req.Scope = "local"
	req.LocalEnsureErr = "brew failed: unison formula missing"
	return nil
}
```

# Scenario

**Feature**: Install --local success via LocalEnsure hook

```
Install(Scope=local, LocalEnsure → "2.54.0")
  -> LocalOK true; LocalVersion 2.54.0; RemoteEnsure not called; nil error
```

## Preconditions

- Scope `local`.
- FakeLocalVersion default pin `2.54.0`.
- No LocalEnsureErr.

## Steps

1. Mode install, Scope local.
2. Assert LocalEnsure called, RemoteEnsure not called, LocalOK, version, nil err.

## Context

- Scenario 2 from P4 design (library equivalent of `--local`).

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
	req.FakeLocalVersion = "2.54.0"
	return nil
}
```

# Scenario

**Feature**: CLI install defaults to both when no flags

```
RunCLI([unison install], LocalEnsure+RemoteEnsure ok)
  -> both hooks called; RunErr empty
```

## Preconditions

- Args: `unison install` only (no `--local` / `--remote` / `--both`).
- Fake versions success.

## Steps

1. Mode cli; Args bare install.
2. Assert both hooks called and RunErr empty.

## Context

- Scenario 6 from P4 design; CLI default maps to scope both.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Mode = "cli"
	req.Args = []string{"unison", "install"}
	req.FakeLocalVersion = "2.54.0"
	req.FakeRemoteVersion = "2.54.0"
	return nil
}
```

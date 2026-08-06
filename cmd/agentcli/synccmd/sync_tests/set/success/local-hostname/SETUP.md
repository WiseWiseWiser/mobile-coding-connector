# Scenario

**Feature**: set local-hostname updates pair field

```
PreArgs: init\noperator -> set --local-hostname mac-mad-max\n  -> localHostname stored
```

## Preconditions

- Focus pair mad-max.

## Steps

1. Pre-init.
2. Set hostname.
3. Assert field.

## Context

- Hostname patch (store field; profile need not embed it in P1).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.FocusPair = "mad-max"
	req.PreArgs = [][]string{
		{"unison", "init", "mad-max", req.LocalPath, req.RemotePath},
	}
	req.Args = []string{"unison", "set", "mad-max", "--local-hostname", "mac-mad-max"}
	return nil
}
```

# Scenario

**Feature**: init accepts prefer, local-hostname, remote-unison flags

```
operator -> init --prefer older --local-hostname mac-x --remote-unison /opt/unison\n  -> pair fields match flags
```

## Preconditions

- Name: `custom`.
- Flags: prefer older, local-hostname mac-x, remote-unison /opt/unison.

## Steps

1. Set Args with flags.
2. Assert stored field values.

## Context

- Custom option path.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.FocusPair = "custom"
	req.Args = []string{
		"unison", "init", "custom", req.LocalPath, req.RemotePath,
		"--prefer", "older",
		"--local-hostname", "mac-x",
		"--remote-unison", "/opt/unison",
	}
	return nil
}
```

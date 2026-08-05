# Scenario

**Feature**: `remote-agent ssh --serve` mode and exclusive rules

```
# serve mode
operator -> sshcmd ["--serve"] -> ServeStarter.Start(ProfileID)
operator -> sshcmd ["--serve", "ls"] -> exclusive error; no Start
```

## Preconditions

- Mode under test is **serve** or **serve exclusive error**.
- Leaves set whether a remote command token appears after `--serve`.
- Session gate is not used for pure serve success (Start only).

## Steps

1. Grouping marks serve-related scenarios.
2. Leaves set Args and Assert Start calls or exclusive error text.

## Context

- `--serve` cannot be combined with a remote command.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Session = nil
	return nil
}
```

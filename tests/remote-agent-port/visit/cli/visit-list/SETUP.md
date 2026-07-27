# Scenario

**Feature**: CLI port visit list (empty and with active)

```
port visit list -> empty message when no sessions
```

When sessions exist, manager list-active + detach-json already cover listing.
This leaf locks the CLI subcommand surface for empty list.

## Steps

1. Args: `port visit list` with no prior sessions.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "cli"
	enableOwnedQuick(req, true, true)
	setCLI(req, "port", "visit", "list")
	return nil
}
```

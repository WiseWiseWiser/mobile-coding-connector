# Scenario

**Feature**: a plain invocation keeps using the default domain

```
Prelude (default domain + xdev alias) -> remote-agent ping -> default server
```

## Preconditions

Same prelude as `resolve/alias-targets-alias-server`: two configured servers.

## Steps

1. `Setup` requests the prelude and runs `ping` with no `--alias`.
2. `Assert` checks the default server was hit with its own token.

## Context

MECE partner of the alias-target leaf: adding an alias must not move the
default target.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	req.Prelude = true
	setCLI(req, "ping")
	return nil
}
```

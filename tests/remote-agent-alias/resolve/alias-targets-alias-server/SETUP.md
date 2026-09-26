# Scenario

**Feature**: `--alias NAME` targets that alias's server and token

```
Prelude (default domain + xdev alias + xdev token)
  -> remote-agent --alias xdev ping -> xdev server, Bearer tok-xdev
```

## Preconditions

`Prelude` seeds the default domain (`tok-default`), the `xdev` alias, and its
token (`tok-xdev`), all through the real CLI.

## Steps

1. `Setup` requests the prelude and runs `--alias xdev ping`.
2. `Assert` checks which server was hit and with which bearer token.

## Context

This is the core of the feature: the wrapper will use exactly this flag.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	req.Prelude = true
	setCLI(req, "--alias", aliasLeafName, "ping")
	return nil
}
```

# Scenario

**Feature**: the wrapper form of `config set` writes the alias's server token

```
Prelude -> remote-agent --alias xdev config set --token tok-new
        -> config JSON: xdev server token = tok-new (default untouched)
```

## Preconditions

`Prelude` seeds the default domain, the `xdev` alias, and an initial token.

## Steps

1. `Setup` requests the prelude and runs the global `--alias` form of
   `config set --token tok-new`.
2. `Assert` checks the written config, the echoed server name, and that no HTTP
   request happened.

## Context

`xdev-agent config set --token-stdin` expands to exactly this argv. The stdin
form itself is covered by the package test (`TestConfigSetWritesTokenForNonDefaultServer`,
`TestReadTokenStdin`); this leaf keeps the alias-targeting behavior.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	req.Prelude = true
	setCLI(req, "--alias", aliasLeafName, "config", "set", "--token", "tok-new")
	return nil
}
```

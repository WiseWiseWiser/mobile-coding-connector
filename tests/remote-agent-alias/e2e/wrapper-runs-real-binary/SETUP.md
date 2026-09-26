# Scenario

**Feature**: the generated wrapper runs the real product binary (L3 smoke)

```
build ./cmd/remote-agent -> alias add xdev --bin <built>
  -> sh ~/.local/bin/xdev-agent ping -> xdev server, Bearer tok-xdev
```

## Preconditions

Session builds the product CLI from source into the leaf HOME; the alias is
installed with `--bin <built binary>`, so the wrapper execs that real binary.

## Steps

1. `Setup` requests the prelude with `BuildBinary`, and `Op: wrapper` with
   `ping` as the wrapper argument.
2. `Run` builds `./cmd/remote-agent`, seeds the alias with `--bin`, then runs
   the wrapper through `sh` with the leaf `HOME`.
3. `Assert` checks exit code, output, and which server the request reached.

## Context

Sparse L3 smoke — keeps the wrapper → binary → HTTP path covered by at least one
leaf (subprocess boundary, real config-file resolution, real bearer auth).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	req.Prelude = true
	req.BuildBinary = true
	req.Op = "wrapper"
	req.WrapperArgs = []string{"ping"}
	return nil
}
```

# Scenario

**Feature**: remote-agent alias wrappers + `config set` targeting

```
# L2: CLI in-process with temp HOME + two fake servers; L3: sh wrapper -> real binary
leaf Setup -> Request (Prelude? Args | Op=wrapper) -> Run -> Assert
```

## Preconditions

1. Product implements `remote-agent alias …`, the global `--alias` flag, and
   `remote-agent config set`.
2. Each leaf gets an isolated temp `HOME` (`testhooks.SetHomeOverride`), so the
   alias store, client config, wrappers, and rc files never touch the real user
   home.
3. Two fake servers exist per leaf: `default` and `xdev`, both recording
   requests and bearer tokens. The `xdev` alias targets the second one.
4. A fake `remote-agent` is first on `PATH`, so a generated wrapper execs the
   bare name `remote-agent` deterministically.
5. No leaf needs a real network or a real remote host.

## Steps

1. `Run` creates the temp HOME, fake PATH, and both fake servers.
2. Optional `SeedForeignWrapper` writes a user-owned script at the wrapper path.
3. With `Prelude`, `Run` seeds the default domain, the `xdev` alias, and the
   `xdev` token through the real CLI, then applies `MutateWrapper` if set.
4. `Run` executes `Args` (in-process CLI) or the installed wrapper (`Op: wrapper`).
5. `Run` captures exit code, stdout/stderr, wrapper bytes before/after, store and
   config JSON, and each server's hit count + bearer tokens.
6. Leaf `Assert` checks those observations.

## Context

Leaves keep one behavioral outcome each; `resolve/` holds the MECE pair
(alias target vs default target). Only `e2e/` builds a product binary.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

// Setup defaults the op so leaves only declare what they exercise.
func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	if req.Op == "" {
		req.Op = "cli"
	}
	return nil
}

// setCLI configures a CLI leaf.
func setCLI(req *Request, args ...string) {
	req.Op = "cli"
	req.Args = args
}
```

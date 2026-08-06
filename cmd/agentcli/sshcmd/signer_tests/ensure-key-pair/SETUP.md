# Scenario

**Feature**: EnsureClientKeyPair — create, reload, fail-closed on corrupt private

```
# disk lifecycle under {ConfigDir}/id_ed25519
tests -> EnsureClientKeyPair(configDir)
  -> create missing | reload same Public | corrupt -> error
```

## Preconditions

- Scenario family is ensure-key-pair (leaf sets concrete Scenario).
- ConfigDir is absolute under d.DOCTEST_CASE (root Setup).

## Steps

1. Grouping does not override Scenario; leaves choose create-missing | reload-stable | corrupt-private.
2. Leaves assert file path, mode, Signer/Public, or EnsureErr as appropriate.

## Context

- Preferred export: `sshcmd.EnsureClientKeyPair(configDir string) (*ClientKeyPair, error)`.

```go
import (
	"fmt"
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	// Leaves set Scenario. ConfigDir already absolute from root Setup.
	if req.ConfigDir == "" {
		return fmt.Errorf("ensure-key-pair: ConfigDir must be set by root Setup")
	}
	return nil
}
```

# Scenario

**Feature**: Second Ensure same configDir reloads the same public key material

```
# stable identity
Ensure(configDir) -> Public A; Ensure(same) -> Public A (equal)
```

## Preconditions

- Scenario: `ensure-reload-stable`.
- ConfigDir starts empty; first call creates the key.

## Steps

1. Call EnsureClientKeyPair twice on the same ConfigDir.
2. Assert both succeed; PublicMaterial == PublicMaterial2; ReloadedSame true.

## Context

- Proves serve CreateSession public key and client Signer share one identity.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Scenario = ScenarioEnsureReloadStable
	return nil
}
```

# Scenario

**Feature**: Missing private key → Ensure creates id_ed25519 with owner-only mode

```
# empty configDir
tests -> EnsureClientKeyPair(configDir)
  -> write id_ed25519 (0600); return Signer + Public
```

## Preconditions

- Scenario: `ensure-create-missing`.
- `{ConfigDir}/id_ed25519` must not exist before Run (empty case dir).

## Steps

1. Call EnsureClientKeyPair on empty ConfigDir.
2. Assert private file exists, mode 0600 (or ModePerm == 0600), Signer and Public non-nil.

## Context

- Optional `id_ed25519.pub` is not required for this leaf.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Scenario = ScenarioEnsureCreateMissing
	return nil
}
```

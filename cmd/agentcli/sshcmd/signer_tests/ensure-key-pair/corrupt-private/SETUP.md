# Scenario

**Feature**: Corrupt id_ed25519 → Ensure returns error (fail-closed, no silent regen)

```
# bad private file
tests -> write garbage to id_ed25519 -> EnsureClientKeyPair
  -> error (do not overwrite with a new key silently)
```

## Preconditions

- Scenario: `ensure-corrupt-private`.
- Harness writes non-key bytes to `{ConfigDir}/id_ed25519` before Ensure.
- Default corrupt payload: `not-a-valid-ssh-private-key\n`.

## Steps

1. Write corrupt private file (mode 0600).
2. Call EnsureClientKeyPair; Assert EnsureErr non-empty.

## Context

- Safety preference from requirement: error over regenerate.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Scenario = ScenarioEnsureCorrupt
	req.CorruptPrivateBytes = []byte("not-a-valid-ssh-private-key\n")
	return nil
}
```

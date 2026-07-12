# Scenario

**Bug**: after Skip, residual banner still contains Update available and was treated as blocking

```
03b-menu-dismissed.snapshot.txt -> IsBlocking=false; writable reason ≠ codex update available
```

## Preconditions

Signed fixture `03b-menu-dismissed.snapshot.txt` (first post-Enter frame with menu gone).
Still has `model: loading` — writable may remain loading for **model** reason only.

## Steps

1. `FixtureFile=03b-menu-dismissed.snapshot.txt`.

## Context

PROTOCOL `confirm_skip` success predicate + writable narrow gate.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.FixtureFile = "03b-menu-dismissed.snapshot.txt"
	return nil
}
```

# Scenario

**Feature**: Empty doctor name with exactly one pair uses that pair

```
pairs=[solo] only, defaultPair empty
  -> Doctor("") -> PairName solo, AllOK
```

## Preconditions

- One pair `solo`; no defaultPair.
- Profile seeded for solo.

## Steps

1. Seed single pair.
2. Seed profile.
3. PairName empty.

## Context

- Sole-pair auto-select when no default.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.SeedPairsJSON = pairJSON("", onePair("solo", req.LocalPath, req.RemotePath))
	req.FocusPair = "solo"
	req.SeedProfile = true
	return nil
}
```

# Scenario

**Feature**: Empty doctor name uses config.defaultPair

```
defaultPair=alpha, pairs=[alpha, beta]
  -> Doctor("") -> PairName alpha, checks run
```

## Preconditions

- Two pairs; defaultPair `alpha`.
- Profile for alpha (FocusPair).
- Local roots exist for both paths (same LocalPath/RemotePath ok for test).

## Steps

1. Seed two pairs with defaultPair alpha.
2. Seed profile for alpha.
3. PairName empty.

## Context

- defaultPair wins over multi-pair ambiguity.

```go
import (
	"path/filepath"
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	// Distinct remotes under case for clarity.
	remoteB := filepath.Join(d.DOCTEST_CASE, "workspace-remote-b")
	req.SeedPairsJSON = pairJSON("alpha",
		onePair("alpha", req.LocalPath, req.RemotePath),
		onePair("beta", req.LocalPath, remoteB),
	)
	req.FocusPair = "alpha"
	req.SeedProfile = true
	return nil
}
```

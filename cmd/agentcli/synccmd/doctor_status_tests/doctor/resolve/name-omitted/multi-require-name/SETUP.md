# Scenario

**Feature**: Empty doctor name with multiple pairs and no default errors

```
pairs=[a,b], defaultPair=""
  -> Doctor("") -> error "pair name required"
```

## Preconditions

- Two pairs; empty defaultPair.
- No profile required (resolution fails).

## Steps

1. Seed two pairs without defaultPair.
2. PairName empty.

## Context

- Operator must pass an explicit name.

```go
import (
	"path/filepath"
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	remoteB := filepath.Join(d.DOCTEST_CASE, "workspace-remote-b")
	req.SeedPairsJSON = pairJSON("",
		onePair("a", req.LocalPath, req.RemotePath),
		onePair("b", req.LocalPath, remoteB),
	)
	// no SeedProfile — resolution should fail first
	return nil
}
```

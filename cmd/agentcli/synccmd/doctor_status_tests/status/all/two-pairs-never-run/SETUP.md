# Scenario

**Feature**: Status without name returns both pairs as never-run

```
pairs alpha + beta, no state files
  -> Status("") -> items for both; LastRun never each
```

## Preconditions

- Two pairs; no state files.
- Serve up.

## Steps

1. Seed two pairs.
2. PairName empty.

## Context

- Multi-pair inventory for operators.

```go
import (
	"path/filepath"
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	remoteB := filepath.Join(d.DOCTEST_CASE, "workspace-remote-b")
	req.PairName = ""
	req.SeedPairsJSON = pairJSON("",
		onePair("alpha", req.LocalPath, req.RemotePath),
		onePair("beta", req.LocalPath, remoteB),
	)
	req.ServeOK = serveUp()
	return nil
}
```

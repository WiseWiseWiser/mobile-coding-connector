# Scenario

**Feature**: CLI doctor exits non-zero when a critical check fails

```
seed mad-max + serve down
  -> RunCLI([unison doctor mad-max])
  -> RunErr non-empty; stdout may list checks
```

## Preconditions

- Pair + profile seeded.
- ServeOK fails; other hooks happy.

## Steps

1. Seed pair and profile.
2. Args `unison doctor mad-max`.
3. Serve down.

## Context

- Wire path for operator-facing exit status.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Mode = "cli"
	req.PairName = "mad-max"
	req.FocusPair = "mad-max"
	req.SeedPairsJSON = pairJSON("", onePair("mad-max", req.LocalPath, req.RemotePath))
	req.SeedProfile = true
	doctorHappyHooks(req)
	req.ServeOK = serveDown("connection refused")
	req.Args = []string{"unison", "doctor", "mad-max"}
	return nil
}
```

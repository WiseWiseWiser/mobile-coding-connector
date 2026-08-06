# Scenario

**Feature**: Doctor reports failed checks when probes or paths fail

```
Doctor(mad-max, broken probe|path) -> AllOK false + DoctorErr non-empty
```

## Preconditions

- Pair and profile seeded so only the intentional check fails.
- Grouping for version / serve / local-root failures.

## Steps

1. Seed mad-max + profile.
2. Leaves override one probe or remove local root.

## Context

- Critical fail → library error + structured check rows.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Mode = "doctor"
	req.PairName = "mad-max"
	req.FocusPair = "mad-max"
	req.SeedPairsJSON = pairJSON("", onePair("mad-max", req.LocalPath, req.RemotePath))
	req.SeedProfile = true
	doctorHappyHooks(req)
	return nil
}
```

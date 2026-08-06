# Scenario

**Feature**: Status surfaces lastRunAt from state file

```
seed state/mad-max.json with lastRunAt
  -> Status(mad-max) -> LastRun contains the timestamp (not never)
```

## Preconditions

- Pair mad-max + state JSON with known lastRunAt.

## Steps

1. Seed pair.
2. Seed state with lastRunAt `2026-01-15T10:00:00Z`.
3. PairName mad-max.

## Context

- State shape documented in root DOCTEST.md.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.PairName = "mad-max"
	req.SeedPairsJSON = pairJSON("", onePair("mad-max", req.LocalPath, req.RemotePath))
	req.SeedStateJSON = map[string]string{
		"mad-max": `{"lastRunAt":"2026-01-15T10:00:00Z","exitCode":0,"message":"ok"}` + "\n",
	}
	req.ServeOK = serveUp()
	return nil
}
```

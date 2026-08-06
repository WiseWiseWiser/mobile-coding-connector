# Scenario

**Feature**: Status shows never when state file is missing

```
pairs has mad-max; no state/mad-max.json
  -> Status(mad-max) -> LastRun contains "never"; ServeOK true
```

## Preconditions

- Seed one pair; do not seed state.
- Serve up.

## Steps

1. Seed mad-max.
2. PairName mad-max.
3. No SeedStateJSON.

## Context

- P2 default before any `run`.

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
	req.ServeOK = serveUp()
	return nil
}
```

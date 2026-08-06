# Scenario

**Feature**: RunPair unknown pair name errors

```
pairs has mad-max only -> RunPair(Name=ghost) -> error "unknown pair"
```

## Preconditions

- Seed one pair `mad-max`; request name `ghost`.

## Steps

1. Seed pairs.json with mad-max.
2. PairName `ghost`.
3. Happy hooks optional (resolution fails first).

## Context

- Same substring as P1 show/set/rm and P2 doctor/status.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Mode = "run"
	req.PairName = "ghost"
	req.FocusPair = "ghost"
	req.SeedPairsJSON = pairJSON("", onePair("mad-max", req.LocalPath, req.RemotePath))
	req.SeedProfile = false
	doctorHappyHooks(req)
	req.FakeExitCode = 0
	return nil
}
```

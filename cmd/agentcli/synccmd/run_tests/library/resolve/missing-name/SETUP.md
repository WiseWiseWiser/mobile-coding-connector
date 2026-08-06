# Scenario

**Feature**: RunPair missing name errors

```
RunPair(Name="") -> error requiring name; no Exec
```

## Preconditions

- Empty PairName; store may be empty or seeded (name still required for run).

## Steps

1. Leave PairName empty.
2. Optional seed irrelevant pair.
3. Assert non-empty error mentioning run/name/require.

## Context

- CLI missing name maps to same library rule; tested here at library layer.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Mode = "run"
	req.PairName = ""
	req.SeedPairsJSON = pairJSON("", onePair("mad-max", req.LocalPath, req.RemotePath))
	doctorHappyHooks(req)
	req.FakeExitCode = 0
	return nil
}
```

# Scenario

**Feature**: Doctor all checks pass with injectable fakes

```
seed pairs.json + profile
  -> Doctor(mad-max, versions=2.54.0/2.54.0, serve up, remote ok)
  -> AllOK, checks local-version…profile all OK
```

## Preconditions

- Pair `mad-max` with LocalPath/RemotePath.
- Profile seeded.
- Local root exists (root mkdir).

## Steps

1. Seed one pair + profile.
2. Install happy hooks.
3. PairName `mad-max`.

## Context

- Baseline greenpath for implementer.

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

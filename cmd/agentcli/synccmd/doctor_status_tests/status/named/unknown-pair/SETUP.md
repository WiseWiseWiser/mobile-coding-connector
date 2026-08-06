# Scenario

**Feature**: Status unknown pair name errors

```
Status(ghost) with only mad-max in store -> error "unknown pair"
```

## Preconditions

- Seed mad-max; request ghost.

## Steps

1. Seed one pair.
2. PairName ghost.

## Context

- Same error substring as doctor/P1.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.PairName = "ghost"
	req.SeedPairsJSON = pairJSON("", onePair("mad-max", req.LocalPath, req.RemotePath))
	return nil
}
```

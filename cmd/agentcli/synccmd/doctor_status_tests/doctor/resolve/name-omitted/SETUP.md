# Scenario

**Feature**: Doctor with empty name resolves defaultPair / sole pair / error

```
Doctor(Name="") -> defaultPair | sole pair | "pair name required"
```

## Preconditions

- PairName left empty.
- Leaves vary pair count and defaultPair.

## Steps

1. Clear PairName.
2. Install happy hooks + profile when resolution should succeed.

## Context

- Resolution precedence: defaultPair, else sole pair, else error.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Mode = "doctor"
	req.PairName = ""
	doctorHappyHooks(req)
	return nil
}
```

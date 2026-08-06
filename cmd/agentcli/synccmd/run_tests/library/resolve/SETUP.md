# Scenario

**Feature**: RunPair name resolution errors

```
unknown name | empty name -> error; no Exec
```

## Preconditions

- Grouping for resolve failures.

## Steps

1. Ensure Mode is `run`.
2. Leaves set PairName ghost or empty.

## Context

- Same unknown-pair substring as P1/P2.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	if req.Mode == "" {
		req.Mode = "run"
	}
	return nil
}
```

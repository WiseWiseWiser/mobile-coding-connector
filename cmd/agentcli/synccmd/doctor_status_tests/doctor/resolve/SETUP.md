# Scenario

**Feature**: Doctor pair name resolution and hard errors

```
Doctor(name|empty) -> resolve defaultPair / sole pair / error
```

## Preconditions

- Grouping for unknown pair and name-omitted resolution.

## Steps

1. Mode doctor.
2. Leaves set seeds and PairName (often empty).

## Context

- Resolution errors before or instead of full check table.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Mode = "doctor"
	return nil
}
```

# Scenario

**Feature**: Status with name omitted lists all pairs

```
Status(Name="") -> StatusItem per pair
```

## Preconditions

- Grouping for all-pairs status.

## Steps

1. Mode status; PairName empty.
2. Leaves seed multiple pairs.

## Context

- List-like status when operator omits name.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Mode = "status"
	req.PairName = ""
	return nil
}
```

# Scenario

**Feature**: per-task nested menu includes Delete…

```
ForEach task -> Menu { …; Delete… }  // confirm then DELETE
```

## Preconditions

Delete is per-task only.

## Steps

1. Set `ClientLeaf=per-task-delete`.

## Context

REQUIREMENT leaf: `client/per-task-delete` (scenario 5).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.ClientLeaf = "per-task-delete"
	return nil
}
```

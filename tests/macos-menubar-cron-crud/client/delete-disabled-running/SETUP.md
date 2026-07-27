# Scenario

**Feature**: Delete… is disabled when task status is running

```
Button("Delete…").disabled(!canDeleteCronTask(status: task.status))
  // or equivalent .disabled(task.status == "running")
```

## Preconditions

Swift (or shared formatter) gates Delete the same way as Run Now.

## Steps

1. Set `ClientLeaf=delete-disabled-running`.

## Context

REQUIREMENT leaf: `client/delete-disabled-running` (scenario 6).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.ClientLeaf = "delete-disabled-running"
	return nil
}
```

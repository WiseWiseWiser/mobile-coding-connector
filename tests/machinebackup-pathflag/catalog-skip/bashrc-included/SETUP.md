# Scenario

**Feature**: ordinary home dotfile is not catalog-excluded

```
MergeExclusions(nil,nil,nil) -> IsExcluded(".bashrc") == false
```

## Preconditions

- No pathflag attribute rule for `.bashrc`.

## Steps

1. RelPath `.bashrc`.
2. Expect not excluded.

## Context

- Negative control for catalog skip.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.RelPath = ".bashrc"
	req.WantExcluded = false
	req.WantExcludedSet = true
	return nil
}
```

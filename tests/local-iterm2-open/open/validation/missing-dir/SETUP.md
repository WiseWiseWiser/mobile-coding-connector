# Scenario

**Feature**: missing dir field is rejected

```
POST {} -> 4xx {"error":...}
```

## Preconditions

None.

## Steps

1. Set `OmitDir=true`.

## Context

REQUIREMENT scenario 3: missing dir → 4xx.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.OmitDir = true
	return nil
}
```

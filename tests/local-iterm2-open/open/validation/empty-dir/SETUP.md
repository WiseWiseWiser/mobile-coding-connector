# Scenario

**Feature**: empty dir string is rejected

```
POST {dir:""} -> 4xx {"error":...}
```

## Preconditions

None.

## Steps

1. Set `Dir=""` (field present but empty).

## Context

REQUIREMENT scenario 3 variant.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Dir = ""
	req.OmitDir = false
	return nil
}
```

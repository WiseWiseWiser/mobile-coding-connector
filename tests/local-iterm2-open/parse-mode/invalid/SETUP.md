# Scenario

**Feature**: unknown mode string is rejected

```
ParseOpenMode("bogus") -> error
```

## Preconditions

None beyond parse group.

## Steps

1. Set `ModeInput=bogus`.

## Context

Invalid mode must not silently default.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.ModeInput = "bogus"
	return nil
}
```

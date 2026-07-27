# Scenario

**Feature**: disabled service shows Enable action

```
ShowEnableAction(false) -> true (show Enable, not Disable)
```

## Preconditions

Service definition has `enabled=false`.

## Steps

1. Set `Enabled=false`.

## Context

REQUIREMENT leaf: `action/toggle-enable`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Enabled = false
	return nil
}
```
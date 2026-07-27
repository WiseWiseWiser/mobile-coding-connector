# Scenario

**Feature**: mode=smart selects ModeSmart

```
POST {dir, mode:"smart"} -> ModeSmart -> 200
```

## Preconditions

Valid temp dir.

## Steps

1. Set `Mode=smart`.
2. Use real OpenConfig for smart script markers.

## Context

REQUIREMENT scenario 2: mode smart maps correctly.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Mode = "smart"
	req.UseRealOpenConfig = true
	return nil
}
```

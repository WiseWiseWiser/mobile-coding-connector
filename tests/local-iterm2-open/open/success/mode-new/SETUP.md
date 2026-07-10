# Scenario

**Feature**: mode=new selects ModeForceNew

```
POST {dir, mode:"new"} -> ModeForceNew -> 200
```

## Preconditions

Valid temp dir.

## Steps

1. Set `Mode=new`.
2. Use real OpenConfig to capture force-new script.

## Context

REQUIREMENT scenario 2: mode new maps correctly.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Mode = "new"
	req.UseRealOpenConfig = true
	return nil
}
```

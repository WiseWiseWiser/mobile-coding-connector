# Scenario

**Feature**: mode "new" maps to ModeForceNew

```
ParseOpenMode("new") -> ModeForceNew
```

## Preconditions

None beyond parse group.

## Steps

1. Set `ModeInput=new`.

## Context

JSON `"new"` → `ModeForceNew` (kool `-n`).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ModeInput = "new"
	return nil
}
```

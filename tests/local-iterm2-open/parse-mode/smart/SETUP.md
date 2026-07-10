# Scenario

**Feature**: mode "smart" maps to ModeSmart

```
ParseOpenMode("smart") -> ModeSmart
```

## Preconditions

None beyond parse group.

## Steps

1. Set `ModeInput=smart`.

## Context

JSON `"smart"` → `ModeSmart`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ModeInput = "smart"
	return nil
}
```

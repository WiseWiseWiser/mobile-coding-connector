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
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ModeInput = "bogus"
	return nil
}
```

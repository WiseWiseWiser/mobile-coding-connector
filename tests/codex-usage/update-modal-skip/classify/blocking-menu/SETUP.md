# Scenario

**Feature**: blocking Update available menu classification

```
menu fixture (1. Update now / 2. Skip / 3. Skip until…) -> IsBlockingUpdateMenu=true
```

## Preconditions

Fixtures `01` and `02` contain full menu options and `Press enter to continue`.

## Steps

1. Leaf picks default selection vs Skip-selected fixture.

## Context

Auto-Skip protocol only runs when blocking menu is detected.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	t.Helper()
	// Grouping: ensure classify op for all menu leaves.
	if req.Op == "" {
		req.Op = "classify"
	}
	return nil
}
```

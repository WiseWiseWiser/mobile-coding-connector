# Scenario

**Feature**: top-level Refresh button remains on both apps

```
local + remote menu body -> Button("Refresh") at level-1 (not removed)
```

## Preconditions

Existing top-level Refresh must be kept when Terminals is added.

## Steps

1. Set `ClientLeaf=top-level-refresh`.

## Context

REQUIREMENT: keep existing top-level Refresh.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ClientLeaf = "top-level-refresh"
	return nil
}
```

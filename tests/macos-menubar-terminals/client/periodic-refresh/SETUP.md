# Scenario

**Feature**: periodic background refresh of services and terminals

```
app launch -> refresh loop / timer -> refreshServices + refreshTerminals (or combined)
```

## Preconditions

App-side poll re-fetches services and terminal sessions without requiring manual Refresh.

## Steps

1. Set `ClientLeaf=periodic-refresh`.

## Context

REQUIREMENT leaf: `client/periodic-refresh`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ClientLeaf = "periodic-refresh"
	return nil
}
```

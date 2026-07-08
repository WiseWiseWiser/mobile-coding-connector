# Scenario

**Feature**: all=1 returns services across project scopes

```
GET /api/services?all=1 -> local-web + other-api
```

## Preconditions

Two services seeded with different `projectDir` values.

## Steps

1. Set `Op=list-all`.

## Context

REQUIREMENT leaf: `list-all`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "list-all"
	return nil
}
```
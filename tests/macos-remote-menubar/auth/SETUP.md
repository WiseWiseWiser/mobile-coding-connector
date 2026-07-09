# Scenario

**Feature**: format Authorization header for remote API calls

```
token -> remoteconfig.AuthorizationHeader -> "Bearer <token>" | ""
```

## Preconditions

Remote profile attaches the header on every request when token is non-empty.
Empty token must not produce a bare `Bearer` prefix.

## Steps

1. Set `Op=auth`.
2. Leaf sets `Token`.

## Context

REQUIREMENT group: `auth/`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "auth"
	return nil
}
```

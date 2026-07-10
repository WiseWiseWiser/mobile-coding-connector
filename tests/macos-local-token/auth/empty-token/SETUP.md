# Scenario

**Feature**: empty token yields empty Authorization header value

```
AuthorizationHeader("") -> ""
```

## Preconditions

Token is empty string (caller should omit header).

## Steps

1. Set `Token` to empty.

## Context

REQUIREMENT leaf: `auth/empty-token`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Token = ""
	return nil
}
```

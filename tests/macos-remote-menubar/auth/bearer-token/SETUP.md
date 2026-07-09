# Scenario

**Feature**: non-empty token formats as Bearer header value

```
AuthorizationHeader("abc") -> "Bearer abc"
```

## Preconditions

Token is non-empty.

## Steps

1. Set `Token=abc`.

## Context

REQUIREMENT leaf: `auth/bearer-token`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Token = "abc"
	return nil
}
```

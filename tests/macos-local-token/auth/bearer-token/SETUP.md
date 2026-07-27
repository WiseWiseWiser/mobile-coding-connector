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
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Token = "abc"
	return nil
}
```

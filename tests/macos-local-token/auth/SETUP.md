# Scenario

**Feature**: format Authorization header for local ServerClient

```
token -> localauth.AuthorizationHeader -> "Bearer <token>" | ""
```

## Preconditions

Same contract as remote/service helpers: non-empty token gets Bearer scheme with
a single space; empty token returns empty string (caller omits header).

## Steps

1. Set `Op=auth`.
2. Leaf sets `Token`.

## Context

REQUIREMENT group: `auth/` (scenario 8 pure helper).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "auth"
	return nil
}
```

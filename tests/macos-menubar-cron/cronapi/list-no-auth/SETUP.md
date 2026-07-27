# Scenario

**Feature**: list request omits Authorization when token empty

```
BuildListCronTasksRequest("https://agent.example.com", "") -> no Authorization header
```

## Preconditions

Local server or unauthenticated remote call.

## Steps

1. Set base URL and empty token.

## Context

REQUIREMENT leaf: `cronapi/list-no-auth`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.CronAPILeaf = "list-no-auth"
	req.BaseURL = "https://agent.example.com"
	req.Token = ""
	return nil
}
```

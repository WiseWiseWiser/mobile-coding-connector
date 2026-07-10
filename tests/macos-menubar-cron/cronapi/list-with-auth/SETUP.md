# Scenario

**Feature**: list request includes Bearer auth when token set

```
BuildListCronTasksRequest("https://agent.example.com/", "secret-token")
  -> GET https://agent.example.com/api/cron-tasks
  -> Authorization: Bearer secret-token
```

## Preconditions

Remote (or local) client has a non-empty token.

## Steps

1. Set base URL and token.

## Context

REQUIREMENT leaf: `cronapi/list-with-auth`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.CronAPILeaf = "list-with-auth"
	req.BaseURL = "https://agent.example.com/"
	req.Token = "secret-token"
	return nil
}
```

# Scenario

**Feature**: delete request requires non-empty id

```
BuildDeleteCronTaskRequest(base, token, "") -> error
```

## Preconditions

Delete must not build with empty task id (same rule as run/enable/disable).

## Steps

1. Set empty TaskID and valid base URL.

## Context

REQUIREMENT leaf: `cronapi/delete-requires-id` (scenario 2).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.CronAPILeaf = "delete-requires-id"
	req.BaseURL = "https://agent.example.com"
	req.Token = "tok"
	req.TaskID = ""
	return nil
}
```

# Scenario

**Feature**: cron action request requires non-empty id

```
BuildCronActionRequest(base, token, run, "") -> error
```

## Preconditions

Run/enable/disable must not build with empty task id.

## Steps

1. Set empty TaskID and valid base URL.

## Context

REQUIREMENT leaf: `cronapi/action-requires-id`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.CronAPILeaf = "action-requires-id"
	req.BaseURL = "https://agent.example.com"
	req.Token = "tok"
	req.TaskID = ""
	req.CronAction = "run"
	return nil
}
```

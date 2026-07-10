# Scenario

**Feature**: send[] maps to FollowUpCommands

```
POST {dir, mode:"reuse", send:["echo hi","ls"]}
  -> Open cfg.FollowUpCommands == send -> 200
```

## Preconditions

Valid temp dir.

## Steps

1. Set mode reuse and `Send` with two commands.
2. Record-only inject is enough (no need for full script).

## Context

REQUIREMENT scenario 7: send[] → FollowUpCommands.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Mode = "reuse"
	req.OmitSend = false
	req.Send = []string{"echo hi", "ls"}
	req.UseRealOpenConfig = false
	return nil
}
```

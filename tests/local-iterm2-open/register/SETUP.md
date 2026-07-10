# Scenario

**Feature**: route registration and auth skip-list placement

```
Register(mux, handler) -> POST /api/local/iterm2/open served
server.Serve skip list must NOT include this path
```

## Preconditions

Package exports `Register`. Skip-list leaf reads `server/server.go`.

## Steps

1. Set `Op` per leaf (`register` or `skip_list`).

## Context

REQUIREMENT registration + not skip-listed.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	// Grouping narrows to registration / skip-list surface; leaf sets concrete Op.
	if req == nil {
		t.Fatal("nil request")
	}
	req.OmitSend = true
	return nil
}
```

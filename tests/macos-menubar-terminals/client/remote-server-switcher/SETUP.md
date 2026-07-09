# Scenario

**Feature**: remote app has level-1 Server/domain switcher

```
remote AICriticApp body -> Menu("Server"…) [level-1, not under Terminals]
  -> on select: write default + reload services/terminals
```

## Preconditions

Remote multi-domain UX requires a top-level domain switcher.

## Steps

1. Set `ClientLeaf=remote-server-switcher`.

## Context

REQUIREMENT leaf: `client/remote-server-switcher`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ClientLeaf = "remote-server-switcher"
	return nil
}
```

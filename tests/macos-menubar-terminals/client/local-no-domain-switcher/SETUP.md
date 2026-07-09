# Scenario

**Feature**: local app must not require a remote domain switcher

```
local AICriticApp -> Terminals + Services + Refresh; no Server domain menu
```

## Preconditions

Local app talks to the local server; domain switcher is remote-only.

## Steps

1. Set `ClientLeaf=local-no-domain-switcher`.

## Context

REQUIREMENT: local has no domain switcher.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ClientLeaf = "local-no-domain-switcher"
	return nil
}
```

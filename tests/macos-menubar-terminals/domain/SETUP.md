# Scenario

**Feature**: remote domain switcher persists `default` in remote-agent-config

```
# user picks domain B in level-1 Server menu
cfg(multi-domain, default=A) -> SelectDefaultDomain(B) -> Save -> Load -> Resolve = B
# same resolved endpoint drives Services + Terminals + Open-in-browser
```

## Preconditions

`Op=select_domain` uses `remoteconfig.SelectDefaultDomain` then Save/Load/Resolve.

## Steps

1. Leaf supplies multi-domain `ConfigJSON` and `SelectServer`.

## Context

REQUIREMENT: selecting a domain writes `default` and reloads clients for one endpoint.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "select_domain"
	return nil
}
```

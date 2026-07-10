# Scenario

**Feature**: config yields a non-empty domain token (source=config)

```
# config path succeeds before credentials
local-agent-config.json (usable domain.token) -> source=config
# credentials may exist but must not win
```

## Preconditions

Config file is present and provides a non-empty trimmed domain token via either
local-loopback domain match or default-domain match.

## Steps

1. Set `ConfigPresent=true` with leaf-specific JSON.
2. Leaf may also seed credentials to prove config precedence.

## Context

REQUIREMENT group: `resolve/from-config/`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ConfigPresent = true
	return nil
}
```

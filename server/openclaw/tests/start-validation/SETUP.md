# Scenario

**Feature**: start prerequisites validated before mock gateway starts

```
# ValidateStartConfig gates Start and POST /api/openclaw/start
Manager.Start -> ValidateStartConfig -> BAD_REQUEST or proceed

# slack disabled skips token requirements
Config (slack off) -> Start OK
```

## Preconditions

Config pre-seeded with slack on/off and token presence per leaf.

## Steps

1. Leaf configures slack and token fields.
2. `Run` calls validate or API start and captures HTTP status.

## Context

Maps validation errors to HTTP 400 BAD_REQUEST via API.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	if req.Op == "" {
		req.Op = OpAPIStart
	}
	if req.GatewayPort == 0 {
		req.GatewayPort = 18789
	}
	return nil
}
```
# Scenario

**Feature**: openclaw.json config load, save, mask, and merge

```
# config store holds plaintext secrets; API masks on read
Client -> PUT/GET /api/openclaw/config -> Config store -> MaskConfig

# partial PUT merges without dropping omitted secrets
PUT partial body -> MergeConfig -> SaveConfig (tokens preserved)
```

## Preconditions

Config file may be absent or pre-seeded by leaf `Setup`.

## Steps

1. Leaf sets `Request.Op` to config lifecycle operation.
2. `Run` executes load, round-trip, or API config call.

## Context

Covers defaults, secret round-trip, GET masking, and partial PUT merge edge cases.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	if req.GatewayPort == 0 {
		req.GatewayPort = 18789
	}
	return nil
}
```
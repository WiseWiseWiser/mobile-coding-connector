# Scenario

**Feature**: RenderGatewayConfig output for OpenClaw gateway

```
# renderer maps ai-critic config to openclaw.json shape
Config -> RenderGatewayConfig -> gateway/agents/channels blocks

# slack socket mode uses env SecretRefs, not inline tokens
Slack enabled -> channels.slack.appToken/botToken (source=env)
```

## Preconditions

Config pre-seeded with gateway, agents, and slack fields per leaf.

## Steps

1. Leaf writes config via `WriteInitialConfig`.
2. `Run` calls `RenderGatewayConfig` and parses JSON.

## Context

Validates rendered structure independent of filesystem write.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	if req.Op == "" {
		req.Op = OpRender
	}
	req.WriteInitialConfig = true
	if req.GatewayPort == 0 {
		req.GatewayPort = 18789
	}
	return nil
}
```
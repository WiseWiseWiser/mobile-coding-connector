# Scenario

**Feature**: rendered gateway port, workspace, and model

```
# RenderGatewayConfig maps top-level config to gateway/agents blocks
Config (port, workspace, model) -> rendered JSON
```

## Steps

1. Seed workspace and model.
2. Render config.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpRender
	req.WriteInitialConfig = true
	req.GatewayPort = 18789
	req.Workspace = "~/.openclaw/workspace"
	req.Model = "anthropic/claude-sonnet-4-6"
	return nil
}
```
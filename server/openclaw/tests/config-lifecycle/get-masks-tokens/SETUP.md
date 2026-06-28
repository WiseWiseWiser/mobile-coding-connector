# Scenario

**Feature**: GET /api/openclaw/config masks secrets

```
# MaskConfig replaces non-empty tokens with ***
GET /api/openclaw/config -> MaskConfig -> JSON response
```

## Steps

1. Seed config with slack tokens.
2. GET config via API.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpAPIGetConfig
	req.WriteInitialConfig = true
	req.SlackEnabled = true
	req.BotToken = "xoxb-secret"
	req.AppToken = "xapp-secret"
	return nil
}
```
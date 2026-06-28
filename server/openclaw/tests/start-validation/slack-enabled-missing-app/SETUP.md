# Scenario

**Feature**: missing app token rejected for socket mode

```
# socket mode requires app token
POST start -> ValidateStartConfig -> 400 BAD_REQUEST
```

## Steps

1. Seed slack enabled with bot token only.
2. POST /api/openclaw/start.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpAPIStart
	req.WriteInitialConfig = true
	req.SlackEnabled = true
	req.BotToken = "xoxb-only"
	return nil
}
```
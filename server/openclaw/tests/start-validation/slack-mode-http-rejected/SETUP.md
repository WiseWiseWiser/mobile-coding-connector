# Scenario

**Feature**: only socket mode supported

```
# http mode rejected at validation
POST start (mode=http) -> 400 BAD_REQUEST
```

## Steps

1. Seed slack with both tokens but mode `http`.
2. POST /api/openclaw/start.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpAPIStart
	req.WriteInitialConfig = true
	req.SlackEnabled = true
	req.SlackMode = "http"
	req.BotToken = "xoxb-1"
	req.AppToken = "xapp-1"
	return nil
}
```
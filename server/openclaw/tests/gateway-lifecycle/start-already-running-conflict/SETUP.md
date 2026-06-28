# Scenario

**Feature**: second start returns 409 CONFLICT

```
# already running gateway rejects duplicate start
POST start (ok) -> POST start -> 409 ALREADY_RUNNING
```

## Steps

1. Seed valid config.
2. POST start twice via API.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpAPIStart
	req.SecondStart = true
	req.WriteInitialConfig = true
	req.SlackEnabled = true
	req.BotToken = "xoxb-test"
	req.AppToken = "xapp-test"
	return nil
}
```
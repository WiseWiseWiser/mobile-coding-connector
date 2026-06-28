# Scenario

**Feature**: start sets mock gateway state and writes config

```
# Start persists running state and generated openclaw.json
Manager.Start -> state.json (running, mock_pid=4242) + openclaw/openclaw.json
```

## Steps

1. Seed valid slack config.
2. Call `OpStart`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpStart
	req.WriteInitialConfig = true
	req.SlackEnabled = true
	req.BotToken = "xoxb-test"
	req.AppToken = "xapp-test"
	return nil
}
```
# Scenario

**Feature**: stop clears running state

```
# Stop flips running=false, clears mock_pid
PreStart -> Stop -> state.running=false
```

## Steps

1. Seed config and pre-start gateway.
2. Call `OpStop`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpStop
	req.WriteInitialConfig = true
	req.SlackEnabled = true
	req.BotToken = "xoxb-test"
	req.AppToken = "xapp-test"
	req.PreStart = true
	return nil
}
```
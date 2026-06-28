# Scenario

**Feature**: running mock gateway passes doctor check

```
# pre-started gateway -> gateway_running ok
PreStart -> Doctor -> gateway_running (ok)
```

## Steps

1. Seed valid config and pre-start.
2. Run doctor.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpDoctor
	req.WriteInitialConfig = true
	req.SlackEnabled = true
	req.BotToken = "xoxb-test"
	req.AppToken = "xapp-test"
	req.PreStart = true
	return nil
}
```
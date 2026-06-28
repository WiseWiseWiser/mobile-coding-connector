# Scenario

**Feature**: generated openclaw.json present after start

```
# start writes generated config; doctor reports ok
PreStart -> Doctor -> generated_config (ok)
```

## Steps

1. Seed valid config and pre-start gateway.
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
# Scenario

**Feature**: status exposes slack config fields

```
# Status merges config slack settings into API payload
LoadConfig (slack on, socket) -> Status -> slack_enabled/mode
```

## Steps

1. Seed slack-enabled socket config.
2. Call `OpStatus`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpStatus
	req.WriteInitialConfig = true
	req.SlackEnabled = true
	req.SlackMode = "socket"
	req.BotToken = "xoxb-test"
	req.AppToken = "xapp-test"
	return nil
}
```
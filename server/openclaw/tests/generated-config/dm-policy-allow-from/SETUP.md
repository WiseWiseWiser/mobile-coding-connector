# Scenario

**Feature**: dmPolicy and allowFrom propagated to rendered slack

```
# slack channel block carries dm policy and allow list
Config dm_policy + allow_from -> channels.slack.dmPolicy/allowFrom
```

## Steps

1. Seed slack with dm_policy and allow_from.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpRender
	req.WriteInitialConfig = true
	req.SlackEnabled = true
	req.BotToken = "xoxb-test"
	req.AppToken = "xapp-test"
	req.DMPolicy = "allowlist"
	req.AllowFrom = []string{"U123", "U456"}
	return nil
}
```
# Scenario

**Feature**: require_mention adds groups wildcard

```
# requireMention=true adds groups.*.requireMention
Slack require_mention=true -> groups {"*": {requireMention: true}}
```

## Steps

1. Seed slack with `require_mention=true`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpRender
	req.WriteInitialConfig = true
	req.SlackEnabled = true
	req.BotToken = "xoxb-test"
	req.AppToken = "xapp-test"
	req.RequireMention = boolPtr(true)
	return nil
}
```
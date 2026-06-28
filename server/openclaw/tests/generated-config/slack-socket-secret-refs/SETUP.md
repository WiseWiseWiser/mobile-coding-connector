# Scenario

**Feature**: slack tokens rendered as env SecretRefs

```
# inline tokens never appear in generated openclaw.json
Slack enabled -> channels.slack.*Token (source=env, id=SLACK_*)
```

## Steps

1. Seed slack enabled (tokens on disk only).
2. Render config.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpRender
	req.WriteInitialConfig = true
	req.SlackEnabled = true
	req.BotToken = "xoxb-render"
	req.AppToken = "xapp-render"
	return nil
}
```
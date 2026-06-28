# Scenario

**Feature**: PUT without slack block preserves entire slack config

```
# top-level partial PUT must not clear nested slack settings
PUT {gateway_port:19002} -> slack block unchanged
```

## Steps

1. Seed slack-enabled config with tokens.
2. PUT only `gateway_port`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpAPIPutConfig
	req.WriteInitialConfig = true
	req.GatewayPort = 18789
	req.SlackEnabled = true
	req.BotToken = "xoxb-noslack"
	req.AppToken = "xapp-noslack"
	req.PutBody = `{"gateway_port":19002}`
	return nil
}
```
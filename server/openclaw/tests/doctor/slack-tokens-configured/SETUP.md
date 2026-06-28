# Scenario

**Feature**: configured slack tokens pass doctor check

```
# slack enabled with tokens -> slack_tokens ok; plugin/socket mocked (warn)
Doctor -> slack_tokens (ok), slack_plugin (warn), slack_socket (warn)
```

## Steps

1. Seed slack with both tokens.
2. Run doctor.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpDoctor
	req.WriteInitialConfig = true
	req.SlackEnabled = true
	req.BotToken = "xoxb-doc"
	req.AppToken = "xapp-doc"
	return nil
}
```
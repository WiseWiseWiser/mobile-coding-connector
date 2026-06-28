# Scenario

**Feature**: dry-run reports mocked integration for valid config

```
# DryRun validates config and lists mocked checks
POST start?dry_run=true -> DryRunResult (mocked, checks)
```

## Steps

1. Seed valid slack config.
2. API dry-run start.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpAPIDryRun
	req.WriteInitialConfig = true
	req.SlackEnabled = true
	req.BotToken = "xoxb-test"
	req.AppToken = "xapp-test"
	return nil
}
```
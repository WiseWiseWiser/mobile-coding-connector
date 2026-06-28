# Scenario

**Feature**: save/load preserves plaintext secrets on disk

```
# disk stores plaintext; LoadConfig returns same tokens
SaveConfig (tokens) -> openclaw.json -> LoadConfig
```

## Steps

1. Write config with slack tokens.
2. Round-trip load and read raw file.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpRoundTrip
	req.WriteInitialConfig = true
	req.SlackEnabled = true
	req.BotToken = "xoxb-roundtrip"
	req.AppToken = "xapp-roundtrip"
	return nil
}
```
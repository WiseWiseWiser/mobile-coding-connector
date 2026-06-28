# Scenario

**Feature**: PUT partial update preserves omitted slack tokens

```
# MergeConfig keeps existing tokens when PUT omits them
PUT {slack:{dm_policy:allowlist}} -> MergeConfig -> tokens unchanged
```

## Steps

1. Seed config with tokens.
2. PUT body changing only `dm_policy`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpAPIPutConfig
	req.WriteInitialConfig = true
	req.SlackEnabled = true
	req.BotToken = "xoxb-keep"
	req.AppToken = "xapp-keep"
	req.DMPolicy = "pairing"
	req.PutBody = `{"slack":{"dm_policy":"allowlist"}}`
	return nil
}
```
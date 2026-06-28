# Scenario

**Feature**: enabling slack applies normalized defaults

```
# normalizeConfig sets mode=socket, dm_policy=pairing, require_mention=true
PUT slack enabled + tokens -> normalizeConfig -> defaults applied
```

## Steps

1. PUT minimal slack enable payload with tokens.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpAPIPutConfig
	req.PutBody = `{"slack":{"enabled":true,"bot_token":"xoxb-new","app_token":"xapp-new"}}`
	return nil
}
```
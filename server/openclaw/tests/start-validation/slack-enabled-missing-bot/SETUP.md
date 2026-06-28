# Scenario

**Feature**: missing bot token rejected on start

```
# slack enabled without bot token fails validation
POST start -> ValidateStartConfig -> 400 BAD_REQUEST
```

## Steps

1. Seed slack enabled with app token only.
2. POST /api/openclaw/start.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpAPIStart
	req.WriteInitialConfig = true
	req.SlackEnabled = true
	req.AppToken = "xapp-only"
	return nil
}
```
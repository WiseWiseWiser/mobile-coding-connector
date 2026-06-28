# Scenario

**Feature**: enabled slack without tokens fails doctor

```
# slack enabled but tokens missing -> slack_tokens fail
Doctor -> slack_tokens (fail + hint)
```

## Steps

1. Seed slack enabled without tokens.
2. Run doctor.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpDoctor
	req.WriteInitialConfig = true
	req.SlackEnabled = true
	return nil
}
```
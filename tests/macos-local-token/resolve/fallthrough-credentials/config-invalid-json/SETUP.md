# Scenario

**Feature**: invalid JSON config falls through to credentials

```
local-agent-config.json = "{not valid json" + credentials "cred-after-bad-json"
  -> token=cred-after-bad-json, source=credentials
```

## Preconditions

Config file exists but is not valid JSON; must not abort resolve — fall through.

## Steps

1. Write invalid JSON as config content.
2. Write credentials with `cred-after-bad-json`.

## Context

REQUIREMENT leaf: scenario 4.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ConfigPresent = true
	req.ConfigJSON = `{not valid json`
	req.CredentialsText = "cred-after-bad-json\n"
	return nil
}
```

# Scenario

**Feature**: missing config falls through to credentials

```
# no local-agent-config.json
server-credentials: "cred-from-missing-config\n"
  -> token=cred-from-missing-config, source=credentials
```

## Preconditions

Config file absent; credentials present with a single non-empty line.

## Steps

1. Leave `ConfigPresent=false` (omit config file).
2. Write credentials with `cred-from-missing-config`.

## Context

REQUIREMENT leaf: scenario 2.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ConfigPresent = false
	req.CredentialsText = "cred-from-missing-config\n"
	return nil
}
```

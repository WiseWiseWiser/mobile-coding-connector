# Scenario

**Feature**: both config and credentials missing → none

```
# empty DataDir (no local-agent-config.json, no server-credentials)
ResolveLocalServerToken -> "", source=none
```

## Preconditions

Temp DataDir has neither file.

## Steps

1. Leave both files absent.

## Context

REQUIREMENT leaf: scenario 6.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ConfigPresent = false
	req.CredentialsPresent = false
	return nil
}
```

# Scenario

**Feature**: local profile UsesAuthToken is true

```
appprofile.Local() -> UsesAuthToken=true, ConfigFileName=local-agent-config.json
```

## Preconditions

Local product uses Bearer tokens for server APIs; profile flag must not claim
auth is unused.

## Steps

1. Set `ProfileName=local`.

## Context

REQUIREMENT leaf: optional UsesAuthToken for local profile.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ProfileName = "local"
	return nil
}
```

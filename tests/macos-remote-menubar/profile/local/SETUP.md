# Scenario

**Feature**: local app profile still spawns daemon

```
appprofile.Local() -> SpawnsDaemon=true (local app intent unchanged)
```

## Preconditions

Local product remains daemon-based.

## Steps

1. Set `ProfileName=local`.

## Context

REQUIREMENT leaf: `profile/local`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ProfileName = "local"
	return nil
}
```

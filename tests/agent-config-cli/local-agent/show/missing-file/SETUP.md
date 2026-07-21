# Scenario

**Feature**: local --show with no local-agent-config.json

```
# missing local config -> empty-ish pretty JSON
local-agent config --show -> empty domains JSON
```

## Preconditions

No local seed. Optional remote sentinel must not be read as local content.

## Steps

1. Seed only remote sentinel (different content).
2. Args = `config --show`.

## Context

Proves empty local dump even when remote file exists.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.SeedConfig = nil
	req.AlsoSeedRemoteConfig = remoteOnlySentinel()
	req.Args = []string{"config", "--show"}
	return nil
}
```

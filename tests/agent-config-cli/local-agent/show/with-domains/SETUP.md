# Scenario

**Feature**: local --show prints local-agent-config.json content

```
# seeded local config + remote sentinel -> stdout is local only
local-agent config --show -> sampleLocalConfig JSON
```

## Preconditions

Seed local sample; also write remote sentinel with different data.

## Steps

1. SeedConfig = sampleLocalConfig().
2. AlsoSeedRemoteConfig = remoteOnlySentinel().
3. Args = `config --show`.

## Context

Isolation: remote file must not appear in dump.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.SeedConfig = sampleLocalConfig()
	req.AlsoSeedRemoteConfig = remoteOnlySentinel()
	req.Args = []string{"config", "--show"}
	return nil
}
```

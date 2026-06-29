# Scenario

**Feature**: remote-agent config file is never modified

```
# sentinel remote-agent-config.json + local-agent-config.json -> ping -> remote bytes unchanged
local-agent ping -> reads local-agent-config.json only
```

## Preconditions

`remote-agent-config.json` contains a distinct sentinel payload.

## Steps

1. Seed remote config sentinel JSON.
2. Start server; seed local config for same localhost URL with valid token.
3. `WatchRemoteConfig = true`; run `ping` with `--server` matching local domain (via sync).

## Context

Ensures profile-specific config path; remote file is not written or migrated.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.SeedRemoteConfig = []byte(`{
  "default": "https://sentinel.remote.example.com",
  "domains": [
    {"server": "https://sentinel.remote.example.com", "token": "remote-only-token"}
  ]
}`)
	req.WatchRemoteConfig = true
	req.StartServer = true
	req.SyncServerFromBoundPort = true
	req.SeedLocalConfigAfterServer = true
	req.Args = []string{"ping"}
	return nil
}
```
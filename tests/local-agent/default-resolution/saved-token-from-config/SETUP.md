# Scenario

**Feature**: saved token applied for matching server URL

```
# local-agent-config.json domain row -> resolveClient fills token -> auth status
local-agent auth status -> ai-critic-server + saved token -> Auth: OK
```

## Preconditions

Server uses `lib.TestPassword` credentials; config stores that token for the server URL.

## Steps

1. Start server; capture port as `http://localhost:P`.
2. Seed `local-agent-config.json` with default domain and token `testpassword`.
3. Run `auth status` with `--server http://localhost:P` and no `--token`.

## Context

Mirrors `TestResolveClientUsesSavedTokenForExplicitMatchingServer` at CLI level.

```go
import (
	"testing"

	"github.com/xhd2015/ai-critic/script/lib"
)

func Setup(t *testing.T, req *Request) error {
	req.StartServer = true
	req.Args = []string{"auth", "status"}
	// Server URL and config written after port known — Run sets Server when SyncServerFromBoundPort.
	req.SyncServerFromBoundPort = true
	req.SeedLocalConfigAfterServer = true
	req.LocalConfigToken = lib.TestPassword
	return nil
}
```
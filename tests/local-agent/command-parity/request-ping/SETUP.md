# Scenario

**Feature**: arbitrary GET via `request` subcommand

```
# request /ping -> plain pong body on stdout
local-agent request /ping -> GET /ping -> pong
```

## Preconditions

Server running with test credentials; valid token on CLI.

## Steps

1. Start server; set `--server` and `--token` to test password.

## Context

REQUIREMENT scenario: `local-agent request /ping` → pong.

```go
import (
	"testing"

	"github.com/xhd2015/ai-critic/script/lib"
)

func Setup(t *testing.T, req *Request) error {
	req.StartServer = true
	req.SyncServerFromBoundPort = true
	req.TokenSpecified = true
	req.Token = lib.TestPassword
	req.Args = []string{"request", "/ping"}
	return nil
}
```
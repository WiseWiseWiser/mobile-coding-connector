# Scenario

**Feature**: remote menu must not expose Restart Daemon

```
remote product / profile-gated Swift menu -> no ungated Restart Daemon for remote
```

## Preconditions

Either a dedicated `ai-critic-remote-macos` target without Restart Daemon, or
shared menu gated by `SpawnsDaemon` / remote profile so remote never shows it.

## Steps

1. Set `ClientLeaf=no-restart-daemon`.

## Context

REQUIREMENT leaf: `client/no-restart-daemon`. RED until remote product/gating exists.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ClientLeaf = "no-restart-daemon"
	return nil
}
```

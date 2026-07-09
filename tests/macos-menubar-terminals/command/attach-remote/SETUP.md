# Scenario

**Feature**: remote attach command line

```
BuildTerminalAttachCommand("remote-agent","web1") -> "remote-agent terminal attach web1"
```

## Preconditions

Remote app uses `remote-agent` with config-covered domain (no Bearer on cmdline).

## Steps

1. Set `Op=attach_cmd`, binary `remote-agent`, session id `web1`.

## Context

REQUIREMENT leaf: `command/attach-remote`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "attach_cmd"
	req.AgentBinary = "remote-agent"
	req.SessionID = "web1"
	return nil
}
```

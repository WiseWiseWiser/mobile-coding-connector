# Scenario

**Feature**: local attach command line

```
BuildTerminalAttachCommand("local-agent","web1") -> "local-agent terminal attach web1"
```

## Preconditions

Local app opens iTerm and runs local-agent attach for session id.

## Steps

1. Set `Op=attach_cmd`, binary `local-agent`, session id `web1`.

## Context

REQUIREMENT leaf: `command/attach-local`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "attach_cmd"
	req.AgentBinary = "local-agent"
	req.SessionID = "web1"
	return nil
}
```

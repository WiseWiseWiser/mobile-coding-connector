# Scenario

**Feature**: stream progress frames consumed for display (not token-only)

```
MachineBackupClient stream -> onProgress / events / FormatBackupProgress*
# must not drop intermediate SSE frames solely to keep archive_token
```

## Preconditions

Client or app code yields/handles section|progress|log|error for the window.

## Steps

1. ClientLeaf=stream-progress-consumed.

## Context

REQUIREMENT #19; problem 3 (frames discarded).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ClientLeaf = "stream-progress-consumed"
	return nil
}
```

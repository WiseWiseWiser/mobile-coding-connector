# Scenario

**Feature**: download path matches CLI stream + archive_token flow

```
POST …/machine/backup/stream -> done.archive_token -> GET archive by token
```

## Preconditions

Remote app (or shared client helper) must not invent a non-token-only backup path
as the sole implementation; stream + token is required.

## Steps

1. Set `ClientLeaf=stream-token-download`.

## Context

REQUIREMENT #27; aligns with `cmd/agentcli/machine.go` / client machine backup.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ClientLeaf = "stream-token-download"
	return nil
}
```

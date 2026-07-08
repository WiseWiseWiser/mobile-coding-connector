# Scenario

**Feature**: business APIs on server port 23712

```
ServerClient -> :23712 /api/grok/usage, /api/codex/usage, /api/debug/log
DaemonClient -> control plane only (no business usage routes)
```

## Preconditions

Port split: daemon `23312` control only; server `23712` business plane.

## Steps

1. Set `Op=client-business-port`.

## Context

REQUIREMENT section E — restart-menu client contract for server-port APIs.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "client-business-port"
	return nil
}
```
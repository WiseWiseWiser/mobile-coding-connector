# Scenario

**Feature**: remote Swift AICriticApp Backup submenu contracts

```
# ai-critic-remote-macos sources (read-only)
AICriticApp -> Menu("Backup") -> Status ▸ Enable|Disable, Backup Now…, Recent, Reveal
default enabled=false; download via machine/backup/stream + archive_token
```

## Preconditions

Swift sources exist under `macos-ai-critic/ai-critic-remote-macos/` (and optional Shared).
Pure source inspection — no subprocess, UI, or network.

## Steps

1. Set `Op=client` and leaf-specific `ClientLeaf`.

## Context

REQUIREMENT Swift contract scenarios 24–27. Local app out of scope.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "client"
	return nil
}
```

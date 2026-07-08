# Scenario

**Feature**: Swift menu-bar Services + server-port business API contract

```
# Swift sources must use ServerClient on :23712 for grok/codex/services/logs
User -> Services Menu -> ServerClient -> GET /api/services?all=1, usage APIs, log SSE
```

## Preconditions

Swift sources exist under `macos-ai-critic/ai-critic-macos/` including
`ServerClient.swift`, `LogTailWindow.swift`, and `AICriticApp.swift`.

## Steps

1. Set `Op=client` and leaf-specific `ClientLeaf`.

## Context

Pure source inspection — no subprocess or HTTP. RED before ServerClient migration.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "client"
	return nil
}
```
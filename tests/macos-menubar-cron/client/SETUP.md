# Scenario

**Feature**: Swift source contracts for Cron menus (local + remote)

```
# local + remote AICriticApp.swift (+ Shared, ServerClient, LogTailWindow)
local  -> Menu("Cron") + cron-menu a11y; nested actions; SSE logs; placement
remote -> same Cron UX; Not configured; ServiceClient base URL + Bearer for cron
refresh loop / Refresh button -> includes cron task list
```

## Preconditions

Swift sources exist under:
- `macos-ai-critic/ai-critic-macos/AICriticApp.swift`
- `macos-ai-critic/ai-critic-remote-macos/AICriticApp.swift`
- optional Shared helpers and log window

## Steps

1. Set `Op=client` and leaf-specific `ClientLeaf`.

## Context

Pure source inspection — no subprocess, UI, or network. RED until Cron UI lands.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "client"
	return nil
}
```

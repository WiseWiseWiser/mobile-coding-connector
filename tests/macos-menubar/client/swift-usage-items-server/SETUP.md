# Scenario

**Feature**: menu-bar usage refresh targets the server usage-items registry

```
AppState.refresh -> ServerClient :23712 GET /api/usage/items
```

## Preconditions

Per-provider usage client helpers (`grokUsage` / `codexUsage`) were retired; the
menu bar is a thin display layer over the server-rendered usage item titles.

## Steps

1. Inspect `ai-critic-macos/AICriticApp.swift` and `ai-critic-macos/ServerClient.swift`.
2. Walk every `*.swift` under `macos-ai-critic/` (skipping `.build`) for legacy
   `grokUsage` / `codexUsage` calls.

## Context

REQUIREMENT leaf: thin menu-bar display layer fed by `GET /api/usage/items`.

```go
import (
"testing"

"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
req.Op = "client"
return nil
}
```

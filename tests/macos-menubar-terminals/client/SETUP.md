# Scenario

**Feature**: Swift source contracts for Terminals menus, domain switcher, iTerm, refresh

```
# local + remote AICriticApp.swift (+ Shared)
local  -> Terminals, New Terminal, Refresh; no Server domain switcher
remote -> Terminals, New Terminal, level-1 Server switcher, Refresh
open path -> iTerm only (no Terminal.app fallback)
background -> periodic services + terminals refresh
```

## Preconditions

Swift sources exist under:
- `macos-ai-critic/ai-critic-macos/AICriticApp.swift`
- `macos-ai-critic/ai-critic-remote-macos/AICriticApp.swift`
- optional Shared helpers

## Steps

1. Set `Op=client` and leaf-specific `ClientLeaf`.

## Context

Pure source inspection — no subprocess, UI, or network. RED until Terminals UI lands.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "client"
	return nil
}
```

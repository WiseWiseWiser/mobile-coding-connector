# Scenario

**Feature**: menu-bar usage refresh targets server port

```
AppState.refresh -> ServerClient :23712 /api/grok/usage and /api/codex/usage
```

## Preconditions

Business usage APIs moved off daemon port `23312`.

## Steps

1. Inspect `AICriticApp.swift`, `ServerClient.swift`, `DaemonClient.swift`.

## Context

REQUIREMENT leaf: menubar grok/codex server-port migration.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "client"
	return nil
}
```
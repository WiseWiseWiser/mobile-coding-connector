# Scenario

**Feature**: local Terminals attach/new go through openITerm2 API

```
openAttachTerminal / openNewTerminal
  -> build command -> openITerm2(dir?, mode, send: [cmd])
  -> POST /api/local/iterm2/open
```

## Preconditions

Local `AICriticApp.swift` + `ServerClient.swift` only (remote out of scope).

## Steps

1. Set `ClientLeaf=open-via-local-api`.

## Context

REQUIREMENT scenario 11: **local** Terminals use `/api/local/iterm2/open` (not
Terminal.app; not raw osascript-only product path). Remote keeps client-side open.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ClientLeaf = "open-via-local-api"
	return nil
}
```

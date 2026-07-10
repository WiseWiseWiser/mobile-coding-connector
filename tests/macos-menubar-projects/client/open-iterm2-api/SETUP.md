# Scenario

**Feature**: ServerClient exposes openITerm2 against /api/local/iterm2/open

```
ServerClient.openITerm2(dir:mode:send:)
  -> POST /api/local/iterm2/open + Authorization Bearer
```

## Preconditions

Local `ServerClient.swift` on port 23712.

## Steps

1. Set `ClientLeaf=open-iterm2-api`.

## Context

REQUIREMENT scenarios 8, 12 (API client + Bearer on open).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ClientLeaf = "open-iterm2-api"
	return nil
}
```

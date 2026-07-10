# Scenario

**Feature**: Swift source contracts for Projects menu UX (parts, loading, API)

```
# inspect local macOS app sources
AICriticApp.swift / ServerClient.swift / ProjectsMenuFormatter.swift
  -> Projects menu, /api/wrk/..., HStack titles, projectsLoading, Loading…
```

## Preconditions

`Op=client` inspects local macOS app sources under `macos-ai-critic/`.

## Steps

1. Set `Op=client`.
2. Leaf sets `ClientLeaf`.

## Context

REQUIREMENT scenarios 12–14, prior menu/API leaves, plus local iTerm2 open
contracts (open-iterm2-api / click-main / click-worktree / create-worktree-opens)
from REQUIREMENT-DESIGN-local-iterm2-open.md.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "client"
	return nil
}
```

# Scenario

**Feature**: project and worktree titles use parts + left/right HStack layout

```
# Swift renders Leading left, Trailing right
ProjectsMenuFormatter.formatProjectTitleParts / formatWorktreeTitleParts
  + AICriticApp HStack { Text(leading); Spacer(); Text(trailing) }
```

## Preconditions

Local app sources under `macos-ai-critic/`.

## Steps

1. Set `ClientLeaf=title-parts-hstack`.

## Context

REQUIREMENT scenario 12: parts + HStack (or equivalent left/right).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ClientLeaf = "title-parts-hstack"
	return nil
}
```

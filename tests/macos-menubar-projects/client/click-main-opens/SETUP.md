# Scenario

**Feature**: project submenu primary action “Open in iTerm2” opens project.path with reuse

```
Projects > {project} > Button("Open in iTerm2")
  -> openITerm2(dir: project.path, mode: reuse|default)
```

## Preconditions

Local `AICriticApp.swift` Projects submenu. Nested Menu titles are not click
targets — locked UX is an explicit open action under the project submenu.

## Steps

1. Set `ClientLeaf=click-main-opens`.

## Context

REQUIREMENT locked decision (orchestrator): **Open in iTerm2** under each project
opens `project.path` with mode reuse. Not Menu-title click alone.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.ClientLeaf = "click-main-opens"
	return nil
}
```

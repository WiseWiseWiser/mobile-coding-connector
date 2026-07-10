# Scenario

**Feature**: AppState tracks projectsLoading and does not clear projects on start/fail

```
# AppState
@Published projectsLoading: Bool
refreshProjects / refresh: set loading; never projects = [] on start or catch
```

## Preconditions

Local `AICriticApp.swift` owns `AppState` and project refresh.

## Steps

1. Set `ClientLeaf=projects-loading-flag`.

## Context

REQUIREMENT scenario 13: `projectsLoading`; keep list on fail/start.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ClientLeaf = "projects-loading-flag"
	return nil
}
```

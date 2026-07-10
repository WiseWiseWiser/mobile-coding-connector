# Scenario

**Feature**: refresh success replaces list and clears error

```
# success with a new list after a prior error state
ApplyProjectsRefreshSuccess(
  {Projects:[old], Loading:true, Error:"timeout"},
  [dot-pkgs, other],
) -> {Projects:[dot-pkgs, other], Loading:false, Error:""}
```

## Preconditions

Refresh in flight; previous error may still be set; new list arrives.

## Steps

1. Set prior projects `["old"]`, prior loading true, prior error `timeout`.
2. Set `RefreshAction=success`, `NewProjects=["dot-pkgs","other"]`.

## Context

REQUIREMENT: success → `projects=list`; `projectsLoadError=nil`; `projectsLoading=false`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.PriorProjects = []string{"old"}
	req.PriorLoading = true
	req.PriorError = "timeout"
	req.RefreshAction = "success"
	req.NewProjects = []string{"dot-pkgs", "other"}
	return nil
}
```

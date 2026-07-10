# Scenario

**Feature**: refresh failure keeps projects and records error

```
# mid-flight loading with stale rows; fetch fails
ApplyProjectsRefreshFailure({Projects:[dot-pkgs], Loading:true, Error:""}, "timeout")
  -> {Projects:[dot-pkgs], Loading:false, Error:"timeout"}
```

## Preconditions

Refresh was in progress with a non-empty prior list; the list call fails.

## Steps

1. Set prior projects `["dot-pkgs"]`, prior loading true, prior error empty.
2. Set `RefreshAction=failure`, `FailError=timeout`.

## Context

REQUIREMENT: failure → keep projects; set `projectsLoadError`; `projectsLoading=false`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.PriorProjects = []string{"dot-pkgs"}
	req.PriorLoading = true
	req.PriorError = ""
	req.RefreshAction = "failure"
	req.FailError = "timeout"
	return nil
}
```

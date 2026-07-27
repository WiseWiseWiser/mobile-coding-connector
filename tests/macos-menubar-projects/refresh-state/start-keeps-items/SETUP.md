# Scenario

**Feature**: refresh start sets loading without clearing existing projects

```
# prior has "dot-pkgs"; start refresh
ApplyProjectsRefreshStart({Projects:[dot-pkgs], Loading:false, Error:""})
  -> {Projects:[dot-pkgs], Loading:true, Error:""}
```

## Preconditions

Last successful list is non-empty; a new list fetch is starting.

## Steps

1. Set prior projects `["dot-pkgs"]`, prior loading false, prior error empty.
2. Set `RefreshAction=start`.

## Context

REQUIREMENT: start → `projectsLoading=true`; do **not** clear projects.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.PriorProjects = []string{"dot-pkgs"}
	req.PriorLoading = false
	req.PriorError = ""
	req.RefreshAction = "start"
	return nil
}
```

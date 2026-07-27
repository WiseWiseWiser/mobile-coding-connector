# Scenario

**Feature**: Projects menu shows Loading… when loading and list is empty

```
# menu body
if projectsLoading && projects.isEmpty { Text(Loading… / formatProjectsLoadingLabel) }
```

## Preconditions

Local Projects menu UI and loading label wiring.

## Steps

1. Set `ClientLeaf=loading-when-empty`.

## Context

REQUIREMENT scenario 14: show Loading… when loading and empty.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.ClientLeaf = "loading-when-empty"
	return nil
}
```

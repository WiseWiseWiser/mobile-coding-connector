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
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ClientLeaf = "loading-when-empty"
	return nil
}
```

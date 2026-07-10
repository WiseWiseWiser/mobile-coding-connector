# Scenario

**Feature**: projects list load-failed label

```
FormatProjectsLoadFailedLabel() -> "Failed to load projects"
```

## Preconditions

Load finished with error and the registry has no rows to show.

## Steps

1. Set `LabelKind=failed`.

## Context

REQUIREMENT: failed label → `Failed to load projects`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.LabelKind = "failed"
	return nil
}
```

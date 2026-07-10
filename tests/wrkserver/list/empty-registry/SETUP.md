# Scenario

**Feature**: empty projects registry returns empty envelope

```
# no projects.json (or empty projects list)
ListProjects -> 200 {"projects":[]}
```

## Preconditions

Isolated `WrkHome` with no recorded projects.

## Steps

1. Ensure `WrkHome` exists empty (no `projects.json` entries).
2. Invoke list handler.

## Context

REQUIREMENT scenario 1.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	ensureWrkHome(t, req)
	// Explicit empty registry file (absent file should also yield empty).
	writeProjectsJSON(t, req.WrkHome, nil)
	return nil
}
```

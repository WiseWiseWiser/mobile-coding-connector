# Scenario

**Feature**: node_modules child size and recursive dir count both appear

```
# nm-entry has top-level node_modules child + nested node_modules deeper
entry shows > node_modules child line AND node_modules N dirs aggregate (N>=2)
```

## Preconditions

`SeedProfile=node-modules`: `nm-entry/node_modules/` and `nm-entry/src/deep/node_modules/`.

## Steps

1. Set `SeedProfile` to `node-modules`.

## Context

REQUIREMENT leaf `stream/node-modules`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.SeedProfile = "node-modules"
	req.Args = []string{"machine", "analyse-files"}
	return nil
}
```
# Scenario

**Feature**: CLI include re-includes builtin `.cache`

```
MergeExclusions(nil, nil, [".cache"]) -> IsExcluded(".cache/x") == false
```

## Preconditions

- Include deletes `.cache` from the effective exclude set.

## Steps

1. Include `.cache`.
2. RelPath `.cache/x`.
3. Expect not excluded.

## Context

- Policy layer must survive pathflag catalog wiring.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Include = []string{".cache"}
	req.Exclude = nil
	req.RelPath = ".cache/x"
	req.WantExcluded = false
	req.WantExcludedSet = true
	return nil
}
```

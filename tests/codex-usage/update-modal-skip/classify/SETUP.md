# Scenario

**Feature**: classify signed update-modal snapshots (menu vs banner vs status)

```
fixture text -> IsBlockingUpdateMenu + UpdateMenuSelection + CheckWritable (+ ParseStatusSnapshot)
```

## Preconditions

Signed fixtures under `testdata/update-modal-skip/`. No live Codex.

## Steps

1. Set `Op=classify`.
2. Leaf sets `FixtureFile` and optional `StripModelLoading`.

## Context

Fast CI leaves (no labels). Drive production classifier surface.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "classify"
	return nil
}
```

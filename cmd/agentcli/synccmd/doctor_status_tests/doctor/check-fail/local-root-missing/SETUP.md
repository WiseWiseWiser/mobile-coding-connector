# Scenario

**Feature**: Doctor fails local-root when pair local path is missing

```
remove LocalPath dir -> Doctor -> local-root OK=false
```

## Preconditions

- Pair points at LocalPath; Run removes it when EnsureLocalRoot=false.

## Steps

1. Set EnsureLocalRoot false so Run deletes LocalPath after seed.
2. Keep other hooks happy.

## Context

- Filesystem check only; no network.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.EnsureLocalRoot = false
	req.EnsureLocalRootSet = true
	return nil
}
```

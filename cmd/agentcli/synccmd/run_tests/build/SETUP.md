# Scenario

**Feature**: BuildUnisonCmd pure argv/env builder

```
seed pair -> BuildUnisonCmd(opts) -> argv + child env (no Exec)
```

## Preconditions

- Grouping for BuildUnisonCmd leaves.
- Leaves seed pairs.json and set Interactive / LocalUnisonPath.

## Steps

1. Default Mode to `build` when empty.
2. Leaves seed pair + profile and flag variants.

## Context

- Library-only; no Exec, no state write.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	if req.Mode == "" {
		req.Mode = "build"
	}
	return nil
}
```

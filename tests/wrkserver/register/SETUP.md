# Scenario

**Feature**: Register mounts fixed leaves under host-owned base

```
# base is host-owned; leaves are fixed
Register(mux, base) -> GET {base}/projects, POST {base}/worktrees
```

## Preconditions

`Op=register` builds a fresh `ServeMux`, calls `Register`, then serves one path.

## Steps

1. Set `Op` to `register`.
2. Leaf sets `Base`, `Method`, and `Path` (and optional body).

## Context

REQUIREMENT scenarios 10–11 — proves prefix is not hardcoded inside wrkserver.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "register"
	return nil
}
```

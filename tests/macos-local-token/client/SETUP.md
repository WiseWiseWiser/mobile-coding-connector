# Scenario

**Feature**: local Swift ServerClient source contracts for Bearer auth

```
# read-only inspection of ServerClient / LocalAuth sources
ServerClient request builders -> set Authorization: Bearer <token> when non-empty
```

## Preconditions

Sources under `macos-ai-critic/` implement (or will implement) attaching the
resolved local token on loopback API requests. Leaves stay RED until wiring exists.

## Steps

1. Set `Op=client`.
2. Leaf sets `ClientLeaf`.

## Context

REQUIREMENT group: `client/` (scenario 8 Swift contract).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "client"
	return nil
}
```

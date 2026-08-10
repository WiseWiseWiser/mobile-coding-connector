# Scenario

**Feature**: ai-critic server/eventbus hub, loopback publish, WS subscribe, flags

```
# L2 in-process: Hub / PublishServer / RegisterSubscribeWS / PublishConfig
NewHub -> Publish/Subscribe/Recent
StartPublishServer(127.0.0.1) -> POST /publish
RegisterSubscribeWS(mux) -> GET /api/event-bus/ws
ResolvePublishConfig / AppendEventBusServerArgs -> keep-alive child argv
```

## Preconditions

1. Intended package path: `github.com/xhd2015/ai-critic/server/eventbus`
   (greenfield until implementer lands code).
2. Shared types: `github.com/xhd2015/dot-pkgs/go-pkgs/eventbus` Event + constants
   (PHASE 1). Module may need a `replace` to the local brought tree.
3. Loopback TCP available for publish + WS leaves.
4. No product binary e2e; no process env mutation (`Setenv`/`Chdir` forbidden).

## Steps

1. Root Setup is a no-op guard (validates Request pointer).
2. Grouping Setup sets `Op` and surface defaults.
3. Leaf Setup fills event fixtures, tokens, flag inputs.
4. Root `Run` calls the intended package API and fills `Response`.
5. Leaf `Assert` checks status, enrichment, fan-out, bind, flags.

## Context

REQUIREMENT-DESIGN PHASE 2 — ai-critic event bus server. Classic TDD RED until
`server/eventbus` exists.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	if req == nil {
		return nil
	}
	// Default ring size for hub leaves that do not override.
	if req.RingSize == 0 {
		req.RingSize = 200
	}
	return nil
}
```

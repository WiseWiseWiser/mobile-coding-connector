# Scenario

**Feature**: GET /api/services list with optional all=1

```
seed multi-project services.json -> ai-critic-server -> GET /api/services -> scoped or all IDs
```

## Preconditions

1. Module builds `ai-critic-server` (`.`).
2. Each test uses isolated `AI_CRITIC_HOME` with `lib.TestPassword` credentials.
3. `services.json` seeds two services with different `projectDir` values.
4. Server starts with `Dir` set to the local project directory.

## Steps

1. Root `Run` builds server, writes `services.json`, starts server on free port.
2. Leaf `Setup` sets `Op` to `list-scoped` or `list-all`.
3. Root `Run` performs authenticated `GET /api/services` with or without `?all=1`.
4. Leaf `Assert` checks returned service IDs.

## Context

Implements REQUIREMENT-DESIGN-menubar-services-server-port.md section B.
Menu bar uses `?all=1` to show every managed service.

```go
import (
	"testing"

	"github.com/xhd2015/ai-critic/script/lib"
)

func Setup(t *testing.T, req *Request) error {
	if req.Token == "" {
		req.Token = lib.TestPassword
	}
	return nil
}
```
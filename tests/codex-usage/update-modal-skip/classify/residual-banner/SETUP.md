# Scenario

**Feature**: residual Update available banner is not a blocking menu

```
03b/04 banner + main TUI chrome (no 2. Skip / Press enter to continue) -> IsBlocking=false
```

## Preconditions

Fixtures show boxed banner + main Codex chrome; menu options gone.

## Steps

1. Leaf sets dismissed banner fixture; optional model-loading strip for idle.

## Context

Critical product implication: bare “update available” must not keep writable loading
forever after Skip, or `/status` never runs.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	t.Helper()
	if req.Op == "" {
		req.Op = "classify"
	}
	return nil
}
```

# Scenario

**Feature**: Tunnel dial fails after session destroy

```
# lifecycle
CreateSession -> DestroySSHSession -> SSHTunnelDial -> error
```

## Preconditions

- Scenario: `tunnel-after-destroy`.

## Steps

1. CreateSession; DestroySSHSession.
2. SSHTunnelDial must fail (TunnelDialErr non-empty).

## Context

- Requirement scenario: tunnel after session destroy → fail.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Scenario = ScenarioTunnelAfterDestroy
	return nil
}
```

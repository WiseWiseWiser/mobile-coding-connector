# Scenario

**Feature**: Wrong bearer token is rejected on CreateSession

```
# auth fail
Client(Token=wrong) -> CreateSSHSession -> CreateErr non-empty; no session_id
```

## Preconditions

- Scenario: `session-create-unauthorized`.
- Manager.RequiredToken = `expected-token`; client Token = `wrong-token`.

## Steps

1. Call CreateSSHSession with mismatched token.
2. Assert CreateErr set; SessionID empty.

## Context

- Covers requirement scenario: unauthorized token → create fails.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Scenario = ScenarioSessionCreateUnauth
	req.ManagerToken = "expected-token"
	req.Token = "wrong-token"
	return nil
}
```

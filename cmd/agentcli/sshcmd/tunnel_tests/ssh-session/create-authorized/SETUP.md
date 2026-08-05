# Scenario

**Feature**: Valid bearer token creates SSH tunnel session

```
# happy create
Client(Token=good) -> CreateSSHSession(public_key) -> session_id non-empty
```

## Preconditions

- Scenario: `session-create-authorized`.
- Token matches Manager.RequiredToken.

## Steps

1. Generate client key; upload OpenSSH public key.
2. Assert SessionID set; CreateErr empty.

## Context

- HostKey may be returned when AdhocServer starts (optional assert soft).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Scenario = ScenarioSessionCreateOK
	req.Token = "test-token"
	req.ManagerToken = "test-token"
	return nil
}
```

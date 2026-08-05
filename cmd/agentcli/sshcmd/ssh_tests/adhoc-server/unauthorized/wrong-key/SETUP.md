# Scenario

**Feature**: Wrong client key is rejected

```
# reject
AdhocServer.SetAuthorizedKeys(good.Public)
ssh.Dial with bad.Signer -> AuthErr non-empty
```

## Preconditions

- Scenario: `adhoc-auth-reject`.

## Steps

1. Generate two key pairs.
2. Authorize only good public key.
3. Dial with bad signer; expect auth failure.

## Context

- No session channel should open.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Scenario = ScenarioAdhocAuthReject
	return nil
}
```

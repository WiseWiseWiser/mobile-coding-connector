# Scenario

**Feature**: AdhocServer — in-process SSH server with public-key auth

```
# authorized client
ClientKeyPair.Public -> AdhocServer.SetAuthorizedKeys
x/crypto client -> AdhocServer.Addr -> session channel (shell | command)

# unauthorized client
wrong Signer -> dial/auth fails
```

## Preconditions

- Scenario family: adhoc-server.
- AdhocServer symbols may be missing until implementer (RED).

## Steps

1. Generate client key pair(s).
2. Start AdhocServer with authorized keys as needed.
3. Dial with x/crypto/ssh client; exercise auth + session.

## Context

- No LocalRelay in this branch; direct dial to Adhoc port.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	_ = req
	return nil
}
```

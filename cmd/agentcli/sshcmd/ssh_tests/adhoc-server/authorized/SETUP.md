# Scenario

**Feature**: AdhocServer accepts authorized public key

```
# right key
GenerateClientKeyPair -> SetAuthorizedKeys(pub)
client{Signer} -> AdhocServer -> auth success -> session
```

## Preconditions

- Auth path: authorized.

## Steps

1. Leaf chooses remote-command vs login-shell after successful auth.

## Context

- User default `"agent"`.

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

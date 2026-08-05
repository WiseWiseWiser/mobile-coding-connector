# Scenario

**Feature**: AdhocServer rejects unauthorized public keys

```
# wrong key
authorized = good.Public
client bad.Signer -> dial/auth fails
```

## Preconditions

- Auth path: unauthorized.

## Steps

1. Leaf dials with a key not in AuthorizedKeys.

## Context

- Distinct key pairs for good vs bad.

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

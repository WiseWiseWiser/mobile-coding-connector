# Scenario

**Feature**: owned ad-hoc visit must not write port-mapping-names

```
Start(owned) -> mapping-names file unchanged
```

## Preconditions

Isolated mapping-names path with optional seed entries.

## Steps

1. Seed mapping file; Start with owned.
2. Assert file contents equal seed.

## Context

Locked: ephemeral random subdomain; no port-mapping-names write.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "visit-mapping"
	req.Port = defaultTestPort
	req.Provider = "owned"
	enableOwnedQuick(req, true, true)
	return nil
}
```

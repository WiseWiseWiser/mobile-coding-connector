# Scenario

**Feature**: corrupt pairs.json causes list to error

```
SeedPairsJSON: not-json\noperator -> list -> error
```

## Preconditions

- SeedPairsJSON is invalid JSON text.

## Steps

1. Seed corrupt file.
2. list; Assert error.

## Context

- Corrupt store.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.SeedPairsJSON = "{not valid json"
	req.Args = []string{"unison", "list"}
	return nil
}
```

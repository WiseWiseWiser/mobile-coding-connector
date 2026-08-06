# Scenario

**Feature**: list on empty store succeeds

```
operator -> RunCLI([unison list]) with empty store -> nil err, no pair names required
```

## Preconditions

- No SeedPairsJSON; fresh StoreDir.

## Steps

1. Args list only.
2. Assert nil error; stdout may say empty or just no names.

## Context

- Empty list path.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Args = []string{"unison", "list"}
	return nil
}
```

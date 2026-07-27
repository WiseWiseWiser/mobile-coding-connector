# Scenario

**Feature**: port list --json

```
seeded listeners -> port list --json -> JSON array
```

## Steps

1. Seed listener; Args `port list --json`.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setCLI(req, "port", "list", "--json")
	seedOneListener(req)
	return nil
}
```

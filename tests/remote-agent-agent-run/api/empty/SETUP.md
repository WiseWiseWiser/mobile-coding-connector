# Scenario

**Feature**: API list with empty store

```
empty home -> GET /api/agent-run/sessions -> 200 {"sessions":[]}
```

## Preconditions

- No session seeds.

## Steps

1. `setAPI`; Seeds empty; omit limit query.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setAPI(req)
	req.Seeds = nil
	req.APILimit = nil
	return nil
}
```
